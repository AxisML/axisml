package mlservice

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/axisml/axisml/components/compute-service/internal/metrics"
	"github.com/axisml/axisml/components/compute-service/internal/store"
	"github.com/axisml/axisml/components/compute-service/pkg/computeruntime"
)

// Reconciler implements the service Outbox loop. Namespace is read from
// the row directly; workloads are driven through the ComputeRuntime contract.
type Reconciler struct {
	db       *gorm.DB
	repo     *Repository
	runtime  computeruntime.ComputeRuntime
	log      logr.Logger
	interval time.Duration
}

func NewReconciler(db *gorm.DB, rt computeruntime.ComputeRuntime, log logr.Logger, interval time.Duration) *Reconciler {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Reconciler{db: db, repo: NewRepository(db), runtime: rt, log: log, interval: interval}
}

func (r *Reconciler) NeedLeaderElection() bool { return true }

func (r *Reconciler) Start(ctx context.Context) error {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Reconciler) runOnce(ctx context.Context) {
	ws, err := r.repo.FindWorkSet(ctx)
	if err != nil {
		r.log.Error(err, "find work set")
		return
	}
	for i := range ws.Creating {
		r.handleCreate(ctx, &ws.Creating[i])
	}
	for i := range ws.Deleting {
		r.handleDelete(ctx, &ws.Deleting[i])
	}
	for i := range ws.SpecDirty {
		r.handleSpecSync(ctx, &ws.SpecDirty[i])
	}
}

// handleCreate applies the desired MLService. ApplyMLService is idempotent:
// it creates the CR when absent and is a no-op when present, so observed_-
// generation advances on first success.
func (r *Reconciler) handleCreate(ctx context.Context, s *store.MLService) {
	cr, err := ToCR(s)
	if err != nil {
		r.log.Error(err, "render service CR")
		return
	}
	if err := r.runtime.ApplyMLService(ctx, cr); err != nil {
		r.log.Error(err, "apply MLService")
		_ = r.repo.Update(ctx, s.ID, map[string]any{"message": err.Error()})
		metrics.ReconcilerActions.WithLabelValues("mlservice", "creating", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, s.ID, map[string]any{"observed_generation": s.Generation})
	metrics.ReconcilerActions.WithLabelValues("mlservice", "creating", "success").Inc()
}

func (r *Reconciler) handleDelete(ctx context.Context, s *store.MLService) {
	key := types.NamespacedName{Namespace: s.Namespace, Name: s.Name}
	if err := r.runtime.DeleteMLService(ctx, key); err != nil {
		r.log.Error(err, "delete MLService")
		metrics.ReconcilerActions.WithLabelValues("mlservice", "deleting", "error").Inc()
		return
	}
	if _, err := r.runtime.ObserveMLService(ctx, key); apierrors.IsNotFound(err) {
		now := time.Now().UTC()
		_ = r.repo.Update(ctx, s.ID, map[string]any{
			"phase":      string(StatusDeleted),
			"deleted_at": now,
		})
		metrics.ReconcilerActions.WithLabelValues("mlservice", "deleting", "noop").Inc()
		return
	}
	metrics.ReconcilerActions.WithLabelValues("mlservice", "deleting", "success").Inc()
}

// handleSpecSync converges the CR onto the PG spec snapshot (only
// roles[0].replicas changes after create) and records observed_generation.
func (r *Reconciler) handleSpecSync(ctx context.Context, s *store.MLService) {
	cr, err := ToCR(s)
	if err != nil {
		r.log.Error(err, "render service CR")
		return
	}
	if err := r.runtime.ApplyMLService(ctx, cr); err != nil {
		r.log.Error(err, "apply MLService spec")
		metrics.ReconcilerActions.WithLabelValues("mlservice", "spec_sync", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, s.ID, map[string]any{"observed_generation": s.Generation})
	metrics.ReconcilerActions.WithLabelValues("mlservice", "spec_sync", "success").Inc()
}
