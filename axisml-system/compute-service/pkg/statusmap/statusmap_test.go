package statusmap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

var fixedNow = time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

func TestMapRun_Transitions(t *testing.T) {
	// Creating -> Pending.
	phase, _ := MapRun(RunCreating, RunStatus{}, mlrunv1alpha1.MLRunStatus{Phase: mlrunv1alpha1.PhasePending}, fixedNow)
	assert.Equal(t, RunPending, phase)

	// Pending -> Running stamps startedAt from the CR.
	started := metav1.NewTime(fixedNow.Add(-time.Minute))
	phase, st := MapRun(RunPending, RunStatus{}, mlrunv1alpha1.MLRunStatus{
		Phase: mlrunv1alpha1.PhaseRunning, StartedAt: &started,
	}, fixedNow)
	assert.Equal(t, RunRunning, phase)
	assert.NotNil(t, st.StartedAt)

	// Running -> Succeeded stamps finishedAt (falls back to now when unset).
	phase, st = MapRun(RunRunning, RunStatus{}, mlrunv1alpha1.MLRunStatus{Phase: mlrunv1alpha1.PhaseSucceeded}, fixedNow)
	assert.Equal(t, RunSucceeded, phase)
	assert.Equal(t, fixedNow, *st.FinishedAt)

	// Failed carries the CR message.
	phase, st = MapRun(RunRunning, RunStatus{}, mlrunv1alpha1.MLRunStatus{
		Phase: mlrunv1alpha1.PhaseFailed, Message: "boom",
	}, fixedNow)
	assert.Equal(t, RunFailed, phase)
	assert.Equal(t, "boom", st.Message)

	// Canceling + Suspended(CancelRequested) -> Cancelled.
	phase, st = MapRun(RunCanceling, RunStatus{}, mlrunv1alpha1.MLRunStatus{
		Conditions: []metav1.Condition{{
			Type:   mlrunv1alpha1.ConditionSuspended,
			Status: metav1.ConditionTrue,
			Reason: mlrunv1alpha1.ReasonCancelRequested,
		}},
	}, fixedNow)
	assert.Equal(t, RunCancelled, phase)
	assert.Equal(t, fixedNow, *st.FinishedAt)
}

func TestMapService_Transitions(t *testing.T) {
	// desired == 0 -> Pending.
	phase, _ := MapService(ServiceReady, ServiceStatus{}, 0, mlservicev1alpha1.MLServiceStatus{})
	assert.Equal(t, ServicePending, phase)

	// ready == desired -> Ready, endpoint reflected.
	phase, st := MapService(ServicePending, ServiceStatus{}, 2, mlservicev1alpha1.MLServiceStatus{
		ReadyReplicas: 2, Endpoint: "svc:8080",
	})
	assert.Equal(t, ServiceReady, phase)
	assert.Equal(t, int32(2), st.ReadyReplicas)
	assert.Equal(t, "svc:8080", st.Endpoint)

	// 0 < ready < desired -> Degraded.
	phase, _ = MapService(ServiceReady, ServiceStatus{}, 3, mlservicev1alpha1.MLServiceStatus{ReadyReplicas: 1})
	assert.Equal(t, ServiceDegraded, phase)

	// ready == 0 && CR Failed -> Failed with message.
	phase, st = MapService(ServicePending, ServiceStatus{}, 2, mlservicev1alpha1.MLServiceStatus{
		ReadyReplicas: 0, Phase: mlservicev1alpha1.PhaseFailed, Message: "crashloop",
	})
	assert.Equal(t, ServiceFailed, phase)
	assert.Equal(t, "crashloop", st.Message)
}

func TestMapTraffic_Transitions(t *testing.T) {
	phase, st := MapTraffic(ServicePending, TrafficStatus{}, mltp.MLTrafficPolicyStatus{
		Phase:    mltp.PhaseReady,
		Endpoint: "/svc/x",
		Backends: []mltp.BackendStatus{{ServiceName: "a", Weight: 70, Ready: true}},
	})
	assert.Equal(t, ServiceReady, phase)
	assert.Equal(t, "/svc/x", st.Endpoint)
	assert.Len(t, st.Backends, 1)
	assert.Equal(t, "a", st.Backends[0].ServiceName)
	assert.Equal(t, int32(70), st.Backends[0].Weight)
}
