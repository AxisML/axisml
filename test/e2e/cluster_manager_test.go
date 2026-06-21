//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	poolv1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// cluster-manager: REST over the cluster-scoped ResourcePool CRD.

func TestClusterManager_CreatePoolRoundTripsToCR(t *testing.T) {
	ctx := context.Background()
	pool := uniqueName("e2e-pool")
	req := cmCreatePoolReq{
		Name: pool,
		Units: []cmCreateUnitReq{{
			Name:     "small",
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("2Gi")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("2Gi")},
		}},
	}
	r := h.clusterManager.mustDo(t, ctx, http.MethodPost, "/api/v1/resourcepools", req)
	require.True(t, r.is2xx(), "create pool: %d: %s", r.status, string(r.body))
	t.Cleanup(func() {
		_, _ = h.clusterManager.do(context.Background(), http.MethodDelete, "/api/v1/resourcepools/"+pool, nil)
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
	require.True(t, h.clusterManager.mustDo(t, ctx, http.MethodPost, "/api/v1/resourcepools",
		cmCreatePoolReq{Name: pool}).is2xx())
	t.Cleanup(func() {
		_, _ = h.clusterManager.do(context.Background(), http.MethodDelete, "/api/v1/resourcepools/"+pool, nil)
	})

	unitPath := "/api/v1/resourcepools/" + pool + "/units"
	add := cmCreateUnitReq{
		Name:     "unit-a",
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
	}
	r := h.clusterManager.mustDo(t, ctx, http.MethodPost, unitPath, add)
	require.True(t, r.is2xx(), "add unit: %d: %s", r.status, string(r.body))

	// Unit fields are mutable (only the unit name is immutable). PATCH the
	// requests and confirm the change is reflected in the pool.
	patch := map[string]any{"requests": map[string]string{"cpu": "4"}, "limits": map[string]string{"cpu": "4"}}
	r = h.clusterManager.mustDo(t, ctx, http.MethodPatch, unitPath+"/unit-a", patch)
	require.True(t, r.is2xx(), "patch unit: %d: %s", r.status, string(r.body))

	g := h.clusterManager.mustDo(t, ctx, http.MethodGet, "/api/v1/resourcepools/"+pool, nil)
	var dto cmPoolDTO
	require.NoError(t, g.decode(&dto))
	require.Len(t, dto.Units, 1)
	assert.Equal(t, "4", dto.Units[0].Requests.Cpu().String(), "patched unit cpu should be 4")
}

func TestClusterManager_DeletePoolGC(t *testing.T) {
	ctx := context.Background()
	pool := uniqueName("e2e-pool")
	require.True(t, h.clusterManager.mustDo(t, ctx, http.MethodPost, "/api/v1/resourcepools",
		cmCreatePoolReq{Name: pool}).is2xx())

	r := h.clusterManager.mustDo(t, ctx, http.MethodDelete, "/api/v1/resourcepools/"+pool, nil)
	require.True(t, r.is2xx(), "delete pool: %d", r.status)

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

func TestClusterManager_MissingIdentity401(t *testing.T) {
	ctx := context.Background()
	r, err := h.clusterManager.doNoAuth(ctx, http.MethodGet, "/api/v1/resourcepools", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, r.status, "missing %s must be 401", headerUser)
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
	r, err := h.createTenant(ctx, cmCreateTenantReq{
		Name:      name,
		Namespace: cmNamespaceSpec{Name: name},
		Quotas:    []cmQuota{{Pool: h.cfg.DefaultPool, Units: []cmQuotaUnit{{UnitName: h.cfg.DefaultUnit, Quantity: 1}}}},
	})
	require.NoError(t, err)
	require.True(t, r.is2xx(), "create tenant: %d: %s", r.status, string(r.body))
	t.Cleanup(func() { removeTenant(name, name) })

	// Namespace provisioned by tenant-operator.
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, name) })

	// GET reflects it.
	g, err := h.getTenant(ctx, name)
	require.NoError(t, err)
	require.True(t, g.is2xx())
	var tn cmTenantResp
	require.NoError(t, g.decode(&tn))
	assert.Equal(t, name, tn.Name)
}

func TestClusterManager_QuotaAllocationViaAPI(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, ensureDefaultPool(ctx), "ensure default pool")
	name := uniqueName("e2e-apiquota")
	r, err := h.createTenant(ctx, cmCreateTenantReq{
		Name:      name,
		Namespace: cmNamespaceSpec{Name: name},
	})
	require.NoError(t, err)
	require.True(t, r.is2xx(), "create tenant: %d: %s", r.status, string(r.body))
	t.Cleanup(func() { removeTenant(name, name) })
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, name) })

	// Allocate a quota for the pool via the tenant sub-route.
	q := cmSetQuotaReq{Pool: h.cfg.DefaultPool, Units: []cmQuotaUnit{{UnitName: h.cfg.DefaultUnit, Quantity: 1}}}
	r, err = h.setTenantQuota(ctx, name, q)
	require.NoError(t, err)
	require.True(t, r.is2xx(), "create quota: %d: %s", r.status, string(r.body))

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
