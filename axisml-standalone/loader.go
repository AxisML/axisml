package standalone

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
// startup: ResourcePool and Tenant bootstrap seeds. API mutations are persisted
// separately in PostgreSQL; changing these files does not overwrite an object
// that has already been imported.
type StaticConfig struct {
	Pools   []*cmv1alpha1.ResourcePool
	Tenants []*tenantv1alpha1.Tenant
}

// LoadStaticConfigOptions customizes static-config loading. LookupEnv resolves
// ${VAR} references; nil uses os.LookupEnv.
type LoadStaticConfigOptions struct {
	LookupEnv func(name string) (string, bool)
}

// LoadStaticConfig reads every ResourcePool under <dir>/resourcepools and every
// Tenant under <dir>/tenants, expands ${VAR} references from the process
// environment, decodes them strictly with the AxisML API types and runs the
// cross-object validation. Referenced variables must be set and non-empty. Any
// failure leaves axisml-standalone not-ready (design §5.1.1).
func LoadStaticConfig(dir string) (*StaticConfig, error) {
	return LoadStaticConfigWithOptions(dir, LoadStaticConfigOptions{LookupEnv: os.LookupEnv})
}

// LoadStaticConfigWithOptions is LoadStaticConfig with an injectable
// environment lookup. This lets embedding hosts provide a scoped environment
// without mutating process-global state.
func LoadStaticConfigWithOptions(dir string, opts LoadStaticConfigOptions) (*StaticConfig, error) {
	lookupEnv := opts.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	pools, err := loadDir[cmv1alpha1.ResourcePool](filepath.Join(dir, poolsSubdir), "resource pool", lookupEnv)
	if err != nil {
		return nil, err
	}
	tenants, err := loadDir[tenantv1alpha1.Tenant](filepath.Join(dir, tenantsSubdir), "tenant", lookupEnv)
	if err != nil {
		return nil, err
	}
	sc := &StaticConfig{Pools: pools, Tenants: tenants}
	if err := sc.Validate(); err != nil {
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
func loadDir[T any](dir, kind string, lookupEnv func(string) (string, bool)) ([]*T, error) {
	paths, err := yamlFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s config dir %s: %w", kind, dir, err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no %s config found in %s", kind, dir)
	}
	out := make([]*T, 0, len(paths))
	for _, p := range paths {
		obj, err := decodeObject[T](p, kind, lookupEnv)
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
func decodeObject[T any](path, kind string, lookupEnv func(string) (string, bool)) (*T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s config %s: %w", kind, path, err)
	}
	b, err = expandEnvironment(b, lookupEnv)
	if err != nil {
		return nil, fmt.Errorf("expand %s config %s: %w", kind, path, err)
	}
	if err := singleDocument(b, path, kind); err != nil {
		return nil, err
	}
	var obj T
	if err := yaml.UnmarshalStrict(b, &obj); err != nil {
		return nil, fmt.Errorf("decode %s config %s: %w", kind, path, err)
	}
	return &obj, nil
}

var environmentReference = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// expandEnvironment replaces only braced environment references. Bare $VAR is
// deliberately left untouched so ordinary dollar signs in YAML values retain
// their literal meaning.
func expandEnvironment(b []byte, lookupEnv func(string) (string, bool)) ([]byte, error) {
	matches := environmentReference.FindAllSubmatchIndex(b, -1)
	if len(matches) == 0 {
		return b, nil
	}

	var out bytes.Buffer
	last := 0
	for _, match := range matches {
		name := string(b[match[2]:match[3]])
		value, ok := lookupEnv(name)
		if !ok || value == "" {
			return nil, fmt.Errorf("environment variable %q is unset or empty", name)
		}
		_, _ = out.Write(b[last:match[0]])
		_, _ = out.WriteString(value)
		last = match[1]
	}
	_, _ = out.Write(b[last:])
	return out.Bytes(), nil
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

// Validate enforces the standalone config invariants: unique pool / tenant
// identities, a tenant namespace equal to its name, empty scheduling
// fields, predefined-volume rules, and every tenant quota referencing an
// existing pool.
func (sc *StaticConfig) Validate() error {
	if sc == nil {
		return fmt.Errorf("static config must not be nil")
	}
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

// validatePools checks each pool's standalone invariants and returns a name→pool index
// for the tenant quota cross-checks.
func validatePools(pools []*cmv1alpha1.ResourcePool) (map[string]*cmv1alpha1.ResourcePool, error) {
	poolByName := make(map[string]*cmv1alpha1.ResourcePool, len(pools))
	for _, pool := range pools {
		if pool == nil {
			return nil, fmt.Errorf("resource pool must not be nil")
		}
		if pool.Name == "" {
			return nil, fmt.Errorf("resource pool name must not be empty")
		}
		if _, dup := poolByName[pool.Name]; dup {
			return nil, fmt.Errorf("duplicate resource pool %q", pool.Name)
		}
		poolByName[pool.Name] = pool

		if len(pool.Spec.NodeSelector) != 0 || len(pool.Spec.Tolerations) != 0 {
			return nil, fmt.Errorf("resource pool %q nodeSelector/tolerations must be empty in standalone", pool.Name)
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
				return nil, fmt.Errorf("resource pool %q: resource unit %q nodeSelector must be empty in standalone", pool.Name, u.Name)
			}
			field := fmt.Sprintf("resource pool %q resource unit %q", pool.Name, u.Name)
			if err := validateStandaloneResourceList(field+" requests", u.Requests); err != nil {
				return nil, err
			}
			if err := validateStandaloneResourceList(field+" limits", u.Limits); err != nil {
				return nil, err
			}
		}
	}
	return poolByName, nil
}

// validateTenants checks each tenant's standalone invariants (unique name, namespace
// equal to name, credential-free initResources, predefined-volume rules) and
// cross-references its quotas against poolByName. hostPathOwner tracks hostPath
// volume names across all tenants: the Standalone Runtime looks them up in a
// single name-keyed map (Config.HostPathVolumes), so they must be globally
// unique even though managed volumes are namespaced per tenant.
func validateTenants(tenants []*tenantv1alpha1.Tenant, poolByName map[string]*cmv1alpha1.ResourcePool) error {
	tenantNames := map[string]struct{}{}
	hostPathOwner := map[string]string{}
	for _, tenant := range tenants {
		if tenant == nil {
			return fmt.Errorf("tenant must not be nil")
		}
		if tenant.Name == "" {
			return fmt.Errorf("tenant name must not be empty")
		}
		if _, dup := tenantNames[tenant.Name]; dup {
			return fmt.Errorf("duplicate tenant %q", tenant.Name)
		}
		tenantNames[tenant.Name] = struct{}{}

		// The standalone tenant scope is the tenant name: the System contract defines the
		// tenant name as the CR name, namespace, and partition string. Requiring
		// them equal keeps Platform's namespace, the compute partition and the
		// runtime label consistent (which also makes namespaces unique per tenant).
		if ns := tenant.Spec.Namespace.Name; ns != tenant.Name {
			return fmt.Errorf("tenant %q namespace %q must equal the tenant name", tenant.Name, ns)
		}

		// Standalone has no tenant-operator to copy Secrets/ConfigMaps/ServiceAccounts, so
		// those stay unsupported; predefined data volumes ARE supported — they are
		// seeded as managed Docker volumes at startup (see seedTenantVolumes).
		if !credentialInitResourcesEmpty(tenant.Spec.InitResources) {
			return fmt.Errorf("tenant %q initResources secrets/configMaps/serviceAccounts must be empty in standalone; only volumes are supported", tenant.Name)
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
			field := fmt.Sprintf("tenant %q quota for pool %q", tenant.Name, q.Pool)
			if err := validateStandaloneResourceList(field+" min", q.Min); err != nil {
				return err
			}
			if err := validateStandaloneResourceList(field+" max", q.Max); err != nil {
				return err
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
// directory instead of a managed Docker volume (standalone only); the path must be
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
