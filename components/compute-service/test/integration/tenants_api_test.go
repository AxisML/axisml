//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	tenantmod "github.com/axisml/axisml/components/compute-service/internal/tenant"
)

// TestTenant_API_Lifecycle is the acceptance test the new compute-service
// design calls out: client → compute-service /api/v1/namespaces → PG row
// with phase=Creating, plus GET/PATCH/LIST/DELETE round-trips.
func TestTenant_API_Lifecycle(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped (docker likely unavailable)")
	}
	ctx := context.Background()

	createReq := map[string]any{
		"name":        "team-alpha",
		"displayName": "Team Alpha",
		"description": "tenant integration test",
		"labels":      map[string]string{"axisml.io/project": "p1"},
		"namespace":   map[string]any{"name": "team-alpha-ns"},
		"quotas": []map[string]any{{
			"pool": "default",
			"name": "default",
			"min":  map[string]string{"cpu": "1"},
			"max":  map[string]string{"cpu": "4"},
		}},
	}
	var created tenantmod.Response
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", createReq, &created)
	requireStatus(t, rr, http.StatusCreated)
	require.Equal(t, "team-alpha", created.Name)
	require.Equal(t, "Team Alpha", created.DisplayName)
	require.Equal(t, tenantmod.PhaseCreating, created.Phase)
	require.Equal(t, "team-alpha-ns", created.Namespace.Name)
	require.Len(t, created.Quotas, 1)
	require.Equal(t, "default", created.Quotas[0].Pool)
	require.Equal(t, int64(1), created.Generation)

	// GET round-trip.
	var got tenantmod.Response
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/team-alpha", nil, &got)
	requireStatus(t, rr, http.StatusOK)
	require.Equal(t, "team-alpha", got.Name)

	// LIST round-trip.
	var listed tenantmod.ListResponse
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces", nil, &listed)
	requireStatus(t, rr, http.StatusOK)
	found := false
	for _, item := range listed.Items {
		if item.Name == "team-alpha" {
			found = true
			break
		}
	}
	require.True(t, found, "tenant team-alpha not in LIST response")

	// PATCH displayName + bump quota max → generation must increment.
	patchReq := map[string]any{
		"displayName": "Team Alpha (renamed)",
		"quotas": []map[string]any{{
			"pool": "default",
			"name": "default",
			"min":  map[string]string{"cpu": "1"},
			"max":  map[string]string{"cpu": "16"},
		}},
	}
	var patched tenantmod.Response
	rr = doJSON(t, ctx, http.MethodPatch, "/api/v1/namespaces/team-alpha", patchReq, &patched)
	requireStatus(t, rr, http.StatusOK)
	require.Equal(t, "Team Alpha (renamed)", patched.DisplayName)
	require.Equal(t, "16", patched.Quotas[0].Max["cpu"])
	require.Greater(t, patched.Generation, created.Generation,
		"spec mutation must bump generation")

	// DELETE soft-deletes (subsequent GET 404s).
	rr = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/team-alpha", nil, nil)
	requireStatus(t, rr, http.StatusNoContent)

	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/team-alpha", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

// TestTenant_API_DuplicateName ensures the partial unique index on
// (name) WHERE deleted_at IS NULL fires a 409 on a fresh duplicate.
func TestTenant_API_DuplicateName(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()
	req := map[string]any{
		"name":      "dup-tenant",
		"namespace": map[string]any{"name": "dup-ns"},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", req, nil)
	requireStatus(t, rr, http.StatusCreated)
	t.Cleanup(func() {
		_ = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/dup-tenant", nil, nil)
	})

	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", req, nil)
	requireStatus(t, rr, http.StatusConflict)
}

// TestTenant_API_InvalidName rejects a bad name before touching PG.
func TestTenant_API_InvalidName(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", map[string]any{
		"name":      "BadName_Underscores",
		"namespace": map[string]any{"name": "x-ns"},
	}, nil)
	requireClientError(t, rr)
}
