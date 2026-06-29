package dispatcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlrun/v1alpha1"
	axishandler "github.com/axisml/axisml/axisml-system/compute-operator/internal/mlrun/handler"
	axislabels "github.com/axisml/axisml/axisml-system/compute-operator/internal/mlrun/labels"
)

// MLRunReconciler is the dispatcher Reconciler. It owns the only write
// path to MLRun.status: handlers return MapStatusResult/ReconcileResult
// values; this Reconciler merges them and Patches.
type MLRunReconciler struct {
	client.Client
	Registry *Registry
}

// Reconcile implements the dispatcher loop described in design §6.
func (r *MLRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("mlrun", req.NamespacedName)

	var mlJob axisv1alpha1.MLRun
	if err := r.Get(ctx, req.NamespacedName, &mlJob); err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted; ownerReference cascade does the rest. The
			// design forbids finalizers, so there's nothing to do.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	backendName := mlJob.Spec.Backend.Name
	backendEngine := mlJob.Spec.Backend.Engine
	h, hasHandler := r.Registry.Lookup(backendName, backendEngine)

	// Terminal-phase precedence (design §4): once status.phase is
	// Succeeded or Failed, do NOT call handler.Reconcile. Owned-resource
	// delete events from TTL GC or manual cleanup would otherwise make
	// the handler observe missing Jobs/Pods and recreate them, rerunning
	// a workload that Compute already considers terminal. We still sync
	// observedGeneration if the user mutated spec post-terminal, and we
	// give handlers an optional Sweep hook for post-terminal cleanup
	// (e.g. TTL GC on backends without native TTL).
	if isTerminal(mlJob.Status.Phase) {
		if mlJob.Status.ObservedGeneration != mlJob.Generation {
			if err := r.writeStatus(ctx, &mlJob,
				axishandler.MapStatusResult{Phase: mlJob.Status.Phase},
				axishandler.ReconcileResult{}); err != nil {
				return ctrl.Result{}, err
			}
		}
		if hasHandler {
			if sweeper, ok := h.(axishandler.Sweeper); ok {
				requeueAfter, err := sweeper.Sweep(ctx, r.Client, &mlJob)
				if err != nil {
					return ctrl.Result{}, err
				}
				if requeueAfter > 0 {
					return ctrl.Result{RequeueAfter: time.Duration(requeueAfter) * time.Second}, nil
				}
			}
		}
		return ctrl.Result{}, nil
	}

	// Spec immutability check (design §3.3 / §10): record a fingerprint
	// of the immutable portion of spec at first observation; reject
	// any subsequent mutation of backend.{name,engine} or role topology
	// (name, replicas) since handlers do not implement scale-down or
	// retargeting.
	if err := r.assertSpecImmutable(ctx, &mlJob); err != nil {
		if statusErr := r.writeStatus(ctx, &mlJob, axishandler.MapStatusResult{
			Phase:   axisv1alpha1.PhaseFailed,
			Message: err.Error(),
		}, axishandler.ReconcileResult{}); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}

	if !hasHandler {
		// Design §5: unregistered tuple → Failed, no underlying resources.
		if err := r.writeStatus(ctx, &mlJob, axishandler.MapStatusResult{
			Phase:   axisv1alpha1.PhaseFailed,
			Message: fmt.Sprintf("no handler for backend=%s engine=%s", backendName, backendEngine),
		}, axishandler.ReconcileResult{}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if errs := h.Validate(&mlJob); len(errs) > 0 {
		if err := r.writeStatus(ctx, &mlJob, axishandler.MapStatusResult{
			Phase:   axisv1alpha1.PhaseFailed,
			Message: errs.ToAggregate().Error(),
		}, axishandler.ReconcileResult{}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	underlying, recRes, err := h.Reconcile(ctx, r.Client, &mlJob)
	if err != nil {
		// Surface the error in the message but keep the Reconcile
		// retry loop running (return non-nil err to controller-runtime).
		phase := mlJob.Status.Phase
		if phase == "" {
			phase = axisv1alpha1.PhasePending
		}
		if statusErr := r.writeStatus(ctx, &mlJob, axishandler.MapStatusResult{
			Phase:   phase,
			Message: fmt.Sprintf("reconcile error: %v", err),
		}, axishandler.ReconcileResult{}); statusErr != nil {
			logger.Error(statusErr, "failed to write status after reconcile error")
		}
		return ctrl.Result{}, err
	}

	mapRes := h.MapStatus(underlying)
	if err := r.writeStatus(ctx, &mlJob, mapRes, recRes); err != nil {
		return ctrl.Result{}, err
	}
	res := ctrl.Result{}
	if recRes.RequeueAfterSeconds > 0 {
		res.RequeueAfter = time.Duration(recRes.RequeueAfterSeconds) * time.Second
	}
	return res, nil
}

// writeStatus is the single status-writer: merges the existing status
// with the handler outputs and persists with a status subresource patch.
// Conflict on resourceVersion is retried by controller-runtime via
// the returned error. Returns nil when the merged status is identical
// to the existing one (no API call issued).
func (r *MLRunReconciler) writeStatus(
	ctx context.Context,
	mlJob *axisv1alpha1.MLRun,
	mapRes axishandler.MapStatusResult,
	recRes axishandler.ReconcileResult,
) error {
	next := mergeStatus(mlJob.Status, mapRes, recRes, mlJob.Generation)
	if statusEqual(mlJob.Status, next) {
		return nil
	}
	patch := client.MergeFrom(mlJob.DeepCopy())
	mlJob.Status = next
	if err := r.Status().Patch(ctx, mlJob, patch); err != nil {
		return fmt.Errorf("patch MLRun status: %w", err)
	}
	return nil
}

// assertSpecImmutable enforces design §3.3: every spec field that a
// handler bakes into the underlying resource at create time is locked
// after first observation. The handlers do not implement update — they
// only patch RunPolicy.Suspend on the live resource — so a mutation to
// e.g. roles[0].template.image would otherwise be silently accepted by
// the dispatcher and ignored by the handler, leaving the user staring
// at a CR whose spec lies about the cluster. RunPolicy.Suspend is the
// one documented mutable field (cancel mechanism) and is excluded from
// the fingerprint.
//
// The first observation is encoded as a deterministic fingerprint and
// stored in an annotation; subsequent mismatches are rejected and
// surfaced via status.message.
func (r *MLRunReconciler) assertSpecImmutable(ctx context.Context, mlJob *axisv1alpha1.MLRun) error {
	want := specFingerprint(mlJob)
	existing := ""
	if mlJob.Annotations != nil {
		existing = mlJob.Annotations[axislabels.AppliedSpecAnnotation]
	}
	if existing == "" {
		// First observation; persist the annotation. We update the CR
		// directly (not status) because annotations live on metadata.
		patch := client.MergeFrom(mlJob.DeepCopy())
		if mlJob.Annotations == nil {
			mlJob.Annotations = map[string]string{}
		}
		mlJob.Annotations[axislabels.AppliedSpecAnnotation] = want
		if err := r.Patch(ctx, mlJob, patch); err != nil {
			return fmt.Errorf("anchor applied-spec annotation: %w", err)
		}
		return nil
	}
	if existing != want {
		return fmt.Errorf("immutable spec fields changed after creation (everything except runPolicy.suspend); fingerprint was %q, attempted %q", existing, want)
	}
	return nil
}

// specFingerprint canonicalises the immutable portion of MLRun.spec
// into a stable string: SHA-256 of the JSON-marshaled snapshot with
// RunPolicy.Suspend zeroed (Suspend is the documented mutable cancel
// mechanism). Using JSON over a typed snapshot keeps the fingerprint
// deterministic — encoding/json sorts map keys and emits struct fields
// in declaration order — without us having to hand-roll serialisation
// for every field we lock.
func specFingerprint(mlJob *axisv1alpha1.MLRun) string {
	snapshot := struct {
		Backend    axisv1alpha1.BackendSpec    `json:"backend"`
		Scheduling axisv1alpha1.SchedulingSpec `json:"scheduling"`
		Roles      []axisv1alpha1.RoleSpec     `json:"roles"`
		RunPolicy  axisv1alpha1.RunPolicySpec  `json:"runPolicy"`
	}{
		Backend:    mlJob.Spec.Backend,
		Scheduling: mlJob.Spec.Scheduling,
		Roles:      mlJob.Spec.Roles,
		RunPolicy:  mlJob.Spec.RunPolicy,
	}
	snapshot.RunPolicy.Suspend = false
	raw, err := json.Marshal(snapshot)
	if err != nil {
		// Fixed types here — Marshal can only fail on a pathological
		// runtime.RawExtension. Surface a sentinel that won't ever match
		// a real fingerprint so the rejection path still fires.
		return "marshal-error:" + err.Error()
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func statusEqual(a, b axisv1alpha1.MLRunStatus) bool {
	return equality.Semantic.DeepEqual(a, b)
}

// SetupWithManager wires the dispatcher to MLRun events and to every
// underlying GVK any registered handler watches. ownerReference reverse
// lookup (Owns) requeues MLRun keys when a derived resource changes.
// Watch targets are deduped by GVK so two handlers claiming the same
// kind don't double-wire informers.
func (r *MLRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(&axisv1alpha1.MLRun{}, builder.WithPredicates(predicate.GenerationChangedPredicate{}))
	seen := map[schema.GroupVersionKind]struct{}{}
	for _, h := range r.Registry.All() {
		for _, target := range h.WatchTargets() {
			gvk, err := apiutil.GVKForObject(target, mgr.GetScheme())
			if err != nil {
				return fmt.Errorf("resolve GVK for watch target: %w", err)
			}
			if _, dup := seen[gvk]; dup {
				continue
			}
			seen[gvk] = struct{}{}
			b = b.Watches(target,
				handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &axisv1alpha1.MLRun{}, handler.OnlyControllerOwner()))
		}
	}
	return b.Complete(r)
}

var _ reconcile.Reconciler = (*MLRunReconciler)(nil)
