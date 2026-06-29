package handler

import (
	"context"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlrun/v1alpha1"
)

// Key is the (backend, engine) tuple a handler claims as its registry
// primary key. Both fields are case-sensitive and must match the values
// Compute writes into MLRun.spec.backend exactly.
type Key struct {
	Backend string
	Engine  string
}

// Handler is the contract every backend implementation must satisfy
// (design §7). All current backends support native suspend, so the
// cancel path is folded into Reconcile (which observes
// spec.runPolicy.suspend and returns SuspendCompleted=true once the
// underlying resource is paused or torn down). ownerReference cascade
// handles the active-path delete; the optional Sweeper extension
// (types.go) covers post-terminal TTL GC for backends without native
// TTL controllers.
//
//   - Reconcile is idempotent: repeated calls with the same spec produce
//     the same underlying resource, no rebuild.
//   - MapStatus is a pure function: no API calls, no I/O. It runs on
//     watch events to compute the next MLRun.status without round-trips.
type Handler interface {
	Key() Key

	Validate(*axisv1alpha1.MLRun) field.ErrorList

	// Reconcile creates or updates the underlying resource and returns
	// the live object together with structured signals for dispatcher.
	// underlying may be nil when no resource has yet been created.
	Reconcile(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLRun) (underlying any, result ReconcileResult, err error)

	// MapStatus translates the handler-specific underlying object into
	// the unified four-state phase + role aggregation. underlying may
	// be nil; in that case the handler returns Phase=Pending.
	MapStatus(underlying any) MapStatusResult

	// WatchTargets enumerates the GVKs whose events should requeue
	// MLRun reconciles via ownerReference reverse-lookup. Returned
	// objects are empty zero-values, used by controller-runtime as
	// type discriminators.
	WatchTargets() []client.Object
}
