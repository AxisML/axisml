package core

import (
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

// DefaultName is the single tenant / pool / namespace identity Lite supports.
const DefaultName = "default"

// StaticConfig is the immutable snapshot parsed from the CR-YAML config at
// startup: the single default ResourcePool and Tenant (design §5.1.1). All
// config-backed providers read from this snapshot; changing it requires an
// axisml-core restart.
type StaticConfig struct {
	Pool   *cmv1alpha1.ResourcePool
	Tenant *tenantv1alpha1.Tenant
}

// LoadStaticConfig reads resource-pool.yaml and tenant.yaml from dir, decodes
// them with the AxisML API types and runs the cross-object validation. Any
// failure leaves axisml-core not-ready (design §5.1.1).
func LoadStaticConfig(dir string) (*StaticConfig, error) {
	pool, err := decodePool(filepath.Join(dir, "resource-pool.yaml"))
	if err != nil {
		return nil, err
	}
	tenant, err := decodeTenant(filepath.Join(dir, "tenant.yaml"))
	if err != nil {
		return nil, err
	}
	sc := &StaticConfig{Pool: pool, Tenant: tenant}
	if err := sc.validate(); err != nil {
		return nil, err
	}
	return sc, nil
}

func decodePool(path string) (*cmv1alpha1.ResourcePool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read resource-pool config: %w", err)
	}
	var p cmv1alpha1.ResourcePool
	if err := yaml.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("decode resource-pool config: %w", err)
	}
	return &p, nil
}

func decodeTenant(path string) (*tenantv1alpha1.Tenant, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tenant config: %w", err)
	}
	var t tenantv1alpha1.Tenant
	if err := yaml.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("decode tenant config: %w", err)
	}
	return &t, nil
}

// validate enforces the Lite config invariants (design §5.1.1).
func (sc *StaticConfig) validate() error {
	if sc.Pool.Name != DefaultName {
		return fmt.Errorf("resource pool name must be %q, got %q", DefaultName, sc.Pool.Name)
	}
	if sc.Tenant.Name != DefaultName {
		return fmt.Errorf("tenant name must be %q, got %q", DefaultName, sc.Tenant.Name)
	}
	if sc.Tenant.Spec.Namespace.Name != DefaultName {
		return fmt.Errorf("tenant namespace must be %q, got %q", DefaultName, sc.Tenant.Spec.Namespace.Name)
	}
	if len(sc.Pool.Spec.NodeSelector) != 0 || len(sc.Pool.Spec.Tolerations) != 0 {
		return fmt.Errorf("resource pool nodeSelector/tolerations must be empty in Lite")
	}
	if !initResourcesEmpty(sc.Tenant.Spec.InitResources) {
		return fmt.Errorf("tenant initResources must be empty in Lite")
	}

	// Unit names unique + unit nodeSelector empty.
	seen := map[string]struct{}{}
	for _, u := range sc.Pool.Spec.Units {
		if u.Name == "" {
			return fmt.Errorf("resource unit name must not be empty")
		}
		if _, dup := seen[u.Name]; dup {
			return fmt.Errorf("duplicate resource unit %q", u.Name)
		}
		seen[u.Name] = struct{}{}
		if len(u.NodeSelector) != 0 {
			return fmt.Errorf("resource unit %q nodeSelector must be empty in Lite", u.Name)
		}
	}

	// Quotas must reference only the default pool.
	if len(sc.Tenant.Spec.Quotas) == 0 {
		return fmt.Errorf("tenant must declare a quota for pool %q", DefaultName)
	}
	var max corev1.ResourceList
	for _, q := range sc.Tenant.Spec.Quotas {
		if q.Pool != DefaultName {
			return fmt.Errorf("tenant quota must reference pool %q, got %q", DefaultName, q.Pool)
		}
		max = q.Max
	}

	// Units must not exceed the declared quota max.
	for _, u := range sc.Pool.Spec.Units {
		if err := withinMax(u.Name, u.Requests, max); err != nil {
			return err
		}
		if err := withinMax(u.Name, u.Limits, max); err != nil {
			return err
		}
	}
	return nil
}

func withinMax(unit string, list, max corev1.ResourceList) error {
	for name, q := range list {
		m, ok := max[name]
		if !ok {
			continue // unconstrained dimension
		}
		if q.Cmp(m) > 0 {
			return fmt.Errorf("resource unit %q %s=%s exceeds quota max %s", unit, string(name), q.String(), m.String())
		}
	}
	return nil
}

func initResourcesEmpty(ir tenantv1alpha1.InitResources) bool {
	return len(ir.ImagePullSecrets) == 0 &&
		len(ir.Secrets) == 0 &&
		len(ir.ConfigMaps) == 0 &&
		len(ir.ServiceAccounts) == 0
}
