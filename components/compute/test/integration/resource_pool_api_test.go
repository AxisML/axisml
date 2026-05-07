//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestResourcePool_CRUD drives the full /resource-pools surface in-process
// against a real Postgres testcontainer: POST → GET → LIST → PATCH → DELETE.
func TestResourcePool_CRUD(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding (testEngine) not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	name := uniquePoolName(t, "crud")

	createBody := map[string]any{
		"name":        name,
		"description": "crud test",
		"nodeSelector": map[string]string{
			"role": "worker",
		},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/resource-pools", createBody, nil)
	requireStatus(t, rr, http.StatusCreated)

	// GET
	var got map[string]any
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/resource-pools/"+name, nil, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, name, got["name"])
	assert.Equal(t, "crud test", got["description"])

	// LIST contains the pool we created.
	var list struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/resource-pools", nil, &list)
	requireStatus(t, rr, http.StatusOK)
	assert.GreaterOrEqual(t, list.Total, int64(1))
	assert.True(t, containsPool(list.Items, name), "list should include the created pool")

	// PATCH description.
	rr = doJSON(t, ctx, http.MethodPatch, "/api/v1/resource-pools/"+name,
		map[string]any{"description": "patched"}, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, "patched", got["description"])

	// DELETE → 204; subsequent GET → 404; a second DELETE is also 204
	// (idempotent — same convention as cluster-manager Tenant DELETE).
	rr = doJSON(t, ctx, http.MethodDelete, "/api/v1/resource-pools/"+name, nil, nil)
	requireStatus(t, rr, http.StatusNoContent)

	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/resource-pools/"+name, nil, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodDelete, "/api/v1/resource-pools/"+name, nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
}

// TestResourcePool_DuplicateConflict ensures a second POST with the same
// name returns 409.
func TestResourcePool_DuplicateConflict(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	name := uniquePoolName(t, "dup")
	body := map[string]any{"name": name}

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/resource-pools", body, nil)
	requireStatus(t, rr, http.StatusCreated)
	t.Cleanup(func() {
		_ = doJSON(t, context.Background(), http.MethodDelete, "/api/v1/resource-pools/"+name, nil, nil)
	})

	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/resource-pools", body, nil)
	requireStatus(t, rr, http.StatusConflict)
}

// TestResourcePool_Validation covers missing / malformed name. Bind failures
// and service-layer validation both surface as 4xx.
func TestResourcePool_Validation(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing name", map[string]any{"description": "x"}},
		{"uppercase name (axisml_name DNS-1123)", map[string]any{"name": "BadName"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := doJSON(t, ctx, http.MethodPost, "/api/v1/resource-pools", tc.body, nil)
			if rr.Code < 400 || rr.Code >= 500 {
				t.Fatalf("expected 4xx, got %d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

// TestResourcePool_NotFound covers the read paths against a name that
// doesn't exist. DELETE is intentionally idempotent (always 204), so it
// is exercised in TestResourcePool_CRUD instead.
func TestResourcePool_NotFound(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const name = "does-not-exist-pool"

	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/resource-pools/"+name, nil, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodPatch, "/api/v1/resource-pools/"+name,
		map[string]any{"description": "x"}, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

// --- helpers --------------------------------------------------------------

func uniquePoolName(t *testing.T, scenario string) string {
	t.Helper()
	return "pool-" + scenario + "-" + randSuffix(t)
}

func containsPool(items []map[string]any, name string) bool {
	for _, it := range items {
		if it["name"] == name {
			return true
		}
	}
	return false
}
