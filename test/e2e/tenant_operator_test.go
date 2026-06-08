//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// tenant-operator. Tenants are created directly as CRs (bypassing
// compute-service) to isolate the operator. These manage their own short-lived
// tenants and never touch the shared `e2e` tenant.

func TestTenant_Provisioning(t *testing.T) {
	ctx := context.Background()
	name := uniqueName("e2e-tenant")
	ns := name
	ten := buildTenant(name, ns, h.cfg.DefaultPool, "2", "4Gi")
	// The operator only provisions RBAC/SA/Secret/ConfigMap when the Tenant spec
	// requests them via InitResources (a bare tenant gets just namespace +
	// ElasticQuota). Request a ServiceAccount with RBAC so the operator also
	// creates a Role + RoleBinding, which we assert below.
	ten.Spec.InitResources = tenantv1.InitResources{
		ServiceAccounts: []tenantv1.ServiceAccountSpec{{
			Name: "e2e-sa",
			RBAC: &tenantv1.RBACSpec{
				Rules: []rbacv1.PolicyRule{{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "list"},
				}},
			},
		}},
	}
	createTenantCR(t, ctx, ten)

	// Namespace materializes.
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, ns) })

	// ElasticQuota exists with the requested max.
	var eqName string
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		names, err := elasticQuotaNames(ctx, ns)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return assertErr("no ElasticQuota yet in %s", ns)
		}
		eqName = names[0]
		return nil
	})
	max, err := quotaMax(ctx, ns, eqName)
	require.NoError(t, err)
	assert.Equal(t, "2", max["cpu"])

	// InitResources: the ServiceAccount and its Role are provisioned. The
	// operator names per-tenant resources "axisml-<tenant>-<sub>", so match by
	// suffix rather than the bare spec name.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var sas corev1.ServiceAccountList
		if err := h.k8s.List(ctx, &sas, client.InNamespace(ns)); err != nil {
			return err
		}
		found := false
		for i := range sas.Items {
			if strings.HasSuffix(sas.Items[i].Name, "-e2e-sa") {
				found = true
			}
		}
		if !found {
			return assertErr("ServiceAccount *-e2e-sa not found in %s", ns)
		}
		n, err := countRoles(ctx, ns)
		if err != nil {
			return err
		}
		if n < 1 {
			return assertErr("expected >=1 Role in %s, got %d", ns, n)
		}
		return nil
	})

	// Tenant status reaches Active.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		cur, err := getTenantCR(ctx, name)
		if err != nil {
			return err
		}
		if cur.Status.Phase != tenantv1.TenantPhaseActive {
			return assertErr("tenant phase = %q, want Active", cur.Status.Phase)
		}
		return nil
	})
}

func TestTenant_QuotaUpdatePropagates(t *testing.T) {
	ctx := context.Background()
	name := uniqueName("e2e-quota")
	ns := name
	ten := buildTenant(name, ns, h.cfg.DefaultPool, "2", "4Gi")
	createTenantCR(t, ctx, ten)
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, ns) })

	var eqName string
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		names, err := elasticQuotaNames(ctx, ns)
		if err != nil || len(names) == 0 {
			return assertErr("no ElasticQuota yet (err=%v)", err)
		}
		eqName = names[0]
		return nil
	})

	// Bump the quota max to 3 CPU. Retry on conflict — the operator writes
	// status concurrently, so a naive get+update races.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		cur, err := getTenantCR(ctx, name)
		if err != nil {
			return err
		}
		cur.Spec.Quotas[0].Max[corev1.ResourceCPU] = resource.MustParse("3")
		return h.k8s.Update(ctx, cur)
	})

	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		max, err := quotaMax(ctx, ns, eqName)
		if err != nil {
			return err
		}
		if max["cpu"] != "3" {
			return assertErr("ElasticQuota max cpu = %q, want 3", max["cpu"])
		}
		return nil
	})
}

// TestTenant_DeletionRetainsNamespace verifies the operator's intended deletion
// semantics: deleting a Tenant CR removes the CR but DELIBERATELY retains the
// namespace (design §6.1: "never delete, no ownerReference" — so user workloads
// and data survive an accidental Tenant deletion). The test runner cleans up the
// orphaned namespace itself.
func TestTenant_DeletionRetainsNamespace(t *testing.T) {
	ctx := context.Background()
	name := uniqueName("e2e-delete")
	ns := name
	ten := buildTenant(name, ns, h.cfg.DefaultPool, "1", "2Gi")
	require.NoError(t, h.k8s.Create(ctx, ten))
	t.Cleanup(func() {
		_ = h.k8s.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
	})
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, ns) })

	require.NoError(t, h.k8s.Delete(ctx, ten))

	// Tenant CR is gone...
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		if _, err := getTenantCR(ctx, name); isNotFound(err) {
			return nil
		} else if err != nil {
			return err
		}
		return assertErr("tenant %s still present", name)
	})
	// ...but the namespace is intentionally retained.
	require.NoError(t, h.namespaceExists(ctx, ns), "namespace must survive tenant deletion")
}

// TestTenant_ElasticQuotaAdmits is the real-cluster payoff: koord-scheduler must
// actually admit/block pods against the ElasticQuota. [edge]
func TestTenant_ElasticQuotaAdmits(t *testing.T) {
	ctx := context.Background()
	name := uniqueName("e2e-admin")
	ns := name
	// Small quota with small pods: two 200m pods (400m) exceed the 300m max so
	// the second is blocked, while 200m always fits the node. Set min=200m so the
	// quota has GUARANTEED capacity for the first pod — koord-scheduler then
	// admits fit-1 deterministically regardless of how many other quotas the rest
	// of the suite has churned (with min=0 admission depends on borrowing shared
	// capacity, which gets flaky under churn).
	ten := buildTenant(name, ns, h.cfg.DefaultPool, "300m", "512Mi")
	ten.Spec.Quotas[0].Min = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("200m"),
		corev1.ResourceMemory: resource.MustParse("256Mi"),
	}
	createTenantCR(t, ctx, ten)
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, ns) })

	var eqName string
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		names, err := elasticQuotaNames(ctx, ns)
		if err != nil || len(names) == 0 {
			return assertErr("no ElasticQuota yet (err=%v)", err)
		}
		eqName = names[0]
		return nil
	})

	p1 := schedulablePod(ns, "fit-1", eqName, "200m")
	p2 := schedulablePod(ns, "fit-2", eqName, "200m")
	require.NoError(t, h.k8s.Create(ctx, p1))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), p1) })
	require.NoError(t, h.k8s.Create(ctx, p2))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), p2) })

	// p1 gets scheduled.
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		ok, err := podScheduled(ctx, ns, "fit-1")
		if err != nil {
			return err
		}
		if !ok {
			return assertErr("fit-1 not scheduled")
		}
		return nil
	})
	// p2 must STAY Pending while p1 holds the quota. A single immediate check
	// would pass simply because the scheduler hasn't processed p2 yet (a false
	// positive that would hide a quota-enforcement regression), so assert it
	// stays unscheduled across a window.
	consistently(t, 15*time.Second, func() error {
		ok, err := podScheduled(ctx, ns, "fit-2")
		if err != nil {
			return err
		}
		if ok {
			return assertErr("fit-2 scheduled while fit-1 holds the quota (ElasticQuota not enforced)")
		}
		return nil
	})

	// Free the quota; p2 should then schedule.
	require.NoError(t, h.k8s.Delete(ctx, p1))
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		ok, err := podScheduled(ctx, ns, "fit-2")
		if err != nil {
			return err
		}
		if !ok {
			return assertErr("fit-2 not scheduled after fit-1 released quota")
		}
		return nil
	})
}
