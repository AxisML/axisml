//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDataVolume_CRUD drives the tenant-scoped data-volume surface end to end
// against the cluster-manager stub: create / list / get / patch / delete.
func TestDataVolume_CRUD(t *testing.T) {
	admin := loginAdmin(t)

	code, tn := do(t, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": "vol-team", "kubernetesNamespace": "axisml-vol-team",
		"displayName": "Vol Team", "initialAdmin": "admin",
	})
	require.Equal(t, http.StatusCreated, code, "create tenant: %v", tn)

	// Create.
	code, body := doTenant(t, http.MethodPost, "/api/v1/datavolumes", admin, "vol-team", map[string]any{
		"name": "shared-data", "size": "10Gi", "accessModes": []string{"ReadWriteMany"}, "description": "team share",
	})
	require.Equal(t, http.StatusCreated, code, body)
	require.Equal(t, "shared-data", body["name"])

	// List shows it.
	code, body = doTenant(t, http.MethodGet, "/api/v1/datavolumes", admin, "vol-team", nil)
	require.Equal(t, http.StatusOK, code, body)
	require.Equal(t, float64(1), body["count"])

	// Get returns description + status.
	code, body = doTenant(t, http.MethodGet, "/api/v1/datavolumes/shared-data", admin, "vol-team", nil)
	require.Equal(t, http.StatusOK, code, body)
	require.Equal(t, "team share", body["description"])
	require.NotNil(t, body["status"])

	// Patch the description.
	code, body = doTenant(t, http.MethodPatch, "/api/v1/datavolumes/shared-data", admin, "vol-team", map[string]any{
		"description": "updated share",
	})
	require.Equal(t, http.StatusOK, code, body)
	require.Equal(t, "updated share", body["description"])

	// Delete.
	code, _ = doTenant(t, http.MethodDelete, "/api/v1/datavolumes/shared-data", admin, "vol-team", nil)
	require.Equal(t, http.StatusNoContent, code)
}

// TestDataVolume_WritesRequireSystemAdmin verifies a non-admin cannot create a
// volume (writes are system-admin only); reads are open to tenant members so the
// mount picker can populate.
func TestDataVolume_WritesRequireSystemAdmin(t *testing.T) {
	admin := loginAdmin(t)
	code, _ := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "voluser", "password": "password123", "displayName": "Vol User",
	})
	require.Equal(t, http.StatusCreated, code)

	tok := loginAs(t, "voluser", "password123")
	code, _ = doTenant(t, http.MethodPost, "/api/v1/datavolumes", tok, "vol-team", map[string]any{
		"name": "nope", "size": "1Gi",
	})
	require.Equal(t, http.StatusForbidden, code)
}

// TestDataVolume_RequiresActiveTenant verifies the active-tenant guard.
func TestDataVolume_RequiresActiveTenant(t *testing.T) {
	admin := loginAdmin(t)
	code, _ := do(t, http.MethodGet, "/api/v1/datavolumes", admin, nil)
	require.Equal(t, http.StatusBadRequest, code)
}
