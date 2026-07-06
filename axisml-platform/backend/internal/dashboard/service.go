// Package dashboard implements the Dashboard tag: landing-page aggregates for
// the active tenant. The recent-activity feed reads the durable Platform audit
// store; cluster-usage / cluster-metrics fold System-layer (cluster-manager)
// per-pool usage and time series.
package dashboard

import (
	"context"
	"time"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

// Service holds dashboard aggregation logic.
type Service struct {
	audit *store.AuditRepo
	cm    *clustermanager.Client
}

// NewService constructs a dashboard Service.
func NewService(audit *store.AuditRepo, cm *clustermanager.Client) *Service {
	return &Service{audit: audit, cm: cm}
}

// ClusterUsage folds the tenant's per-pool resource utilisation: one entry per
// pool the tenant has quota in (or just poolFilter when set), each from
// cluster-manager's per-(tenant, pool) usage (N2).
func (s *Service) ClusterUsage(ctx context.Context, tenant, poolFilter string) (server.ClusterUsage, error) {
	pools, err := s.tenantPools(ctx, tenant, poolFilter)
	if err != nil {
		return server.ClusterUsage{}, err
	}
	out := server.ClusterUsage{Pools: make([]server.ClusterPoolUsage, 0, len(pools)), UpdatedAt: time.Now().UTC()}
	for _, p := range pools {
		u, err := s.cm.ResourcePoolUsage(ctx, p, tenant)
		if err != nil {
			// Degrade gracefully: one unreadable pool must not blank the whole
			// landing page. Omit it and flag the snapshot partial.
			out.Partial = true
			continue
		}
		out.Pools = append(out.Pools, toClusterPoolUsage(u))
	}
	return out, nil
}

// ClusterMetrics proxies a (tenant, pool) resource-utilisation time series from
// cluster-manager (N3).
func (s *Service) ClusterMetrics(ctx context.Context, tenant, pool, metric, rng string, step *string) (server.MetricSeries, error) {
	m, err := s.cm.ResourcePoolMetrics(ctx, pool, tenant, metric, rng, step)
	if err != nil {
		return server.MetricSeries{}, err
	}
	return poolMetricToServer(m), nil
}

// tenantPools returns the pools the tenant has quota in, or [poolFilter] alone.
func (s *Service) tenantPools(ctx context.Context, tenant, poolFilter string) ([]string, error) {
	if poolFilter != "" {
		return []string{poolFilter}, nil
	}
	quotas, err := s.cm.ListQuotas(ctx, tenant)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(quotas))
	for _, q := range quotas {
		if !seen[q.Pool] {
			seen[q.Pool] = true
			out = append(out, q.Pool)
		}
	}
	return out, nil
}

func toClusterPoolUsage(u *clustermanager.PoolUsage) server.ClusterPoolUsage {
	meters := make([]server.ClusterMeter, 0, len(u.Meters))
	for _, m := range u.Meters {
		cm := server.ClusterMeter{Resource: m.Resource, Used: m.Used, Total: m.Total}
		if m.Unit != nil {
			cm.Unit = *m.Unit
		}
		meters = append(meters, cm)
	}
	return server.ClusterPoolUsage{Pool: u.Pool, Meters: meters}
}

func poolMetricToServer(m *clustermanager.PoolMetricSeries) server.MetricSeries {
	out := server.MetricSeries{Metric: m.Metric, Range: m.Range, Series: make([]server.MetricPoint, 0, len(m.Series))}
	if m.Step != nil {
		out.Step = *m.Step
	}
	if m.Unit != nil {
		out.Unit = *m.Unit
	}
	for _, p := range m.Series {
		out.Series = append(out.Series, server.MetricPoint{Timestamp: p.Timestamp, Value: p.Value})
	}
	return out
}

// Activity returns the tenant's recent-activity feed, newest first.
func (s *Service) Activity(ctx context.Context, tenant string, limit int) (server.ActivityList, error) {
	evs, err := s.audit.ListByTenant(ctx, tenant, limit)
	if err != nil {
		return server.ActivityList{}, apperrors.Wrap(apperrors.ClassInternal, "list activity", err)
	}
	items := make([]server.ActivityItem, 0, len(evs))
	for i := range evs {
		e := &evs[i]
		items = append(items, server.ActivityItem{
			ID:        e.ID,
			Kind:      e.Kind,
			Name:      e.Name,
			Action:    e.Action,
			Phase:     e.Phase,
			Actor:     e.Actor,
			Timestamp: e.CreatedAt,
		})
	}
	return server.ActivityList{Items: items, Count: len(items)}, nil
}
