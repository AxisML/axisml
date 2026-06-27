package mlrun

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/axisml/axisml/components/compute-service/internal/metrics"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	"github.com/axisml/axisml/components/compute-service/internal/store"
	"github.com/axisml/axisml/components/compute-service/pkg/extensions"
)

// Reconciler implements the job Outbox loop. Reads namespace directly off
// the row and drives the workload through the ComputeRuntime contract rather
// than a raw apiserver client, so the same loop serves any runtime form.
type Reconciler struct {
	db       *gorm.DB
	repo     *Repository
	runtime  extensions.ComputeRuntime
	log      logr.Logger
	interval time.Duration
}

func NewReconciler(
	db *gorm.DB,
	rt extensions.ComputeRuntime,
	log logr.Logger,
	interval time.Duration,
) *Reconciler {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Reconciler{
		db:       db,
		repo:     NewRepository(db),
		runtime:  rt,
		log:      log,
		interval: interval,
	}
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
	for i := range ws.Canceling {
		r.handleCancel(ctx, &ws.Canceling[i])
	}
	for i := range ws.Deleting {
		r.handleDelete(ctx, &ws.Deleting[i])
	}
}

func (r *Reconciler) handleCreate(ctx context.Context, j *store.MLRun) {
	cr, err := ToCR(j)
	if err != nil {
		r.log.Error(err, "render job CR")
		return
	}
	if err := r.runtime.ApplyMLRun(ctx, cr); err != nil {
		r.log.Error(err, "apply MLRun", "name", j.Name)
		// The error surfaces via status.message (a jsonb field); there is no
		// top-level `message` column.
		next := mergeStatusFields(j.StatusJSON, func(s *server.MLRunStatus) {
			s.Message = err.Error()
		})
		_ = r.repo.Update(ctx, j.ID, map[string]any{"status": next})
		metrics.ReconcilerActions.WithLabelValues("mlrun", "creating", "error").Inc()
		return
	}
	metrics.ReconcilerActions.WithLabelValues("mlrun", "creating", "success").Inc()
}

func (r *Reconciler) handleCancel(ctx context.Context, j *store.MLRun) {
	key := types.NamespacedName{Namespace: j.Namespace, Name: j.Name}
	if err := r.runtime.CancelMLRun(ctx, key); err != nil {
		r.log.Error(err, "cancel MLRun", "mlrun", j.Name)
		metrics.ReconcilerActions.WithLabelValues("mlrun", "canceling", "error").Inc()
		return
	}
	metrics.ReconcilerActions.WithLabelValues("mlrun", "canceling", "success").Inc()
}

func (r *Reconciler) handleDelete(ctx context.Context, j *store.MLRun) {
	key := types.NamespacedName{Namespace: j.Namespace, Name: j.Name}
	if err := r.runtime.DeleteMLRun(ctx, key); err != nil {
		r.log.Error(err, "delete MLRun", "name", j.Name)
		metrics.ReconcilerActions.WithLabelValues("mlrun", "deleting", "error").Inc()
		return
	}
	// Confirm the workload is gone so the row advances even when no informer
	// DELETE event arrives (e.g. the CR was already absent).
	if _, err := r.runtime.ObserveMLRun(ctx, key); apierrors.IsNotFound(err) {
		now := time.Now().UTC()
		_ = r.repo.Update(ctx, j.ID, map[string]any{
			"phase":      string(StatusDeleted),
			"deleted_at": now,
		})
		metrics.ReconcilerActions.WithLabelValues("mlrun", "deleting", "noop").Inc()
		return
	}
	metrics.ReconcilerActions.WithLabelValues("mlrun", "deleting", "success").Inc()
}
