//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// sharedTenantReady reports whether the System-layer shared tenant provisioned
// (workload tests gate on it; Platform tests don't need it).
var sharedTenantReady bool

// TestMain wires up the process-wide harness: a K8s client against the ambient
// kubeconfig, port-forwards to the three HTTP services, and the shared `e2e`
// test tenant that the workload tests run inside. It assumes the cluster and
// the infra + system Helm layers are already installed (see design §2) and
// fails fast with guidance if they are not.
func TestMain(m *testing.M) {
	s, err := newSuite()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: cannot reach the axisml cluster: %v\n", err)
		fmt.Fprintf(os.Stderr, "e2e: bring it up first: `make cluster-up && make helm-install`\n")
		os.Exit(1)
	}
	h = s

	ctx := context.Background()
	// The shared tenant backs the System-layer workload tests. Its setup is
	// non-fatal: the Platform tests provision their own tenants via the Platform
	// API, so they run regardless. Workload tests that need the shared tenant
	// detect its absence and skip.
	if err := ensureSharedTenant(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: WARN shared tenant setup failed (workload tests will skip): %v\n", err)
		sharedTenantReady = false
	} else {
		sharedTenantReady = true
	}

	code := m.Run()

	cleanupSharedTenant(ctx)
	h.close()
	os.Exit(code)
}

// ensureSharedTenant creates the shared tenant via the cluster-manager API
// (idempotent: an existing tenant is fine) and waits for tenant-operator to
// provision its namespace + ElasticQuota. It first ensures the default
// ResourcePool/unit the quota references exists. Retries over HTTPReadyTimeout
// to tolerate a service that is still becoming ready.
func ensureSharedTenant(ctx context.Context) error {
	cfg := h.cfg
	if err := ensureDefaultPool(ctx); err != nil {
		return fmt.Errorf("ensure default pool: %w", err)
	}
	req := cmCreateTenantReq{
		Name:        cfg.SharedTenant,
		DisplayName: "AxisML E2E shared tenant",
		Namespace:   cmNamespaceSpec{Name: cfg.SharedNamespace},
		// cpu-small has requests==limits, so the fold sets ElasticQuota
		// min==max==quantity. Keep the quantity small: a large guaranteed `min`
		// reserves node capacity in koord and starves admission for the other
		// tenants the suite churns. 4 leaves ample headroom for the workload
		// pods (≤2 concurrent cpu-small) while keeping min modest on the node.
		Quotas: []cmQuota{{
			Pool:  cfg.DefaultPool,
			Units: []cmQuotaUnit{{UnitName: cfg.DefaultUnit, Quantity: 4}},
		}},
	}

	deadline := time.Now().Add(cfg.HTTPReadyTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		r, err := h.createTenant(ctx, req)
		if err != nil {
			lastErr = err
		} else if r.is2xx() || r.status == http.StatusConflict {
			lastErr = nil
			break
		} else {
			lastErr = fmt.Errorf("POST /tenants: status %d: %s", r.status, string(r.body))
		}
		time.Sleep(cfg.PollInterval)
	}
	if lastErr != nil {
		return lastErr
	}

	// Wait for the namespace AND its ElasticQuota to materialize. Workload
	// tests resolve the quota immediately, so racing on just the namespace would
	// flake.
	deadline = time.Now().Add(cfg.CRProvisionTimeout)
	for time.Now().Before(deadline) {
		if err := h.namespaceExists(ctx, cfg.SharedNamespace); err == nil {
			if names, qerr := elasticQuotaNames(ctx, cfg.SharedNamespace); qerr == nil && len(names) > 0 {
				return nil
			}
		}
		time.Sleep(cfg.PollInterval)
	}
	return fmt.Errorf("shared tenant %q namespace/quota not provisioned within %s", cfg.SharedTenant, cfg.CRProvisionTimeout)
}

// defaultPoolUnits mirrors the authoritative `default` pool seed (Helm
// post-install hook / `cluster-manager bootstrap`): cpu-small 1/2Gi and
// cpu-medium 4/8Gi. The preflight test asserts both units are present, so the
// harness fallback must seed the same shape.
func defaultPoolUnits() []cmCreateUnitReq {
	u := func(name, cpu, mem string) cmCreateUnitReq {
		rl := corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(mem),
		}
		return cmCreateUnitReq{Name: name, Requests: rl, Limits: rl}
	}
	return []cmCreateUnitReq{u("cpu-small", "1", "2Gi"), u("cpu-medium", "4", "8Gi")}
}

// ensureDefaultPool idempotently ensures the default ResourcePool with its two
// units exists so the shared tenant's quota can fold into an ElasticQuota. A
// properly installed cluster seeds this pool (Helm hook); the harness recreates
// it as a fallback for clusters where the seed did not run. The pool and unit
// creates tolerate a 409 (already present from the seed, a prior run, or
// another caller).
func ensureDefaultPool(ctx context.Context) error {
	cfg := h.cfg
	units := defaultPoolUnits()
	pool := cmCreatePoolReq{Name: cfg.DefaultPool, Units: units}

	deadline := time.Now().Add(cfg.HTTPReadyTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		r, err := h.clusterManager.do(ctx, http.MethodPost, "/api/v1/resourcepools", pool)
		if err != nil {
			lastErr = err
		} else if r.is2xx() || r.status == http.StatusConflict {
			lastErr = nil
			break
		} else {
			lastErr = fmt.Errorf("POST /resourcepools: status %d: %s", r.status, string(r.body))
		}
		time.Sleep(cfg.PollInterval)
	}
	if lastErr != nil {
		return lastErr
	}

	// The pool may have pre-existed without one of the units; ensure each.
	for _, u := range units {
		r, err := h.clusterManager.do(ctx, http.MethodPost, "/api/v1/resourcepools/"+cfg.DefaultPool+"/units", u)
		if err != nil {
			return err
		}
		if !r.is2xx() && r.status != http.StatusConflict {
			return fmt.Errorf("POST unit %q: status %d: %s", u.Name, r.status, string(r.body))
		}
	}
	return nil
}

// requireSharedTenant skips the calling test when the shared System-layer tenant
// failed to provision (its setup is non-fatal in TestMain). Workload tests call
// this so a missing tenant skips cleanly instead of failing on a missing
// namespace/quota.
func requireSharedTenant(t *testing.T) {
	t.Helper()
	if !sharedTenantReady {
		t.Skip("shared tenant not provisioned (see TestMain WARN); skipping workload test")
	}
}

// cleanupSharedTenant best-effort removes the shared tenant at the end of the
// run. Failures are logged, not fatal — leaving it behind is harmless and aids
// post-mortem debugging.
func cleanupSharedTenant(ctx context.Context) {
	if os.Getenv("E2E_KEEP_TENANT") != "" {
		return
	}
	// Soft-delete via the API (keeps compute-service's view consistent)...
	_, _ = h.deleteTenant(ctx, h.cfg.SharedTenant)
	// ...then hard-remove the CR + namespace via the admin client so the next
	// run starts clean (the operator never deletes the namespace itself).
	ten := &tenantv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: h.cfg.SharedTenant}}
	_ = h.k8s.Delete(ctx, ten)
	nsObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: h.cfg.SharedNamespace}}
	_ = h.k8s.Delete(ctx, nsObj)
}

// sharedNS returns the namespace of the shared test tenant, skipping the test
// when that tenant failed to provision (its setup is non-fatal in TestMain).
func sharedNS(t *testing.T) string {
	t.Helper()
	requireSharedTenant(t)
	return h.cfg.SharedNamespace
}

// sharedQuota returns the koord ElasticQuota CR name that workloads in the
// shared tenant must schedule under. The job/service create API stamps this
// string verbatim onto spec.scheduling.quota, so it must be the real CR name
// (tenant-operator composes it as <tenant>-<pool>-<name>). We read it back from
// the cluster rather than reconstruct the naming. Skips when the shared tenant
// is absent.
func sharedQuota(t *testing.T, ctx context.Context) string {
	t.Helper()
	requireSharedTenant(t)
	names := listQuotaNames(t, ctx, h.cfg.SharedNamespace)
	if len(names) == 0 {
		t.Fatalf("shared tenant namespace %q has no ElasticQuota", h.cfg.SharedNamespace)
	}
	return names[0]
}

// listQuotaNames returns the names of all ElasticQuota CRs in a namespace.
func listQuotaNames(t *testing.T, ctx context.Context, ns string) []string {
	t.Helper()
	names, err := elasticQuotaNames(ctx, ns)
	if err != nil {
		t.Fatalf("list ElasticQuota in %s: %v", ns, err)
	}
	return names
}
