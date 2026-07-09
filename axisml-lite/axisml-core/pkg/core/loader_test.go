package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"

	"github.com/axisml/axisml/axisml-lite/axisml-core/pkg/core"
)

// validPool returns a ResourcePool that satisfies every Lite invariant so tests
// can mutate a single field to exercise one validation branch at a time.
func validPool() *cmv1alpha1.ResourcePool {
	return &cmv1alpha1.ResourcePool{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: cmv1alpha1.ResourcePoolSpec{
			Units: []cmv1alpha1.ResourceUnit{
				{
					Name:     "small",
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
					Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
				},
			},
		},
	}
}

func validTenant() *tenantv1alpha1.Tenant {
	return &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: tenantv1alpha1.TenantSpec{
			Namespace: tenantv1alpha1.NamespaceSpec{Name: "default"},
			Quotas: []tenantv1alpha1.QuotaSpec{
				{Pool: "default", Name: "default", Max: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10")}},
			},
		},
	}
}

// writeConfig marshals the given pool/tenant into <dir>/resource-pool.yaml and
// <dir>/tenant.yaml. A nil argument skips writing that file (to test the missing
// -file decode path).
func writeConfig(t *testing.T, pool *cmv1alpha1.ResourcePool, tenant *tenantv1alpha1.Tenant) string {
	t.Helper()
	dir := t.TempDir()
	if pool != nil {
		b, err := yaml.Marshal(pool)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "resource-pool.yaml"), b, 0o600))
	}
	if tenant != nil {
		b, err := yaml.Marshal(tenant)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "tenant.yaml"), b, 0o600))
	}
	return dir
}

func TestLoadStaticConfig_Valid(t *testing.T) {
	dir := writeConfig(t, validPool(), validTenant())
	sc, err := core.LoadStaticConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, sc)
	assert.Equal(t, "default", sc.Pool.Name)
	assert.Equal(t, "default", sc.Tenant.Name)
	require.Len(t, sc.Pool.Spec.Units, 1)
	assert.Equal(t, "small", sc.Pool.Spec.Units[0].Name)
}

func TestLoadStaticConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(p *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant)
		wantSub string
	}{
		{
			name:    "pool name not default",
			mutate:  func(p *cmv1alpha1.ResourcePool, _ *tenantv1alpha1.Tenant) { p.Name = "other" },
			wantSub: "resource pool name must be",
		},
		{
			name:    "tenant name not default",
			mutate:  func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) { tn.Name = "other" },
			wantSub: "tenant name must be",
		},
		{
			name:    "tenant namespace not default",
			mutate:  func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) { tn.Spec.Namespace.Name = "other" },
			wantSub: "tenant namespace must be",
		},
		{
			name: "pool nodeSelector not empty",
			mutate: func(p *cmv1alpha1.ResourcePool, _ *tenantv1alpha1.Tenant) {
				p.Spec.NodeSelector = map[string]string{"gpu": "true"}
			},
			wantSub: "nodeSelector/tolerations must be empty",
		},
		{
			name: "pool tolerations not empty",
			mutate: func(p *cmv1alpha1.ResourcePool, _ *tenantv1alpha1.Tenant) {
				p.Spec.Tolerations = []corev1.Toleration{{Key: "dedicated"}}
			},
			wantSub: "nodeSelector/tolerations must be empty",
		},
		{
			name: "tenant initResources not empty",
			mutate: func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) {
				tn.Spec.InitResources.ConfigMaps = []tenantv1alpha1.ConfigMapSpec{{Name: "cm"}}
			},
			wantSub: "initResources must be empty",
		},
		{
			name:    "unit name empty",
			mutate:  func(p *cmv1alpha1.ResourcePool, _ *tenantv1alpha1.Tenant) { p.Spec.Units[0].Name = "" },
			wantSub: "resource unit name must not be empty",
		},
		{
			name: "duplicate unit name",
			mutate: func(p *cmv1alpha1.ResourcePool, _ *tenantv1alpha1.Tenant) {
				p.Spec.Units = append(p.Spec.Units, p.Spec.Units[0])
			},
			wantSub: "duplicate resource unit",
		},
		{
			name: "unit nodeSelector not empty",
			mutate: func(p *cmv1alpha1.ResourcePool, _ *tenantv1alpha1.Tenant) {
				p.Spec.Units[0].NodeSelector = map[string]string{"gpu": "true"}
			},
			wantSub: "nodeSelector must be empty",
		},
		{
			name:    "no quotas declared",
			mutate:  func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) { tn.Spec.Quotas = nil },
			wantSub: "tenant must declare a quota",
		},
		{
			name:    "quota references wrong pool",
			mutate:  func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) { tn.Spec.Quotas[0].Pool = "other" },
			wantSub: "tenant quota must reference pool",
		},
		{
			name: "unit requests exceed max",
			mutate: func(p *cmv1alpha1.ResourcePool, _ *tenantv1alpha1.Tenant) {
				p.Spec.Units[0].Requests = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("20")}
			},
			wantSub: "exceeds quota max",
		},
		{
			name: "unit limits exceed max",
			mutate: func(p *cmv1alpha1.ResourcePool, _ *tenantv1alpha1.Tenant) {
				p.Spec.Units[0].Limits = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50")}
			},
			wantSub: "exceeds quota max",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool, tenant := validPool(), validTenant()
			tc.mutate(pool, tenant)
			dir := writeConfig(t, pool, tenant)
			_, err := core.LoadStaticConfig(dir)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantSub)
		})
	}
}

func TestLoadStaticConfig_MissingPoolFile(t *testing.T) {
	dir := writeConfig(t, nil, validTenant())
	_, err := core.LoadStaticConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read resource-pool config")
}

func TestLoadStaticConfig_MissingTenantFile(t *testing.T) {
	dir := writeConfig(t, validPool(), nil)
	_, err := core.LoadStaticConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read tenant config")
}

func TestLoadStaticConfig_UndecodablePool(t *testing.T) {
	dir := t.TempDir()
	// A top-level YAML sequence decodes to a JSON array, which cannot unmarshal
	// into the ResourcePool struct.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "resource-pool.yaml"), []byte("- a\n- b\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tenant.yaml"), []byte("{}"), 0o600))
	_, err := core.LoadStaticConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode resource-pool config")
}

// unconstrainedDimension confirms withinMax skips resource dimensions the quota
// does not cap (e.g. memory when only cpu is bounded).
func TestLoadStaticConfig_UnconstrainedDimensionAllowed(t *testing.T) {
	pool, tenant := validPool(), validTenant()
	// Unit asks for memory the quota does not constrain; cpu stays within max.
	pool.Spec.Units[0].Requests[corev1.ResourceMemory] = resource.MustParse("999Gi")
	dir := writeConfig(t, pool, tenant)
	_, err := core.LoadStaticConfig(dir)
	require.NoError(t, err)
}
