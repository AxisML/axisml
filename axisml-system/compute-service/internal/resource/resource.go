// Package resource holds compute-internal helpers for translating a resolved
// resource snapshot into the Kubernetes shapes the workload CR spec expects.
package resource

import (
	corev1 "k8s.io/api/core/v1"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/cluster-manager/api/v1alpha1"
)

// Expanded is the resource snapshot frozen into the workload spec at create
// time, produced by merging a ResourcePool with one of its ResourceUnits.
type Expanded struct {
	NodeSelector map[string]string
	Tolerations  []corev1.Toleration
	Requests     corev1.ResourceList
	Limits       corev1.ResourceList
}

// Expand merges a pool and one of its units into the snapshot per the design
// §5.4 merge rules: pool nodeSelector keys win and unit-only keys fill the
// gaps; pool tolerations pass through verbatim; the unit's requests/limits go
// to the role template. The unit must belong to pool (the caller resolves both
// via the ResourceResolver seam).
func Expand(pool *cmv1alpha1.ResourcePool, unit *cmv1alpha1.ResourceUnit) Expanded {
	return Expanded{
		NodeSelector: mergeNodeSelector(pool.Spec.NodeSelector, unit.NodeSelector),
		Tolerations:  pool.Spec.Tolerations,
		Requests:     unit.Requests,
		Limits:       unit.Limits,
	}
}

// mergeNodeSelector applies the design merge rule: pool keys are preserved,
// unit-only keys fill in gaps the pool didn't declare.
func mergeNodeSelector(poolSel, unitSel map[string]string) map[string]string {
	if len(poolSel) == 0 && len(unitSel) == 0 {
		return nil
	}
	out := make(map[string]string, len(poolSel)+len(unitSel))
	for k, v := range poolSel {
		out[k] = v
	}
	for k, v := range unitSel {
		if _, ok := out[k]; !ok {
			out[k] = v
		}
	}
	return out
}

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
