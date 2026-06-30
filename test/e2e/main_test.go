//go:build (e2e || standard) && !lite

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
)

// requiredCRDs are the CustomResourceDefinitions the operators reconcile (plus
// the external ones they consume). A workload test against a cluster missing one
// of these hangs on "no matches for kind X" rather than failing clearly, so the
// readiness gate waits for all of them to be Established before any test runs.
// The TestPreflight_CRDsEstablished diagnostic asserts the same list.
var requiredCRDs = []string{
	"tenants.axisml.io",
	"resourcepools.axisml.io",
	"mlruns.axisml.io",
	"mlservices.axisml.io",
	"elasticquotas.scheduling.x-k8s.io",
	"podgroups.scheduling.x-k8s.io",
	"httproutes.gateway.networking.k8s.io",
	"gateways.gateway.networking.k8s.io",
}

// gateReady is the fail-fast readiness gate. It checks the conditions whose
// absence would otherwise make workload tests hang rather than fail clearly: the
// required CRDs must be Established, and the default ResourcePool the tenants'
// quotas reference must exist. The HTTP services are already proven reachable by
// newSuite's port-forwards.
func gateReady(ctx context.Context) error {
	deadline := time.Now().Add(h.cfg.CRProvisionTimeout)
	for _, name := range requiredCRDs {
		for {
			err := crdEstablished(ctx, name)
			if err == nil {
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("CRD %s not Established within %s: %w", name, h.cfg.CRProvisionTimeout, err)
			}
			time.Sleep(h.cfg.PollInterval)
		}
	}
	if err := ensureDefaultPool(ctx); err != nil {
		return fmt.Errorf("default ResourcePool not available: %w", err)
	}
	return nil
}

// provisionTenant creates a fresh tenant via the cluster-manager API, waits for
// tenant-operator to materialize its namespace + ElasticQuota, registers
// teardown, and returns the namespace plus the ElasticQuota CR name that
// workloads in it must schedule under. Each workload test file calls this once
// in its top-level Test and shares (ns, quota) across its subtests — explicit,
// scoped state in place of a process-global shared tenant.
func provisionTenant(t *testing.T) (ns, quota string) {
	t.Helper()
	ctx := context.Background()

	name := uniqueName("e2e")
	ns = name
	r, err := h.createTenant(ctx, clustermanager.CreateTenantRequest{
		Name:      ptr(name),
		Namespace: &clustermanager.Apiv1alpha1NamespaceSpec{Name: ns},
		// cpu-small has requests==limits, so the fold sets ElasticQuota
		// min==max==quantity. Keep it small: a large guaranteed `min` reserves
		// node capacity in axisml-scheduler and starves admission. 4 leaves headroom for the
		// file's subtests (each ≤2 cpu-small pods, torn down before the next).
		Quotas: &[]clustermanager.ServerQuota{{
			Pool:  h.cfg.DefaultPool,
			Units: []clustermanager.ServerQuotaUnit{{UnitName: h.cfg.DefaultUnit, Quantity: 4}},
		}},
	})
	require.NoError(t, err)
	require.True(t, is2xx(r.StatusCode()), "create tenant %s: %d: %s", name, r.StatusCode(), string(r.Body))
	t.Cleanup(func() { removeTenant(name, ns) })

	// Wait for the namespace AND its ElasticQuota — subtests resolve the quota
	// immediately, so racing on just the namespace would flake.
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, ns) })
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		names, err := elasticQuotaNames(ctx, ns)
		if err != nil || len(names) == 0 {
			return assertErr("no ElasticQuota in %s yet (err=%v)", ns, err)
		}
		quota = names[0]
		return nil
	})
	return ns, quota
}

// defaultPoolUnits mirrors the authoritative `default` pool seed (Helm
// post-install hook / `cluster-manager bootstrap`): cpu-small 1/2Gi and
// cpu-medium 4/8Gi. The preflight test asserts both units are present, so the
// harness fallback must seed the same shape.
func defaultPoolUnits() []clustermanager.ServerCreateResourceUnitRequest {
	u := func(name, cpu, mem string) clustermanager.ServerCreateResourceUnitRequest {
		rl := map[string]string{"cpu": cpu, "memory": mem}
		return clustermanager.ServerCreateResourceUnitRequest{Name: name, Requests: rl, Limits: rl}
	}
	return []clustermanager.ServerCreateResourceUnitRequest{u("cpu-small", "1", "2Gi"), u("cpu-medium", "4", "8Gi")}
}

// ensureDefaultPool idempotently ensures the default ResourcePool with its two
// units exists so a tenant's quota can fold into an ElasticQuota. A properly
// installed cluster seeds this pool (Helm hook); the harness recreates it as a
// fallback for clusters where the seed did not run. The pool and unit creates
// tolerate a 409 (already present from the seed, a prior run, or another caller).
func ensureDefaultPool(ctx context.Context) error {
	cfg := h.cfg
	units := defaultPoolUnits()
	pool := clustermanager.CreateResourcePoolRequest{Name: ptr(cfg.DefaultPool), Units: &units}

	deadline := time.Now().Add(cfg.HTTPReadyTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		r, err := h.clusterManager.CreateResourcePoolWithResponse(ctx, pool)
		if err != nil {
			lastErr = err
		} else if is2xx(r.StatusCode()) || r.StatusCode() == http.StatusConflict {
			lastErr = nil
			break
		} else {
			lastErr = fmt.Errorf("POST /resourcepools: status %d: %s", r.StatusCode(), string(r.Body))
		}
		time.Sleep(cfg.PollInterval)
	}
	if lastErr != nil {
		return lastErr
	}

	// The pool may have pre-existed without one of the units; ensure each.
	for _, u := range units {
		body := clustermanager.CreateResourceUnitRequest{
			Name:     ptr(u.Name),
			Requests: ptr(u.Requests),
			Limits:   ptr(u.Limits),
		}
		r, err := h.clusterManager.CreateResourceUnitWithResponse(ctx, cfg.DefaultPool, body)
		if err != nil {
			return err
		}
		if !is2xx(r.StatusCode()) && r.StatusCode() != http.StatusConflict {
			return fmt.Errorf("POST unit %q: status %d: %s", u.Name, r.StatusCode(), string(r.Body))
		}
	}
	return nil
}
