package mlrun

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/metrics"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

const dispatchTimeout = 2 * time.Minute

// Reconciler implements the job Outbox loop. Reads namespace directly off
// the row and drives the workload through the ComputeRuntime contract rather
// than a raw apiserver client, so the same loop serves any runtime form.
type Reconciler struct {
	db           *gorm.DB
	repo         *Repository
	runtime      extensions.ComputeRuntime
	log          logr.Logger
	interval     time.Duration
	tenantPrefix bool
}

func NewReconciler(
	db *gorm.DB,
	rt extensions.ComputeRuntime,
	log logr.Logger,
	interval time.Duration,
	tenantPrefix bool,
) *Reconciler {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Reconciler{
		db:           db,
		repo:         NewRepository(db),
		runtime:      rt,
		log:          log,
		interval:     interval,
		tenantPrefix: tenantPrefix,
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
	recovery, err := r.repo.FindDispatchRecoverySet(ctx, time.Now().UTC().Add(-dispatchTimeout))
	if err != nil {
		r.log.Error(err, "find MLRun dispatch recovery set")
		return
	}
	for i := range recovery {
		r.handleDispatchRecovery(ctx, &recovery[i])
	}

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

func (r *Reconciler) handleDispatchRecovery(ctx context.Context, j *store.MLRun) {
	key := types.NamespacedName{Namespace: j.Namespace, Name: j.Name}
	observed, err := r.runtime.ObserveMLRun(ctx, key)
	if apierrors.IsNotFound(err) {
		next := mergeStatusFields(j.StatusJSON, func(s *server.MLRunStatus) {
			s.QueueReason = ""
			s.Message = "runtime object absent after dispatch timeout; waiting for readmission"
		})
		changed, updateErr := r.repo.UpdatePhase(ctx, j.ID, Status(j.Phase), map[string]any{
			"phase":        string(StatusQueued),
			"status":       next,
			"scheduled_at": nil,
		})
		if updateErr != nil {
			r.log.Error(updateErr, "requeue stale MLRun dispatch", "name", j.Name)
			return
		}
		if changed {
			metrics.ReconcilerActions.WithLabelValues("mlrun", "dispatch-recovery", "requeued").Inc()
		}
		return
	}
	if err != nil {
		r.log.Error(err, "observe stale MLRun dispatch", "name", j.Name)
		metrics.ReconcilerActions.WithLabelValues("mlrun", "dispatch-recovery", "error").Inc()
		return
	}

	if Status(j.Phase) == StatusCreating {
		now := time.Now().UTC()
		changed, updateErr := r.repo.UpdatePhase(ctx, j.ID, StatusCreating, map[string]any{
			"phase":        string(StatusPending),
			"scheduled_at": now,
		})
		if updateErr != nil {
			r.log.Error(updateErr, "repair stale MLRun dispatch", "name", j.Name)
			return
		}
		if changed {
			j.Phase = string(StatusPending)
			j.ScheduledAt = &now
			reflectObserved(ctx, r.repo, j, observed, now)
			metrics.ReconcilerActions.WithLabelValues("mlrun", "dispatch-recovery", "repaired").Inc()
		}
		return
	}
	now := time.Now().UTC()
	updates := map[string]any{"updated_at": now}
	if j.ScheduledAt == nil {
		updates["scheduled_at"] = now
	}
	changed, updateErr := r.repo.UpdatePhase(ctx, j.ID, StatusPending, updates)
	if updateErr != nil {
		r.log.Error(updateErr, "repair stale MLRun scheduled state", "name", j.Name)
		return
	}
	if changed && j.ScheduledAt == nil {
		j.ScheduledAt = &now
	}
	reflectObserved(ctx, r.repo, j, observed, now)
}

func (r *Reconciler) handleCreate(ctx context.Context, j *store.MLRun) {
	cr, err := ToCR(j, r.tenantPrefix)
	if err != nil {
		r.log.Error(err, "render job CR")
		return
	}
	if err := r.runtime.ApplyMLRun(ctx, cr); err != nil {
		// The message surfaces via status.message (a jsonb field); there is no
		// top-level `message` column.
		terminal := extensions.IsTerminalApplyError(err)
		now := time.Now().UTC()
		next := mergeStatusFields(j.StatusJSON, func(s *server.MLRunStatus) {
			s.Message = err.Error()
			if terminal {
				s.FinishedAt = &now
			}
		})
		updates := map[string]any{"status": next}
		if terminal {
			updates["phase"] = string(StatusFailed)
		}
		_ = r.repo.Update(ctx, j.ID, updates)
		if terminal {
			r.log.Error(err, "apply MLRun terminal failure", "name", j.Name)
			metrics.ReconcilerActions.WithLabelValues("mlrun", "creating", "failed").Inc()
			return
		}
		if extensions.IsResourceUnavailable(err) {
			// The runtime remains the final atomic guard (notably for Docker GPU
			// assignment). Requeue only if Apply did not create any instance; a
			// partial apply stays Creating and converges idempotently.
			instances, listErr := r.runtime.ListMLRunInstances(ctx, types.NamespacedName{Namespace: j.Namespace, Name: j.Name})
			if listErr == nil && len(instances.Items) == 0 {
				next = mergeStatusFields(j.StatusJSON, func(s *server.MLRunStatus) {
					s.QueueReason = "InsufficientResources"
					s.Message = err.Error()
				})
				_ = r.repo.Update(ctx, j.ID, map[string]any{
					"phase":        string(StatusQueued),
					"status":       next,
					"scheduled_at": nil,
				})
				metrics.ReconcilerActions.WithLabelValues("mlrun", "creating", "requeued").Inc()
				return
			}
			metrics.ReconcilerActions.WithLabelValues("mlrun", "creating", "pending").Inc()
			return
		}
		r.log.Error(err, "apply MLRun", "name", j.Name)
		metrics.ReconcilerActions.WithLabelValues("mlrun", "creating", "error").Inc()
		return
	}
	if err := r.repo.Update(ctx, j.ID, map[string]any{
		"phase":        string(StatusPending),
		"scheduled_at": time.Now().UTC(),
	}); err != nil {
		r.log.Error(err, "mark MLRun pending after apply", "name", j.Name)
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
