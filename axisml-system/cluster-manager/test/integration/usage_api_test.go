//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

// TestResourcePoolUsage drives the N2 endpoint: a tenant's used-vs-total quota
// utilisation in a pool, folding the quota ceiling and the reflected Used.
func TestResourcePoolUsage(t *testing.T) {
	const pool = "usage-pool"
	const tenant = "usage-team"
	seedPool(t, pool)

	body := `{"name":"` + tenant + `","quotas":[{"pool":"` + pool + `","units":[{"unitName":"cpu-small","quantity":3}]}]}`
	rr := doRequest(t, "POST", "/api/v1/tenants", body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// Envtest runs no controllers, so reflect an ElasticQuota Used into the CR
	// status by hand (tenant-operator would normally do this).
	var cr tenantv1alpha1.Tenant
	require.NoError(t, testCli.Get(context.Background(), types.NamespacedName{Name: tenant}, &cr))
	cr.Status.Quotas = []tenantv1alpha1.QuotaStatus{{
		Pool:  pool,
		Ready: true,
		Used: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	}}
	require.NoError(t, testCli.Status().Update(context.Background(), &cr))

	rr = doRequest(t, "GET", "/api/v1/resourcepools/"+pool+"/usage?tenant="+tenant, "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var usage srv.PoolUsage
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &usage))
	assert.Equal(t, pool, usage.Pool)
	assert.Equal(t, tenant, usage.Tenant)

	byName := map[string]srv.ResourceMeter{}
	for _, m := range usage.Meters {
		byName[m.Resource] = m
	}
	// Ceiling = 3 × cpu-small limits {cpu:2, memory:4Gi}.
	assert.Equal(t, 6.0, byName["cpu"].Total)
	assert.Equal(t, 2.0, byName["cpu"].Used)
	assert.Equal(t, 12.0, byName["memory"].Total)
	assert.Equal(t, 4.0, byName["memory"].Used)

	// tenant is required.
	rr = doRequest(t, "GET", "/api/v1/resourcepools/"+pool+"/usage", "")
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestResourcePoolMetrics_Unavailable covers the N3 endpoint when no Prometheus
// backend is configured (the harness injects a nil metrics querier).
func TestResourcePoolMetrics_Unavailable(t *testing.T) {
	const pool = "metrics-pool"
	seedPool(t, pool)

	rr := doRequest(t, "GET", "/api/v1/resourcepools/"+pool+"/metrics?tenant=whoever&metric=cpu_util&range=1h", "")
	require.Equal(t, http.StatusServiceUnavailable, rr.Code, rr.Body.String())

	// tenant is required, checked before availability.
	rr = doRequest(t, "GET", "/api/v1/resourcepools/"+pool+"/metrics?metric=cpu_util&range=1h", "")
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
