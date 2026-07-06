//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkspaceImagesCatalog covers the curated workspace base-image catalog.
func TestWorkspaceImagesCatalog(t *testing.T) {
	tok := loginAdmin(t)

	code, body := do(t, http.MethodGet, "/api/v1/workspace-images", tok, nil)
	require.Equal(t, http.StatusOK, code, "%v", body)
	items, _ := body["items"].([]any)
	require.NotEmpty(t, items, "catalog should not be empty")
	require.EqualValues(t, len(items), body["count"])
	first, _ := items[0].(map[string]any)
	require.NotEmpty(t, first["ref"])
	require.NotEmpty(t, first["displayName"])

	// Unauthenticated access is rejected.
	code, _ = do(t, http.MethodGet, "/api/v1/workspace-images", "", nil)
	require.Equal(t, http.StatusUnauthorized, code)
}

// TestDashboardActivityFeed verifies the audit middleware records a mutation and
// the dashboard activity feed reads it back, scoped to the tenant.
func TestDashboardActivityFeed(t *testing.T) {
	admin := loginAdmin(t)

	// A user to own the fresh tenant.
	code, _ := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "auditor", "password": "password123", "displayName": "Auditor",
	})
	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, code)

	// Creating a tenant is a mutation the audit middleware should record.
	code, tn := do(t, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": "audit-demo", "kubernetesNamespace": "axisml-tenant",
		"displayName": "Audit Demo", "initialAdmin": "auditor",
	})
	require.Equal(t, http.StatusCreated, code, "create tenant: %v", tn)

	// The event appears in the tenant's activity feed.
	code, feed := do(t, http.MethodGet, "/api/v1/dashboard/activity?activeTenant=audit-demo", admin, nil)
	require.Equal(t, http.StatusOK, code, "%v", feed)
	items, _ := feed["items"].([]any)
	require.NotEmpty(t, items, "feed should contain the tenant-create event")
	first, _ := items[0].(map[string]any)
	require.Equal(t, "tenant", first["kind"])
	require.Equal(t, "audit-demo", first["name"])
	require.Equal(t, "created", first["action"])
	require.Equal(t, "admin", first["actor"])

	// Activity requires an active tenant scope.
	code, _ = do(t, http.MethodGet, "/api/v1/dashboard/activity", admin, nil)
	require.Equal(t, http.StatusBadRequest, code)
}

// TestDashboardClusterUsageAndMetrics verifies the per-pool fold of cluster-usage
// (via the tenant's quotas) and the cluster-metrics proxy.
func TestDashboardClusterUsageAndMetrics(t *testing.T) {
	admin := loginAdmin(t)
	cmStub.seedQuota("dash-team", "gpu-a100")

	// cluster-usage folds one entry per pool the tenant has quota in.
	code, usage := doTenant(t, http.MethodGet, "/api/v1/dashboard/cluster-usage", admin, "dash-team", nil)
	require.Equal(t, http.StatusOK, code, "%v", usage)
	pools, _ := usage["pools"].([]any)
	require.Len(t, pools, 1)
	first, _ := pools[0].(map[string]any)
	assert.Equal(t, "gpu-a100", first["pool"])
	meters, _ := first["meters"].([]any)
	require.NotEmpty(t, meters)

	// cluster-metrics proxies N3 for the given pool.
	code, series := doTenant(t, http.MethodGet, "/api/v1/dashboard/cluster-metrics?pool=gpu-a100&metric=cpu_util&range=1h", admin, "dash-team", nil)
	require.Equal(t, http.StatusOK, code, "%v", series)
	assert.Equal(t, "cpu_util", series["metric"])

	// cluster-metrics requires a pool (the series is per (tenant, pool)).
	code, _ = doTenant(t, http.MethodGet, "/api/v1/dashboard/cluster-metrics?metric=cpu_util&range=1h", admin, "dash-team", nil)
	require.Equal(t, http.StatusBadRequest, code)

	// Unauthenticated is rejected.
	code, _ = doTenant(t, http.MethodGet, "/api/v1/dashboard/cluster-usage", "", "dash-team", nil)
	require.Equal(t, http.StatusUnauthorized, code)
}
