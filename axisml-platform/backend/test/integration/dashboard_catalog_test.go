//go:build integration

package integration

import (
	"net/http"
	"testing"

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
