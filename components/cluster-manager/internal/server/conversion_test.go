package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

func TestParseResourceList(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]string
		want corev1.ResourceList
	}{
		{
			name: "empty returns nil",
			in:   nil,
			want: nil,
		},
		{
			name: "valid quantities parsed",
			in:   map[string]string{"cpu": "4", "memory": "8Gi"},
			want: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
		{
			name: "malformed values dropped, valid kept",
			in:   map[string]string{"cpu": "not-a-quantity", "memory": "1Gi"},
			want: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
		{
			name: "all malformed → empty map (not nil)",
			in:   map[string]string{"cpu": "wat", "memory": "huh"},
			want: corev1.ResourceList{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseResourceList(tc.in)
			assert.True(t, resourceListEqual(got, tc.want),
				"got=%v want=%v", got, tc.want)
		})
	}
}

func TestToTenantSpec_FullRoundTrip(t *testing.T) {
	req := CreateTenantRequest{
		Name:        "alpha",
		DisplayName: "Alpha Team",
		Annotations: map[string]string{"team": "ml-platform"},
		Namespace: NamespaceSpec{
			Name:        "alpha-ns",
			Labels:      map[string]string{"env": "prod"},
			Annotations: map[string]string{"owner": "alice"},
		},
		Quotas: []QuotaSpec{{
			Pool: "default",
			Name: "alpha-quota",
			Min:  map[string]string{"cpu": "1"},
			Max:  map[string]string{"cpu": "8", "memory": "16Gi"},
		}},
		Init: &InitResources{
			ImagePullSecrets: []ImagePullSecretSpec{{
				Name:            "registry",
				SourceSecretRef: ObjectRef{Namespace: "platform", Name: "src-pull"},
			}},
			Secrets: []SecretSpec{{
				Name:            "creds",
				Type:            "Opaque",
				SourceSecretRef: ObjectRef{Namespace: "platform", Name: "src-creds"},
			}},
			ConfigMaps: []ConfigMapSpec{{
				Name:               "tuning",
				SourceConfigMapRef: ObjectRef{Namespace: "platform", Name: "src-cm"},
			}},
			ServiceAccounts: []ServiceAccountSpec{{
				Name:             "runner",
				ImagePullSecrets: []string{"registry"},
				RBAC: &RBAC{
					Rules: []map[string]any{{
						"verbs":     []string{"get", "list"},
						"apiGroups": []string{""},
						"resources": []string{"pods"},
					}},
					RoleRef: &RoleRef{Kind: "Role", Name: "viewer"},
				},
			}},
		},
		Suspended: true,
	}

	spec := req.ToTenantSpec()

	assert.Equal(t, "Alpha Team", spec.DisplayName)
	assert.Equal(t, map[string]string{"team": "ml-platform"}, spec.Annotations)
	assert.True(t, spec.Suspended)
	assert.Equal(t, "alpha-ns", spec.Namespace.Name)

	require.Len(t, spec.Quotas, 1)
	assert.Equal(t, "default", spec.Quotas[0].Pool)
	assert.True(t, spec.Quotas[0].Max.Cpu().Equal(resource.MustParse("8")))
	assert.True(t, spec.Quotas[0].Max.Memory().Equal(resource.MustParse("16Gi")))

	require.Len(t, spec.InitResources.ImagePullSecrets, 1)
	assert.Equal(t, "registry", spec.InitResources.ImagePullSecrets[0].Name)
	assert.Equal(t, "src-pull", spec.InitResources.ImagePullSecrets[0].SourceSecretRef.Name)

	require.Len(t, spec.InitResources.Secrets, 1)
	assert.Equal(t, corev1.SecretType("Opaque"), spec.InitResources.Secrets[0].Type)

	require.Len(t, spec.InitResources.ConfigMaps, 1)
	assert.Equal(t, "tuning", spec.InitResources.ConfigMaps[0].Name)

	require.Len(t, spec.InitResources.ServiceAccounts, 1)
	sa := spec.InitResources.ServiceAccounts[0]
	require.NotNil(t, sa.RBAC)
	require.Len(t, sa.RBAC.Rules, 1)
	assert.Equal(t, []string{"get", "list"}, sa.RBAC.Rules[0].Verbs)
	require.NotNil(t, sa.RBAC.RoleRef)
	assert.Equal(t, "viewer", sa.RBAC.RoleRef.Name)
}

func TestToTenantSpec_NoInit_LeavesEmpty(t *testing.T) {
	req := CreateTenantRequest{
		Name:      "beta",
		Namespace: NamespaceSpec{Name: "beta-ns"},
	}
	spec := req.ToTenantSpec()
	assert.Empty(t, spec.InitResources.ImagePullSecrets)
	assert.Empty(t, spec.InitResources.Secrets)
	assert.Empty(t, spec.InitResources.ConfigMaps)
	assert.Empty(t, spec.InitResources.ServiceAccounts)
	assert.Empty(t, spec.Quotas)
}

func TestStrSlice(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"nil", nil, nil},
		{"unknown type", 42, nil},
		{"[]string passthrough", []string{"a", "b"}, []string{"a", "b"}},
		{"[]any with strings", []any{"x", "y"}, []string{"x", "y"}},
		{"[]any with non-strings filtered", []any{"x", 1, "y", true}, []string{"x", "y"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, strSlice(tc.in))
		})
	}
}

func TestFromTenant_StatusAndQuotas(t *testing.T) {
	now := metav1.NewTime(time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC))
	in := &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "alpha",
			Labels:            map[string]string{tenantv1alpha1.LabelTenantID: "tnt-uuid-1"},
			CreationTimestamp: now,
		},
		Spec: tenantv1alpha1.TenantSpec{
			DisplayName: "Alpha",
			Namespace:   tenantv1alpha1.NamespaceSpec{Name: "alpha-ns"},
			Quotas: []tenantv1alpha1.QuotaSpec{{
				Pool: "default",
				Name: "q1",
				Max: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("8"),
				},
			}},
			Suspended: false,
		},
		Status: tenantv1alpha1.TenantStatus{
			Phase:   tenantv1alpha1.TenantPhaseActive,
			Message: "ok",
			Quotas: []tenantv1alpha1.QuotaStatus{{
				Pool:    "default",
				Name:    "q1",
				Ready:   true,
				Used:    corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
				Message: "in use",
			}},
		},
	}

	out := FromTenant(in)
	assert.Equal(t, "tnt-uuid-1", out.ID)
	assert.Equal(t, "alpha", out.Name)
	assert.Equal(t, "Alpha", out.DisplayName)
	assert.Equal(t, "alpha-ns", out.Namespace.Name)
	require.Len(t, out.Quotas, 1)
	assert.Equal(t, "8", out.Quotas[0].Max["cpu"])
	assert.Equal(t, "Active", out.Status.Phase)
	require.Len(t, out.Status.Quotas, 1)
	assert.Equal(t, "2", out.Status.Quotas[0].Used["cpu"])
	assert.Equal(t, now.Time, out.CreatedAt)
}

func TestFromTenant_NoInit_DropsBlock(t *testing.T) {
	in := &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "beta"},
		Spec:       tenantv1alpha1.TenantSpec{Namespace: tenantv1alpha1.NamespaceSpec{Name: "beta-ns"}},
	}
	out := FromTenant(in)
	assert.Nil(t, out.Init,
		"empty InitResources must round-trip back to nil so omitempty drops the JSON field")
}

func TestFromTenant_PolicyRulesPreserveAllFields(t *testing.T) {
	in := &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "gamma"},
		Spec: tenantv1alpha1.TenantSpec{
			Namespace: tenantv1alpha1.NamespaceSpec{Name: "gamma-ns"},
			InitResources: tenantv1alpha1.InitResources{
				ServiceAccounts: []tenantv1alpha1.ServiceAccountSpec{{
					Name: "runner",
					RBAC: &tenantv1alpha1.RBACSpec{
						Rules: []rbacv1.PolicyRule{{
							Verbs:           []string{"get"},
							APIGroups:       []string{""},
							Resources:       []string{"pods"},
							ResourceNames:   []string{"named-pod"},
							NonResourceURLs: []string{"/metrics"},
						}},
					},
				}},
			},
		},
	}
	out := FromTenant(in)
	require.NotNil(t, out.Init)
	require.Len(t, out.Init.ServiceAccounts, 1)
	require.NotNil(t, out.Init.ServiceAccounts[0].RBAC)
	rules := out.Init.ServiceAccounts[0].RBAC.Rules
	require.Len(t, rules, 1)
	assert.Equal(t, []string{"get"}, rules[0]["verbs"])
	assert.Equal(t, []string{"named-pod"}, rules[0]["resourceNames"])
	assert.Equal(t, []string{"/metrics"}, rules[0]["nonResourceURLs"])
}

func TestApplyPatchToTenant(t *testing.T) {
	original := &tenantv1alpha1.Tenant{
		Spec: tenantv1alpha1.TenantSpec{
			DisplayName: "Old",
			Annotations: map[string]string{"old": "true"},
			InitResources: tenantv1alpha1.InitResources{
				Secrets: []tenantv1alpha1.SecretSpec{{Name: "stale"}},
			},
		},
	}

	t.Run("display name only", func(t *testing.T) {
		t1 := original.DeepCopy()
		newName := "New"
		ApplyPatchToTenant(t1, PatchTenantRequest{DisplayName: &newName})
		assert.Equal(t, "New", t1.Spec.DisplayName)
		assert.Equal(t, map[string]string{"old": "true"}, t1.Spec.Annotations,
			"untouched fields must not change")
	})

	t.Run("nil display name preserves existing", func(t *testing.T) {
		t1 := original.DeepCopy()
		ApplyPatchToTenant(t1, PatchTenantRequest{
			Annotations: map[string]string{"new": "yes"},
		})
		assert.Equal(t, "Old", t1.Spec.DisplayName)
		assert.Equal(t, map[string]string{"new": "yes"}, t1.Spec.Annotations)
	})

	t.Run("init replaces wholesale", func(t *testing.T) {
		t1 := original.DeepCopy()
		ApplyPatchToTenant(t1, PatchTenantRequest{
			Init: &InitResources{ConfigMaps: []ConfigMapSpec{{Name: "fresh"}}},
		})
		assert.Empty(t, t1.Spec.InitResources.Secrets,
			"PATCH init must replace the InitResources block, not merge")
		require.Len(t, t1.Spec.InitResources.ConfigMaps, 1)
		assert.Equal(t, "fresh", t1.Spec.InitResources.ConfigMaps[0].Name)
	})
}

func TestEnsureMetadata(t *testing.T) {
	t1 := &tenantv1alpha1.Tenant{}
	EnsureMetadata(t1, "delta", "tnt-uuid-7")
	assert.Equal(t, "delta", t1.Name)
	assert.Equal(t, "tnt-uuid-7", t1.Labels[tenantv1alpha1.LabelTenantID])
}

func TestEnsureMetadata_PreservesExistingLabels(t *testing.T) {
	t1 := &tenantv1alpha1.Tenant{}
	t1.Labels = map[string]string{"keep": "yes"}
	EnsureMetadata(t1, "delta", "tnt-uuid-7")
	assert.Equal(t, "yes", t1.Labels["keep"])
	assert.Equal(t, "tnt-uuid-7", t1.Labels[tenantv1alpha1.LabelTenantID])
}

func TestNewTenantList_TypeMeta(t *testing.T) {
	l := NewTenantList()
	assert.Equal(t, "TenantList", l.Kind)
	assert.Equal(t, tenantv1alpha1.GroupVersion.String(), l.APIVersion)
}

// resourceListEqual compares two ResourceLists by parsed Quantity values.
// resource.Quantity has internal state that makes raw == comparison flaky.
func resourceListEqual(a, b corev1.ResourceList) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok || !v.Equal(bv) {
			return false
		}
	}
	return true
}
