//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHealthz_APIPort asserts the liveness/readiness probes are also served on
// the API router (not only the dedicated probes server), as JSON HealthStatus —
// clients reaching the API service can health-check it, matching
// compute-service / platform.
func TestHealthz_APIPort(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz"} {
		rr := doRequest(t, http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, rr.Code, "%s -> %s", path, rr.Body.String())
		var hs map[string]any
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &hs), "%s body must be JSON", path)
		require.Equal(t, "ok", hs["status"], path)
	}
}
