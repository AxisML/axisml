// Package k8sstore is the Kubernetes implementation of the cluster-manager
// provider stores: full CRUD over the cluster-scoped ResourcePool and Tenant
// CRs via a controller-runtime client.Client. The Lite form supplies read-only
// config-backed stores instead.
package k8sstore

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	"github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
)

// ResourcePoolStore backs extensions.ResourcePoolProvider with a client.Client.
type ResourcePoolStore struct {
	c client.Client
}

var _ extensions.ResourcePoolProvider = (*ResourcePoolStore)(nil)

// NewResourcePoolStore builds a ResourcePoolStore.
func NewResourcePoolStore(c client.Client) *ResourcePoolStore { return &ResourcePoolStore{c: c} }

func (s *ResourcePoolStore) Get(ctx context.Context, name string) (*cmv1alpha1.ResourcePool, error) {
	pool := &cmv1alpha1.ResourcePool{}
	if err := s.c.Get(ctx, types.NamespacedName{Name: name}, pool); err != nil {
		return nil, err
	}
	return pool, nil
}

func (s *ResourcePoolStore) List(ctx context.Context, listOpts metav1.ListOptions) (*cmv1alpha1.ResourcePoolList, error) {
	opts, err := listOptions(listOpts)
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
	// MergeFromWithOptimisticLock embeds base's resourceVersion as a precondition
	// so a concurrent write makes the API server return 409; mutateWithRetry then
	// re-reads and replays the mutation. A plain MergeFrom would silently clobber.
	return s.c.Patch(ctx, obj, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{}))
}

func (s *ResourcePoolStore) Delete(ctx context.Context, name string) error {
	pool := &cmv1alpha1.ResourcePool{}
	pool.Name = name
	return s.c.Delete(ctx, pool)
}

// Writable reports the Kubernetes store accepts full CRUD.
func (s *ResourcePoolStore) Writable() bool { return true }

// TenantStore backs extensions.TenantProvider with a client.Client.
type TenantStore struct {
	c client.Client
}

var _ extensions.TenantProvider = (*TenantStore)(nil)

// NewTenantStore builds a TenantStore.
func NewTenantStore(c client.Client) *TenantStore { return &TenantStore{c: c} }

func (s *TenantStore) Get(ctx context.Context, name string) (*tenantv1alpha1.Tenant, error) {
	cr := &tenantv1alpha1.Tenant{}
	if err := s.c.Get(ctx, types.NamespacedName{Name: name}, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

func (s *TenantStore) List(ctx context.Context, listOpts metav1.ListOptions) (*tenantv1alpha1.TenantList, error) {
	opts, err := listOptions(listOpts)
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
	// Optimistic lock: surface concurrent writes as 409 so mutateWithRetry replays
	// the mutation against a fresh read rather than silently overwriting.
	return s.c.Patch(ctx, obj, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{}))
}

func (s *TenantStore) Delete(ctx context.Context, name string) error {
	cr := &tenantv1alpha1.Tenant{}
	cr.Name = name
	return s.c.Delete(ctx, cr)
}

// Writable reports the Kubernetes store supports multi-tenant CRUD.
func (s *TenantStore) Writable() bool { return true }

// listOptions translates the standard metav1.ListOptions into controller-runtime
// options. The selector is re-parsed here (handlers pre-validate it for the 400
// path).
func listOptions(listOpts metav1.ListOptions) ([]client.ListOption, error) {
	opts := []client.ListOption{}
	if listOpts.LabelSelector != "" {
		ps, err := labels.Parse(listOpts.LabelSelector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: ps})
	}
	if listOpts.Limit > 0 {
		opts = append(opts, client.Limit(listOpts.Limit))
	}
	if listOpts.Continue != "" {
		opts = append(opts, client.Continue(listOpts.Continue))
	}
	return opts, nil
}
