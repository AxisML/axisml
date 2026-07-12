package core

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

// Config subdirectory names under PoolConfigDir. Each holds one YAML file per
// object; every *.yaml / *.yml file is loaded (design §5.1.1).
const (
	poolsSubdir   = "resourcepools"
	tenantsSubdir = "tenants"
)

// StaticConfig is the immutable snapshot parsed from the CR-YAML config at
// startup: one or more read-only ResourcePools and Tenants (design §5.1.1). All
// config-backed providers read from this snapshot; changing it requires an
// axisml-core restart.
type StaticConfig struct {
	Pools   []*cmv1alpha1.ResourcePool
	Tenants []*tenantv1alpha1.Tenant
}

// LoadStaticConfig reads every ResourcePool under <dir>/resourcepools and every
// Tenant under <dir>/tenants, decodes them with the AxisML API types and runs
// the cross-object validation. Any failure leaves axisml-core not-ready
// (design §5.1.1).
func LoadStaticConfig(dir string) (*StaticConfig, error) {
	pools, err := loadDir[cmv1alpha1.ResourcePool](filepath.Join(dir, poolsSubdir), "resource pool")
	if err != nil {
		return nil, err
	}
	tenants, err := loadDir[tenantv1alpha1.Tenant](filepath.Join(dir, tenantsSubdir), "tenant")
	if err != nil {
		return nil, err
	}
	sc := &StaticConfig{Pools: pools, Tenants: tenants}
	if err := sc.validate(); err != nil {
		return nil, err
	}
	return sc, nil
}

// yamlFiles returns the sorted *.yaml / *.yml paths directly under dir. Sorting
// keeps the load order (and thus List order) deterministic across restarts.
func yamlFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if name := e.Name(); strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			out = append(out, filepath.Join(dir, name))
		}
	}
	sort.Strings(out)
	return out, nil
}

// loadDir decodes every YAML file under dir into a *T. kind names the object for
// error messages ("resource pool" / "tenant").
func loadDir[T any](dir, kind string) ([]*T, error) {
	paths, err := yamlFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s config dir %s: %w", kind, dir, err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no %s config found in %s", kind, dir)
	}
	out := make([]*T, 0, len(paths))
	for _, p := range paths {
		obj, err := decodeObject[T](p, kind)
		if err != nil {
			return nil, err
		}
		out = append(out, obj)
	}
	return out, nil
}

// decodeObject reads one CR-YAML file into a *T. Each file must hold exactly one
// object (design §5.1.1); a file with multiple YAML documents is rejected rather
// than silently loading only the first.
func decodeObject[T any](path, kind string) (*T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s config %s: %w", kind, path, err)
	}
	if err := singleDocument(b, path, kind); err != nil {
		return nil, err
	}
	var obj T
	if err := yaml.Unmarshal(b, &obj); err != nil {
		return nil, fmt.Errorf("decode %s config %s: %w", kind, path, err)
	}
	return &obj, nil
}

// singleDocument rejects a config file that packs more than one YAML document,
// which yaml.Unmarshal would otherwise collapse to its first document silently.
// A decode error on the first document is left for the typed decode to report.
func singleDocument(b []byte, path, kind string) error {
	dec := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 4096)
	var first map[string]any
	if err := dec.Decode(&first); err != nil {
		return nil // real errors surface in the typed decode
	}
	var next map[string]any
	if err := dec.Decode(&next); err == nil && len(next) > 0 {
		return fmt.Errorf("%s config %s must contain exactly one object (found multiple YAML documents)", kind, path)
	}
	return nil
}

// validate enforces the Lite config invariants (design §5.1.1): unique pool /
// tenant identities, a tenant namespace equal to its name, Lite-empty scheduling
// fields, predefined-volume rules, and every tenant quota referencing an
// existing pool.
func (sc *StaticConfig) validate() error {
	if len(sc.Pools) == 0 {
		return fmt.Errorf("at least one resource pool must be defined")
	}
	if len(sc.Tenants) == 0 {
		return fmt.Errorf("at least one tenant must be defined")
	}

	poolByName, err := validatePools(sc.Pools)
	if err != nil {
		return err
	}
	return validateTenants(sc.Tenants, poolByName)
}

// validatePools checks each pool's Lite invariants and returns a name→pool index
// for the tenant quota cross-checks.
func validatePools(pools []*cmv1alpha1.ResourcePool) (map[string]*cmv1alpha1.ResourcePool, error) {
	poolByName := make(map[string]*cmv1alpha1.ResourcePool, len(pools))
	for _, pool := range pools {
		if pool.Name == "" {
			return nil, fmt.Errorf("resource pool name must not be empty")
		}
		if _, dup := poolByName[pool.Name]; dup {
			return nil, fmt.Errorf("duplicate resource pool %q", pool.Name)
		}
		poolByName[pool.Name] = pool

		if len(pool.Spec.NodeSelector) != 0 || len(pool.Spec.Tolerations) != 0 {
			return nil, fmt.Errorf("resource pool %q nodeSelector/tolerations must be empty in Lite", pool.Name)
		}

		seen := map[string]struct{}{}
		for _, u := range pool.Spec.Units {
			if u.Name == "" {
				return nil, fmt.Errorf("resource pool %q: resource unit name must not be empty", pool.Name)
			}
			if _, dup := seen[u.Name]; dup {
				return nil, fmt.Errorf("resource pool %q: duplicate resource unit %q", pool.Name, u.Name)
			}
			seen[u.Name] = struct{}{}
			if len(u.NodeSelector) != 0 {
				return nil, fmt.Errorf("resource pool %q: resource unit %q nodeSelector must be empty in Lite", pool.Name, u.Name)
			}
		}
	}
	return poolByName, nil
}

// validateTenants checks each tenant's Lite invariants (unique name, namespace
// equal to name, credential-free initResources, predefined-volume rules) and
// cross-references its quotas against poolByName. hostPathOwner tracks hostPath
// volume names across all tenants: the Standalone Runtime looks them up in a
// single name-keyed map (Config.HostPathVolumes), so they must be globally
// unique even though managed volumes are namespaced per tenant.
func validateTenants(tenants []*tenantv1alpha1.Tenant, poolByName map[string]*cmv1alpha1.ResourcePool) error {
	tenantNames := map[string]struct{}{}
	hostPathOwner := map[string]string{}
	for _, tenant := range tenants {
		if tenant.Name == "" {
			return fmt.Errorf("tenant name must not be empty")
		}
		if _, dup := tenantNames[tenant.Name]; dup {
			return fmt.Errorf("duplicate tenant %q", tenant.Name)
		}
		tenantNames[tenant.Name] = struct{}{}

		// The Lite tenant scope IS the tenant name: the System contract defines the
		// tenant name as the CR name, namespace, and partition string. Requiring
		// them equal keeps Platform's namespace, the compute partition and the
		// runtime label consistent (which also makes namespaces unique per tenant).
		if ns := tenant.Spec.Namespace.Name; ns != tenant.Name {
			return fmt.Errorf("tenant %q namespace %q must equal the tenant name", tenant.Name, ns)
		}

		// Lite has no tenant-operator to copy Secrets/ConfigMaps/ServiceAccounts, so
		// those stay unsupported; predefined data volumes ARE supported — they are
		// seeded as managed Docker volumes at startup (see seedTenantVolumes).
		if !credentialInitResourcesEmpty(tenant.Spec.InitResources) {
			return fmt.Errorf("tenant %q initResources secrets/configMaps/serviceAccounts must be empty in Lite; only volumes are supported", tenant.Name)
		}
		if err := validateTenantVolumes(tenant.Name, tenant.Spec.InitResources.Volumes, hostPathOwner); err != nil {
			return err
		}

		if len(tenant.Spec.Quotas) == 0 {
			return fmt.Errorf("tenant %q must declare at least one quota", tenant.Name)
		}
		for _, q := range tenant.Spec.Quotas {
			if _, ok := poolByName[q.Pool]; !ok {
				return fmt.Errorf("tenant %q quota references unknown pool %q", tenant.Name, q.Pool)
			}
		}
	}
	return nil
}

// credentialInitResourcesEmpty reports whether the credential/RBAC init
// resources (everything except predefined volumes) are empty. Volumes are
// handled separately (validateTenantVolumes + seedTenantVolumes).
func credentialInitResourcesEmpty(ir tenantv1alpha1.InitResources) bool {
	return len(ir.ImagePullSecrets) == 0 &&
		len(ir.Secrets) == 0 &&
		len(ir.ConfigMaps) == 0 &&
		len(ir.ServiceAccounts) == 0
}

// validateTenantVolumes checks a tenant's predefined data volumes: each needs a
// non-empty name unique within the tenant (it becomes the Docker volume / claim
// name a workload mounts). A volume may set hostPath to bind-mount a host
// directory instead of a managed Docker volume (Lite-only); the path must be
// absolute, and because hostPath volumes resolve through the runtime's single
// name-keyed map, their names must be unique across ALL tenants (tracked in
// hostPathOwner). Size/storageClass/accessModes are accepted but ignored by the
// single-host Docker runtime.
func validateTenantVolumes(tenant string, vols []tenantv1alpha1.VolumeSpec, hostPathOwner map[string]string) error {
	seen := map[string]struct{}{}
	for i, v := range vols {
		if v.Name == "" {
			return fmt.Errorf("tenant %q initResources.volumes[%d].name is required", tenant, i)
		}
		if _, dup := seen[v.Name]; dup {
			return fmt.Errorf("tenant %q duplicate volume %q", tenant, v.Name)
		}
		seen[v.Name] = struct{}{}
		if v.HostPath != "" {
			if !filepath.IsAbs(v.HostPath) {
				return fmt.Errorf("tenant %q initResources.volumes[%d].hostPath %q must be an absolute path", tenant, i, v.HostPath)
			}
			if other, ok := hostPathOwner[v.Name]; ok {
				return fmt.Errorf("hostPath volume name %q is declared by tenants %q and %q; hostPath volume names must be unique across tenants", v.Name, other, tenant)
			}
			hostPathOwner[v.Name] = tenant
		}
	}
	return nil
}
