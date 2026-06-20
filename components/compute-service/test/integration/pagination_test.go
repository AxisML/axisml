//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// mlrunListPage is the subset of the MLRun list envelope the pagination /
// label-selector tests assert on.
type mlrunListPage struct {
	Items []struct {
		ID     string            `json:"id"`
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"items"`
	ContinueToken string `json:"continueToken"`
}

// TestPagination_ContinueTokenRoundTrip verifies the K8s-style `?continue=`
// token works for sequential page fetches and the last page reports no
// continueToken. Driven on the MLRun list — the same pagination helper powers
// services / artifact lists.
func TestPagination_ContinueTokenRoundTrip(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()
	seedResourcePool(t, ctx, "pg-pool", "pg-unit")
	const ns = "pg-ns"
	mustCreateNamespace(t, ctx, ns)

	// Seed 5 MLRuns so we can request limit=2 and walk three pages.
	for _, n := range []string{"pg-a", "pg-b", "pg-c", "pg-d", "pg-e"} {
		rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
			buildMLRunCreateBody(n, "pg-pool", "pg-unit"), nil)
		requireStatus(t, rr, http.StatusCreated)
	}

	// Page 1.
	var page1 mlrunListPage
	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns?limit=2", nil, &page1)
	requireStatus(t, rr, http.StatusOK)
	require.NotEmpty(t, page1.ContinueToken, "page 1 must surface continueToken")

	// Page 2 via continueToken.
	var page2 mlrunListPage
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns?limit=2&continue="+page1.ContinueToken, nil, &page2)
	requireStatus(t, rr, http.StatusOK)
	require.NotEmpty(t, page2.ContinueToken, "page 2 must still have continueToken")
	require.NotEqual(t, page1.Items[0].ID, page2.Items[0].ID, "pages must be disjoint")

	// Page 3 — last page; continueToken should be absent.
	var page3 mlrunListPage
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns?limit=2&continue="+page2.ContinueToken, nil, &page3)
	requireStatus(t, rr, http.StatusOK)
	require.Empty(t, page3.ContinueToken, "last page must NOT surface a continueToken")
}

// TestPagination_InvalidContinue returns 400 on a malformed token.
func TestPagination_InvalidContinue(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	rr := doJSON(t, context.Background(), http.MethodGet,
		"/api/v1/namespaces/pg-invalid-ns/mlruns?continue=!!!not-base64!!!", nil, nil)
	requireClientError(t, rr)
}
