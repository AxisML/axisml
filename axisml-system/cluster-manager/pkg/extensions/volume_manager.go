package extensions

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// VolumeManager is the persistence seam for durable data volumes, each modelled
// as a Kubernetes PersistentVolumeClaim (design §3.4). The Kubernetes
// implementation materialises the PVC directly; a single-host implementation maps it
// onto a managed Docker volume, reading only the fields a single-host volume can
// honour. cluster-manager does not interpret the volume's purpose — the
// deterministic claim naming and the pod mount are the caller's job; this seam
// materialises / reads / reclaims by the claim's key and reports occupancy.
//
// Idempotency: Ensure treats an already-existing volume as success; Delete
// treats a missing volume as success. Volumes are writable in every deployment
// form, though Patch (expand / relabel) may be unavailable in a single-host
// deployment.
type VolumeManager interface {
	// Ensure materialises the backing volume from the supplied PVC spec.
	Ensure(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error
	// Get reads back the backing volume; NotFound when absent.
	Get(ctx context.Context, key types.NamespacedName) (*corev1.PersistentVolumeClaim, error)
	// List returns the managed volumes in a namespace, optionally filtered by a
	// K8s label selector.
	List(ctx context.Context, namespace, labelSelector string) ([]corev1.PersistentVolumeClaim, error)
	// Patch expands the volume and/or updates its description / labels. Size is
	// expand-only; storageClass and accessModes are immutable.
	Patch(ctx context.Context, key types.NamespacedName, patch VolumePatch) (*corev1.PersistentVolumeClaim, error)
	// Mounts reports the workloads currently mounting the volume, used for
	// delete-time occupancy checks and the detail view.
	Mounts(ctx context.Context, key types.NamespacedName) ([]VolumeMount, error)
	// Delete reclaims the backing volume.
	Delete(ctx context.Context, key types.NamespacedName) error
	// ListStorageClasses returns the storage classes available for new volumes
	// (empty in deployment forms without a StorageClass concept).
	ListStorageClasses(ctx context.Context) ([]StorageClass, error)
}

// StorageClass is a cluster-level storage backend a volume can be provisioned on.
type StorageClass struct {
	Name                 string
	Provisioner          string
	Default              bool
	AllowVolumeExpansion bool
}

// VolumePatch carries the mutable fields of a Patch. A nil pointer leaves the
// field unchanged; a non-nil Labels replaces the user-defined label set.
type VolumePatch struct {
	Size        *string
	Description *string
	Labels      map[string]string
}

// VolumeMount is one workload currently mounting a volume — the unit of
// occupancy reporting used for delete-protection and the detail view.
type VolumeMount struct {
	Workload  string // controlling workload (or pod) name
	Kind      string // Kubernetes controller kind (Deployment/StatefulSet/Job/Pod)
	MountPath string // first matching mount path inside the pod
	Running   bool   // whether the mounting pod is currently running
}
