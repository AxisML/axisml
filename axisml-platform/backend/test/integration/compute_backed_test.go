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

// TestJobMetadataOnlyPatch covers a metadata-only Job PATCH (no spec): the
// embedded JobSpec's dns1123/min validators must not trip when spec is omitted,
// so JobPatchRequest.Spec is a pointer left nil.
func TestJobMetadataOnlyPatch(t *testing.T) {
	admin := loginAdmin(t)
	code, _ := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "jobowner", "password": "password123", "displayName": "Job Owner",
	})
	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, code)
	code, _ = do(t, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": "job-team", "kubernetesNamespace": "axisml-tenant",
		"displayName": "Job", "initialAdmin": "jobowner",
	})
	require.Equal(t, http.StatusCreated, code)

	code, created := doTenant(t, http.MethodPost, "/api/v1/jobs", admin, "job-team", map[string]any{
		"name": "echo", "displayName": "Echo",
		"spec": map[string]any{
			"backend":  map[string]any{"name": "native", "engine": "job"},
			"poolName": "gpu-a100", "unitName": "small",
			"roles": []map[string]any{{
				"name": "worker", "replicas": 1,
				"template": map[string]any{"image": "busybox:latest", "command": []string{"sh", "-c", "echo hi"}},
			}},
		},
	})
	require.Equal(t, http.StatusCreated, code, "%v", created)

	code, patched := doTenant(t, http.MethodPatch, "/api/v1/jobs/echo", admin, "job-team", map[string]any{
		"displayName": "Renamed", "description": "metadata only",
	})
	require.Equal(t, http.StatusOK, code, "%v", patched)
	assert.Equal(t, "Renamed", patched["displayName"])
}

// TestJobRunForwardsVolumes covers the platform → compute Run path: a Job
// definition declaring a PVC-backed dataset volume must forward that volume
// (with its source) and the matching mount to compute when a run is triggered.
func TestJobRunForwardsVolumes(t *testing.T) {
	admin := loginAdmin(t)
	code, _ := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "runvolowner", "password": "password123", "displayName": "Run Vol Owner",
	})
	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, code)
	code, _ = do(t, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": "runvol-team", "kubernetesNamespace": "axisml-tenant",
		"displayName": "Run Vol", "initialAdmin": "runvolowner",
	})
	require.Equal(t, http.StatusCreated, code)

	code, created := doTenant(t, http.MethodPost, "/api/v1/jobs", admin, "runvol-team", map[string]any{
		"name": "trainer", "displayName": "Trainer",
		"spec": map[string]any{
			"backend":  map[string]any{"name": "native", "engine": "job"},
			"poolName": "gpu-a100", "unitName": "small",
			"roles": []map[string]any{{
				"name": "worker", "replicas": 1,
				"template": map[string]any{
					"image":   "busybox:latest",
					"command": []string{"sh", "-c", "ls /data"},
					"volumes": []map[string]any{
						{"name": "data", "persistentVolumeClaim": map[string]any{"claimName": "dataset-1"}},
					},
					"volumeMounts": []map[string]any{
						{"name": "data", "mountPath": "/data"},
					},
				},
			}},
		},
	})
	require.Equal(t, http.StatusCreated, code, "%v", created)

	// Trigger a run with no overrides — the stored spec (incl. volumes) drives it.
	code, run := doTenant(t, http.MethodPost, "/api/v1/jobs/trainer/runs", admin, "runvol-team", nil)
	require.Equal(t, http.StatusCreated, code, "%v", run)

	tmpl := computeStub.lastRunTmpl()
	require.NotNil(t, tmpl, "compute must have received an MLRun create")

	vols, _ := tmpl["volumes"].([]any)
	require.Len(t, vols, 1, "volume must be forwarded to compute: %v", tmpl)
	vol0, _ := vols[0].(map[string]any)
	assert.Equal(t, "data", vol0["name"])
	pvc, _ := vol0["persistentVolumeClaim"].(map[string]any)
	require.NotNil(t, pvc, "volume source must survive the typed round-trip: %v", vol0)
	assert.Equal(t, "dataset-1", pvc["claimName"])

	mounts, _ := tmpl["volumeMounts"].([]any)
	require.Len(t, mounts, 1, "volumeMount must be forwarded to compute")
	m0, _ := mounts[0].(map[string]any)
	assert.Equal(t, "data", m0["name"])
	assert.Equal(t, "/data", m0["mountPath"])
}

// TestWorkspaceMountsVolume covers the platform → compute Workspace path: a
// workspace referencing an existing data volume must forward it to compute as a
// PVC volume + matching mount in the role template.
func TestWorkspaceMountsVolume(t *testing.T) {
	admin := loginAdmin(t)
	code, _ := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "wsvolowner", "password": "password123", "displayName": "WS Vol Owner",
	})
	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, code)
	code, _ = do(t, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": "wsvol-team", "kubernetesNamespace": "axisml-tenant",
		"displayName": "WS Vol", "initialAdmin": "wsvolowner",
	})
	require.Equal(t, http.StatusCreated, code)

	code, ws := doTenant(t, http.MethodPost, "/api/v1/workspaces", admin, "wsvol-team", map[string]any{
		"name": "notebook", "image": "jupyter:3", "containerPort": 8888,
		"poolName": "gpu-a100", "unitName": "small",
		"volumes": []map[string]any{{"name": "shared-data", "mountPath": "/home/jovyan/work"}},
	})
	require.Equal(t, http.StatusCreated, code, "%v", ws)

	tmpl := computeStub.lastServiceTmpl()
	require.NotNil(t, tmpl, "compute must have received an MLService create")

	vols, _ := tmpl["volumes"].([]any)
	require.Len(t, vols, 1, "workspace volume must be forwarded: %v", tmpl)
	vol0, _ := vols[0].(map[string]any)
	assert.Equal(t, "shared-data", vol0["name"])
	pvc, _ := vol0["persistentVolumeClaim"].(map[string]any)
	require.NotNil(t, pvc, "workspace volume must be a PVC reference: %v", vol0)
	assert.Equal(t, "shared-data", pvc["claimName"])

	mounts, _ := tmpl["volumeMounts"].([]any)
	require.Len(t, mounts, 1)
	m0, _ := mounts[0].(map[string]any)
	assert.Equal(t, "shared-data", m0["name"])
	assert.Equal(t, "/home/jovyan/work", m0["mountPath"])
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
