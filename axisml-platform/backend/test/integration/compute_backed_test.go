//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMLServiceCreateModelPrecheck covers the served-model preflight: a
// non-Ready model is rejected, and a Ready model is created with its resolved
// pull URI injected as env.
func TestMLServiceCreateModelPrecheck(t *testing.T) {
	admin := loginAdmin(t)
	code, _ := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "svcowner", "password": "password123", "displayName": "Svc Owner",
	})
	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, code)
	code, _ = do(t, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": "svc-team", "kubernetesNamespace": "axisml-tenant",
		"displayName": "Svc", "initialAdmin": "svcowner",
	})
	require.Equal(t, http.StatusCreated, code)

	body := func() map[string]any {
		return map[string]any{
			"name": "chatbot", "modelName": "m", "modelVersion": "v1", "image": "serve:1",
			"ports":    []map[string]any{{"name": "http", "port": 8080}},
			"poolName": "gpu-a100", "unitName": "small", "replicas": 1,
		}
	}

	// Model not Ready → rejected before any compute call.
	artStub.seedModel("svc-team", "m", "v1", "Pending")
	code, prob := doTenant(t, http.MethodPost, "/api/v1/mlservices", admin, "svc-team", body())
	require.Equal(t, http.StatusBadRequest, code, "%v", prob)

	// Model Ready → created, with the resolved model URI injected as env.
	artStub.seedModel("svc-team", "m", "v1", "Ready")
	code, svc := doTenant(t, http.MethodPost, "/api/v1/mlservices", admin, "svc-team", body())
	require.Equal(t, http.StatusCreated, code, "%v", svc)
	assert.Equal(t, "chatbot", svc["name"])

	var names []string
	for _, e := range computeStub.lastServiceEnv() {
		if m, ok := e.(map[string]any); ok {
			if n, ok := m["name"].(string); ok {
				names = append(names, n)
			}
		}
	}
	assert.Contains(t, names, "AXISML_MODEL_URI", "resolved model URI must be injected")
}

// TestPlatformMetricsProxy covers the four metrics endpoints proxying to
// compute-service N1.
func TestPlatformMetricsProxy(t *testing.T) {
	admin := loginAdmin(t)
	code, _ := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "metricsowner", "password": "password123", "displayName": "Metrics Owner",
	})
	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, code)
	code, _ = do(t, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": "metrics-team", "kubernetesNamespace": "axisml-tenant",
		"displayName": "Metrics", "initialAdmin": "metricsowner",
	})
	require.Equal(t, http.StatusCreated, code)

	for _, path := range []string{
		"/api/v1/jobs/somejob/runs/somerun/metrics?metric=cpu_util&range=1h",
		"/api/v1/experiments/someexp/runs/somerun/metrics?metric=cpu_util&range=1h",
		"/api/v1/mlservices/somesvc/metrics?metric=cpu_util&range=1h",
		"/api/v1/trafficpolicies/somepolicy/metrics?metric=cpu_util&range=1h",
	} {
		code, series := doTenant(t, http.MethodGet, path, admin, "metrics-team", nil)
		require.Equal(t, http.StatusOK, code, "%s -> %v", path, series)
		assert.Equal(t, "cpu_util", series["metric"], path)
		pts, _ := series["series"].([]any)
		require.Len(t, pts, 1, path)
	}
}
