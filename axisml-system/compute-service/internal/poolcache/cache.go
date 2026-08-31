// Package poolcache wraps the controller-runtime Informer cache for the
// ResourcePool CRD (cluster-scoped). Compute consumes this cache at
// Job/Service create time to look up the ResourcePool and ResourceUnit it
// expands (internal/resource) into the nodeSelector / resources
// snapshot that lands in spec jsonb — there is no PG mirror of ResourcePool.
// See design §5.4.
package poolcache

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axismlv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// Reader exposes the read-only lookups needed by job/service. Backed by
// a controller-runtime client.Client whose cache the parent manager has
// already wired (so Get is served from the local Informer cache). It is the
// Kubernetes implementation of extensions.ResourceResolver.
type Reader struct {
	c client.Reader
}

var _ extensions.ResourceResolver = (*Reader)(nil)

// New returns a Reader that calls c.Get to satisfy lookups.
func New(c client.Reader) *Reader { return &Reader{c: c} }

// ResolveResourcePool returns the named cluster-scoped ResourcePool from the
// Informer cache, or a validation error if no such pool exists.
func (r *Reader) ResolveResourcePool(ctx context.Context, name string) (*axismlv1alpha1.ResourcePool, error) {
	if name == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "poolName is required")
	}
	var pool axismlv1alpha1.ResourcePool
	if err := r.c.Get(ctx, types.NamespacedName{Name: name}, &pool); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperrors.Newf(apperrors.CodeValidation,
				"resource pool %q not found", name)
		}
		return nil, apperrors.Wrap(apperrors.CodeUnavailable,
			fmt.Sprintf("read ResourcePool %q", name), err)
	}
	return &pool, nil
}

// ResolveResourceUnit returns the named unit embedded in the named pool, or a
// validation error if the pool or the unit within it does not exist.
func (r *Reader) ResolveResourceUnit(ctx context.Context, poolName, unitName string) (*axismlv1alpha1.ResourceUnit, error) {
	if unitName == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "unitName is required")
	}
	pool, err := r.ResolveResourcePool(ctx, poolName)
	if err != nil {
		return nil, err
	}
	for i := range pool.Spec.Units {
		if pool.Spec.Units[i].Name == unitName {
			return &pool.Spec.Units[i], nil
		}
	}
	return nil, apperrors.Newf(apperrors.CodeValidation,
		"resource unit %q not found in pool %q", unitName, poolName)
}
