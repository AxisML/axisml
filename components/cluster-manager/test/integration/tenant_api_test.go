//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"

	srv "github.com/axisml/axisml/components/cluster-manager/internal/server"
	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// seedPool creates a ResourcePool with one cpu-small unit (requests cpu=1,
// memory=2Gi; limits cpu=2, memory=4Gi) so tenant quotas have something to
// fold against.
func seedPool(t *testing.T, name string) {
	t.Helper()
	body := `{
	  "name": "` + name + `",
	  "units": [
	    {
	      "name": "cpu-small",
	      "requests": {"cpu": "1", "memory": "2Gi"},
	      "limits":   {"cpu": "2", "memory": "4Gi"}
	    }
	  ]
	}`
	rr := doRequest(t, "POST", "/api/v1/resourcepools", body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
}

// TestTenant_Lifecycle drives Create → Get → List → Patch → Delete plus the
// quota folding round-trip for a Tenant, against the live envtest apiserver.
func TestTenant_Lifecycle(t *testing.T) {
	const pool = "tnt-pool"
	const name = "team-alpha"
	seedPool(t, pool)

	// Create with a quota expressed as unit × quantity.
	body := `{
	  "name": "` + name + `",
	  "labels": {"axisml.io/tier": "gold"},
	  "quotas": [
	    {"pool": "` + pool + `", "units": [{"unitName": "cpu-small", "quantity": 3}]}
	  ]
	}`
	rr := doRequest(t, "POST", "/api/v1/tenants", body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var created srv.TenantDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &created))
	require.Equal(t, name, created.Name)
	require.Equal(t, name, created.Namespace.Name) // namespace defaults to name
	require.Len(t, created.Quotas, 1)
	require.Equal(t, pool, created.Quotas[0].Pool)
	require.Equal(t, "cpu-small", created.Quotas[0].Units[0].UnitName)
	require.Equal(t, 3, created.Quotas[0].Units[0].Quantity)

	// The underlying CR carries folded ElasticQuota min/max (3×{cpu1,mem2Gi}
	// for min; 3×{cpu2,mem4Gi} for max).
	var cr tenantv1alpha1.Tenant
	require.NoError(t, testCli.Get(context.Background(), types.NamespacedName{Name: name}, &cr))
	require.Len(t, cr.Spec.Quotas, 1)
	minCPU := cr.Spec.Quotas[0].Min["cpu"]
	maxCPU := cr.Spec.Quotas[0].Max["cpu"]
	require.Equal(t, "3", minCPU.String())
	require.Equal(t, "6", maxCPU.String())
	minMem := cr.Spec.Quotas[0].Min["memory"]
	require.Equal(t, "6Gi", minMem.String())

	// Get round-trips the business form from the annotation.
	rr = doRequest(t, "GET", "/api/v1/tenants/"+name, "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got srv.TenantDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Len(t, got.Quotas, 1)
	require.Equal(t, 3, got.Quotas[0].Units[0].Quantity)

	// List by label selector.
	rr = doRequest(t, "GET", "/api/v1/tenants?labelSelector=axisml.io/tier=gold", "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var list srv.TenantList
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	require.GreaterOrEqual(t, list.Count, 1)

	// Patch CR labels.
	rr = doRequest(t, "PATCH", "/api/v1/tenants/"+name, `{"labels": {"axisml.io/tier": "silver"}}`)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, "silver", got.Labels["axisml.io/tier"])

	// Delete.
	rr = doRequest(t, "DELETE", "/api/v1/tenants/"+name, "")
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())
	rr = doRequest(t, "GET", "/api/v1/tenants/"+name, "")
	require.Equal(t, http.StatusNotFound, rr.Code)
}

// TestTenant_QuotaSubRoutes drives the per-pool quota sub-routes.
func TestTenant_QuotaSubRoutes(t *testing.T) {
	const pool = "q-pool"
	const name = "team-beta"
	seedPool(t, pool)

	rr := doRequest(t, "POST", "/api/v1/tenants", `{"name": "`+name+`"}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// Set a quota for the pool.
	rr = doRequest(t, "POST", "/api/v1/tenants/"+name+"/quotas",
		`{"pool": "`+pool+`", "units": [{"unitName": "cpu-small", "quantity": 2}]}`)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	// List shows it.
	rr = doRequest(t, "GET", "/api/v1/tenants/"+name+"/quotas", "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var ql srv.QuotaList
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ql))
	require.Len(t, ql.Items, 1)

	// Patch the unit selection.
	rr = doRequest(t, "PATCH", "/api/v1/tenants/"+name+"/quotas/"+pool,
		`{"units": [{"unitName": "cpu-small", "quantity": 5}]}`)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var q srv.QuotaDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &q))
	require.Equal(t, 5, q.Units[0].Quantity)

	// Patch a missing pool quota 404s.
	rr = doRequest(t, "PATCH", "/api/v1/tenants/"+name+"/quotas/nope",
		`{"units": [{"unitName": "cpu-small", "quantity": 1}]}`)
	require.Equal(t, http.StatusNotFound, rr.Code, rr.Body.String())

	// Delete the quota (idempotent).
	rr = doRequest(t, "DELETE", "/api/v1/tenants/"+name+"/quotas/"+pool, "")
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())
	rr = doRequest(t, "DELETE", "/api/v1/tenants/"+name+"/quotas/"+pool, "")
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())

	_ = doRequest(t, "DELETE", "/api/v1/tenants/"+name, "")
}

// TestTenant_Create_UnknownPool rejects a quota against a missing pool.
func TestTenant_Create_UnknownPool(t *testing.T) {
	rr := doRequest(t, "POST", "/api/v1/tenants", `{
	  "name": "team-gamma",
	  "quotas": [{"pool": "ghost-pool", "units": [{"unitName": "x", "quantity": 1}]}]
	}`)
	require.Equal(t, http.StatusUnprocessableEntity, rr.Code, rr.Body.String())
}
