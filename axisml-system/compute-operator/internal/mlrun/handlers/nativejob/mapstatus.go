package nativejob

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	axishandler "github.com/axisml/axisml/axisml-system/compute-operator/internal/mlrun/handler"
)

// MapStatus is a pure function: given the live batch/v1.Job, derive the
// MLRun phase per design §8.1. Pure means no Get/Watch calls — only the
// fields on the input.
func (h *Handler) MapStatus(underlying any) axishandler.MapStatusResult {
	if underlying == nil {
		return axishandler.MapStatusResult{Phase: axisv1alpha1.PhasePending}
	}
	job, ok := underlying.(*batchv1.Job)
	if !ok {
		return axishandler.MapStatusResult{
			Phase:   axisv1alpha1.PhaseFailed,
			Message: "internal: unexpected underlying type for (native, job)",
		}
	}

	res := axishandler.MapStatusResult{
		Phase:      axisv1alpha1.PhasePending,
		StartedAt:  job.Status.StartTime,
		FinishedAt: job.Status.CompletionTime,
		Roles: []axisv1alpha1.RoleStatus{{
			Name:              roleName,
			Replicas:          deref32(job.Spec.Parallelism),
			ActiveReplicas:    job.Status.Active,
			SucceededReplicas: job.Status.Succeeded,
			FailedReplicas:    job.Status.Failed,
		}},
	}

	for _, c := range job.Status.Conditions {
		if c.Status != corev1.ConditionTrue {
			continue
		}
		switch c.Type {
		case batchv1.JobComplete:
			res.Phase = axisv1alpha1.PhaseSucceeded
			if res.FinishedAt == nil {
				res.FinishedAt = &c.LastTransitionTime
			}
			return res
		case batchv1.JobFailed:
			res.Phase = axisv1alpha1.PhaseFailed
			res.Message = c.Message
			if res.FinishedAt == nil {
				res.FinishedAt = &c.LastTransitionTime
			}
			return res
		}
	}

	if job.Status.Active > 0 {
		res.Phase = axisv1alpha1.PhaseRunning
	}
	return res
}

func deref32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}
