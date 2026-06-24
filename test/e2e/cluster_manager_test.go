//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
)

// cluster-manager: tenant lifecycle that fans out to the real tenant-operator.
//
// Pure ResourcePool / ResourceUnit CRUD is NOT exercised here — cluster-manager
// is a stateless REST shell with no reconciler, so that path is fully covered by
// the hermetic integration suite (TestResourcePool_Lifecycle /
// TestResourceUnit_Lifecycle). These e2e tests cover only what needs a real
// cluster: the cross-component chain cluster-manager -> Tenant CR ->
// tenant-operator -> namespace + koord ElasticQuota.

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
