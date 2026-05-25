package resourceunit

import (
	corev1 "k8s.io/api/core/v1"
)

// MergeNodeSelector implements the design-doc rule: pool keys win, then
// resource-unit keys fill in gaps that pool didn't declare.
func MergeNodeSelector(poolSel, unitSel map[string]string) map[string]string {
	out := make(map[string]string, len(poolSel)+len(unitSel))
	for k, v := range poolSel {
		out[k] = v
	}
	for k, v := range unitSel {
		if _, exists := out[k]; !exists {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// BuildResources turns ResourceUnit requests/limits into corev1.ResourceRequirements.
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
