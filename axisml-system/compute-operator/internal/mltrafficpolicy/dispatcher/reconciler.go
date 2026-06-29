// Package dispatcher hosts the MLTrafficPolicy Reconciler. It enforces the
// (backend, engine) → Handler routing and the single status-write boundary,
// mirroring the MLService dispatcher (compute-operator.md §4.3 / §5.1).
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

	mltp "github.com/axisml/axisml/axisml-system/compute-operator/api/mltrafficpolicy/v1alpha1"
	hpkg "github.com/axisml/axisml/axisml-system/compute-operator/internal/mltrafficpolicy/handler"
)

// Reconciler is the dispatcher. It looks up the Handler by
// spec.backend.{name, engine}, calls Validate / Reconcile / MapStatus in
// sequence, and merges the StatusUpdate back to the CR status subresource.
//
// Unlike the MLService dispatcher there is no immutable-spec hash baseline:
// compute-service is the sole spec writer and only mutates backends[*].weight
// (and backends[*].role on canary promote). Spec immutability is enforced at
// the compute-service REST layer; an admission webhook is the future hard
// boundary (compute-operator.md §6 防御等级).
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
// MLTrafficPolicy primary resource plus every WatchTarget the active handlers
// declared, mapping child events back to their owning MLTrafficPolicy.
func (r *Reconciler) SetupWithManager(mgr manager.Manager, allHandlers []hpkg.Handler) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(&mltp.MLTrafficPolicy{}, builder.WithPredicates(predicate.GenerationChangedPredicate{}))

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
				&mltp.MLTrafficPolicy{}, handler.OnlyControllerOwner(),
			))
		}
	}
	return b.Complete(r)
}

// Reconcile is the controller-runtime entry point.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("mltrafficpolicy", req.NamespacedName)

	p := &mltp.MLTrafficPolicy{}
	if err := r.client.Get(ctx, req.NamespacedName, p); err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted; ownerReference cascade handles the HTTPRoute.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// compute-service stamps axisml.io/traffic-policy-id on every CR before
	// submission. Handlers use it as the stable child-filtering anchor; an
	// empty value would silently widen the listChildren selector.
	if p.Labels[mltp.LabelTrafficPolicyID] == "" {
		return ctrl.Result{}, r.writeFailedStatus(ctx, p,
			fmt.Sprintf("missing required label %q; compute-service must stamp it before CR creation", mltp.LabelTrafficPolicyID))
	}

	key := hpkg.Key{Backend: p.Spec.Backend.Name, Engine: p.Spec.Backend.Engine}
	h, ok := r.handlers[key]
	if !ok {
		return ctrl.Result{}, r.writeFailedStatus(ctx, p,
			fmt.Sprintf("no handler for backend=%s engine=%s", key.Backend, key.Engine))
	}

	if v := h.Validate(&p.Spec); !v.OK() {
		return ctrl.Result{}, r.writeFailedStatus(ctx, p,
			fmt.Sprintf("validation failed: %s", joinErrors(v.Errors)))
	}

	if _, err := h.Reconcile(ctx, p); err != nil {
		logger.Error(err, "handler reconcile failed")
		_ = r.writeFailedStatus(ctx, p, fmt.Sprintf("reconcile error: %v", err))
		return ctrl.Result{}, err
	}

	children, err := r.listChildren(ctx, p, h)
	if err != nil {
		logger.Error(err, "list children for status")
		return ctrl.Result{}, err
	}

	upd := h.MapStatus(hpkg.Snapshot{Policy: p, Children: children})
	if err := r.writeStatus(ctx, p, upd); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// listChildren retrieves every owned child the handler cares about, filtered
// by axisml.io/traffic-policy-id so MapStatus gets a self-contained snapshot.
func (r *Reconciler) listChildren(ctx context.Context, p *mltp.MLTrafficPolicy, h hpkg.Handler) ([]client.Object, error) {
	id := p.Labels[mltp.LabelTrafficPolicyID]
	if id == "" {
		return nil, nil
	}
	selector := client.MatchingLabels{mltp.LabelTrafficPolicyID: id}
	ns := client.InNamespace(p.Namespace)
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

func (r *Reconciler) writeFailedStatus(ctx context.Context, p *mltp.MLTrafficPolicy, message string) error {
	upd := hpkg.StatusUpdate{
		Phase:   mltp.PhaseFailed,
		Message: message,
		Conditions: []metav1.Condition{{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			Reason:             string(mltp.PhaseFailed),
			Message:            message,
			ObservedGeneration: p.Generation,
		}},
	}
	return r.writeStatus(ctx, p, upd)
}

func (r *Reconciler) writeStatus(ctx context.Context, p *mltp.MLTrafficPolicy, upd hpkg.StatusUpdate) error {
	desired := p.DeepCopy()
	desired.Status.ObservedGeneration = p.Generation
	desired.Status.Phase = upd.Phase
	desired.Status.Message = upd.Message
	desired.Status.Endpoint = upd.Endpoint
	desired.Status.Backends = upd.Backends
	desired.Status.Conditions = mergeConditions(p.Status.Conditions, upd.Conditions, p.Generation)

	if equality.Semantic.DeepEqual(p.Status, desired.Status) {
		return nil
	}
	return r.client.Status().Patch(ctx, desired, client.MergeFrom(p))
}

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
