package extensions

import (
	"context"

	cmv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
)

// ResourceResolver reads the admin resource vocabulary (the ResourcePool CR and
// its embedded units) that compute-service expands into the workload spec at
// create time. It owns lookup only; merging a (pool, unit) pair into the spec
// snapshot per design §5.4 is the business layer's concern (internal/resource).
type ResourceResolver interface {
	// ResolveResourcePool returns the named cluster-scoped ResourcePool, or a
	// validation error if no such pool exists.
	ResolveResourcePool(ctx context.Context, name string) (*cmv1alpha1.ResourcePool, error)
	// ResolveResourceUnit returns the named unit embedded in the named pool, or
	// a validation error if the pool or the unit within it does not exist.
	ResolveResourceUnit(ctx context.Context, poolName, unitName string) (*cmv1alpha1.ResourceUnit, error)
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
