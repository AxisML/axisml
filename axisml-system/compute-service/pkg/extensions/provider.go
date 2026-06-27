package extensions

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// Expanded is the resource snapshot a ResourceResolver returns for a
// (pool, unit) pair, per design §5.4 merge rules.
type Expanded struct {
	NodeSelector map[string]string
	Tolerations  []corev1.Toleration
	Requests     corev1.ResourceList
	Limits       corev1.ResourceList
}

// ResourceResolver resolves (poolName, unitName) into the expanded snapshot that
// is frozen into the workload spec at create time.
type ResourceResolver interface {
	Resolve(ctx context.Context, poolName, unitName string) (*Expanded, error)
}

// WorkspaceVolumeProvisioner manages the durable volume that backs a
// kind=workspace MLService. Implementations must be idempotent.
type WorkspaceVolumeProvisioner interface {
	// EnsureWorkspaceVolume provisions (or confirms) the volume backing the
	// named workspace. size is a Kubernetes Quantity string; storageClass may
	// be empty for the cluster default.
	EnsureWorkspaceVolume(ctx context.Context, namespace, name, size, storageClass string) error
	// DeleteWorkspaceVolume removes the workspace's backing volume. A missing
	// volume is not an error.
	DeleteWorkspaceVolume(ctx context.Context, namespace, name string) error
}
