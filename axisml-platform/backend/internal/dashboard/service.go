// Package dashboard implements the Dashboard tag: landing-page aggregates for
// the active tenant. The recent-activity feed reads the durable Platform audit
// store; cluster-usage / cluster-metrics fold System-layer (cluster-manager)
// per-pool usage and time series.
package dashboard

import (
	"context"

	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

// Service holds dashboard aggregation logic.
type Service struct {
	audit *store.AuditRepo
}

// NewService constructs a dashboard Service.
func NewService(audit *store.AuditRepo) *Service {
	return &Service{audit: audit}
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
