package core

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	cmv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	cmprovider "github.com/axisml/axisml/components/cluster-manager/pkg/provider"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	csprovider "github.com/axisml/axisml/components/compute-service/pkg/provider"
	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	"context"
)

const apiGroup = "axisml.io"

// ConfigResourceCatalog serves the single default ResourcePool from the static
// CR-YAML config. It satisfies BOTH the cluster-manager ResourcePoolStore
// (read-only REST surface) and the compute-service ResourceCatalog (resolve a
// (pool, unit) into a resource snapshot). Writes return ErrCapabilityUnavailable
// (design §5.1); the handlers map that to 409 CapabilityUnavailable.
type ConfigResourceCatalog struct {
	pool *cmv1alpha1.ResourcePool
}

var (
	_ cmprovider.ResourcePoolStore = (*ConfigResourceCatalog)(nil)
	_ csprovider.ResourceCatalog   = (*ConfigResourceCatalog)(nil)
)

// NewConfigResourceCatalog builds the catalog over the parsed default pool.
func NewConfigResourceCatalog(pool *cmv1alpha1.ResourcePool) *ConfigResourceCatalog {
	return &ConfigResourceCatalog{pool: pool}
}

func poolGR() schema.GroupResource {
	return schema.GroupResource{Group: apiGroup, Resource: "resourcepools"}
}

// Get returns the default pool; any other name is NotFound.
func (c *ConfigResourceCatalog) Get(_ context.Context, name string) (*cmv1alpha1.ResourcePool, error) {
	if name != c.pool.Name {
		return nil, apierrors.NewNotFound(poolGR(), name)
	}
	return c.pool.DeepCopy(), nil
}

// List returns the single default pool.
func (c *ConfigResourceCatalog) List(_ context.Context, _ cmprovider.ListParams) (*cmv1alpha1.ResourcePoolList, error) {
	return &cmv1alpha1.ResourcePoolList{Items: []cmv1alpha1.ResourcePool{*c.pool.DeepCopy()}}, nil
}

// Create is unavailable in Lite (single read-only default pool).
func (c *ConfigResourceCatalog) Create(context.Context, *cmv1alpha1.ResourcePool) error {
	return cmprovider.ErrCapabilityUnavailable
}

// Patch is unavailable in Lite.
func (c *ConfigResourceCatalog) Patch(context.Context, *cmv1alpha1.ResourcePool, *cmv1alpha1.ResourcePool) error {
	return cmprovider.ErrCapabilityUnavailable
}

// Delete is unavailable in Lite.
func (c *ConfigResourceCatalog) Delete(context.Context, string) error {
	return cmprovider.ErrCapabilityUnavailable
}

// Writable reports the Lite config-backed pool store is read-only.
func (c *ConfigResourceCatalog) Writable() bool { return false }

// Resolve expands (poolName, unitName) into the resource snapshot frozen into
// the workload spec at create time. Lite keeps nodeSelector/tolerations empty
// (validated at load); only requests/limits carry through to Docker limits.
func (c *ConfigResourceCatalog) Resolve(_ context.Context, poolName, unitName string) (*csprovider.Expanded, error) {
	if poolName == "" || unitName == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "poolName and unitName are required")
	}
	if poolName != c.pool.Name {
		return nil, apperrors.Newf(apperrors.CodeValidation, "resource pool %q not found", poolName)
	}
	for i := range c.pool.Spec.Units {
		u := &c.pool.Spec.Units[i]
		if u.Name != unitName {
			continue
		}
		return &csprovider.Expanded{
			Requests: u.Requests,
			Limits:   u.Limits,
		}, nil
	}
	return nil, apperrors.Newf(apperrors.CodeValidation, "resource unit %q not found in pool %q", unitName, poolName)
}

// StaticTenantStore serves the single default Tenant from the static CR-YAML
// config. Get/List are available; Create is idempotent for the default tenant
// only; all other writes return ErrCapabilityUnavailable (design §5.1).
type StaticTenantStore struct {
	tenant *tenantv1alpha1.Tenant
}

var _ cmprovider.TenantStore = (*StaticTenantStore)(nil)

// NewStaticTenantStore builds the store over the parsed default tenant.
func NewStaticTenantStore(tenant *tenantv1alpha1.Tenant) *StaticTenantStore {
	return &StaticTenantStore{tenant: tenant}
}

func tenantGR() schema.GroupResource {
	return schema.GroupResource{Group: apiGroup, Resource: "tenants"}
}

// Get returns the default tenant; any other name is NotFound.
func (s *StaticTenantStore) Get(_ context.Context, name string) (*tenantv1alpha1.Tenant, error) {
	if name != s.tenant.Name {
		return nil, apierrors.NewNotFound(tenantGR(), name)
	}
	return s.tenant.DeepCopy(), nil
}

// List returns the single default tenant.
func (s *StaticTenantStore) List(_ context.Context, _ cmprovider.ListParams) (*tenantv1alpha1.TenantList, error) {
	return &tenantv1alpha1.TenantList{Items: []tenantv1alpha1.Tenant{*s.tenant.DeepCopy()}}, nil
}

// Create is idempotent for the default tenant and unavailable for any other
// name (Lite is single-tenant; the handler maps the error to 409).
func (s *StaticTenantStore) Create(_ context.Context, t *tenantv1alpha1.Tenant) error {
	if t != nil && t.Name == s.tenant.Name {
		return nil
	}
	return cmprovider.ErrCapabilityUnavailable
}

// Patch is unavailable in Lite.
func (s *StaticTenantStore) Patch(context.Context, *tenantv1alpha1.Tenant, *tenantv1alpha1.Tenant) error {
	return cmprovider.ErrCapabilityUnavailable
}

// Delete is unavailable in Lite.
func (s *StaticTenantStore) Delete(context.Context, string) error {
	return cmprovider.ErrCapabilityUnavailable
}

// Writable reports the Lite single-tenant config store is not multi-tenant.
func (s *StaticTenantStore) Writable() bool { return false }
