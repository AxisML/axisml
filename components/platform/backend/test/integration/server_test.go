//go:build integration

package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/components/platform/backend/internal/app"
)

// TestProbeRouter asserts the health/readiness probes boot and report 200.
func TestProbeRouter(t *testing.T) {
	srv := httptest.NewServer(app.NewProbeRouter())
	defer srv.Close()

	for _, path := range []string{"/healthz", "/readyz"} {
		resp, err := http.Get(srv.URL + path)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "GET %s", path)
		require.NoError(t, resp.Body.Close())
	}
}

// TestAPIRouterNotImplemented pins the current contract-only behaviour: the API
// engine boots but every resource route returns 501 until handlers land. When
// real handlers are added, replace this with per-resource coverage.
func TestAPIRouterNotImplemented(t *testing.T) {
	srv := httptest.NewServer(app.NewRouter())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/tenants")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotImplemented, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}
