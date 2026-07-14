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
				{Pool: "default", Max: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10")}},
			},
		},
	}
}

// writeConfig marshals each pool into <dir>/resourcepools/<name>.yaml and each
// tenant into <dir>/tenants/<name>.yaml. Empty slices skip creating that subdir
// (to exercise the missing-config paths).
func writeConfig(t *testing.T, pools []*cmv1alpha1.ResourcePool, tenants []*tenantv1alpha1.Tenant) string {
	t.Helper()
	dir := t.TempDir()
	if len(pools) > 0 {
		sub := filepath.Join(dir, "resourcepools")
		require.NoError(t, os.MkdirAll(sub, 0o755))
		for _, p := range pools {
			b, err := yaml.Marshal(p)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(sub, p.Name+".yaml"), b, 0o600))
		}
	}
	if len(tenants) > 0 {
		sub := filepath.Join(dir, "tenants")
		require.NoError(t, os.MkdirAll(sub, 0o755))
		for _, tn := range tenants {
			b, err := yaml.Marshal(tn)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(sub, tn.Name+".yaml"), b, 0o600))
		}
	}
	return dir
}

// oneConfig is the common single-pool / single-tenant fixture.
func oneConfig(t *testing.T, pool *cmv1alpha1.ResourcePool, tenant *tenantv1alpha1.Tenant) string {
	t.Helper()
	var pools []*cmv1alpha1.ResourcePool
	var tenants []*tenantv1alpha1.Tenant
	if pool != nil {
		pools = []*cmv1alpha1.ResourcePool{pool}
	}
	if tenant != nil {
		tenants = []*tenantv1alpha1.Tenant{tenant}
	}
	return writeConfig(t, pools, tenants)
}

func TestLoadStaticConfig_Valid(t *testing.T) {
	dir := oneConfig(t, validPool(), validTenant())
	sc, err := core.LoadStaticConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, sc)
	require.Len(t, sc.Pools, 1)
	require.Len(t, sc.Tenants, 1)
	assert.Equal(t, "default", sc.Pools[0].Name)
	assert.Equal(t, "default", sc.Tenants[0].Name)
	require.Len(t, sc.Pools[0].Spec.Units, 1)
	assert.Equal(t, "small", sc.Pools[0].Spec.Units[0].Name)
}

// TestLoadStaticConfig_MultipleValid loads two pools and two tenants, each
// tenant scoped to its own namespace and referencing a distinct pool.
func TestLoadStaticConfig_MultipleValid(t *testing.T) {
	gpuPool := validPool()
	gpuPool.Name = "gpu"
	gpuPool.Spec.Units[0].Name = "gpu-1x"

	teamA := validTenant()
	teamA.Name = "team-a"
	teamA.Spec.Namespace.Name = "team-a"
	teamA.Spec.Quotas[0].Pool = "gpu"

	dir := writeConfig(t,
		[]*cmv1alpha1.ResourcePool{validPool(), gpuPool},
		[]*tenantv1alpha1.Tenant{validTenant(), teamA},
	)
	sc, err := core.LoadStaticConfig(dir)
	require.NoError(t, err)
	require.Len(t, sc.Pools, 2)
	require.Len(t, sc.Tenants, 2)
	// Load order follows sorted filenames: "default" before "gpu"/"team-a".
	assert.Equal(t, "default", sc.Pools[0].Name)
	assert.Equal(t, "gpu", sc.Pools[1].Name)
	assert.Equal(t, "default", sc.Tenants[0].Name)
	assert.Equal(t, "team-a", sc.Tenants[1].Name)
}

func TestLoadStaticConfig_ValidWithPredefinedVolumes(t *testing.T) {
	tn := validTenant()
	tn.Spec.InitResources.Volumes = []tenantv1alpha1.VolumeSpec{
		{Name: "dataset", Size: "50Gi", Description: "shared training data"},
		{Name: "checkpoints"},
		{Name: "hostdata", HostPath: "/data/host-datasets"},
	}
	dir := oneConfig(t, validPool(), tn)
	sc, err := core.LoadStaticConfig(dir)
	require.NoError(t, err)
	require.Len(t, sc.Tenants[0].Spec.InitResources.Volumes, 3)
	assert.Equal(t, "dataset", sc.Tenants[0].Spec.InitResources.Volumes[0].Name)
	assert.Equal(t, "/data/host-datasets", sc.Tenants[0].Spec.InitResources.Volumes[2].HostPath)
}

// TestLoadStaticConfig_HostPathVolumeNames covers how predefined volume names
// scope across tenants: hostPath names must be globally unique (they resolve
// through the runtime's single name-keyed map), while managed volume names may
// repeat across tenants (they are namespaced per tenant).
func TestLoadStaticConfig_HostPathVolumeNames(t *testing.T) {
	t.Run("same hostPath name across tenants rejected", func(t *testing.T) {
		a := validTenant()
		a.Spec.InitResources.Volumes = []tenantv1alpha1.VolumeSpec{{Name: "shared", HostPath: "/data/a"}}
		b := validTenant()
		b.Name = "team-b"
		b.Spec.Namespace.Name = "team-b"
		b.Spec.InitResources.Volumes = []tenantv1alpha1.VolumeSpec{{Name: "shared", HostPath: "/data/b"}}
		dir := writeConfig(t, []*cmv1alpha1.ResourcePool{validPool()}, []*tenantv1alpha1.Tenant{a, b})
		_, err := core.LoadStaticConfig(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be unique across tenants")
	})

	t.Run("same managed volume name across tenants allowed", func(t *testing.T) {
		a := validTenant()
		a.Spec.InitResources.Volumes = []tenantv1alpha1.VolumeSpec{{Name: "data"}}
		b := validTenant()
		b.Name = "team-b"
		b.Spec.Namespace.Name = "team-b"
		b.Spec.InitResources.Volumes = []tenantv1alpha1.VolumeSpec{{Name: "data"}}
		dir := writeConfig(t, []*cmv1alpha1.ResourcePool{validPool()}, []*tenantv1alpha1.Tenant{a, b})
		_, err := core.LoadStaticConfig(dir)
		require.NoError(t, err)
	})
}

func TestLoadStaticConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(p *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant)
		wantSub string
	}{
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
			name: "tenant credential initResources not empty",
			mutate: func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) {
				tn.Spec.InitResources.ConfigMaps = []tenantv1alpha1.ConfigMapSpec{{Name: "cm"}}
			},
			wantSub: "secrets/configMaps/serviceAccounts must be empty",
		},
		{
			name: "tenant volume without name",
			mutate: func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) {
				tn.Spec.InitResources.Volumes = []tenantv1alpha1.VolumeSpec{{Name: ""}}
			},
			wantSub: "volumes[0].name is required",
		},
		{
			name: "tenant duplicate volume name",
			mutate: func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) {
				tn.Spec.InitResources.Volumes = []tenantv1alpha1.VolumeSpec{{Name: "data"}, {Name: "data"}}
			},
			wantSub: "duplicate volume",
		},
		{
			name: "tenant hostPath volume relative path",
			mutate: func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) {
				tn.Spec.InitResources.Volumes = []tenantv1alpha1.VolumeSpec{{Name: "hostdata", HostPath: "relative/path"}}
			},
			wantSub: "must be an absolute path",
		},
		{
			name:    "tenant namespace does not match name",
			mutate:  func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) { tn.Spec.Namespace.Name = "other" },
			wantSub: "must equal the tenant name",
		},
		{
			name:    "tenant namespace empty",
			mutate:  func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) { tn.Spec.Namespace.Name = "" },
			wantSub: "must equal the tenant name",
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
			wantSub: "must declare at least one quota",
		},
		{
			name:    "quota references unknown pool",
			mutate:  func(_ *cmv1alpha1.ResourcePool, tn *tenantv1alpha1.Tenant) { tn.Spec.Quotas[0].Pool = "other" },
			wantSub: "references unknown pool",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool, tenant := validPool(), validTenant()
			tc.mutate(pool, tenant)
			dir := oneConfig(t, pool, tenant)
			_, err := core.LoadStaticConfig(dir)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantSub)
		})
	}
}

// TestLoadStaticConfig_DuplicateNames covers the cross-object uniqueness checks
// that only apply once more than one object exists.
func TestLoadStaticConfig_DuplicateNames(t *testing.T) {
	t.Run("duplicate pool name", func(t *testing.T) {
		// Two pool files decode to the same metadata.name.
		dir := t.TempDir()
		sub := filepath.Join(dir, "resourcepools")
		require.NoError(t, os.MkdirAll(sub, 0o755))
		for _, fn := range []string{"a.yaml", "b.yaml"} {
			b, err := yaml.Marshal(validPool())
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(sub, fn), b, 0o600))
		}
		tsub := filepath.Join(dir, "tenants")
		require.NoError(t, os.MkdirAll(tsub, 0o755))
		b, err := yaml.Marshal(validTenant())
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(tsub, "default.yaml"), b, 0o600))

		_, err = core.LoadStaticConfig(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate resource pool")
	})

	t.Run("duplicate tenant name", func(t *testing.T) {
		// Two tenant files decode to the same metadata.name.
		dir := t.TempDir()
		psub := filepath.Join(dir, "resourcepools")
		require.NoError(t, os.MkdirAll(psub, 0o755))
		b, err := yaml.Marshal(validPool())
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(psub, "default.yaml"), b, 0o600))

		tsub := filepath.Join(dir, "tenants")
		require.NoError(t, os.MkdirAll(tsub, 0o755))
		for _, fn := range []string{"a.yaml", "b.yaml"} {
			tb, err := yaml.Marshal(validTenant())
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(tsub, fn), tb, 0o600))
		}

		_, err = core.LoadStaticConfig(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate tenant")
	})
}

func TestLoadStaticConfig_MissingPoolDir(t *testing.T) {
	dir := oneConfig(t, nil, validTenant())
	_, err := core.LoadStaticConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource pool config dir")
}

func TestLoadStaticConfig_MissingTenantDir(t *testing.T) {
	dir := oneConfig(t, validPool(), nil)
	_, err := core.LoadStaticConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant config dir")
}

func TestLoadStaticConfig_EmptyPoolDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "resourcepools"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "tenants"), 0o755))
	_, err := core.LoadStaticConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no resource pool config found")
}

func TestLoadStaticConfig_UndecodablePool(t *testing.T) {
	dir := t.TempDir()
	psub := filepath.Join(dir, "resourcepools")
	require.NoError(t, os.MkdirAll(psub, 0o755))
	// A top-level YAML sequence decodes to a JSON array, which cannot unmarshal
	// into the ResourcePool struct.
	require.NoError(t, os.WriteFile(filepath.Join(psub, "bad.yaml"), []byte("- a\n- b\n"), 0o600))
	tsub := filepath.Join(dir, "tenants")
	require.NoError(t, os.MkdirAll(tsub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tsub, "default.yaml"), []byte("{}"), 0o600))
	_, err := core.LoadStaticConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode resource pool config")
}

// TestLoadStaticConfig_MultipleDocsRejected covers the single-document guard: a
// file packing two YAML documents must error rather than silently loading only
// the first.
func TestLoadStaticConfig_MultipleDocsRejected(t *testing.T) {
	dir := t.TempDir()
	psub := filepath.Join(dir, "resourcepools")
	require.NoError(t, os.MkdirAll(psub, 0o755))
	p1, err := yaml.Marshal(validPool())
	require.NoError(t, err)
	p2 := validPool()
	p2.Name = "second"
	p2b, err := yaml.Marshal(p2)
	require.NoError(t, err)
	multi := append(append(append([]byte{}, p1...), []byte("\n---\n")...), p2b...)
	require.NoError(t, os.WriteFile(filepath.Join(psub, "pools.yaml"), multi, 0o600))

	tsub := filepath.Join(dir, "tenants")
	require.NoError(t, os.MkdirAll(tsub, 0o755))
	tb, err := yaml.Marshal(validTenant())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tsub, "default.yaml"), tb, 0o600))

	_, err = core.LoadStaticConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must contain exactly one object")
}
