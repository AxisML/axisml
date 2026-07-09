package dispatcher

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	"github.com/axisml/axisml/axisml-system/compute-operator/internal/mlrun/handler"
)

func TestMergeStatus_TerminalPhasePrecedence(t *testing.T) {
	// Case A: existing already terminal; incoming non-terminal +
	// suspend signal must NOT roll back the phase, and Suspended must
	// not be written (terminal-phase precedence — design §4 last paragraph).
	existing := axisv1alpha1.MLRunStatus{Phase: axisv1alpha1.PhaseSucceeded}
	got := mergeStatus(existing,
		handler.MapStatusResult{Phase: axisv1alpha1.PhasePending, Message: "stale"},
		handler.ReconcileResult{SuspendCompleted: true, SuspendReason: "CancelRequested"},
		1,
	)
	if got.Phase != axisv1alpha1.PhaseSucceeded {
		t.Fatalf("phase regressed from terminal: got %q", got.Phase)
	}
	for _, c := range got.Conditions {
		if c.Type == axisv1alpha1.ConditionSuspended {
			t.Fatalf("Suspended condition must not be written when already terminal")
		}
	}
}

func TestMergeStatus_FreshTerminalAccepted(t *testing.T) {
	// Case B: existing is Running, MapStatus reports Failed → accept.
	existing := axisv1alpha1.MLRunStatus{Phase: axisv1alpha1.PhaseRunning}
	now := metav1.Now()
	got := mergeStatus(existing,
		handler.MapStatusResult{Phase: axisv1alpha1.PhaseFailed, Message: "pod OOM", FinishedAt: &now},
		handler.ReconcileResult{},
		2,
	)
	if got.Phase != axisv1alpha1.PhaseFailed {
		t.Fatalf("expected Failed, got %q", got.Phase)
	}
	if got.FinishedAt == nil {
		t.Fatalf("FinishedAt must propagate")
	}
}

func TestMergeStatus_SuspendCondition(t *testing.T) {
	// Case C: non-terminal + SuspendCompleted → Suspended condition written
	// with reason CancelRequested; phase stays Pending (design §4 cancel signal).
	existing := axisv1alpha1.MLRunStatus{Phase: axisv1alpha1.PhasePending}
	got := mergeStatus(existing,
		handler.MapStatusResult{Phase: axisv1alpha1.PhasePending},
		handler.ReconcileResult{SuspendCompleted: true},
		1,
	)
	if got.Phase != axisv1alpha1.PhasePending {
		t.Fatalf("phase must stay Pending during suspend, got %q", got.Phase)
	}
	var found bool
	for _, c := range got.Conditions {
		if c.Type == axisv1alpha1.ConditionSuspended {
			found = true
			if c.Status != metav1.ConditionTrue {
				t.Fatalf("Suspended status: want True, got %q", c.Status)
			}
			if c.Reason != axisv1alpha1.ReasonCancelRequested {
				t.Fatalf("Suspended reason: want CancelRequested, got %q", c.Reason)
			}
		}
	}
	if !found {
		t.Fatalf("Suspended condition missing")
	}
}

func TestMergeStatus_ConditionMergeByType(t *testing.T) {
	old := axisv1alpha1.MLRunStatus{
		Phase: axisv1alpha1.PhaseRunning,
		Conditions: []metav1.Condition{
			{Type: axisv1alpha1.ConditionInitialized, Status: metav1.ConditionTrue, Reason: "Created"},
		},
	}
	got := mergeStatus(old,
		handler.MapStatusResult{
			Phase: axisv1alpha1.PhaseRunning,
			Conditions: []metav1.Condition{
				{Type: axisv1alpha1.ConditionScheduled, Status: metav1.ConditionTrue, Reason: "Scheduled"},
			},
		},
		handler.ReconcileResult{},
		1,
	)
	if len(got.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(got.Conditions))
	}
}

func TestMergeStatus_LastTransitionTimePreservedOnSameStatus(t *testing.T) {
	// Kubernetes condition convention: LastTransitionTime advances only
	// when Status flips. A re-emitted condition with the same Status
	// must keep the old timestamp so kubectl describe history is honest.
	original := metav1.NewTime(time.Now().Add(-10 * time.Minute))
	old := axisv1alpha1.MLRunStatus{
		Phase: axisv1alpha1.PhaseRunning,
		Conditions: []metav1.Condition{{
			Type:               axisv1alpha1.ConditionScheduled,
			Status:             metav1.ConditionTrue,
			Reason:             "Scheduled",
			LastTransitionTime: original,
		}},
	}
	got := mergeStatus(old,
		handler.MapStatusResult{
			Phase: axisv1alpha1.PhaseRunning,
			Conditions: []metav1.Condition{{
				Type:    axisv1alpha1.ConditionScheduled,
				Status:  metav1.ConditionTrue,
				Reason:  "Scheduled",
				Message: "still scheduled",
			}},
		},
		handler.ReconcileResult{},
		2,
	)
	if len(got.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(got.Conditions))
	}
	if !got.Conditions[0].LastTransitionTime.Equal(&original) {
		t.Fatalf("LastTransitionTime advanced on unchanged Status: was %v, got %v",
			original, got.Conditions[0].LastTransitionTime)
	}
	if got.Conditions[0].Message != "still scheduled" {
		t.Fatalf("Message: want updated, got %q", got.Conditions[0].Message)
	}
}

func TestMergeStatus_LastTransitionTimeAdvancesOnFlip(t *testing.T) {
	original := metav1.NewTime(time.Now().Add(-10 * time.Minute))
	old := axisv1alpha1.MLRunStatus{
		Phase: axisv1alpha1.PhasePending,
		Conditions: []metav1.Condition{{
			Type:               axisv1alpha1.ConditionScheduled,
			Status:             metav1.ConditionFalse,
			Reason:             "Pending",
			LastTransitionTime: original,
		}},
	}
	got := mergeStatus(old,
		handler.MapStatusResult{
			Phase: axisv1alpha1.PhaseRunning,
			Conditions: []metav1.Condition{{
				Type:   axisv1alpha1.ConditionScheduled,
				Status: metav1.ConditionTrue,
				Reason: "Scheduled",
			}},
		},
		handler.ReconcileResult{},
		2,
	)
	if got.Conditions[0].LastTransitionTime.Equal(&original) {
		t.Fatalf("LastTransitionTime must advance when Status flips")
	}
}
