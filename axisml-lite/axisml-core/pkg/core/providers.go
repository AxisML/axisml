package core

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	cmext "github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	csext "github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"

	"context"
)

const apiGroup = "axisml.io"

// ConfigResourceCatalog serves the read-only ResourcePools parsed from the
// static CR-YAML config. It satisfies BOTH the cluster-manager
// ResourcePoolProvider (read-only REST surface) and the compute-service
// ResourceResolver (look up a pool / unit by name). Writes return
// ErrCapabilityUnavailable (design §5.1); the handlers map that to 409
// CapabilityUnavailable.
type ConfigResourceCatalog struct {
	// order preserves the config load order for List; byName indexes it for Get.
	order  []*cmv1alpha1.ResourcePool
	byName map[string]*cmv1alpha1.ResourcePool
}

var (
	_ cmext.ResourcePoolProvider = (*ConfigResourceCatalog)(nil)
	_ csext.ResourceResolver     = (*ConfigResourceCatalog)(nil)
)

// NewConfigResourceCatalog builds the catalog over the parsed pools (one or
// more). Pool names are unique — validated at config load; a duplicate name here
// is dropped first-wins so order and byName stay consistent.
func NewConfigResourceCatalog(pools ...*cmv1alpha1.ResourcePool) *ConfigResourceCatalog {
	byName := make(map[string]*cmv1alpha1.ResourcePool, len(pools))
	order := make([]*cmv1alpha1.ResourcePool, 0, len(pools))
	for _, p := range pools {
		if _, dup := byName[p.Name]; dup {
			continue
		}
		byName[p.Name] = p
		order = append(order, p)
	}
	return &ConfigResourceCatalog{order: order, byName: byName}
}

func poolGR() schema.GroupResource {
	return schema.GroupResource{Group: apiGroup, Resource: "resourcepools"}
}

// Get returns the named pool; an unknown name is NotFound.
func (c *ConfigResourceCatalog) Get(_ context.Context, name string) (*cmv1alpha1.ResourcePool, error) {
	pool, ok := c.byName[name]
	if !ok {
		return nil, apierrors.NewNotFound(poolGR(), name)
	}
	return pool.DeepCopy(), nil
}

// List returns all config pools in load order.
func (c *ConfigResourceCatalog) List(_ context.Context, _ metav1.ListOptions) (*cmv1alpha1.ResourcePoolList, error) {
	items := make([]cmv1alpha1.ResourcePool, 0, len(c.order))
	for _, p := range c.order {
		items = append(items, *p.DeepCopy())
	}
	return &cmv1alpha1.ResourcePoolList{Items: items}, nil
}

// Create is unavailable in Lite (pools are read-only config).
func (c *ConfigResourceCatalog) Create(context.Context, *cmv1alpha1.ResourcePool) error {
	return cmext.ErrCapabilityUnavailable
}

// Patch is unavailable in Lite.
func (c *ConfigResourceCatalog) Patch(context.Context, *cmv1alpha1.ResourcePool, *cmv1alpha1.ResourcePool) error {
	return cmext.ErrCapabilityUnavailable
}

// Delete is unavailable in Lite.
func (c *ConfigResourceCatalog) Delete(context.Context, string) error {
	return cmext.ErrCapabilityUnavailable
}

// Writable reports the Lite config-backed pool store is read-only.
func (c *ConfigResourceCatalog) Writable() bool { return false }

// ResolveResourcePool returns the named pool, or a validation error for an
// unknown name (the business layer maps it to 400).
func (c *ConfigResourceCatalog) ResolveResourcePool(_ context.Context, name string) (*cmv1alpha1.ResourcePool, error) {
	if name == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "poolName is required")
	}
	pool, ok := c.byName[name]
	if !ok {
		return nil, apperrors.Newf(apperrors.CodeValidation, "resource pool %q not found", name)
	}
	return pool.DeepCopy(), nil
}

// ResolveResourceUnit returns the named unit from the named pool. Lite keeps
// nodeSelector/tolerations empty (validated at load); only requests/limits
// carry through to Docker limits when the business layer expands the snapshot.
func (c *ConfigResourceCatalog) ResolveResourceUnit(ctx context.Context, poolName, unitName string) (*cmv1alpha1.ResourceUnit, error) {
	if unitName == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "unitName is required")
	}
	pool, err := c.ResolveResourcePool(ctx, poolName)
	if err != nil {
		return nil, err
	}
	for i := range pool.Spec.Units {
		if pool.Spec.Units[i].Name == unitName {
			return &pool.Spec.Units[i], nil
		}
	}
	return nil, apperrors.Newf(apperrors.CodeValidation, "resource unit %q not found in pool %q", unitName, poolName)
}

// StaticTenantStore serves the read-only Tenants parsed from the static CR-YAML
// config. Get/List are available; Create is idempotent for a configured tenant
// only; all other writes return ErrCapabilityUnavailable (design §5.1).
type StaticTenantStore struct {
	// order preserves the config load order for List; byName indexes it for Get.
	order  []*tenantv1alpha1.Tenant
	byName map[string]*tenantv1alpha1.Tenant
}

var _ cmext.TenantProvider = (*StaticTenantStore)(nil)

// NewStaticTenantStore builds the store over the parsed tenants (one or more).
// Tenant names are unique — validated at config load; a duplicate name here is
// dropped first-wins so order and byName stay consistent.
func NewStaticTenantStore(tenants ...*tenantv1alpha1.Tenant) *StaticTenantStore {
	byName := make(map[string]*tenantv1alpha1.Tenant, len(tenants))
	order := make([]*tenantv1alpha1.Tenant, 0, len(tenants))
	for _, t := range tenants {
		if _, dup := byName[t.Name]; dup {
			continue
		}
		byName[t.Name] = t
		order = append(order, t)
	}
	return &StaticTenantStore{order: order, byName: byName}
}

func tenantGR() schema.GroupResource {
	return schema.GroupResource{Group: apiGroup, Resource: "tenants"}
}

// Get returns the named tenant; an unknown name is NotFound.
func (s *StaticTenantStore) Get(_ context.Context, name string) (*tenantv1alpha1.Tenant, error) {
	tenant, ok := s.byName[name]
	if !ok {
		return nil, apierrors.NewNotFound(tenantGR(), name)
	}
	return tenant.DeepCopy(), nil
}

// List returns all config tenants in load order.
func (s *StaticTenantStore) List(_ context.Context, _ metav1.ListOptions) (*tenantv1alpha1.TenantList, error) {
	items := make([]tenantv1alpha1.Tenant, 0, len(s.order))
	for _, t := range s.order {
		items = append(items, *t.DeepCopy())
	}
	return &tenantv1alpha1.TenantList{Items: items}, nil
}

// Create is idempotent for a configured tenant and unavailable for any other
// name (Lite tenants are read-only config; the handler maps the error to 409).
func (s *StaticTenantStore) Create(_ context.Context, t *tenantv1alpha1.Tenant) error {
	if t != nil {
		if _, ok := s.byName[t.Name]; ok {
			return nil
		}
	}
	return cmext.ErrCapabilityUnavailable
}

// Patch is unavailable in Lite.
func (s *StaticTenantStore) Patch(context.Context, *tenantv1alpha1.Tenant, *tenantv1alpha1.Tenant) error {
	return cmext.ErrCapabilityUnavailable
}

// Delete is unavailable in Lite.
func (s *StaticTenantStore) Delete(context.Context, string) error {
	return cmext.ErrCapabilityUnavailable
}

// Writable reports the Lite config-backed tenant store is read-only (tenants are
// config-defined presets, not created through the API).
func (s *StaticTenantStore) Writable() bool { return false }
