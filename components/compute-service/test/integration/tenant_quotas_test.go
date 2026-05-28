//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	tenantmod "github.com/axisml/axisml/components/compute-service/internal/tenant"
)

// TestTenant_Quotas_CRUD exercises GET / POST / PATCH / DELETE under
// /api/v1/namespaces/{namespace}/quotas with the spec.quotas[] jsonb path.
func TestTenant_Quotas_CRUD(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", map[string]any{
		"name":      "team-q",
		"namespace": map[string]any{"name": "team-q-ns"},
		"quotas": []map[string]any{{
			"pool": "default", "name": "default",
			"min": map[string]string{"cpu": "1"},
			"max": map[string]string{"cpu": "4"},
		}},
	}, nil)
	requireStatus(t, rr, http.StatusCreated)
	t.Cleanup(func() {
		_ = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/team-q", nil, nil)
	})

	// LIST starts with one quota.
	var list struct {
		Items []tenantmod.QuotaSpec `json:"items"`
	}
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/team-q/quotas", nil, &list)
	requireStatus(t, rr, http.StatusOK)
	require.Len(t, list.Items, 1)

	// POST a second quota.
	var added tenantmod.QuotaSpec
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/team-q/quotas", map[string]any{
		"pool": "gpu", "name": "small",
		"min": map[string]string{"nvidia.com/gpu": "0"},
		"max": map[string]string{"nvidia.com/gpu": "2"},
	}, &added)
	requireStatus(t, rr, http.StatusCreated)
	require.Equal(t, "gpu", added.Pool)
	require.Equal(t, "2", added.Max["nvidia.com/gpu"])

	// Duplicate POST returns 409.
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/team-q/quotas", map[string]any{
		"pool": "gpu", "name": "small",
		"max": map[string]string{"nvidia.com/gpu": "1"},
	}, nil)
	requireStatus(t, rr, http.StatusConflict)

	// PATCH bumps max.
	var patched tenantmod.QuotaSpec
	rr = doJSON(t, ctx, http.MethodPatch,
		"/api/v1/namespaces/team-q/quotas/gpu/small",
		map[string]any{"max": map[string]string{"nvidia.com/gpu": "8"}}, &patched)
	requireStatus(t, rr, http.StatusOK)
	require.Equal(t, "8", patched.Max["nvidia.com/gpu"])

	// DELETE the gpu quota.
	rr = doJSON(t, ctx, http.MethodDelete,
		"/api/v1/namespaces/team-q/quotas/gpu/small", nil, nil)
	requireStatus(t, rr, http.StatusNoContent)

	// PATCH of the deleted quota → 404.
	rr = doJSON(t, ctx, http.MethodPatch,
		"/api/v1/namespaces/team-q/quotas/gpu/small",
		map[string]any{"max": map[string]string{"nvidia.com/gpu": "8"}}, nil)
	requireStatus(t, rr, http.StatusNotFound)

	// LIST is back to one entry.
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/team-q/quotas", nil, &list)
	requireStatus(t, rr, http.StatusOK)
	require.Len(t, list.Items, 1)
	require.Equal(t, "default", list.Items[0].Pool)
}
