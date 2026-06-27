// Package extensions declares the deployment-form-neutral extension seams the
// cluster-manager REST handlers depend on — the interfaces an alternate
// deployment form (notably AxisML Lite's axisml-core) must implement. A
// composition root — the Kubernetes binary, or Lite's axisml-core — injects
// concrete stores:
//
//   - Kubernetes injects stores backed by the cluster-scoped ResourcePool /
//     Tenant CRs (full CRUD with optimistic locking) and a VolumeStore that
//     materialises PersistentVolumeClaims.
//   - Lite injects read-only stores backed by the static CR-YAML config; write
//     operations return ErrCapabilityUnavailable, which the handlers surface as
//     409 CapabilityUnavailable (design §5.1). Its VolumeStore is writable,
//     backed by managed Docker volumes — workspace volumes are created on demand
//     in every deployment form.
//
// The pool / tenant stores traffic in the shared CR API types; the VolumeStore
// trafficks in a neutral Volume value. The handlers own all request validation,
// business folding and HTTP translation.
package extensions

import (
	"context"
	"errors"

	cmv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// ErrCapabilityUnavailable is returned by read-only (Lite) stores for any write
// operation. Handlers map it to 409 CapabilityUnavailable.
var ErrCapabilityUnavailable = errors.New("capability unavailable in this deployment form")

// ListParams are the neutral list controls parsed from the HTTP query.
type ListParams struct {
	Selector string // raw K8s labelSelector ("" = match all)
	Limit    int    // 0 = server default
	Continue string // opaque page cursor
}

// ResourcePoolStore is the persistence seam for the ResourcePool CR (with its
// embedded spec.units[]).
type ResourcePoolStore interface {
	Get(ctx context.Context, name string) (*cmv1alpha1.ResourcePool, error)
	List(ctx context.Context, params ListParams) (*cmv1alpha1.ResourcePoolList, error)
	Create(ctx context.Context, pool *cmv1alpha1.ResourcePool) error
	// Patch applies an optimistic merge of obj against its pre-mutation base.
	Patch(ctx context.Context, obj, base *cmv1alpha1.ResourcePool) error
	Delete(ctx context.Context, name string) error
	// Writable reports whether the store accepts writes (Create/Patch/Delete).
	// The Kubernetes store returns true; the Lite read-only config store returns
	// false. It backs the cluster-manager capability document.
	Writable() bool
}

// Volume is the neutral representation of a durable volume cluster-manager
// materialises (design §3.4). The Kubernetes store backs it with a
// PersistentVolumeClaim; the Lite store backs it with a managed Docker volume.
// cluster-manager does not interpret the volume's purpose — the deterministic
// naming (e.g. a workspace's axisml-ws-<svc>-data) and the pod mount are the
// caller's job; this service only materialises / reclaims by (Namespace, Name).
type Volume struct {
	Namespace string
	Name      string
	// Size is a Kubernetes Quantity string. Required for the Kubernetes store;
	// the Lite Docker store accepts it for parity but ignores it (a single-host
	// volume has no fixed size and grows on demand).
	Size string
	// StorageClass selects the backing StorageClass ("" = cluster default).
	// Ignored by the Lite store.
	StorageClass string
}

// VolumeStore is the persistence seam for durable volumes. Both operations are
// idempotent: Ensure treats an already-existing volume as success; Delete treats
// a missing volume as success. Unlike the pool / tenant stores there is no
// Writable() variance — volumes are writable in every deployment form.
type VolumeStore interface {
	Ensure(ctx context.Context, v Volume) error
	Delete(ctx context.Context, namespace, name string) error
}

// TenantStore is the persistence seam for the Tenant CR.
type TenantStore interface {
	Get(ctx context.Context, name string) (*tenantv1alpha1.Tenant, error)
	List(ctx context.Context, params ListParams) (*tenantv1alpha1.TenantList, error)
	Create(ctx context.Context, tenant *tenantv1alpha1.Tenant) error
	Patch(ctx context.Context, obj, base *tenantv1alpha1.Tenant) error
	Delete(ctx context.Context, name string) error
	// Writable reports whether multi-tenant writes are available (true for the
	// Kubernetes store, false for the Lite single-tenant config store).
	Writable() bool
}
