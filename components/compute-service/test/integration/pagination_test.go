//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	tenantmod "github.com/axisml/axisml/components/compute-service/internal/tenant"
)

// TestPagination_ContinueTokenRoundTrip verifies the K8s-style `?continue=`
// token works for sequential page fetches and the last page reports no
// continueToken. Driven on the tenants list — the same pagination helper
// powers jobs / services / artifact lists.
func TestPagination_ContinueTokenRoundTrip(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()

	// Seed 5 tenants so we can request limit=2 and walk three pages.
	names := []string{"pg-a", "pg-b", "pg-c", "pg-d", "pg-e"}
	for _, n := range names {
		rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", map[string]any{
			"name":      n,
			"namespace": map[string]any{"name": n + "-ns"},
		}, nil)
		requireStatus(t, rr, http.StatusCreated)
	}
	t.Cleanup(func() {
		for _, n := range names {
			_ = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/"+n, nil, nil)
		}
	})

	// Page 1.
	var page1 tenantmod.ListResponse
	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces?limit=2", nil, &page1)
	requireStatus(t, rr, http.StatusOK)
	require.NotEmpty(t, page1.ContinueToken, "page 1 must surface continueToken")

	// Page 2 via continueToken.
	var page2 tenantmod.ListResponse
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces?limit=2&continue="+page1.ContinueToken, nil, &page2)
	requireStatus(t, rr, http.StatusOK)
	require.NotEmpty(t, page2.ContinueToken, "page 2 must still have continueToken")
	require.NotEqual(t, page1.Items[0].ID, page2.Items[0].ID, "pages must be disjoint")

	// Page 3 — last page; continueToken should be absent.
	var page3 tenantmod.ListResponse
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces?limit=2&continue="+page2.ContinueToken, nil, &page3)
	requireStatus(t, rr, http.StatusOK)
	require.Empty(t, page3.ContinueToken, "last page must NOT surface a continueToken")
}

// TestPagination_InvalidContinue returns 400 on a malformed token.
func TestPagination_InvalidContinue(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	rr := doJSON(t, context.Background(), http.MethodGet,
		"/api/v1/namespaces?continue=!!!not-base64!!!", nil, nil)
	requireClientError(t, rr)
}
