package trafficpolicy

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/metrics"
	"github.com/axisml/axisml/components/compute-service/internal/store"
)

// Reconciler implements the traffic-policy Outbox loop (leader-only). It scans
// the predicate work set and Create/Patch/Delete the MLTrafficPolicy CR to
// match PG (compute-service.md §5.1).
type Reconciler struct {
	db        *gorm.DB
	repo      *Repository
	k8sClient client.Client
	log       logr.Logger
	interval  time.Duration
}

func NewReconciler(db *gorm.DB, cl client.Client, log logr.Logger, interval time.Duration) *Reconciler {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Reconciler{db: db, repo: NewRepository(db), k8sClient: cl, log: log, interval: interval}
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
	if err := r.k8sClient.Create(ctx, cr); err != nil {
		if apierrors.IsAlreadyExists(err) {
			_ = r.repo.Update(ctx, p.ID, map[string]any{"observed_generation": p.Generation})
			metrics.ReconcilerActions.WithLabelValues("traffic_policy", "creating", "noop").Inc()
			return
		}
		r.log.Error(err, "create MLTrafficPolicy")
		_ = r.repo.Update(ctx, p.ID, map[string]any{"message": err.Error()})
		metrics.ReconcilerActions.WithLabelValues("traffic_policy", "creating", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, p.ID, map[string]any{"observed_generation": p.Generation})
	metrics.ReconcilerActions.WithLabelValues("traffic_policy", "creating", "success").Inc()
}

func (r *Reconciler) handleDelete(ctx context.Context, p *store.TrafficPolicy) {
	cr := &mltp.MLTrafficPolicy{ObjectMeta: metav1.ObjectMeta{Name: p.Name, Namespace: p.Namespace}}
	err := r.k8sClient.Delete(ctx, cr)
	if err == nil {
		metrics.ReconcilerActions.WithLabelValues("traffic_policy", "deleting", "success").Inc()
		return
	}
	if apierrors.IsNotFound(err) {
		_ = r.repo.Update(ctx, p.ID, map[string]any{"status": string(StatusDeleted)})
		metrics.ReconcilerActions.WithLabelValues("traffic_policy", "deleting", "noop").Inc()
		return
	}
	r.log.Error(err, "delete MLTrafficPolicy")
	metrics.ReconcilerActions.WithLabelValues("traffic_policy", "deleting", "error").Inc()
}

func (r *Reconciler) handleSpecSync(ctx context.Context, p *store.TrafficPolicy) {
	current := &mltp.MLTrafficPolicy{}
	if err := r.k8sClient.Get(ctx, client.ObjectKey{Namespace: p.Namespace, Name: p.Name}, current); err != nil {
		if apierrors.IsNotFound(err) {
			r.handleCreate(ctx, p)
			return
		}
		r.log.Error(err, "get MLTrafficPolicy")
		return
	}
	var desiredSpec mltp.MLTrafficPolicySpec
	if err := json.Unmarshal(p.Spec, &desiredSpec); err != nil {
		return
	}
	// Only backends (weight + role) change after create; mode / endpoint /
	// backend are immutable. Patch the whole backends slice.
	current.Spec.Backends = desiredSpec.Backends
	if err := r.k8sClient.Update(ctx, current); err != nil {
		r.log.Error(err, "patch MLTrafficPolicy backends")
		metrics.ReconcilerActions.WithLabelValues("traffic_policy", "spec_sync", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, p.ID, map[string]any{"observed_generation": p.Generation})
	metrics.ReconcilerActions.WithLabelValues("traffic_policy", "spec_sync", "success").Inc()
}
