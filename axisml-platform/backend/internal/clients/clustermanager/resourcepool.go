package clustermanager

import (
	"context"
	"encoding/json"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clienterr"
	gen "github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager/generated"
)

// Clean-named aliases for the generated ResourcePool / ResourceUnit wire types.
type (
	Pool       = gen.ResourcePool
	Unit       = gen.ServerResourceUnit
	PoolCreate = gen.CreateResourcePoolRequest
	PoolPatch  = gen.PatchResourcePoolRequest
	UnitCreate = gen.CreateResourceUnitRequest
	UnitPatch  = gen.PatchResourceUnitRequest
	UnitInline = gen.ServerCreateResourceUnitRequest
	Toleration = gen.Corev1Toleration

	PoolUsage        = gen.PoolUsage
	PoolMeter        = gen.ServerResourceMeter
	PoolMetricSeries = gen.PoolMetricSeries
	PoolMetricPoint  = gen.ServerPoolMetricPoint
)

// ResourcePoolUsage returns a tenant's used-vs-total resource utilisation in a
// pool (N2).
func (c *Client) ResourcePoolUsage(ctx context.Context, pool, tenant string) (*PoolUsage, error) {
	res, err := c.gen.GetResourcePoolUsageWithResponse(ctx, pool, &gen.GetResourcePoolUsageParams{Tenant: tenant})
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ResourcePoolMetrics returns a (tenant, pool) resource-utilisation time series
// (N3).
func (c *Client) ResourcePoolMetrics(ctx context.Context, pool, tenant, metric, rng string, step *string) (*PoolMetricSeries, error) {
	res, err := c.gen.GetResourcePoolMetricsWithResponse(ctx, pool, &gen.GetResourcePoolMetricsParams{Tenant: tenant, Metric: metric, Range: rng, Step: step})
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ListResourcePools returns all pools (each with embedded units).
func (c *Client) ListResourcePools(ctx context.Context, labelSelector string) ([]Pool, error) {
	params := &gen.ListResourcePoolsParams{}
	if labelSelector != "" {
		params.LabelSelector = &labelSelector
	}
	res, err := c.gen.ListResourcePoolsWithResponse(ctx, params)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	out := make([]Pool, 0, len(res.JSON200.Items))
	for i := range res.JSON200.Items {
		var p Pool
		b, _ := json.Marshal(res.JSON200.Items[i])
		if err := json.Unmarshal(b, &p); err != nil {
			return nil, clienterr.Transport(service, err)
		}
		out = append(out, p)
	}
	return out, nil
}

// GetResourcePool returns one pool.
func (c *Client) GetResourcePool(ctx context.Context, name string) (*Pool, error) {
	res, err := c.gen.GetResourcePoolWithResponse(ctx, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// CreateResourcePool creates a pool (with optional inline units).
func (c *Client) CreateResourcePool(ctx context.Context, body PoolCreate) (*Pool, error) {
	res, err := c.gen.CreateResourcePoolWithResponse(ctx, body)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON201 != nil {
		return res.JSON201, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// UpdateResourcePool patches pool-level fields.
func (c *Client) UpdateResourcePool(ctx context.Context, name string, body PoolPatch) (*Pool, error) {
	res, err := c.gen.UpdateResourcePoolWithResponse(ctx, name, body)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// DeleteResourcePool deletes a pool (idempotent).
func (c *Client) DeleteResourcePool(ctx context.Context, name string) error {
	res, err := c.gen.DeleteResourcePoolWithResponse(ctx, name)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if res.HTTPResponse != nil && res.HTTPResponse.StatusCode < 300 {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// GetResourceUnit returns one unit of a pool.
func (c *Client) GetResourceUnit(ctx context.Context, pool, unit string) (*Unit, error) {
	res, err := c.gen.GetResourceUnitWithResponse(ctx, pool, unit)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return asUnit(res.JSON200)
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// CreateResourceUnit adds a unit to a pool.
func (c *Client) CreateResourceUnit(ctx context.Context, pool string, body UnitCreate) (*Unit, error) {
	res, err := c.gen.CreateResourceUnitWithResponse(ctx, pool, body)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON201 != nil {
		return asUnit(res.JSON201)
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// UpdateResourceUnit patches a unit.
func (c *Client) UpdateResourceUnit(ctx context.Context, pool, unit string, body UnitPatch) (*Unit, error) {
	res, err := c.gen.UpdateResourceUnitWithResponse(ctx, pool, unit, body)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return asUnit(res.JSON200)
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// asUnit normalises the single-op ResourceUnit to the Unit (ServerResourceUnit)
// shape used for embedded pool units (the two generated types are identical).
func asUnit(in *gen.ResourceUnit) (*Unit, error) {
	var u Unit
	b, _ := json.Marshal(in)
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, clienterr.Transport(service, err)
	}
	return &u, nil
}

// DeleteResourceUnit removes a unit from a pool (idempotent).
func (c *Client) DeleteResourceUnit(ctx context.Context, pool, unit string) error {
	res, err := c.gen.DeleteResourceUnitWithResponse(ctx, pool, unit)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if res.HTTPResponse != nil && res.HTTPResponse.StatusCode < 300 {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}
