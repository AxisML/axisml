// Package dispatcher hosts the MLService Reconciler. It enforces the
// (backend, engine) → Handler routing, immutability rules, and the single
// status-write boundary required by mlservice-operator.md §2 / §6 / §7.
package dispatcher

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	axisml "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	hpkg "github.com/axisml/axisml/axisml-system/compute-operator/internal/mlservice/handler"
)

// Reconciler is the dispatcher. It owns no Handler-specific knowledge: it
// looks up the Handler by spec.backend.{name, engine}, calls Validate /
// Reconcile / MapStatus in sequence, and merges the StatusUpdate back to the
// CR via the status subresource.
type Reconciler struct {
	client   client.Client
	scheme   *runtime.Scheme
	handlers map[hpkg.Key]hpkg.Handler
}

// NewReconciler wires the dispatcher with the registered handlers.
func NewReconciler(mgr manager.Manager, handlers map[hpkg.Key]hpkg.Handler) *Reconciler {
	return &Reconciler{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		handlers: handlers,
	}
}

// SetupWithManager registers the controller with the manager. It watches the
// MLService primary resource plus every WatchTarget the active handlers
// declared, mapping child events back to their owning MLService.
//
// The primary watch uses GenerationChangedPredicate so that operator-driven
// metadata patches (e.g. stamping the immutable-spec hash annotation) do not
// re-enqueue the CR. Status writes already go through Status().Patch and do
// not bump generation. Child watches are unaffected — they enqueue via owner
// reference regardless of predicate.
func (r *Reconciler) SetupWithManager(mgr manager.Manager, allHandlers []hpkg.Handler) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(&axisml.MLService{}, builder.WithPredicates(predicate.GenerationChangedPredicate{}))

	seen := map[schema.GroupVersionKind]struct{}{}
	for _, h := range allHandlers {
		for _, obj := range h.WatchTargets() {
			gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
			if err != nil {
				return fmt.Errorf("resolve GVK for watch target: %w", err)
			}
			if _, ok := seen[gvk]; ok {
				continue
			}
			seen[gvk] = struct{}{}
			b = b.Watches(obj, handler.EnqueueRequestForOwner(
				mgr.GetScheme(), mgr.GetRESTMapper(),
				&axisml.MLService{}, handler.OnlyControllerOwner(),
			))
		}
	}
	return b.Complete(r)
}

// Reconcile is the controller-runtime entry point.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("mlservice", req.NamespacedName)

	mls := &axisml.MLService{}
	if err := r.client.Get(ctx, req.NamespacedName, mls); err != nil {
		if apierrors.IsNotFound(err) {
			// CR was deleted; ownerReference cascade handles children.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Compute is required to stamp compute.axisml.io/service-id on every CR before
	// submission (mlservice-operator.md §3.1). Handlers use this label as the
	// stable selector / orphan-detection anchor; an empty value would produce
	// a Deployment selector with "" as the value, and Deployment selectors
	// are immutable — so the next reconcile (after the label finally lands)
	// would fail the upsert forever. Fail fast and explicitly instead.
	if mls.Labels[axisml.LabelServiceID] == "" {
		return ctrl.Result{}, r.writeFailedStatus(ctx, mls,
			fmt.Sprintf("missing required label %q; Compute must stamp it before CR creation", axisml.LabelServiceID))
	}

	serviceKind := mls.Labels[axisml.LabelServiceKind]
	if mls.Spec.Route != nil && mls.Spec.Route.Enabled &&
		(serviceKind == axisml.ServiceKindWorkspace || serviceKind == axisml.ServiceKindTensorBoard) {
		return ctrl.Result{}, r.writeFailedStatus(ctx, mls,
			fmt.Sprintf("%s external routes are disabled until SecurityPolicy derivation is implemented", serviceKind))
	}

	key := hpkg.Key{Backend: mls.Spec.Backend.Name, Engine: mls.Spec.Backend.Engine}
	h, ok := r.handlers[key]
	if !ok {
		return ctrl.Result{}, r.writeFailedStatus(ctx, mls,
			fmt.Sprintf("no handler for backend=%s engine=%s", key.Backend, key.Engine))
	}

	// Immutable-field enforcement per §6: detect mutations to anything other
	// than spec.roles[*].replicas. The baseline is recorded only after a
	// fully successful reconcile (see stampImmutabilityBaseline below) so
	// that an initially invalid spec stays editable.
	if msg, err := r.checkImmutability(mls); err != nil {
		return ctrl.Result{}, err
	} else if msg != "" {
		return ctrl.Result{}, r.writeFailedStatus(ctx, mls, msg)
	}

	// Pure-spec validation. Validation errors are terminal — surface as
	// phase=Failed; do not retry.
	if v := h.Validate(&mls.Spec); !v.OK() {
		return ctrl.Result{}, r.writeFailedStatus(ctx, mls,
			fmt.Sprintf("validation failed: %s", joinErrors(v.Errors)))
	}

	if _, err := h.Reconcile(ctx, mls); err != nil {
		logger.Error(err, "handler reconcile failed")
		// Surface the failure but allow controller-runtime to retry.
		_ = r.writeFailedStatus(ctx, mls, fmt.Sprintf("reconcile error: %v", err))
		return ctrl.Result{}, err
	}

	// Build the snapshot for MapStatus. We keep this minimal — list children
	// by compute.axisml.io/service-id label across each handler's declared GVKs.
	children, err := r.listChildren(ctx, mls, h)
	if err != nil {
		logger.Error(err, "list children for status")
		return ctrl.Result{}, err
	}

	upd := h.MapStatus(hpkg.Snapshot{Service: mls, Children: children})
	if err := r.writeStatus(ctx, mls, upd); err != nil {
		return ctrl.Result{}, err
	}

	// Validate + Reconcile + status write all succeeded — safe to lock the
	// immutable baseline now. Patching annotations here will trigger another
	// reconcile, on which checkImmutability becomes a true comparison.
	if err := r.stampImmutabilityBaseline(ctx, mls); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// listChildren retrieves every owned child object the handler cares about,
// filtered by compute.axisml.io/service-id so we don't paginate the whole namespace.
// MapStatus requires the snapshot be self-contained — no dispatcher state may
// leak into the pure function.
func (r *Reconciler) listChildren(ctx context.Context, mls *axisml.MLService, h hpkg.Handler) ([]client.Object, error) {
	serviceID := mls.Labels[axisml.LabelServiceID]
	if serviceID == "" {
		return nil, nil
	}
	selector := client.MatchingLabels{axisml.LabelServiceID: serviceID}
	ns := client.InNamespace(mls.Namespace)
	var children []client.Object
	for _, proto := range h.WatchTargets() {
		listObj, err := newListFor(proto, r.scheme)
		if err != nil {
			return nil, err
		}
		if err := r.client.List(ctx, listObj, ns, selector); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		items, err := extractItems(listObj)
		if err != nil {
			return nil, err
		}
		children = append(children, items...)
	}
	return children, nil
}

// writeFailedStatus is the dispatcher-level status writer used for
// pre-handler failures (unknown backend, immutable field changes,
// validation errors).
func (r *Reconciler) writeFailedStatus(ctx context.Context, mls *axisml.MLService, message string) error {
	upd := hpkg.StatusUpdate{
		Phase:   axisml.PhaseFailed,
		Message: message,
		Conditions: []metav1.Condition{{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			Reason:             string(axisml.PhaseFailed),
			Message:            message,
			ObservedGeneration: mls.Generation,
		}},
	}
	return r.writeStatus(ctx, mls, upd)
}

// writeStatus merges the StatusUpdate into the CR via the status subresource.
// resourceVersion conflict retries are handled by controller-runtime via
// requeue; we keep this idempotent by skipping no-op updates.
func (r *Reconciler) writeStatus(ctx context.Context, mls *axisml.MLService, upd hpkg.StatusUpdate) error {
	desired := mls.DeepCopy()
	desired.Status.ObservedGeneration = mls.Generation
	desired.Status.Phase = upd.Phase
	desired.Status.Message = upd.Message
	desired.Status.Endpoint = upd.Endpoint
	desired.Status.ReadyReplicas = upd.ReadyReplicas
	desired.Status.Selector = upd.Selector
	desired.Status.Roles = upd.Roles
	desired.Status.Conditions = mergeConditions(mls.Status.Conditions, upd.Conditions, mls.Generation)

	if equality.Semantic.DeepEqual(mls.Status, desired.Status) {
		return nil
	}
	return r.client.Status().Patch(ctx, desired, client.MergeFrom(mls))
}

// mergeConditions de-dupes by Type, preserving lastTransitionTime when the
// status didn't actually change. controller-runtime's status patch leaves
// untouched fields alone, so we only need to compute the desired set.
func mergeConditions(existing, updates []metav1.Condition, generation int64) []metav1.Condition {
	byType := map[string]metav1.Condition{}
	for _, c := range existing {
		byType[c.Type] = c
	}
	for _, u := range updates {
		prev, hadPrev := byType[u.Type]
		if !hadPrev || prev.Status != u.Status {
			u.LastTransitionTime = metav1.Now()
		} else {
			u.LastTransitionTime = prev.LastTransitionTime
		}
		u.ObservedGeneration = generation
		byType[u.Type] = u
	}
	out := make([]metav1.Condition, 0, len(byType))
	for _, c := range byType {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

func joinErrors(errs []string) string {
	out := ""
	for i, e := range errs {
		if i > 0 {
			out += "; "
		}
		out += e
	}
	return out
}
