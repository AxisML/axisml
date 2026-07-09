package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
)

func TestApplyTenantPatch(t *testing.T) {
	t.Run("updates all fields and stamps user", func(t *testing.T) {
		cr := &tenantv1alpha1.Tenant{}
		cr.Annotations = map[string]string{
			srv.LastModifiedByAnnotation: "old",
			srv.QuotasAnnotation:         "quota-json",
			"team":                       "gone",
		}
		ir := tenantv1alpha1.InitResources{Secrets: []tenantv1alpha1.SecretSpec{{Name: "s"}}}
		req := srv.PatchTenantRequest{
			NamespaceLabels:      map[string]string{"nl": "1"},
			NamespaceAnnotations: map[string]string{"na": "1"},
			InitResources:        &ir,
			Labels:               map[string]string{"tier": "prod"},
			Annotations:          map[string]string{"team": "ml"},
		}
		applyTenantPatch(cr, req, "bob")

		assert.Equal(t, map[string]string{"nl": "1"}, cr.Spec.Namespace.Labels)
		assert.Equal(t, map[string]string{"na": "1"}, cr.Spec.Namespace.Annotations)
		require.Len(t, cr.Spec.InitResources.Secrets, 1)
		assert.Equal(t, map[string]string{"tier": "prod"}, cr.Labels)
		// User annotations replace user-visible keys; reserved ones survive.
		assert.Equal(t, "ml", cr.Annotations["team"])
		assert.Equal(t, "quota-json", cr.Annotations[srv.QuotasAnnotation])
		assert.Equal(t, "bob", cr.Annotations[srv.LastModifiedByAnnotation])
	})

	t.Run("user stamp on nil annotations", func(t *testing.T) {
		cr := &tenantv1alpha1.Tenant{}
		applyTenantPatch(cr, srv.PatchTenantRequest{}, "carol")
		assert.Equal(t, "carol", cr.Annotations[srv.LastModifiedByAnnotation])
	})

	t.Run("no-op leaves annotations nil", func(t *testing.T) {
		cr := &tenantv1alpha1.Tenant{}
		applyTenantPatch(cr, srv.PatchTenantRequest{}, "")
		assert.Nil(t, cr.Annotations)
	})
}

func TestSetQuotaAnnotation(t *testing.T) {
	t.Run("sets on nil annotations", func(t *testing.T) {
		cr := &tenantv1alpha1.Tenant{}
		setQuotaAnnotation(cr, "payload")
		assert.Equal(t, "payload", cr.Annotations[srv.QuotasAnnotation])
	})

	t.Run("empty deletes existing", func(t *testing.T) {
		cr := &tenantv1alpha1.Tenant{}
		cr.Annotations = map[string]string{srv.QuotasAnnotation: "x", "keep": "y"}
		setQuotaAnnotation(cr, "")
		_, ok := cr.Annotations[srv.QuotasAnnotation]
		assert.False(t, ok)
		assert.Equal(t, "y", cr.Annotations["keep"])
	})

	t.Run("empty on nil annotations is safe", func(t *testing.T) {
		cr := &tenantv1alpha1.Tenant{}
		setQuotaAnnotation(cr, "")
		assert.Nil(t, cr.Annotations)
	})
}

func TestIndexOfPool(t *testing.T) {
	quotas := []srv.Quota{{Pool: "a"}, {Pool: "b"}, {Pool: "c"}}
	assert.Equal(t, 1, indexOfPool(quotas, "b"))
	assert.Equal(t, -1, indexOfPool(quotas, "missing"))
	assert.Equal(t, -1, indexOfPool(nil, "b"))
}

func TestReplacePoolQuota(t *testing.T) {
	t.Run("replaces existing in place", func(t *testing.T) {
		quotas := []srv.Quota{{Pool: "a"}, {Pool: "b"}}
		out := replacePoolQuota(quotas, srv.Quota{Pool: "b", Units: []srv.QuotaUnit{{UnitName: "u", Quantity: 2}}})
		require.Len(t, out, 2)
		assert.Len(t, out[1].Units, 1)
	})

	t.Run("appends when pool absent", func(t *testing.T) {
		quotas := []srv.Quota{{Pool: "a"}}
		out := replacePoolQuota(quotas, srv.Quota{Pool: "z"})
		require.Len(t, out, 2)
		assert.Equal(t, "z", out[1].Pool)
	})
}

func TestIsBadQuotaInput(t *testing.T) {
	bad := []srv.QuotaErrorReason{
		srv.QuotaBadQuantity, srv.QuotaDuplicatePool, srv.QuotaModeConflict,
		srv.QuotaModeRequired, srv.QuotaMaxRequired, srv.QuotaNegativeResource,
		srv.QuotaMinWithoutMax, srv.QuotaMinExceedsMax,
	}
	for _, r := range bad {
		assert.True(t, isBadQuotaInput(r), "reason %s should be 400", r)
	}
	// PoolNotFound / UnitNotFound are semantic (422), not bad input.
	assert.False(t, isBadQuotaInput(srv.QuotaPoolNotFound))
	assert.False(t, isBadQuotaInput(srv.QuotaUnitNotFound))
}

func TestTenantGroupResource(t *testing.T) {
	gr := tenantGroupResource()
	assert.Equal(t, "tenants", gr.Resource)
	assert.Equal(t, tenantv1alpha1.GroupVersion.Group, gr.Group)
}
