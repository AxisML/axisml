//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	tenantmod "github.com/axisml/axisml/components/compute-service/internal/tenant"
)

// TestTenant_RestoreLifecycle drives Create → Delete → wait for Deleted →
// Restore, and asserts the row flips back to Creating with bumped
// generation (so the reconciler re-creates the CR).
func TestTenant_RestoreLifecycle(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()

	body := map[string]any{
		"name":      "team-restore",
		"namespace": map[string]any{"name": "team-restore-ns"},
	}
	var created tenantmod.Response
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", body, &created)
	requireStatus(t, rr, http.StatusCreated)
	t.Cleanup(func() {
		_ = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/team-restore", nil, nil)
	})

	// Force the row to phase=Deleted directly via gorm — the reconciler
	// path that takes a Tenant CR to Deleted requires envtest support that's
	// outside the scope of this PG-only test.
	require.NoError(t, gormDB.Exec(
		"UPDATE tenants SET phase='Deleted', deleted_at=now() WHERE name=?",
		"team-restore").Error)

	// Restore the row.
	var restored tenantmod.Response
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/team-restore/restore", nil, &restored)
	requireStatus(t, rr, http.StatusOK)
	require.Equal(t, tenantmod.PhaseCreating, restored.Phase)
	require.Greater(t, restored.Generation, created.Generation,
		"restore must bump generation so reconciler re-creates the CR")

	// GET on the active path should now succeed.
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/team-restore", nil, nil)
	requireStatus(t, rr, http.StatusOK)
}

// TestTenant_Restore_WrongPhase refuses to restore a row that's not in
// phase=Deleted (412 PreconditionFailed).
func TestTenant_Restore_WrongPhase(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()

	body := map[string]any{
		"name":      "team-restore-wrong",
		"namespace": map[string]any{"name": "team-restore-wrong-ns"},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", body, nil)
	requireStatus(t, rr, http.StatusCreated)
	t.Cleanup(func() {
		_ = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/team-restore-wrong", nil, nil)
	})

	rr = doJSON(t, ctx, http.MethodPost,
		"/api/v1/namespaces/team-restore-wrong/restore", nil, nil)
	requireStatus(t, rr, http.StatusPreconditionFailed)
}

// TestTenant_Restore_NotFound returns 404 when the tenant doesn't exist.
func TestTenant_Restore_NotFound(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()
	rr := doJSON(t, ctx, http.MethodPost,
		"/api/v1/namespaces/ghost-tenant/restore", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)
}
