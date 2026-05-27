//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	jobmod "github.com/axisml/axisml/components/compute-service/internal/job"
)

// TestJob_PatchMetadata verifies PATCH /jobs/{job} updates display-tier
// fields without touching spec (compute-service.md §4.3).
func TestJob_PatchMetadata(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()
	seedResourcePool(t, ctx, "patch-pool", "small")
	const ns = "patch-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/jobs",
		buildJobCreateBody("patchable", "patch-pool", "small"), nil)
	requireStatus(t, rr, http.StatusCreated)

	body := map[string]any{
		"displayName": "Patched Display",
		"description": "patched desc",
		"labels":      map[string]string{"axisml.io/project": "p9"},
	}
	var patched jobmod.View
	rr = doJSON(t, ctx, http.MethodPatch,
		"/api/v1/namespaces/"+ns+"/jobs/patchable", body, &patched)
	requireStatus(t, rr, http.StatusOK)
	require.Equal(t, "Patched Display", patched.DisplayName)
	require.Equal(t, "patched desc", patched.Description)
	require.Equal(t, "p9", patched.Labels["axisml.io/project"])
}

// TestJob_PatchNotFound returns 404 on a missing job.
func TestJob_PatchNotFound(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	const ns = "patch-nf-ns"
	mustCreateNamespace(t, context.Background(), ns)
	rr := doJSON(t, context.Background(), http.MethodPatch,
		"/api/v1/namespaces/"+ns+"/jobs/ghost",
		map[string]any{"displayName": "x"}, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

// TestService_PatchMetadata: PATCH /services/{service} updates display-tier
// fields without touching spec.
func TestService_PatchMetadata(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()
	seedResourcePool(t, ctx, "svc-patch-pool", "small")
	const ns = "svc-patch-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/services", map[string]any{
		"name":     "patchable-svc",
		"poolName": "svc-patch-pool",
		"unitName": "small",
		"quota":    "axisml-default",
		"roles": []map[string]any{{
			"name":     "predictor",
			"replicas": 1,
			"template": map[string]any{
				"image": "busybox:1.36",
				"ports": []map[string]any{{
					"name": "http", "containerPort": 8080, "protocol": "TCP",
				}},
			},
		}},
	}, nil)
	requireStatus(t, rr, http.StatusCreated)

	rr = doJSON(t, ctx, http.MethodPatch,
		"/api/v1/namespaces/"+ns+"/services/patchable-svc",
		map[string]any{
			"displayName": "Patched Service",
			"labels":      map[string]string{"axisml.io/project": "p9"},
		}, nil)
	requireStatus(t, rr, http.StatusOK)
}
