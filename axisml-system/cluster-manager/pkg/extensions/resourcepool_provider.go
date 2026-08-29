// Package extensions declares the deployment-form-neutral extension seams the
// cluster-manager REST handlers depend on — the interfaces an alternate
// deployment form (notably an external standalone deployment) must implement. A
// composition root — the Kubernetes binary, or an external standalone composition root — injects
// concrete providers:
//
//   - Kubernetes injects providers backed by the cluster-scoped ResourcePool /
//     Tenant CRs (full CRUD with optimistic locking) and a VolumeManager that
//     materialises PersistentVolumeClaims.
//   - A standalone deployment injects PostgreSQL-backed writable ResourcePool
//     and Tenant providers. Startup CR YAML is bootstrap input for those stores.
//     Its VolumeManager is writable, backed by managed Docker volumes — workspace
//     volumes are created on demand in every deployment form.
//
// The pool / tenant providers traffic in the shared CR API types; the
// VolumeManager trafficks in a neutral Volume value. The handlers own all
// request validation, business folding and HTTP translation.
package extensions

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
)

// ErrCapabilityUnavailable is returned by read-only providers for any
// write operation. Handlers map it to 409 CapabilityUnavailable.
var ErrCapabilityUnavailable = errors.New("capability unavailable in this deployment form")

// ResourcePoolProvider is the persistence seam for the ResourcePool CR (with its
// embedded spec.units[]). List controls travel in the standard
// metav1.ListOptions (LabelSelector / Limit / Continue).
type ResourcePoolProvider interface {
	Get(ctx context.Context, name string) (*cmv1alpha1.ResourcePool, error)
	List(ctx context.Context, opts metav1.ListOptions) (*cmv1alpha1.ResourcePoolList, error)
	Create(ctx context.Context, pool *cmv1alpha1.ResourcePool) error
	// Patch applies an optimistic merge of obj against its pre-mutation base.
	Patch(ctx context.Context, obj, base *cmv1alpha1.ResourcePool) error
	Delete(ctx context.Context, name string) error
}
