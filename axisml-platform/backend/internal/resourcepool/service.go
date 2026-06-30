// Package resourcepool implements the ResourcePools and ResourceUnits tags:
// cluster-scoped pool/unit CRUD proxied to cluster-manager (backend.md §4.6).
// Pools/units are global admin objects, not tenant-partitioned.
package resourcepool

import (
	"context"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// Service holds resource-pool business logic.
type Service struct {
	cm *clustermanager.Client
}

// NewService constructs a Service.
func NewService(cm *clustermanager.Client) *Service { return &Service{cm: cm} }

// ListPools returns all pools with embedded units.
func (s *Service) ListPools(ctx context.Context, q string) ([]server.ResourcePool, error) {
	pools, err := s.cm.ListResourcePools(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]server.ResourcePool, 0, len(pools))
	for i := range pools {
		v := toPoolView(&pools[i])
		if q != "" && !matchPool(v, q) {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

// GetPool returns one pool.
func (s *Service) GetPool(ctx context.Context, name string) (*server.ResourcePool, error) {
	p, err := s.cm.GetResourcePool(ctx, name)
	if err != nil {
		return nil, err
	}
	v := toPoolView(p)
	return &v, nil
}

// CreatePool creates a pool (with optional inline units).
func (s *Service) CreatePool(ctx context.Context, req server.ResourcePoolCreateRequest) (*server.ResourcePool, error) {
	p, err := s.cm.CreateResourcePool(ctx, poolCreateBody(req))
	if err != nil {
		return nil, err
	}
	v := toPoolView(p)
	return &v, nil
}

// UpdatePool patches pool-level fields (name immutable).
func (s *Service) UpdatePool(ctx context.Context, name string, req server.ResourcePoolPatchRequest) (*server.ResourcePool, error) {
	p, err := s.cm.UpdateResourcePool(ctx, name, poolPatchBody(req))
	if err != nil {
		return nil, err
	}
	v := toPoolView(p)
	return &v, nil
}

// DeletePool deletes a pool.
//
// TODO(workloads): enforce the §4.6 in-use precheck (enumerate tenant scopes and
// reject when an active MLRun/MLService carries resource.axisml.io/pool=<name>)
// once the compute client wrapper exposes labelSelector workload listing.
func (s *Service) DeletePool(ctx context.Context, name string) error {
	return s.cm.DeleteResourcePool(ctx, name)
}

// ListUnits returns a pool's units.
func (s *Service) ListUnits(ctx context.Context, pool string) ([]server.ResourceUnit, error) {
	p, err := s.cm.GetResourcePool(ctx, pool)
	if err != nil {
		return nil, err
	}
	out := make([]server.ResourceUnit, 0, len(p.Units))
	for i := range p.Units {
		out = append(out, toUnitView(&p.Units[i]))
	}
	return out, nil
}

// GetUnit returns one unit.
func (s *Service) GetUnit(ctx context.Context, pool, unit string) (*server.ResourceUnit, error) {
	u, err := s.cm.GetResourceUnit(ctx, pool, unit)
	if err != nil {
		return nil, err
	}
	v := toUnitView(u)
	return &v, nil
}

// CreateUnit adds a unit to a pool.
func (s *Service) CreateUnit(ctx context.Context, pool string, req server.ResourceUnitCreateRequest) (*server.ResourceUnit, error) {
	u, err := s.cm.CreateResourceUnit(ctx, pool, unitCreateBody(req))
	if err != nil {
		return nil, err
	}
	v := toUnitView(u)
	return &v, nil
}

// UpdateUnit patches a unit (name immutable).
func (s *Service) UpdateUnit(ctx context.Context, pool, unit string, req server.ResourceUnitPatchRequest) (*server.ResourceUnit, error) {
	u, err := s.cm.UpdateResourceUnit(ctx, pool, unit, unitPatchBody(req))
	if err != nil {
		return nil, err
	}
	v := toUnitView(u)
	return &v, nil
}

// DeleteUnit removes a unit from a pool.
func (s *Service) DeleteUnit(ctx context.Context, pool, unit string) error {
	return s.cm.DeleteResourceUnit(ctx, pool, unit)
}
