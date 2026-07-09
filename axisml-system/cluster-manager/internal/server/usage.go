package server

import (
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

// TenantPoolUsage projects a tenant's used-vs-total resource utilisation in one
// pool. The ceiling comes from the folded quota (spec.quotas[pool].Max, falling
// back to Min); usage comes from the ElasticQuota status the tenant-operator
// reflects into status.quotas[pool].Used. An unknown pool yields empty meters.
func TenantPoolUsage(t *tenantv1alpha1.Tenant, pool string) PoolUsage {
	var total, used corev1.ResourceList
	for _, q := range t.Spec.Quotas {
		if q.Pool == pool {
			total = q.Max
			if len(total) == 0 {
				total = q.Min
			}
			break
		}
	}
	for _, q := range t.Status.Quotas {
		if q.Pool == pool {
			used = q.Used
			break
		}
	}
	return PoolUsage{Pool: pool, Tenant: t.Name, Meters: buildMeters(total, used)}
}

// buildMeters emits one meter per resource dimension present in either list.
func buildMeters(total, used corev1.ResourceList) []ResourceMeter {
	names := map[corev1.ResourceName]struct{}{}
	for n := range total {
		names[n] = struct{}{}
	}
	for n := range used {
		names[n] = struct{}{}
	}
	ordered := make([]string, 0, len(names))
	for n := range names {
		ordered = append(ordered, string(n))
	}
	sort.Strings(ordered)

	meters := make([]ResourceMeter, 0, len(ordered))
	for _, n := range ordered {
		name := corev1.ResourceName(n)
		usedV, unit := meterValue(name, used)
		totalV, _ := meterValue(name, total)
		meters = append(meters, ResourceMeter{Resource: n, Used: usedV, Total: totalV, Unit: unit})
	}
	return meters
}

// meterValue converts a resource quantity to a display float + unit: CPU in
// cores, memory in GiB, GPU (any nvidia.com/gpu-like name) in cards, else raw.
func meterValue(name corev1.ResourceName, rl corev1.ResourceList) (float64, string) {
	q, ok := rl[name]
	var v float64
	if ok {
		v = q.AsApproximateFloat64()
	}
	switch {
	case name == corev1.ResourceCPU:
		return v, "cores"
	case name == corev1.ResourceMemory:
		return v / (1 << 30), "GiB"
	case strings.Contains(string(name), "gpu"):
		return v, "cards"
	default:
		return v, ""
	}
}
