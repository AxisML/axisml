package trafficpolicy

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/axisml/axisml/components/compute-service/internal/metrics"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	"github.com/axisml/axisml/components/compute-service/internal/store"
	"github.com/axisml/axisml/components/compute-service/pkg/computeruntime"
)

// Reconciler implements the traffic-policy Outbox loop (leader-only). It scans
// the predicate work set and drives the MLTrafficPolicy workload through the
// ComputeRuntime contract to match PG (compute-service.md §5.1).
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

func (r *Reconciler) handleCreate(ctx context.Context, p *store.TrafficPolicy) {
	cr, err := ToCR(p)
	if err != nil {
		r.log.Error(err, "render traffic-policy CR")
		return
	}
	if err := r.runtime.ApplyMLTrafficPolicy(ctx, cr); err != nil {
		r.log.Error(err, "apply MLTrafficPolicy")
		// The error surfaces via status.message (a jsonb field); there is no
		// top-level `message` column.
		var sf server.TrafficPolicyStatus
		if len(p.StatusJSON) > 0 {
			_ = json.Unmarshal(p.StatusJSON, &sf)
		}
		sf.Message = err.Error()
		b, _ := json.Marshal(sf)
		_ = r.repo.Update(ctx, p.ID, map[string]any{"status": b})
		metrics.ReconcilerActions.WithLabelValues("traffic_policy", "creating", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, p.ID, map[string]any{"observed_generation": p.Generation})
	metrics.ReconcilerActions.WithLabelValues("traffic_policy", "creating", "success").Inc()
}

func (r *Reconciler) handleDelete(ctx context.Context, p *store.TrafficPolicy) {
	key := types.NamespacedName{Namespace: p.Namespace, Name: p.Name}
	if err := r.runtime.DeleteMLTrafficPolicy(ctx, key); err != nil {
		r.log.Error(err, "delete MLTrafficPolicy")
		metrics.ReconcilerActions.WithLabelValues("traffic_policy", "deleting", "error").Inc()
		return
	}
	if _, err := r.runtime.ObserveMLTrafficPolicy(ctx, key); apierrors.IsNotFound(err) {
		now := time.Now().UTC()
		_ = r.repo.Update(ctx, p.ID, map[string]any{
			"phase":      string(StatusDeleted),
			"deleted_at": now,
		})
		metrics.ReconcilerActions.WithLabelValues("traffic_policy", "deleting", "noop").Inc()
		return
	}
	metrics.ReconcilerActions.WithLabelValues("traffic_policy", "deleting", "success").Inc()
}

// handleSpecSync converges the CR onto the PG spec snapshot. Only
// backends (weight + role) change after create; mode / endpoint / backend
// are immutable.
func (r *Reconciler) handleSpecSync(ctx context.Context, p *store.TrafficPolicy) {
	cr, err := ToCR(p)
	if err != nil {
		r.log.Error(err, "render traffic-policy CR")
		return
	}
	if err := r.runtime.ApplyMLTrafficPolicy(ctx, cr); err != nil {
		r.log.Error(err, "apply MLTrafficPolicy spec")
		metrics.ReconcilerActions.WithLabelValues("traffic_policy", "spec_sync", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, p.ID, map[string]any{"observed_generation": p.Generation})
	metrics.ReconcilerActions.WithLabelValues("traffic_policy", "spec_sync", "success").Inc()
}
