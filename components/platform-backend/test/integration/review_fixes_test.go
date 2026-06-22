//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUserDisableNotReEnabled pins the fix for the PATCH-clobbers-disabled bug:
// a profile-only PATCH must not silently re-enable a disabled account.
func TestUserDisableNotReEnabled(t *testing.T) {
	admin := loginAdmin(t)

	code, u := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "carol", "password": "password123", "displayName": "Carol",
	})
	require.Equal(t, http.StatusCreated, code, "create: %v", u)
	id := u["id"].(string)

	// Disable the account.
	code, _ = do(t, http.MethodPatch, "/api/v1/users/"+id, admin, map[string]any{"disabled": true})
	require.Equal(t, http.StatusOK, code)

	// Profile-only PATCH (no disabled field) must leave it disabled.
	code, _ = do(t, http.MethodPatch, "/api/v1/users/"+id, admin, map[string]any{"displayName": "Carol R"})
	require.Equal(t, http.StatusOK, code)

	code, got := do(t, http.MethodGet, "/api/v1/users/"+id, admin, nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, true, got["disabled"], "disabled must survive a profile-only patch")

	// And it can be explicitly re-enabled.
	code, _ = do(t, http.MethodPatch, "/api/v1/users/"+id, admin, map[string]any{"disabled": false})
	require.Equal(t, http.StatusOK, code)
	code, got = do(t, http.MethodGet, "/api/v1/users/"+id, admin, nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, nil, got["disabled"], "disabled omitempty drops to absent when false")
}

// TestUserReadsRequireAdmin pins that user reads are system-admin only.
func TestUserReadsRequireAdmin(t *testing.T) {
	admin := loginAdmin(t)
	code, _ := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "dave", "password": "password123", "displayName": "Dave",
	})
	require.Equal(t, http.StatusCreated, code)

	dave := loginAs(t, "dave", "password123")
	code, _ = do(t, http.MethodGet, "/api/v1/users", dave, nil)
	require.Equal(t, http.StatusForbidden, code)
	code, _ = do(t, http.MethodGet, "/api/v1/users/whatever", dave, nil)
	require.Equal(t, http.StatusForbidden, code)
}

// TestUnimplementedRouteReturns501 pins that a documented-but-unwired endpoint
// returns a problem+json 501, not a bare 404.
func TestUnimplementedRouteReturns501(t *testing.T) {
	// An /api/v1 path with no registered handler must be a problem+json 501,
	// not a bare 404.
	code, body := do(t, http.MethodGet, "/api/v1/not-a-real-endpoint-xyz", "", nil)
	require.Equal(t, http.StatusNotImplemented, code)
	require.Equal(t, "not-implemented", body["code"])
}
