//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestResourceUnit_CRUD drives all five /resource-pools/:pool/resource-units
// routes against a real PG testcontainer.
func TestResourceUnit_CRUD(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding (testEngine) not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := uniquePoolName(t, "unit")
	requireCreatePool(t, ctx, pool)
	t.Cleanup(func() { _ = deletePool(t, pool) })

	const unit = "small"

	// POST a unit.
	rr := doJSON(t, ctx, http.MethodPost,
		"/api/v1/resource-pools/"+pool+"/resource-units",
		map[string]any{
			"name":     unit,
			"requests": map[string]string{"cpu": "100m", "memory": "128Mi"},
		}, nil)
	requireStatus(t, rr, http.StatusCreated)

	// GET reflects the unit.
	var got map[string]any
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/resource-pools/"+pool+"/resource-units/"+unit, nil, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, unit, got["name"])

	// LIST returns the unit.
	var list struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/resource-pools/"+pool+"/resource-units", nil, &list)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, int64(1), list.Total)

	// PATCH description.
	rr = doJSON(t, ctx, http.MethodPatch,
		"/api/v1/resource-pools/"+pool+"/resource-units/"+unit,
		map[string]any{"description": "patched"}, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, "patched", got["description"])

	// DELETE → 204.
	rr = doJSON(t, ctx, http.MethodDelete,
		"/api/v1/resource-pools/"+pool+"/resource-units/"+unit, nil, nil)
	requireStatus(t, rr, http.StatusNoContent)

	// Subsequent GET → 404.
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/resource-pools/"+pool+"/resource-units/"+unit, nil, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

// TestResourceUnit_DuplicateConflict creates the same unit twice in the
// same pool.
func TestResourceUnit_DuplicateConflict(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := uniquePoolName(t, "unit-dup")
	requireCreatePool(t, ctx, pool)
	t.Cleanup(func() { _ = deletePool(t, pool) })

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

// TestResourceUnit_OrphanedParent verifies that the handler returns 404
// when the parent pool doesn't exist (resolvePool surfaces as not-found
// rather than internal-error).
func TestResourceUnit_OrphanedParent(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
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

// TestResourceUnit_Validation rejects empty requests / bad name.
func TestResourceUnit_Validation(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := uniquePoolName(t, "unit-val")
	requireCreatePool(t, ctx, pool)
	t.Cleanup(func() { _ = deletePool(t, pool) })

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			"missing requests",
			map[string]any{"name": "small"},
		},
		{
			"bad name (binding axisml_resource_unit)",
			map[string]any{
				"name":     "Bad_Name",
				"requests": map[string]string{"cpu": "100m"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := doJSON(t, ctx, http.MethodPost,
				"/api/v1/resource-pools/"+pool+"/resource-units", tc.body, nil)
			if rr.Code < 400 || rr.Code >= 500 {
				t.Fatalf("expected 4xx, got %d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

// TestResourceUnit_NotFound covers the read paths against a unit that
// doesn't exist under an existing pool. DELETE is intentionally
// idempotent (always 204) and is exercised in TestResourceUnit_CRUD.
func TestResourceUnit_NotFound(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := uniquePoolName(t, "unit-nf")
	requireCreatePool(t, ctx, pool)
	t.Cleanup(func() { _ = deletePool(t, pool) })

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

// deletePool tears down a pool. Ignores the result — the units it owns may
// already have been deleted by their own cleanup, in which case the pool
// delete just succeeds.
func deletePool(t *testing.T, name string) error {
	t.Helper()
	rr := doJSON(t, context.Background(), http.MethodDelete,
		"/api/v1/resource-pools/"+name, nil, nil)
	_ = rr // best-effort
	return nil
}
