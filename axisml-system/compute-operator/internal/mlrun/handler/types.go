package handler

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlrun/v1alpha1"
)

// MapStatusResult is the pure output a handler emits from observing the
// underlying resource. dispatcher merges it into MLRun.status.
type MapStatusResult struct {
	Phase      axisv1alpha1.MLRunPhase
	Message    string
	StartedAt  *metav1.Time
	FinishedAt *metav1.Time
	// Conditions are merged by type into the existing status.conditions[].
	Conditions []metav1.Condition
	// Roles is the per-role replica aggregation.
	Roles []axisv1alpha1.RoleStatus
}

// IsTerminal returns true when the phase indicates the job has stopped
// for good. Used by dispatcher to enforce "terminal-phase precedence":
// once terminal, suspend signals must not roll the phase back.
func (r MapStatusResult) IsTerminal() bool {
	switch r.Phase {
	case axisv1alpha1.PhaseSucceeded, axisv1alpha1.PhaseFailed:
		return true
	}
	return false
}

// Sweeper is an optional Handler extension invoked by dispatcher when
// the MLRun is already in a terminal phase. It exists because
// dispatcher's terminal-phase short-circuit skips Reconcile entirely
// (to avoid recreating GC'd resources), but some backends still need
// post-terminal work — e.g. TTL GC for handlers without a native
// TTLSecondsAfterFinished equivalent.
//
// Returning requeueAfterSeconds > 0 schedules another Sweep call
// (typically the time remaining until TTL expiry).
type Sweeper interface {
	Sweep(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLRun) (requeueAfterSeconds int32, err error)
}

// ReconcileResult is the structured signal a handler returns from
// Reconcile to feed dispatcher's status merge. It must NOT be filled
// when the underlying resource has already reached a terminal state;
// in that case the handler returns the terminal MapStatusResult and
// leaves SuspendCompleted false (terminal-phase precedence, design §4).
type ReconcileResult struct {
	// SuspendCompleted is true once the handler has fully paused or
	// torn down the underlying resource on the cancel path.
	SuspendCompleted bool
	// SuspendReason is the reason placed on the Suspended condition.
	// Required when SuspendCompleted is true; conventionally
	// axisv1alpha1.ReasonCancelRequested.
	SuspendReason string
	// Warnings are surfaced as status.message suffix when phase is
	// otherwise unchanged (e.g. unknown backend.config keys).
	Warnings []string
	// RequeueAfter, if positive, schedules a follow-up reconcile.
	RequeueAfterSeconds int32
}
