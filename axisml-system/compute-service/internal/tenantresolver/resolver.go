// Package tenantresolver maps logical tenant names to the Kubernetes
// Namespace declared by the cluster-scoped Tenant CR.
package tenantresolver

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

var _ extensions.QuotaResolver = (*Reader)(nil)

// Reader resolves tenant namespaces through a live API reader. Avoiding the
// informer cache here prevents a newly-created Tenant from briefly resolving
// to the legacy identity mapping before the cache observes it.
type Reader struct {
	c client.Reader
}

// New returns a tenant namespace resolver over c.
func New(c client.Reader) *Reader { return &Reader{c: c} }

// ResolveNamespace returns Tenant.spec.namespace.name. A missing Tenant falls
// back to the logical tenant name, preserving the existing one-namespace-per-
// tenant layout where the logical name is already the physical Namespace.
func (r *Reader) ResolveNamespace(ctx context.Context, tenant string) (string, error) {
	var cr tenantv1alpha1.Tenant
	if err := r.c.Get(ctx, types.NamespacedName{Name: tenant}, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			return tenant, nil
		}
		return "", fmt.Errorf("read Tenant %q: %w", tenant, err)
	}
	if cr.Spec.Namespace.Name == "" {
		return "", fmt.Errorf("tenant %q has empty spec.namespace.name", tenant)
	}
	return cr.Spec.Namespace.Name, nil
}

// ResolveQuota returns Tenant.spec.quotas[pool].max for queue admission.
func (r *Reader) ResolveQuota(ctx context.Context, tenant, pool string) (corev1.ResourceList, error) {
	var cr tenantv1alpha1.Tenant
	if err := r.c.Get(ctx, types.NamespacedName{Name: tenant}, &cr); err != nil {
		return nil, fmt.Errorf("read Tenant %q: %w", tenant, err)
	}
	for _, quota := range cr.Spec.Quotas {
		if quota.Pool == pool {
			return quota.Max.DeepCopy(), nil
		}
	}
	return nil, fmt.Errorf("tenant %q has no quota for resource pool %q", tenant, pool)
}
