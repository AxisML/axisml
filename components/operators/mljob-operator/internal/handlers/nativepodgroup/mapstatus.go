package nativepodgroup

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisv1alpha1 "github.com/axisml/axisml/components/operators/mljob-operator/api/v1alpha1"
	axishandler "github.com/axisml/axisml/components/operators/mljob-operator/internal/handler"
)

// MapStatus aggregates Pod phases into the four-state MLJob phase per
// design §8.2. Pure function: only inspects the input.
func (h *Handler) MapStatus(input any) axishandler.MapStatusResult {
	if input == nil {
		return axishandler.MapStatusResult{Phase: axisv1alpha1.PhasePending}
	}
	u, ok := input.(underlying)
	if !ok {
		return axishandler.MapStatusResult{
			Phase:   axisv1alpha1.PhaseFailed,
			Message: "internal: unexpected underlying type for (native, podgroup)",
		}
	}

	res := axishandler.MapStatusResult{
		Phase: axisv1alpha1.PhasePending,
		Roles: []axisv1alpha1.RoleStatus{{Name: roleName, Replicas: u.DesiredReplicas}},
	}

	if len(u.Pods) == 0 {
		// PodGroup may exist but no Pods yet (gang waiting on resources).
		return res
	}

	var (
		active, ready, succeeded, failed int32
		anyRunning                       bool
		earliestStart                    *metav1.Time
		latestFinish                     *metav1.Time
	)
	for i := range u.Pods {
		p := &u.Pods[i]
		switch p.Status.Phase {
		case corev1.PodRunning:
			active++
			anyRunning = true
			if isPodReady(p) {
				ready++
			}
			if p.Status.StartTime != nil && (earliestStart == nil || p.Status.StartTime.Before(earliestStart)) {
				earliestStart = p.Status.StartTime
			}
		case corev1.PodSucceeded:
			succeeded++
			if t := lastTerminationTime(p); t != nil && (latestFinish == nil || t.After(latestFinish.Time)) {
				latestFinish = &metav1.Time{Time: *t}
			}
		case corev1.PodFailed:
			failed++
			if t := lastTerminationTime(p); t != nil && (latestFinish == nil || t.After(latestFinish.Time)) {
				latestFinish = &metav1.Time{Time: *t}
			}
		}
		// PodPending / PodUnknown are intentionally not counted: Active is
		// reserved for Pods that have actually started. Their implicit
		// count is DesiredReplicas - (active + succeeded + failed).
	}
	res.Roles[0].ActiveReplicas = active
	res.Roles[0].ReadyReplicas = ready
	res.Roles[0].SucceededReplicas = succeeded
	res.Roles[0].FailedReplicas = failed
	res.StartedAt = earliestStart

	switch {
	case failed > 0:
		res.Phase = axisv1alpha1.PhaseFailed
		res.Message = "at least one Pod failed"
		res.FinishedAt = latestFinish
	case succeeded == u.DesiredReplicas && u.DesiredReplicas > 0:
		res.Phase = axisv1alpha1.PhaseSucceeded
		res.FinishedAt = latestFinish
	case anyRunning:
		res.Phase = axisv1alpha1.PhaseRunning
	}
	return res
}

func isPodReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// lastTerminationTime returns the latest container termination time
// across init and regular containers — that's the moment the Pod as a
// whole stopped doing work. Walking only ContainerStatuses misses Pods
// that failed in init phase (no main containers ever started); returning
// the first hit instead of the latest understates the timestamp on
// multi-container Pods.
func lastTerminationTime(p *corev1.Pod) *time.Time {
	var latest *time.Time
	consider := func(statuses []corev1.ContainerStatus) {
		for i := range statuses {
			t := statuses[i].State.Terminated
			if t == nil {
				continue
			}
			tm := t.FinishedAt.Time
			if latest == nil || tm.After(*latest) {
				copyTm := tm
				latest = &copyTm
			}
		}
	}
	consider(p.Status.InitContainerStatuses)
	consider(p.Status.ContainerStatuses)
	return latest
}
