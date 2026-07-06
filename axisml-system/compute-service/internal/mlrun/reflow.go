package mlrun

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlrun/v1alpha1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/statusmap"
)

// reflectObserved reflects an observed MLRun CR status onto the row's PG
// (phase, status) via the shared statusmap (design §9.1). It is the single
// writeback path shared by the Kubernetes informer (apiserver events) and the
// Lite status poller (runtime Observe). A no-op write is skipped.
func reflectObserved(ctx context.Context, repo *Repository, j *store.MLRun, observed mlrunv1alpha1.MLRunStatus, now time.Time) {
	var prevStatus server.MLRunStatus
	if len(j.StatusJSON) > 0 {
		_ = json.Unmarshal(j.StatusJSON, &prevStatus)
	}
	newPhase, mapped := statusmap.MapRun(j.Phase, statusmap.RunStatus{
		Message:    prevStatus.Message,
		StartedAt:  prevStatus.StartedAt,
		FinishedAt: prevStatus.FinishedAt,
	}, observed, now)

	nextStatus := prevStatus
	nextStatus.Message = mapped.Message
	nextStatus.StartedAt = mapped.StartedAt
	nextStatus.FinishedAt = mapped.FinishedAt

	b, _ := json.Marshal(nextStatus)
	prevB, _ := json.Marshal(prevStatus)
	if newPhase == j.Phase && string(b) == string(prevB) {
		return
	}
	_ = repo.Update(ctx, j.ID, map[string]any{
		"phase":  newPhase,
		"status": b,
	})
}

// reflectGone advances the row when the underlying workload no longer exists:
// a Deleting row converges to Deleted; an active row that vanished externally
// converges to Cancelled (design §5.4).
func reflectGone(ctx context.Context, repo *Repository, j *store.MLRun) {
	now := time.Now().UTC()
	switch Status(j.Phase) {
	case StatusDeleting:
		_ = repo.Update(ctx, j.ID, map[string]any{
			"phase":      string(StatusDeleted),
			"deleted_at": now,
		})
	case StatusPending, StatusRunning:
		next := mergeStatusFields(j.StatusJSON, func(s *server.MLRunStatus) {
			s.Message = "external delete"
			s.FinishedAt = &now
		})
		_ = repo.Update(ctx, j.ID, map[string]any{
			"phase":  string(StatusCancelled),
			"status": next,
		})
	case StatusCanceling:
		next := mergeStatusFields(j.StatusJSON, func(s *server.MLRunStatus) {
			s.FinishedAt = &now
		})
		_ = repo.Update(ctx, j.ID, map[string]any{
			"phase":  string(StatusCancelled),
			"status": next,
		})
	}
}

func mergeStatusFields(raw []byte, mutate func(*server.MLRunStatus)) []byte {
	var sf server.MLRunStatus
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &sf)
	}
	mutate(&sf)
	b, _ := json.Marshal(sf)
	return b
}

func jobIDFromLabels(cr *mlrunv1alpha1.MLRun) (uuid.UUID, error) {
	if v, ok := cr.Labels[mlrunv1alpha1.LabelRunID]; ok {
		return uuid.Parse(v)
	}
	return uuid.Nil, errMissingID
}

var errMissingID = sentinel("missing compute.axisml.io/run-id label")

type sentinel string

func (s sentinel) Error() string { return string(s) }
