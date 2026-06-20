//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLabelSelector_MLRuns creates two MLRuns with different labels and
// verifies the K8s grammar selector filters correctly against the row's
// labels jsonb.
func TestLabelSelector_MLRuns(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()
	seedResourcePool(t, ctx, "sel-pool", "sel-unit")
	const ns = "sel-ns"
	mustCreateNamespace(t, ctx, ns)

	mk := func(name string, labels map[string]string) {
		body := buildMLRunCreateBody(name, "sel-pool", "sel-unit")
		body["labels"] = labels
		rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns", body, nil)
		requireStatus(t, rr, http.StatusCreated)
	}
	mk("sel-alpha", map[string]string{"axisml.io/project": "p1"})
	mk("sel-beta", map[string]string{"axisml.io/project": "p2"})

	// equality
	expectNames(t, ctx, ns, "axisml.io/project=p1", []string{"sel-alpha"})

	// inequality
	expectNames(t, ctx, ns, "axisml.io/project!=p1", []string{"sel-beta"})

	// existence
	expectNames(t, ctx, ns, "axisml.io/project", []string{"sel-alpha", "sel-beta"})

	// non-existence
	rr := doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns?labelSelector="+url.QueryEscape("!axisml.io/project"),
		nil, nil)
	requireStatus(t, rr, http.StatusOK)
	var resp mlrunListPage
	require.NoError(t, decodeJSONBody(rr, &resp))
	for _, item := range resp.Items {
		require.NotContains(t, item.Labels, "axisml.io/project",
			"mlrun %s must not carry the label", item.Name)
	}

	// invalid selector → 400
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns?labelSelector=%21%21bad",
		nil, nil)
	require.GreaterOrEqual(t, rr.Code, 400)
}

func expectNames(t *testing.T, ctx context.Context, ns, selector string, want []string) {
	t.Helper()
	rr := doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns?labelSelector="+url.QueryEscape(selector),
		nil, nil)
	requireStatus(t, rr, http.StatusOK)
	var resp mlrunListPage
	require.NoError(t, decodeJSONBody(rr, &resp))
	got := map[string]struct{}{}
	for _, item := range resp.Items {
		got[item.Name] = struct{}{}
	}
	for _, w := range want {
		_, ok := got[w]
		require.Truef(t, ok, "selector %q expected to match %q (got %v)", selector, w, keysOf(got))
	}
}

func keysOf(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
