//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	poolv1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
)

// cluster-manager: REST over the cluster-scoped ResourcePool CRD.

func TestClusterManager_CreatePoolRoundTripsToCR(t *testing.T) {
	ctx := context.Background()
	pool := uniqueName("e2e-pool")
	rl := map[string]string{"cpu": "1", "memory": "2Gi"}
	r, err := h.clusterManager.CreateResourcePoolWithResponse(ctx, clustermanager.CreateResourcePoolRequest{
		Name: ptr(pool),
		Units: &[]clustermanager.ServerCreateResourceUnitRequest{{
			Name:     "small",
			Requests: rl,
			Limits:   rl,
		}},
	})
	require.NoError(t, err)
	require.True(t, is2xx(r.StatusCode()), "create pool: %d: %s", r.StatusCode(), string(r.Body))
	t.Cleanup(func() {
		_, _ = h.clusterManager.DeleteResourcePoolWithResponse(context.Background(), pool)
	})

	// The ResourcePool CR materializes in-cluster.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var rp poolv1.ResourcePool
		return h.k8s.Get(ctx, client.ObjectKey{Name: pool}, &rp)
	})
}

func TestClusterManager_AddAndPatchUnit(t *testing.T) {
	ctx := context.Background()
	pool := uniqueName("e2e-pool")
	cp, err := h.clusterManager.CreateResourcePoolWithResponse(ctx, clustermanager.CreateResourcePoolRequest{Name: ptr(pool)})
	require.NoError(t, err)
	require.True(t, is2xx(cp.StatusCode()))
	t.Cleanup(func() {
		_, _ = h.clusterManager.DeleteResourcePoolWithResponse(context.Background(), pool)
	})

	addRL := map[string]string{"cpu": "2"}
	r, err := h.clusterManager.CreateResourceUnitWithResponse(ctx, pool, clustermanager.CreateResourceUnitRequest{
		Name:     ptr("unit-a"),
		Requests: &addRL,
		Limits:   &addRL,
	})
	require.NoError(t, err)
	require.True(t, is2xx(r.StatusCode()), "add unit: %d: %s", r.StatusCode(), string(r.Body))

	// Unit fields are mutable (only the unit name is immutable). PATCH the
	// requests and confirm the change is reflected in the pool.
	patchRL := map[string]string{"cpu": "4"}
	pr, err := h.clusterManager.UpdateResourceUnitWithResponse(ctx, pool, "unit-a", clustermanager.PatchResourceUnitRequest{
		Requests: &patchRL,
		Limits:   &patchRL,
	})
	require.NoError(t, err)
	require.True(t, is2xx(pr.StatusCode()), "patch unit: %d: %s", pr.StatusCode(), string(pr.Body))

	g, err := h.clusterManager.GetResourcePoolWithResponse(ctx, pool)
	require.NoError(t, err)
	require.True(t, is2xx(g.StatusCode()))
	require.NotNil(t, g.JSON200)
	require.Len(t, g.JSON200.Units, 1)
	assert.Equal(t, "4", g.JSON200.Units[0].Requests["cpu"], "patched unit cpu should be 4")
}

func TestClusterManager_DeletePoolGC(t *testing.T) {
	ctx := context.Background()
	pool := uniqueName("e2e-pool")
	cp, err := h.clusterManager.CreateResourcePoolWithResponse(ctx, clustermanager.CreateResourcePoolRequest{Name: ptr(pool)})
	require.NoError(t, err)
	require.True(t, is2xx(cp.StatusCode()))

	d, err := h.clusterManager.DeleteResourcePoolWithResponse(ctx, pool)
	require.NoError(t, err)
	require.True(t, is2xx(d.StatusCode()), "delete pool: %d", d.StatusCode())

	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var rp poolv1.ResourcePool
		err := h.k8s.Get(ctx, client.ObjectKey{Name: pool}, &rp)
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return assertErr("ResourcePool %s still present", pool)
	})
}

// removeTenant best-effort cleans up a tenant: soft-delete via the API, then
// hard-remove the CR + namespace via the admin client (the operator never
// deletes the namespace itself).
func removeTenant(name, ns string) {
	bg := context.Background()
	_, _ = h.deleteTenant(bg, name)
	_ = h.k8s.Delete(bg, &tenantv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: name}})
	_ = h.k8s.Delete(bg, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
}

func TestClusterManager_CreateTenantViaAPI(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, ensureDefaultPool(ctx), "ensure default pool")
	name := uniqueName("e2e-apitenant")
	r, err := h.createTenant(ctx, clustermanager.CreateTenantRequest{
		Name:      ptr(name),
		Namespace: &clustermanager.Apiv1alpha1NamespaceSpec{Name: name},
		Quotas: &[]clustermanager.ServerQuota{{
			Pool:  h.cfg.DefaultPool,
			Units: []clustermanager.ServerQuotaUnit{{UnitName: h.cfg.DefaultUnit, Quantity: 1}},
		}},
	})
	require.NoError(t, err)
	require.True(t, is2xx(r.StatusCode()), "create tenant: %d: %s", r.StatusCode(), string(r.Body))
	t.Cleanup(func() { removeTenant(name, name) })

	// Namespace provisioned by tenant-operator.
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, name) })

	// GET reflects it.
	g, err := h.getTenant(ctx, name)
	require.NoError(t, err)
	require.True(t, is2xx(g.StatusCode()))
	require.NotNil(t, g.JSON200)
	assert.Equal(t, name, g.JSON200.Name)
}

func TestClusterManager_QuotaAllocationViaAPI(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, ensureDefaultPool(ctx), "ensure default pool")
	name := uniqueName("e2e-apiquota")
	r, err := h.createTenant(ctx, clustermanager.CreateTenantRequest{
		Name:      ptr(name),
		Namespace: &clustermanager.Apiv1alpha1NamespaceSpec{Name: name},
	})
	require.NoError(t, err)
	require.True(t, is2xx(r.StatusCode()), "create tenant: %d: %s", r.StatusCode(), string(r.Body))
	t.Cleanup(func() { removeTenant(name, name) })
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, name) })

	// Allocate a quota for the pool via the tenant sub-route.
	q, err := h.setTenantQuota(ctx, name, clustermanager.SetQuotaRequest{
		Pool:  ptr(h.cfg.DefaultPool),
		Units: &[]clustermanager.ServerQuotaUnit{{UnitName: h.cfg.DefaultUnit, Quantity: 1}},
	})
	require.NoError(t, err)
	require.True(t, is2xx(q.StatusCode()), "create quota: %d: %s", q.StatusCode(), string(q.Body))

	// ElasticQuota materializes in the namespace.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		names, err := elasticQuotaNames(ctx, name)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return assertErr("no ElasticQuota in %s yet", name)
		}
		return nil
	})
}
