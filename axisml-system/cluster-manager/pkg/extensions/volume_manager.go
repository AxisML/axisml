package extensions

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// VolumeManager is the persistence seam for durable volumes, each modelled as a
// Kubernetes PersistentVolumeClaim (design §3.4). The Kubernetes implementation
// materialises the PVC directly; the Lite implementation maps it onto a managed
// Docker volume, reading only ObjectMeta (Namespace/Name) and ignoring the
// K8s-specific spec fields a single-host volume has no use for. cluster-manager
// does not interpret the volume's purpose — the deterministic claim naming
// (e.g. a workspace's axisml-ws-<svc>-data) and the pod mount are the caller's
// job; this seam only materialises / reclaims by the claim's key.
//
// Both operations are idempotent: Ensure treats an already-existing volume as
// success; Delete treats a missing volume as success. Unlike the pool / tenant
// providers there is no Writable() variance — volumes are writable in every
// deployment form.
type VolumeManager interface {
	Ensure(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error
	Delete(ctx context.Context, key types.NamespacedName) error
}
