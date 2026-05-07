//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResourceUnit_CRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := uniquePoolName(t, "unit")
	requireCreatePool(t, ctx, pool)
	t.Cleanup(func() { deletePool(t, pool) })

	const unit = "small"

	rr := doJSON(t, ctx, http.MethodPost,
		"/api/v1/resource-pools/"+pool+"/resource-units",
		map[string]any{
			"name":     unit,
			"requests": map[string]string{"cpu": "100m", "memory": "128Mi"},
		}, nil)
	requireStatus(t, rr, http.StatusCreated)

	var got map[string]any
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/resource-pools/"+pool+"/resource-units/"+unit, nil, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, unit, got["name"])

	var list struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/resource-pools/"+pool+"/resource-units", nil, &list)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, int64(1), list.Total)

	rr = doJSON(t, ctx, http.MethodPatch,
		"/api/v1/resource-pools/"+pool+"/resource-units/"+unit,
		map[string]any{"description": "patched"}, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, "patched", got["description"])

	rr = doJSON(t, ctx, http.MethodDelete,
		"/api/v1/resource-pools/"+pool+"/resource-units/"+unit, nil, nil)
	requireStatus(t, rr, http.StatusNoContent)

	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/resource-pools/"+pool+"/resource-units/"+unit, nil, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

func TestResourceUnit_DuplicateConflict(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := uniquePoolName(t, "unit-dup")
	requireCreatePool(t, ctx, pool)
	t.Cleanup(func() { deletePool(t, pool) })

	body := map[string]any{
		"name":     "small",
		"requests": map[string]string{"cpu": "100m"},
	}
	rr := doJSON(t, ctx, http.MethodPost,
		"/api/v1/resource-pools/"+pool+"/resource-units", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	rr = doJSON(t, ctx, http.MethodPost,
		"/api/v1/resource-pools/"+pool+"/resource-units", body, nil)
	requireStatus(t, rr, http.StatusConflict)
}

// TestResourceUnit_OrphanedParent: resolvePool must surface as 404, not
// 500 — the handler resolves the parent FK before binding the body.
func TestResourceUnit_OrphanedParent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rr := doJSON(t, ctx, http.MethodPost,
		"/api/v1/resource-pools/no-such-pool/resource-units",
		map[string]any{
			"name":     "small",
			"requests": map[string]string{"cpu": "100m"},
		}, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/resource-pools/no-such-pool/resource-units", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

func TestResourceUnit_Validation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := uniquePoolName(t, "unit-val")
	requireCreatePool(t, ctx, pool)
	t.Cleanup(func() { deletePool(t, pool) })

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing requests", map[string]any{"name": "small"}},
		{"bad name (binding axisml_resource_unit)", map[string]any{
			"name":     "Bad_Name",
			"requests": map[string]string{"cpu": "100m"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := doJSON(t, ctx, http.MethodPost,
				"/api/v1/resource-pools/"+pool+"/resource-units", tc.body, nil)
			requireClientError(t, rr)
		})
	}
}

func TestResourceUnit_NotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := uniquePoolName(t, "unit-nf")
	requireCreatePool(t, ctx, pool)
	t.Cleanup(func() { deletePool(t, pool) })

	rr := doJSON(t, ctx, http.MethodGet,
		"/api/v1/resource-pools/"+pool+"/resource-units/ghost", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodPatch,
		"/api/v1/resource-pools/"+pool+"/resource-units/ghost",
		map[string]any{"description": "x"}, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

// --- helpers --------------------------------------------------------------

func requireCreatePool(t *testing.T, ctx context.Context, name string) {
	t.Helper()
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/resource-pools",
		map[string]any{"name": name}, nil)
	requireStatus(t, rr, http.StatusCreated)
}

// deletePool best-effort tears down a pool. Idempotent on the server side
// (DELETE on a missing pool returns 204) — we don't surface the result to
// avoid masking the test's actual failure during cleanup.
func deletePool(t *testing.T, name string) {
	t.Helper()
	_ = doJSON(t, context.Background(), http.MethodDelete,
		"/api/v1/resource-pools/"+name, nil, nil)
}
