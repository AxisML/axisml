package tenant

import (
	"context"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// ListQuotas returns a tenant's per-pool quotas plus live statuses.
func (s *Service) ListQuotas(ctx context.Context, identifier string) (*server.QuotaList, error) {
	if _, err := s.getRow(ctx, identifier); err != nil {
		return nil, err
	}
	quotas, err := s.cm.ListQuotas(ctx, identifier)
	if err != nil {
		return nil, err
	}
	items := mapQuotas(quotas)
	list := &server.QuotaList{Items: items, Count: len(items)}
	if cr, err := s.cm.GetTenant(ctx, identifier); err == nil && cr.Status != nil {
		list.Statuses = mapQuotaStatuses(cr.Status.Quotas)
	}
	return list, nil
}

// SetQuota creates or replaces a pool quota (units or direct min/max) and
// returns the updated tenant.
func (s *Service) SetQuota(ctx context.Context, identifier string, q QuotaSpec) (*server.Tenant, error) {
	if _, err := s.getRow(ctx, identifier); err != nil {
		return nil, err
	}
	units, direct := toCMQuotaInput(q)
	if err := s.cm.SetQuota(ctx, identifier, q.Pool, units, direct); err != nil {
		return nil, err
	}
	return s.Get(ctx, identifier)
}

// UpdateQuota replaces an existing pool quota's input (units or direct min/max).
func (s *Service) UpdateQuota(ctx context.Context, identifier string, q QuotaSpec) (*server.Tenant, error) {
	if _, err := s.getRow(ctx, identifier); err != nil {
		return nil, err
	}
	units, direct := toCMQuotaInput(q)
	if err := s.cm.UpdateQuota(ctx, identifier, q.Pool, units, direct); err != nil {
		return nil, err
	}
	return s.Get(ctx, identifier)
}

// DeleteQuota removes a pool quota.
func (s *Service) DeleteQuota(ctx context.Context, identifier, pool string) error {
	if _, err := s.getRow(ctx, identifier); err != nil {
		return err
	}
	return s.cm.DeleteQuota(ctx, identifier, pool)
}

// toCMQuotaInput splits a QuotaSpec into the cluster-manager client's units or
// direct min/max arguments; direct wins when present.
func toCMQuotaInput(q QuotaSpec) ([]clustermanager.QuotaUnit, *clustermanager.QuotaResources) {
	if q.Direct != nil {
		return nil, toCMResources(q.Direct)
	}
	return toCMUnits(q.Units), nil
}

func toCMUnits(units []QuotaUnitSpec) []clustermanager.QuotaUnit {
	out := make([]clustermanager.QuotaUnit, 0, len(units))
	for _, u := range units {
		if u.Quantity < 0 {
			continue
		}
		out = append(out, clustermanager.QuotaUnit{UnitName: u.UnitName, Quantity: u.Quantity})
	}
	return out
}

func toCMResources(r *QuotaResourcesSpec) *clustermanager.QuotaResources {
	if r == nil {
		return nil
	}
	out := &clustermanager.QuotaResources{Max: r.Max}
	if len(r.Min) > 0 {
		min := r.Min
		out.Min = &min
	}
	return out
}
