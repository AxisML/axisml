//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Preflight — environment readiness. These run first and assert the cluster + Helm
// layers are healthy. If these fail, the rest will too; treat them as the
// preflight gate.

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
	crds := []string{
		"tenants.axisml.io",
		"resourcepools.axisml.io",
		"mlruns.axisml.io",
		"mlservices.axisml.io",
		"elasticquotas.scheduling.sigs.k8s.io",
		"podgroups.scheduling.sigs.k8s.io",
		"httproutes.gateway.networking.k8s.io",
		"gateways.gateway.networking.k8s.io",
	}
	for _, name := range crds {
		name := name
		eventually(t, h.cfg.CRProvisionTimeout, func() error { return crdEstablished(ctx, name) })
	}
}

func TestPreflight_SeedDefaultPool(t *testing.T) {
	ctx := context.Background()
	r := h.clusterManager.mustDo(t, ctx, http.MethodGet, "/api/v1/resourcepools/"+h.cfg.DefaultPool, nil)
	require.True(t, r.is2xx(), "GET default pool: status %d: %s", r.status, string(r.body))

	var pool cmPoolDTO
	require.NoError(t, r.decode(&pool))
	names := map[string]bool{}
	for _, u := range pool.Units {
		names[u.Name] = true
	}
	assert.True(t, names["cpu-small"], "default pool should seed cpu-small unit")
	assert.True(t, names["cpu-medium"], "default pool should seed cpu-medium unit")
}

func TestPreflight_HTTPReachable(t *testing.T) {
	ctx := context.Background()
	// cluster-manager
	r := h.clusterManager.mustDo(t, ctx, http.MethodGet, "/api/v1/resourcepools", nil)
	assert.True(t, r.is2xx(), "cluster-manager list pools: %d", r.status)
	// compute-service
	r = h.computeService.mustDo(t, ctx, http.MethodGet, "/api/v1/namespaces", nil)
	assert.True(t, r.is2xx(), "compute-service list namespaces: %d", r.status)
	// artifact-hub (list models in the shared namespace)
	r = h.artifactHub.mustDo(t, ctx, http.MethodGet, "/api/v1/namespaces/"+sharedNS()+"/models", nil)
	assert.True(t, r.is2xx(), "artifact-hub list models: %d", r.status)
}
