package server_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
)

func TestTenantToAPI_FullCR(t *testing.T) {
	cr := &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "acme",
			ResourceVersion: "7",
			Labels:          map[string]string{"tier": "gold"},
			Annotations: map[string]string{
				srv.LastModifiedByAnnotation: "alice",
				srv.QuotasAnnotation:         "ignored-here",
				"team":                       "ml",
			},
		},
		Spec: tenantv1alpha1.TenantSpec{
			Namespace: tenantv1alpha1.NamespaceSpec{Name: "acme"},
			Quotas: []tenantv1alpha1.QuotaSpec{{
				Pool: "gpu",
				Max:  corev1.ResourceList{"cpu": resource.MustParse("8")},
			}},
			InitResources: tenantv1alpha1.InitResources{
				Secrets: []tenantv1alpha1.SecretSpec{{Name: "s"}},
			},
		},
	}
	cr.Status = tenantv1alpha1.TenantStatus{
		Phase:              tenantv1alpha1.TenantPhase("Ready"),
		ObservedGeneration: 3,
		NamespaceReady:     true,
		Message:            "ok",
		Quotas: []tenantv1alpha1.QuotaStatus{{
			Pool: "gpu", Ready: true,
			Used: corev1.ResourceList{"cpu": resource.MustParse("2")},
		}},
	}

	dto := srv.TenantToAPI(cr)
	assert.Equal(t, "acme", dto.Name)
	assert.Equal(t, "7", dto.ResourceVersion)
	assert.Equal(t, "Ready", dto.Phase)
	// Reserved annotations stripped; user annotation retained.
	assert.Equal(t, map[string]string{"team": "ml"}, dto.Annotations)
	require.NotNil(t, dto.InitResources)
	require.Len(t, dto.InitResources.Secrets, 1)
	// A direct quota (max present, no round-trip annotation) renders as direct form.
	require.Len(t, dto.Quotas, 1)
	require.NotNil(t, dto.Quotas[0].Quota)
	assert.Equal(t, "8", dto.Quotas[0].Quota.Max.Cpu().String())
	// Status projected.
	require.NotNil(t, dto.Status)
	assert.Equal(t, int64(3), dto.Status.ObservedGeneration)
	assert.True(t, dto.Status.NamespaceReady)
	require.Len(t, dto.Status.Quotas, 1)
	assert.Equal(t, "gpu", dto.Status.Quotas[0].Pool)
}

func TestTenantToAPI_EmptyStatusAndInitResources(t *testing.T) {
	cr := &tenantv1alpha1.Tenant{}
	cr.Name = "bare"
	dto := srv.TenantToAPI(cr)
	assert.Nil(t, dto.Status)
	assert.Nil(t, dto.InitResources)
	assert.Empty(t, dto.Quotas)
}

func TestAPIToTenant_NamespaceDefaulting(t *testing.T) {
	t.Run("nil namespace defaults to name", func(t *testing.T) {
		cr := srv.APIToTenant(srv.CreateTenantRequest{Name: "acme"}, nil, "", "")
		assert.Equal(t, "acme", cr.Spec.Namespace.Name)
	})

	t.Run("empty namespace name defaults to tenant name", func(t *testing.T) {
		ns := tenantv1alpha1.NamespaceSpec{Labels: map[string]string{"l": "1"}}
		cr := srv.APIToTenant(srv.CreateTenantRequest{Name: "acme", Namespace: &ns}, nil, "", "")
		assert.Equal(t, "acme", cr.Spec.Namespace.Name)
		assert.Equal(t, map[string]string{"l": "1"}, cr.Spec.Namespace.Labels)
	})

	t.Run("explicit namespace name preserved", func(t *testing.T) {
		ns := tenantv1alpha1.NamespaceSpec{Name: "custom-ns"}
		cr := srv.APIToTenant(srv.CreateTenantRequest{Name: "acme", Namespace: &ns}, nil, "", "")
		assert.Equal(t, "custom-ns", cr.Spec.Namespace.Name)
	})
}

func TestAPIToTenant_AnnotationsAndFolded(t *testing.T) {
	folded := []tenantv1alpha1.QuotaSpec{{Pool: "gpu",
		Max: corev1.ResourceList{"cpu": resource.MustParse("4")}}}
	ir := tenantv1alpha1.InitResources{Secrets: []tenantv1alpha1.SecretSpec{{Name: "s"}}}
	cr := srv.APIToTenant(srv.CreateTenantRequest{
		Name:          "acme",
		Labels:        map[string]string{"tier": "gold"},
		Annotations:   map[string]string{"team": "ml"},
		InitResources: &ir,
	}, folded, "quota-json", "alice")

	assert.Equal(t, "acme", cr.Name)
	assert.Equal(t, folded, cr.Spec.Quotas)
	require.Len(t, cr.Spec.InitResources.Secrets, 1)
	assert.Equal(t, "ml", cr.Annotations["team"])
	assert.Equal(t, "alice", cr.Annotations[srv.LastModifiedByAnnotation])
	assert.Equal(t, "quota-json", cr.Annotations[srv.QuotasAnnotation])
}

func TestAPIToTenant_NoAnnotationsYieldsNil(t *testing.T) {
	cr := srv.APIToTenant(srv.CreateTenantRequest{Name: "acme"}, nil, "", "")
	assert.Nil(t, cr.Annotations)
}

func TestPoolNames_Dedupe(t *testing.T) {
	names := srv.PoolNames([]srv.Quota{
		{Pool: "a"}, {Pool: "b"}, {Pool: "a"}, {Pool: "c"}, {Pool: "b"},
	})
	assert.Equal(t, []string{"a", "b", "c"}, names)
	assert.Empty(t, srv.PoolNames(nil))
}

func TestQuotaError_Error(t *testing.T) {
	cases := map[srv.QuotaErrorReason]string{
		srv.QuotaPoolNotFound:     "resource pool not found",
		srv.QuotaUnitNotFound:     "unit not found in pool",
		srv.QuotaBadQuantity:      "quantity must be >= 0",
		srv.QuotaDuplicatePool:    "duplicate quota for pool",
		srv.QuotaModeConflict:     "must use either units or quota",
		srv.QuotaModeRequired:     "must specify either units or quota",
		srv.QuotaMaxRequired:      "quota.max is required",
		srv.QuotaInvalidResource:  "invalid quota resources",
		srv.QuotaNegativeResource: "must be >= 0 in pool",
		srv.QuotaMinWithoutMax:    "must also be present in quota.max",
		srv.QuotaMinExceedsMax:    "exceeds quota.max",
	}
	for reason, want := range cases {
		e := &srv.QuotaError{Reason: reason, Pool: "p", Unit: "u", Resource: "cpu"}
		assert.Contains(t, e.Error(), want, "reason %s", reason)
	}
	// Unknown reason falls through to the default.
	e := &srv.QuotaError{Reason: srv.QuotaErrorReason("bogus")}
	assert.Equal(t, "invalid quota", e.Error())
}

func TestQuotaMarshalJSON(t *testing.T) {
	t.Run("units mode emits units array", func(t *testing.T) {
		b, err := json.Marshal(srv.Quota{Pool: "p", Units: []srv.QuotaUnit{{UnitName: "u", Quantity: 2}}})
		require.NoError(t, err)
		assert.JSONEq(t, `{"pool":"p","units":[{"unitName":"u","quantity":2}]}`, string(b))
	})

	t.Run("explicit empty units preserved", func(t *testing.T) {
		b, err := json.Marshal(srv.Quota{Pool: "p", Units: []srv.QuotaUnit{}})
		require.NoError(t, err)
		assert.JSONEq(t, `{"pool":"p","units":[]}`, string(b))
	})

	t.Run("direct mode omits units", func(t *testing.T) {
		b, err := json.Marshal(srv.Quota{Pool: "p", Quota: &srv.QuotaResources{
			Max: corev1.ResourceList{"cpu": resource.MustParse("4")},
		}})
		require.NoError(t, err)
		assert.NotContains(t, string(b), "units")
		assert.Contains(t, string(b), `"quota"`)
	})
}
