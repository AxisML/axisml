package extensions

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

// TenantProvider is the persistence seam for the Tenant CR. List controls travel
// in the standard metav1.ListOptions (LabelSelector / Limit / Continue).
type TenantProvider interface {
	Get(ctx context.Context, name string) (*tenantv1alpha1.Tenant, error)
	List(ctx context.Context, opts metav1.ListOptions) (*tenantv1alpha1.TenantList, error)
	Create(ctx context.Context, tenant *tenantv1alpha1.Tenant) error
	Patch(ctx context.Context, obj, base *tenantv1alpha1.Tenant) error
	Delete(ctx context.Context, name string) error
	// Writable reports whether multi-tenant writes are available (true for the
	// Kubernetes provider, false for the Lite single-tenant config provider).
	Writable() bool
}
