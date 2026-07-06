package server_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

func TestTenantPoolUsage(t *testing.T) {
	cr := &tenantv1alpha1.Tenant{}
	cr.Name = "acme"
	cr.Spec.Quotas = []tenantv1alpha1.QuotaSpec{{
		Pool: "gpu-a100",
		Max: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("32"),
			corev1.ResourceMemory: resource.MustParse("64Gi"),
			"nvidia.com/gpu":      resource.MustParse("8"),
		},
	}}
	cr.Status.Quotas = []tenantv1alpha1.QuotaStatus{{
		Pool: "gpu-a100",
		Used: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("8"),
			"nvidia.com/gpu":   resource.MustParse("2"),
		},
	}}

	u := srv.TenantPoolUsage(cr, "gpu-a100")
	assert.Equal(t, "gpu-a100", u.Pool)
	assert.Equal(t, "acme", u.Tenant)
	require.Len(t, u.Meters, 3)

	byName := map[string]srv.ResourceMeter{}
	for _, m := range u.Meters {
		byName[m.Resource] = m
	}
	assert.Equal(t, srv.ResourceMeter{Resource: "cpu", Used: 8, Total: 32, Unit: "cores"}, byName["cpu"])
	assert.Equal(t, srv.ResourceMeter{Resource: "memory", Used: 0, Total: 64, Unit: "GiB"}, byName["memory"])
	assert.Equal(t, srv.ResourceMeter{Resource: "nvidia.com/gpu", Used: 2, Total: 8, Unit: "cards"}, byName["nvidia.com/gpu"])
}

func TestTenantPoolUsage_UnknownPool(t *testing.T) {
	cr := &tenantv1alpha1.Tenant{}
	cr.Name = "acme"
	u := srv.TenantPoolUsage(cr, "missing")
	assert.Equal(t, "missing", u.Pool)
	assert.Empty(t, u.Meters)
}
