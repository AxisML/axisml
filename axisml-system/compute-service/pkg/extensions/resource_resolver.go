package extensions

import (
	"context"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
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
