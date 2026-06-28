package mlservice

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/server"
	"github.com/axisml/axisml/components/compute-service/internal/store"
	"github.com/axisml/axisml/components/compute-service/pkg/statusmap"
)

// reflectObserved reflects an observed MLService CR status onto the row's PG
// (phase, status) via the shared statusmap (design §9.1), preserving PG-only
// fields (conditions). Shared by the Kubernetes informer and the Lite poller.
// desiredReplicas is the spec's role[0] replica count.
func reflectObserved(ctx context.Context, repo *Repository, row *store.MLService, desiredReplicas int32, observed mlservicev1alpha1.MLServiceStatus) {
	// Don't override Deleting/Deleted.
	if Status(row.Phase) == StatusDeleting || Status(row.Phase) == StatusDeleted {
		return
	}

	var sf server.MLServiceStatus
	if len(row.StatusJSON) > 0 {
		_ = json.Unmarshal(row.StatusJSON, &sf)
	}
	newPhase, mapped := statusmap.MapService(row.Phase, statusmap.ServiceStatus{
		Message:       sf.Message,
		ReadyReplicas: sf.ReadyReplicas,
		Endpoint:      sf.Endpoint,
	}, desiredReplicas, observed)
	sf.Message = mapped.Message
	sf.ReadyReplicas = mapped.ReadyReplicas
	sf.Endpoint = mapped.Endpoint
	b, _ := json.Marshal(sf)

	_ = repo.Update(ctx, row.ID, map[string]any{
		"phase":  newPhase,
		"status": b,
	})
}

// reflectGone advances a row whose underlying workload no longer exists: a
// Deleting row converges to Deleted; an active row that vanished externally is
// pushed to Deleting (design §5.4).
func reflectGone(ctx context.Context, repo *Repository, row *store.MLService) {
	switch Status(row.Phase) {
	case StatusDeleting:
		now := time.Now().UTC()
		_ = repo.Update(ctx, row.ID, map[string]any{
			"phase":      string(StatusDeleted),
			"deleted_at": now,
		})
	case StatusPending, StatusReady, StatusDegraded, StatusFailed:
		var sf server.MLServiceStatus
		if len(row.StatusJSON) > 0 {
			_ = json.Unmarshal(row.StatusJSON, &sf)
		}
		sf.Message = "external delete"
		b, _ := json.Marshal(sf)
		// Push to Deleting only — do NOT stamp deleted_at here. The row is still
		// mid-teardown and must remain visible to read APIs (which filter
		// deleted_at IS NULL); deleted_at is set only on the Deleting→Deleted
		// convergence above, matching the mlrun reflow.
		_ = repo.Update(ctx, row.ID, map[string]any{
			"phase":  string(StatusDeleting),
			"status": b,
		})
	}
}

// desiredReplicas reads role[0].replicas off the row's stored spec.
func desiredReplicas(row *store.MLService) int32 {
	cr, err := ToCR(row)
	if err != nil || len(cr.Spec.Roles) == 0 {
		return 0
	}
	return cr.Spec.Roles[0].Replicas
}

func serviceIDFromLabels(cr *mlservicev1alpha1.MLService) (uuid.UUID, error) {
	if v, ok := cr.Labels[mlservicev1alpha1.LabelServiceID]; ok {
		return uuid.Parse(v)
	}
	return uuid.Nil, errMissingID
}

var errMissingID = sentinel("missing axisml.io/service-id label")

type sentinel string

func (s sentinel) Error() string { return string(s) }
