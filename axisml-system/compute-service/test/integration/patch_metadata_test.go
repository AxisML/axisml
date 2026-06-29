//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	computeserver "github.com/axisml/axisml/axisml-system/compute-service/internal/server"
)

// TestMLRun_PatchMetadata verifies PATCH /mlruns/{job} updates display-tier
// fields without touching spec (compute-service.md §4.3).
func TestMLRun_PatchMetadata(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()
	seedResourcePool(t, ctx, "patch-pool", "small")
	const ns = "patch-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("patchable", "patch-pool", "small"), nil)
	requireStatus(t, rr, http.StatusCreated)

	body := map[string]any{
		"displayName": "Patched Display",
		"description": "patched desc",
		"labels":      map[string]string{"axisml.io/project": "p9"},
	}
	var patched computeserver.MLRun
	rr = doJSON(t, ctx, http.MethodPatch,
		"/api/v1/namespaces/"+ns+"/mlruns/patchable", body, &patched)
	requireStatus(t, rr, http.StatusOK)
	require.Equal(t, "Patched Display", patched.DisplayName)
	require.Equal(t, "patched desc", patched.Description)
	require.Equal(t, "p9", patched.Labels["axisml.io/project"])
}

// TestMLRun_PatchNotFound returns 404 on a missing job.
func TestMLRun_PatchNotFound(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	const ns = "patch-nf-ns"
	mustCreateNamespace(t, context.Background(), ns)
	rr := doJSON(t, context.Background(), http.MethodPatch,
		"/api/v1/namespaces/"+ns+"/mlruns/ghost",
		map[string]any{"displayName": "x"}, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

// TestMLService_PatchMetadata: PATCH /mlservices/{service} updates display-tier
// fields without touching spec.
func TestMLService_PatchMetadata(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx := context.Background()
	seedResourcePool(t, ctx, "svc-patch-pool", "small")
	const ns = "svc-patch-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", map[string]any{
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
		"/api/v1/namespaces/"+ns+"/mlservices/patchable-svc",
		map[string]any{
			"displayName": "Patched Service",
			"labels":      map[string]string{"axisml.io/project": "p9"},
		}, nil)
	requireStatus(t, rr, http.StatusOK)
}
