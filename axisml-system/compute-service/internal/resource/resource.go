// Package resource holds compute-internal helpers for translating a resolved
// resource snapshot into the Kubernetes shapes the workload CR spec expects.
package resource

import (
	corev1 "k8s.io/api/core/v1"
)

// BuildResources converts a unit's request/limit ResourceLists into the K8s
// ResourceRequirements shape expected by Pod templates.
func BuildResources(req, lim corev1.ResourceList) corev1.ResourceRequirements {
	rr := corev1.ResourceRequirements{}
	if len(req) > 0 {
		rr.Requests = req
	}
	if len(lim) > 0 {
		rr.Limits = lim
	}
	return rr
}
