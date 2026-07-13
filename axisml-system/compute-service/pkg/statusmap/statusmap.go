// Package statusmap publishes the shared CR-Status → PG (phase, status)
// mapping — the Compute domain state machine. Both runtime forms feed observed
// CR status through these pure functions: the Kubernetes form from an apiserver
// informer, the Lite form from a runtime Observe poll (design §4.2 / §9.1). The
// functions take the current PG phase + standardized status and the observed CR
// status and return the next phase + status; they never touch the database.
//
// Phase string constants are the canonical Job / Service / TrafficPolicy state
// machine; the PG-side status structs mirror the standardized status fields the
// mapping reads and writes (callers convert to/from their persistence DTOs).
package statusmap

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"
)

// Run phases — the Job state machine (design §4.1).
const (
	RunCreating  = "Creating"
	RunPending   = "Pending"
	RunRunning   = "Running"
	RunSucceeded = "Succeeded"
	RunFailed    = "Failed"
	RunCanceling = "Canceling"
	RunCancelled = "Cancelled"
)

// Service / TrafficPolicy phases — the Service state machine (design §4.2/§4.3).
const (
	ServicePending  = "Pending"
	ServiceReady    = "Ready"
	ServiceDegraded = "Degraded"
	ServiceFailed   = "Failed"
)

// RunStatus mirrors the standardized PG status fields for a Run.
type RunStatus struct {
	Message    string
	StartedAt  *time.Time
	FinishedAt *time.Time
}

// ServiceStatus mirrors the standardized PG status fields for a Service.
type ServiceStatus struct {
	Message       string
	ReadyReplicas int32
	Endpoint      string
}

// TrafficBackend mirrors one entry of a TrafficPolicy's status.backends[].
type TrafficBackend struct {
	ServiceName string
	Weight      int32
	Ready       bool
}

// TrafficStatus mirrors the standardized PG status fields for a TrafficPolicy.
type TrafficStatus struct {
	Message  string
	Endpoint string
	Backends []TrafficBackend
}

// MapRun reflects an observed MLRun CR status onto the next PG (phase, status),
// given the current PG phase + status. now stamps the finishedAt for a
// cancel-confirmed transition.
func MapRun(curPhase string, cur RunStatus, observed mlrunv1alpha1.MLRunStatus, now time.Time) (string, RunStatus) {
	phase := curPhase
	next := cur

	switch observed.Phase {
	case mlrunv1alpha1.PhasePending:
		if curPhase == RunCreating {
			phase = RunPending
		}
	case mlrunv1alpha1.PhaseRunning:
		if curPhase == RunCreating || curPhase == RunPending {
			phase = RunRunning
		}
		if cur.StartedAt == nil && observed.StartedAt != nil {
			t := observed.StartedAt.Time
			next.StartedAt = &t
		}
		// Clear any placement message (e.g. "waiting for GPU") now that the Run
		// is actually running.
		next.Message = ""
	case mlrunv1alpha1.PhaseSucceeded:
		phase = RunSucceeded
		t := terminalTime(observed.FinishedAt, now)
		next.FinishedAt = &t
	case mlrunv1alpha1.PhaseFailed:
		phase = RunFailed
		t := terminalTime(observed.FinishedAt, now)
		next.FinishedAt = &t
		if observed.Message != "" {
			next.Message = observed.Message
		}
	}

	// Suspended condition with reason=CancelRequested confirms a cancel.
	if curPhase == RunCanceling &&
		hasCondition(observed.Conditions, mlrunv1alpha1.ConditionSuspended, mlrunv1alpha1.ReasonCancelRequested) {
		phase = RunCancelled
		t := now
		next.FinishedAt = &t
	}
	return phase, next
}

// MapService reflects an observed MLService CR status onto the next PG (phase,
// status). desiredReplicas is the spec's role[0] replica count.
func MapService(curPhase string, cur ServiceStatus, desiredReplicas int32, observed mlservicev1alpha1.MLServiceStatus) (string, ServiceStatus) {
	phase := curPhase
	switch {
	case desiredReplicas == 0:
		phase = ServicePending
	case observed.ReadyReplicas == 0 && observed.Phase == mlservicev1alpha1.PhasePending:
		phase = ServicePending
	case observed.ReadyReplicas == desiredReplicas:
		phase = ServiceReady
	case observed.ReadyReplicas > 0 && observed.ReadyReplicas < desiredReplicas:
		phase = ServiceDegraded
	case observed.ReadyReplicas == 0 && observed.Phase == mlservicev1alpha1.PhaseFailed:
		phase = ServiceFailed
	}

	next := cur
	next.ReadyReplicas = observed.ReadyReplicas
	next.Endpoint = observed.Endpoint
	if observed.Phase == mlservicev1alpha1.PhaseFailed && observed.Message != "" {
		next.Message = observed.Message
	}
	if phase == ServiceReady {
		// Clear any placement message (e.g. "waiting for GPU") now that the
		// Service is ready.
		next.Message = ""
	}
	return phase, next
}

// MapTraffic reflects an observed MLTrafficPolicy CR status onto the next PG
// (phase, status). The CR phase maps 1:1 onto the PG phase.
func MapTraffic(curPhase string, cur TrafficStatus, observed mltp.MLTrafficPolicyStatus) (string, TrafficStatus) {
	phase := curPhase
	switch observed.Phase {
	case mltp.PhasePending:
		phase = ServicePending
	case mltp.PhaseReady:
		phase = ServiceReady
	case mltp.PhaseDegraded:
		phase = ServiceDegraded
	case mltp.PhaseFailed:
		phase = ServiceFailed
	}

	next := cur
	next.Endpoint = observed.Endpoint
	next.Message = observed.Message
	next.Backends = next.Backends[:0]
	for _, b := range observed.Backends {
		next.Backends = append(next.Backends, TrafficBackend{
			ServiceName: b.ServiceName,
			Weight:      b.Weight,
			Ready:       b.Ready,
		})
	}
	return phase, next
}

func terminalTime(finishedAt *metav1.Time, now time.Time) time.Time {
	if finishedAt != nil {
		return finishedAt.Time
	}
	return now
}

func hasCondition(conds []metav1.Condition, kind, reason string) bool {
	for _, c := range conds {
		if c.Type == kind && c.Reason == reason && c.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}
