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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

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
	if err := ensureSharedTenant(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: shared tenant setup failed: %v\n", err)
		h.close()
		os.Exit(1)
	}

	code := m.Run()

	cleanupSharedTenant(ctx)
	h.close()
	os.Exit(code)
}

// ensureSharedTenant creates the shared tenant via the compute-service API
// (idempotent: an existing tenant is fine) and waits for tenant-operator to
// provision its namespace. Retries over HTTPReadyTimeout to tolerate a
// service that is still becoming ready.
func ensureSharedTenant(ctx context.Context) error {
	cfg := h.cfg
	req := csCreateTenantReq{
		Name:        cfg.SharedTenant,
		DisplayName: "AxisML E2E shared tenant",
		Namespace:   csNamespaceSpec{Name: cfg.SharedNamespace},
		Quotas: []csQuotaSpec{{
			Pool: cfg.DefaultPool,
			Name: "default",
			Max:  map[string]string{"cpu": "8", "memory": "16Gi"},
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
			lastErr = fmt.Errorf("POST /namespaces: status %d: %s", r.status, string(r.body))
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

// sharedNS returns the namespace of the shared test tenant.
func sharedNS() string { return h.cfg.SharedNamespace }

// sharedQuota returns the koord ElasticQuota CR name that workloads in the
// shared tenant must schedule under. The job/service create API stamps this
// string verbatim onto spec.scheduling.quota, so it must be the real CR name
// (tenant-operator composes it as <tenant>-<pool>-<name>). We read it back from
// the cluster rather than reconstruct the naming.
func sharedQuota(t *testing.T, ctx context.Context) string {
	t.Helper()
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
