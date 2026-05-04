package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlservicev1alpha1 "github.com/axisml/axisml/components/operators/mlservice-operator/api/v1alpha1"

	"github.com/axisml/axisml/components/compute/internal/metrics"
	"github.com/axisml/axisml/components/compute/internal/tenant"
)

// Reconciler implements the service Outbox loop.
type Reconciler struct {
	db        *gorm.DB
	repo      *Repository
	k8sClient client.Client
	tenants   *tenant.Service
	log       logr.Logger
	interval  time.Duration
}

// NewReconciler builds a service Reconciler.
func NewReconciler(db *gorm.DB, cl client.Client, tenants *tenant.Service, log logr.Logger, interval time.Duration) *Reconciler {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Reconciler{db: db, repo: NewRepository(db), k8sClient: cl, tenants: tenants, log: log, interval: interval}
}

// NeedLeaderElection ensures only the leader runs.
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

func (r *Reconciler) handleCreate(ctx context.Context, s *Service) {
	tnt, err := r.tenants.GetByID(ctx, s.TenantID)
	if err != nil {
		r.log.Error(err, "lookup tenant")
		return
	}
	cr, err := ToCR(s, tnt.Name, tnt.Namespace.Name)
	if err != nil {
		r.log.Error(err, "render service CR")
		return
	}
	if err := r.k8sClient.Create(ctx, cr); err != nil {
		if apierrors.IsAlreadyExists(err) {
			_ = r.repo.Update(ctx, s.ID, map[string]any{"applied_spec_hash": s.DesiredSpecHash})
			metrics.ReconcilerActions.WithLabelValues("service", "creating", "noop").Inc()
			return
		}
		r.log.Error(err, "create MLService")
		_ = r.repo.Update(ctx, s.ID, map[string]any{"message": err.Error()})
		metrics.ReconcilerActions.WithLabelValues("service", "creating", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, s.ID, map[string]any{"applied_spec_hash": s.DesiredSpecHash})
	metrics.ReconcilerActions.WithLabelValues("service", "creating", "success").Inc()
}

func (r *Reconciler) handleDelete(ctx context.Context, s *Service) {
	tnt, err := r.tenants.GetByID(ctx, s.TenantID)
	if err != nil {
		return
	}
	cr := &mlservicev1alpha1.MLService{ObjectMeta: metav1.ObjectMeta{Name: s.Name, Namespace: tnt.Namespace.Name}}
	err = r.k8sClient.Delete(ctx, cr)
	if err == nil {
		metrics.ReconcilerActions.WithLabelValues("service", "deleting", "success").Inc()
		return
	}
	if apierrors.IsNotFound(err) {
		_ = r.repo.Update(ctx, s.ID, map[string]any{"status": string(StatusDeleted)})
		metrics.ReconcilerActions.WithLabelValues("service", "deleting", "noop").Inc()
		return
	}
	r.log.Error(err, "delete MLService")
	metrics.ReconcilerActions.WithLabelValues("service", "deleting", "error").Inc()
}

func (r *Reconciler) handleSpecSync(ctx context.Context, s *Service) {
	tnt, err := r.tenants.GetByID(ctx, s.TenantID)
	if err != nil {
		return
	}
	current := &mlservicev1alpha1.MLService{}
	if err := r.k8sClient.Get(ctx, client.ObjectKey{Namespace: tnt.Namespace.Name, Name: s.Name}, current); err != nil {
		if apierrors.IsNotFound(err) {
			r.handleCreate(ctx, s)
			return
		}
		r.log.Error(err, "get MLService")
		return
	}
	var desiredSpec mlservicev1alpha1.MLServiceSpec
	if err := json.Unmarshal(s.Spec, &desiredSpec); err != nil {
		return
	}
	// We only mutate replicas (the only mutable field) per design §6.3.2.
	if len(current.Spec.Roles) == 0 || len(desiredSpec.Roles) == 0 {
		return
	}
	if current.Spec.Roles[0].Replicas == desiredSpec.Roles[0].Replicas {
		// CR already at desired replicas; idempotently catch up
		// applied_spec_hash and skip the API patch.
		_ = r.repo.Update(ctx, s.ID, map[string]any{"applied_spec_hash": s.DesiredSpecHash})
		metrics.ReconcilerActions.WithLabelValues("service", "spec_sync", "noop").Inc()
		return
	}
	current.Spec.Roles[0].Replicas = desiredSpec.Roles[0].Replicas
	if err := r.k8sClient.Update(ctx, current); err != nil {
		r.log.Error(err, "patch MLService replicas")
		metrics.ReconcilerActions.WithLabelValues("service", "spec_sync", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, s.ID, map[string]any{"applied_spec_hash": s.DesiredSpecHash})
	metrics.ReconcilerActions.WithLabelValues("service", "spec_sync", "success").Inc()
}

// Compile guard: keep uuid import alive.
var _ = uuid.Nil
