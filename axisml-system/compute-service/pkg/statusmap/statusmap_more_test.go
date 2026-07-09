package statusmap_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/statusmap"
)

var now = time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

// terminalTime prefers the CR-supplied FinishedAt over now (line 169-171).
func TestMapRun_SucceededPrefersObservedFinishedAt(t *testing.T) {
	finished := metav1.NewTime(now.Add(-5 * time.Minute))
	phase, st := statusmap.MapRun(statusmap.RunRunning, statusmap.RunStatus{},
		mlrunv1alpha1.MLRunStatus{Phase: mlrunv1alpha1.PhaseSucceeded, FinishedAt: &finished}, now)
	assert.Equal(t, statusmap.RunSucceeded, phase)
	assert.Equal(t, finished.Time, *st.FinishedAt)
}

func TestMapRun_FailedPrefersObservedFinishedAt(t *testing.T) {
	finished := metav1.NewTime(now.Add(-time.Minute))
	phase, st := statusmap.MapRun(statusmap.RunRunning, statusmap.RunStatus{},
		mlrunv1alpha1.MLRunStatus{Phase: mlrunv1alpha1.PhaseFailed, FinishedAt: &finished}, now)
	assert.Equal(t, statusmap.RunFailed, phase)
	assert.Equal(t, finished.Time, *st.FinishedAt)
}

// hasCondition returns false when no matching condition exists (line 181): the
// Canceling phase must be held until a real cancel-confirmed condition arrives.
func TestMapRun_CancelingHeldWithoutMatchingCondition(t *testing.T) {
	tests := []struct {
		name  string
		conds []metav1.Condition
	}{
		{name: "no conditions", conds: nil},
		{
			name: "wrong reason",
			conds: []metav1.Condition{{
				Type:   mlrunv1alpha1.ConditionSuspended,
				Status: metav1.ConditionTrue,
				Reason: "SomethingElse",
			}},
		},
		{
			name: "matching type+reason but status false",
			conds: []metav1.Condition{{
				Type:   mlrunv1alpha1.ConditionSuspended,
				Status: metav1.ConditionFalse,
				Reason: mlrunv1alpha1.ReasonCancelRequested,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, st := statusmap.MapRun(statusmap.RunCanceling, statusmap.RunStatus{},
				mlrunv1alpha1.MLRunStatus{Conditions: tt.conds}, now)
			assert.Equal(t, statusmap.RunCanceling, phase, "still canceling")
			assert.Nil(t, st.FinishedAt)
		})
	}
}

// MapService: ready==0 with an observed Pending CR phase → Pending (line 120-121).
func TestMapService_ZeroReadyPendingPhase(t *testing.T) {
	phase, st := statusmap.MapService(statusmap.ServiceReady, statusmap.ServiceStatus{}, 2,
		mlservicev1alpha1.MLServiceStatus{ReadyReplicas: 0, Phase: mlservicev1alpha1.PhasePending})
	assert.Equal(t, statusmap.ServicePending, phase)
	assert.Equal(t, int32(0), st.ReadyReplicas)
}

// MapTraffic: every CR phase maps 1:1 onto the PG phase (lines 144-151).
func TestMapTraffic_PhaseMapping(t *testing.T) {
	tests := []struct {
		crPhase   mltp.Phase
		wantPhase string
	}{
		{mltp.PhasePending, statusmap.ServicePending},
		{mltp.PhaseReady, statusmap.ServiceReady},
		{mltp.PhaseDegraded, statusmap.ServiceDegraded},
		{mltp.PhaseFailed, statusmap.ServiceFailed},
	}
	for _, tt := range tests {
		t.Run(string(tt.crPhase), func(t *testing.T) {
			phase, st := statusmap.MapTraffic("whatever", statusmap.TrafficStatus{},
				mltp.MLTrafficPolicyStatus{Phase: tt.crPhase, Message: "m", Endpoint: "/e"})
			assert.Equal(t, tt.wantPhase, phase)
			assert.Equal(t, "m", st.Message)
			assert.Equal(t, "/e", st.Endpoint)
		})
	}
}

// MapTraffic reuses the existing backing array (next.Backends[:0]) and rewrites
// it from the observed backends.
func TestMapTraffic_BackendsReset(t *testing.T) {
	cur := statusmap.TrafficStatus{Backends: []statusmap.TrafficBackend{
		{ServiceName: "old", Weight: 100, Ready: true},
	}}
	_, st := statusmap.MapTraffic(statusmap.ServiceReady, cur, mltp.MLTrafficPolicyStatus{
		Phase: mltp.PhaseReady,
		Backends: []mltp.BackendStatus{
			{ServiceName: "a", Weight: 60, Ready: true},
			{ServiceName: "b", Weight: 40, Ready: false},
		},
	})
	assert.Len(t, st.Backends, 2)
	assert.Equal(t, "a", st.Backends[0].ServiceName)
	assert.Equal(t, int32(40), st.Backends[1].Weight)
	assert.False(t, st.Backends[1].Ready)
}

// An unknown observed CR phase leaves the current phase unchanged.
func TestMapTraffic_UnknownPhaseKeepsCurrent(t *testing.T) {
	phase, _ := statusmap.MapTraffic(statusmap.ServiceReady, statusmap.TrafficStatus{},
		mltp.MLTrafficPolicyStatus{Phase: mltp.Phase("Bogus")})
	assert.Equal(t, statusmap.ServiceReady, phase)
}
