package dispatcher

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisv1alpha1 "axisml.io/operators/mljob/api/v1alpha1"
	"axisml.io/operators/mljob/internal/handler"
)

// mergeStatus computes the next MLJobStatus from the existing status,
// the handler's MapStatus output, and the handler's Reconcile signal.
// It is a pure function; the dispatcher only needs to feed it inputs
// and persist the returned value via Status().Patch.
//
// Two design invariants drive the logic:
//
//  1. Terminal-phase precedence (design §4): once status.phase is
//     Succeeded/Failed (either previously, or freshly observed), the
//     phase must NOT regress to Pending — even if a cancel signal
//     arrives. The Suspended condition is also suppressed in that case
//     because there's nothing to cancel.
//
//  2. Single-writer status: handlers never patch status directly. Their
//     observed phase (MapStatus) and structured suspend signal
//     (ReconcileResult) are merged here, then handed back to dispatcher
//     for one Patch call.
func mergeStatus(
	existing axisv1alpha1.MLJobStatus,
	mapRes handler.MapStatusResult,
	recRes handler.ReconcileResult,
	generation int64,
) axisv1alpha1.MLJobStatus {
	next := existing
	next.ObservedGeneration = generation

	// Terminal-phase precedence: keep the prior terminal phase across
	// late-arriving non-terminal mapRes events.
	if !isTerminal(existing.Phase) {
		next.Phase = mapRes.Phase
	}

	if mapRes.Message != "" {
		next.Message = mapRes.Message
	}
	if len(recRes.Warnings) > 0 {
		w := strings.Join(recRes.Warnings, "; ")
		if next.Message == "" {
			next.Message = w
		} else {
			next.Message = next.Message + " | warnings: " + w
		}
	}

	if mapRes.StartedAt != nil && (next.StartedAt == nil || mapRes.StartedAt.Before(next.StartedAt)) {
		next.StartedAt = mapRes.StartedAt
	}
	if mapRes.FinishedAt != nil {
		next.FinishedAt = mapRes.FinishedAt
	}

	if mapRes.Roles != nil {
		next.Roles = mapRes.Roles
	}

	// Conditions: meta.SetStatusCondition encodes the Kubernetes
	// convention (LastTransitionTime preserved when Status is unchanged,
	// stamped now() on a flip). It also handles the empty-slice case.
	for _, c := range mapRes.Conditions {
		meta.SetStatusCondition(&next.Conditions, c)
	}

	// Cancel signal: terminal-phase precedence rules out a successful
	// cancel signal because the work is already finished.
	if recRes.SuspendCompleted && !isTerminal(next.Phase) {
		reason := recRes.SuspendReason
		if reason == "" {
			reason = axisv1alpha1.ReasonCancelRequested
		}
		meta.SetStatusCondition(&next.Conditions, metav1.Condition{
			Type:    axisv1alpha1.ConditionSuspended,
			Status:  metav1.ConditionTrue,
			Reason:  reason,
			Message: "Handler completed suspend; underlying resources paused or torn down",
		})
	}

	return next
}

func isTerminal(p axisv1alpha1.MLJobPhase) bool {
	return p == axisv1alpha1.PhaseSucceeded || p == axisv1alpha1.PhaseFailed
}
