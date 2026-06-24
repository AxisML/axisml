// Package k8sstore is the Kubernetes implementation of the cluster-manager
// provider stores: full CRUD over the cluster-scoped ResourcePool and Tenant
// CRs via a controller-runtime client.Client. The Lite form supplies read-only
// config-backed stores instead.
package k8sstore

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cmv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	"github.com/axisml/axisml/components/cluster-manager/pkg/provider"
	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// ResourcePoolStore backs provider.ResourcePoolStore with a client.Client.
type ResourcePoolStore struct {
	c client.Client
}

var _ provider.ResourcePoolStore = (*ResourcePoolStore)(nil)

// NewResourcePoolStore builds a ResourcePoolStore.
func NewResourcePoolStore(c client.Client) *ResourcePoolStore { return &ResourcePoolStore{c: c} }

func (s *ResourcePoolStore) Get(ctx context.Context, name string) (*cmv1alpha1.ResourcePool, error) {
	pool := &cmv1alpha1.ResourcePool{}
	if err := s.c.Get(ctx, types.NamespacedName{Name: name}, pool); err != nil {
		return nil, err
	}
	return pool, nil
}

func (s *ResourcePoolStore) List(ctx context.Context, params provider.ListParams) (*cmv1alpha1.ResourcePoolList, error) {
	opts, err := listOptions(params)
	if err != nil {
		return nil, err
	}
	list := &cmv1alpha1.ResourcePoolList{}
	if err := s.c.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ResourcePoolStore) Create(ctx context.Context, pool *cmv1alpha1.ResourcePool) error {
	return s.c.Create(ctx, pool)
}

func (s *ResourcePoolStore) Patch(ctx context.Context, obj, base *cmv1alpha1.ResourcePool) error {
	return s.c.Patch(ctx, obj, client.MergeFrom(base))
}

func (s *ResourcePoolStore) Delete(ctx context.Context, name string) error {
	pool := &cmv1alpha1.ResourcePool{}
	pool.Name = name
	return s.c.Delete(ctx, pool)
}

// Writable reports the Kubernetes store accepts full CRUD.
func (s *ResourcePoolStore) Writable() bool { return true }

// TenantStore backs provider.TenantStore with a client.Client.
type TenantStore struct {
	c client.Client
}

var _ provider.TenantStore = (*TenantStore)(nil)

// NewTenantStore builds a TenantStore.
func NewTenantStore(c client.Client) *TenantStore { return &TenantStore{c: c} }

func (s *TenantStore) Get(ctx context.Context, name string) (*tenantv1alpha1.Tenant, error) {
	cr := &tenantv1alpha1.Tenant{}
	if err := s.c.Get(ctx, types.NamespacedName{Name: name}, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

func (s *TenantStore) List(ctx context.Context, params provider.ListParams) (*tenantv1alpha1.TenantList, error) {
	opts, err := listOptions(params)
	if err != nil {
		return nil, err
	}
	list := &tenantv1alpha1.TenantList{}
	if err := s.c.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *TenantStore) Create(ctx context.Context, tenant *tenantv1alpha1.Tenant) error {
	return s.c.Create(ctx, tenant)
}

func (s *TenantStore) Patch(ctx context.Context, obj, base *tenantv1alpha1.Tenant) error {
	return s.c.Patch(ctx, obj, client.MergeFrom(base))
}

func (s *TenantStore) Delete(ctx context.Context, name string) error {
	cr := &tenantv1alpha1.Tenant{}
	cr.Name = name
	return s.c.Delete(ctx, cr)
}

// Writable reports the Kubernetes store supports multi-tenant CRUD.
func (s *TenantStore) Writable() bool { return true }

// listOptions translates neutral ListParams into controller-runtime options.
// The selector is re-parsed here (handlers pre-validate it for the 400 path).
func listOptions(params provider.ListParams) ([]client.ListOption, error) {
	opts := []client.ListOption{}
	if params.Selector != "" {
		ps, err := labels.Parse(params.Selector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: ps})
	}
	if params.Limit > 0 {
		opts = append(opts, client.Limit(int64(params.Limit)))
	}
	if params.Continue != "" {
		opts = append(opts, client.Continue(params.Continue))
	}
	return opts, nil
}
