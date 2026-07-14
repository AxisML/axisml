// Package tenantresolver maps logical tenant names to the Kubernetes
// Namespace declared by the cluster-scoped Tenant CR.
package tenantresolver

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

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
