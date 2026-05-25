//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
)

func TestService_NotFound(t *testing.T) {
	ctx := context.Background()
	const ns = "services-nf-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/services/ghost", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodPost,
		"/api/v1/namespaces/"+ns+"/services/ghost/scale",
		map[string]any{"replicas": 2}, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

func TestService_ScaleValidation(t *testing.T) {
	// gte=0 fires at bind time, before the row lookup — any path works.
	rr := doJSON(t, context.Background(), http.MethodPost,
		"/api/v1/namespaces/x-ns/services/whatever/scale",
		map[string]any{"replicas": -1}, nil)
	requireClientError(t, rr)
}
