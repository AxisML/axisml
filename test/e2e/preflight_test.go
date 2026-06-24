//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Preflight — environment-readiness diagnostics. The actual fail-fast gate runs
// in TestMain (gateReady) before any test, because test execution order is by
// file name and cannot be relied on to run these first. These remain as
// individually-runnable checks (`-run TestPreflight_…`) for pinpointing which
// part of the environment is unhealthy.

func TestPreflight_InfraPodsReady(t *testing.T) {
	ctx := context.Background()
	ns := h.cfg.InfraNamespace
	// Required infra components (matched by pod-name substring).
	for _, sub := range []string{"database", "zot", "rustfs", "koord-scheduler", "koord-manager", "envoy"} {
		requireReadyPod(t, ctx, ns, sub)
	}
	// GPU operator is optional on minikube — log only.
	logReadyPod(t, ctx, ns, "gpu-operator")
}

func TestPreflight_SystemPodsReady(t *testing.T) {
	ctx := context.Background()
	ns := h.cfg.SystemNamespace
	for _, sub := range []string{
		"tenant-operator", "compute-operator",
		"cluster-manager", "compute-service", "artifact-hub",
	} {
		requireReadyPod(t, ctx, ns, sub)
	}
}

func TestPreflight_CRDsEstablished(t *testing.T) {
	ctx := context.Background()
	for _, name := range requiredCRDs {
		name := name
		eventually(t, h.cfg.CRProvisionTimeout, func() error { return crdEstablished(ctx, name) })
	}
}

func TestPreflight_SeedDefaultPool(t *testing.T) {
	ctx := context.Background()
	r, err := h.clusterManager.GetResourcePoolWithResponse(ctx, h.cfg.DefaultPool)
	require.NoError(t, err)
	require.True(t, is2xx(r.StatusCode()), "GET default pool: status %d: %s", r.StatusCode(), string(r.Body))
	require.NotNil(t, r.JSON200)

	names := map[string]bool{}
	for _, u := range r.JSON200.Units {
		names[u.Name] = true
	}
	assert.True(t, names["cpu-small"], "default pool should seed cpu-small unit")
	assert.True(t, names["cpu-medium"], "default pool should seed cpu-medium unit")
}

func TestPreflight_HTTPReachable(t *testing.T) {
	ctx := context.Background()
	// compute-service / artifact-hub serve no top-level collection; list within a
	// real namespace (tenant CRUD lives in cluster-manager).
	ns, _ := provisionTenant(t)
	// cluster-manager
	rp, err := h.clusterManager.ListResourcePoolsWithResponse(ctx, nil)
	require.NoError(t, err)
	assert.True(t, is2xx(rp.StatusCode()), "cluster-manager list pools: %d", rp.StatusCode())
	// compute-service (list mlruns in the namespace)
	mr, err := h.computeService.ListMLRunsWithResponse(ctx, ns, nil)
	require.NoError(t, err)
	assert.True(t, is2xx(mr.StatusCode()), "compute-service list mlruns: %d", mr.StatusCode())
	// artifact-hub (list models in the namespace)
	am, err := h.artifactHub.ListModelsWithResponse(ctx, ns, nil)
	require.NoError(t, err)
	assert.True(t, is2xx(am.StatusCode()), "artifact-hub list models: %d", am.StatusCode())
}
