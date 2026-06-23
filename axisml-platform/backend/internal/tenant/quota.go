package tenant

import (
	"context"

	"github.com/axisml/axisml/components/platform/internal/clients/clustermanager"
	"github.com/axisml/axisml/components/platform/internal/server"
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

// SetQuota creates or replaces a pool quota and returns the updated tenant.
func (s *Service) SetQuota(ctx context.Context, identifier, pool string, units []QuotaUnitSpec) (*server.Tenant, error) {
	if _, err := s.getRow(ctx, identifier); err != nil {
		return nil, err
	}
	if err := s.cm.SetQuota(ctx, identifier, pool, toCMUnits(units)); err != nil {
		return nil, err
	}
	return s.Get(ctx, identifier)
}

// UpdateQuota replaces an existing pool quota's unit selection.
func (s *Service) UpdateQuota(ctx context.Context, identifier, pool string, units []QuotaUnitSpec) (*server.Tenant, error) {
	if _, err := s.getRow(ctx, identifier); err != nil {
		return nil, err
	}
	if err := s.cm.UpdateQuota(ctx, identifier, pool, toCMUnits(units)); err != nil {
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
