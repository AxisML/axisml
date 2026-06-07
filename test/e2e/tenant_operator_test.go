//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// tenant-operator. Tenants are created directly as CRs (bypassing
// compute-service) to isolate the operator. These manage their own short-lived
// tenants and never touch the shared `e2e` tenant.

func TestTenant_Provisioning(t *testing.T) {
	ctx := context.Background()
	name := uniqueName("e2e-l1")
	ns := name
	ten := buildTenant(name, ns, h.cfg.DefaultPool, "2", "4Gi")
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

	// RBAC: tenant-operator creates at least one Role in the namespace.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
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
	name := uniqueName("e2e-l1q")
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

	// Bump the quota max to 3 CPU.
	cur, err := getTenantCR(ctx, name)
	require.NoError(t, err)
	cur.Spec.Quotas[0].Max[corev1.ResourceCPU] = resource.MustParse("3")
	require.NoError(t, h.k8s.Update(ctx, cur))

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

func TestTenant_DeletionGC(t *testing.T) {
	ctx := context.Background()
	name := uniqueName("e2e-l1d")
	ns := name
	ten := buildTenant(name, ns, h.cfg.DefaultPool, "1", "2Gi")
	require.NoError(t, h.k8s.Create(ctx, ten))
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, ns) })

	require.NoError(t, h.k8s.Delete(ctx, ten))
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		if err := h.namespaceExists(ctx, ns); isNotFound(err) {
			return nil
		} else if err != nil {
			return err
		}
		return assertErr("namespace %s still present", ns)
	})
}

// TestTenant_ElasticQuotaAdmits is the real-cluster payoff: koord-scheduler must
// actually admit/block pods against the ElasticQuota. [edge]
func TestTenant_ElasticQuotaAdmits(t *testing.T) {
	ctx := context.Background()
	name := uniqueName("e2e-l1adm")
	ns := name
	// 1 CPU quota; two 700m pods cannot both fit.
	ten := buildTenant(name, ns, h.cfg.DefaultPool, "1", "2Gi")
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

	p1 := schedulablePod(ns, "fit-1", eqName, "700m")
	p2 := schedulablePod(ns, "fit-2", eqName, "700m")
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
	// p2 stays Pending (unscheduled) while p1 holds the quota.
	ok, err := podScheduled(ctx, ns, "fit-2")
	require.NoError(t, err)
	assert.False(t, ok, "fit-2 should be blocked by ElasticQuota while fit-1 runs")

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
