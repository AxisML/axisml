package tenant

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"

	"github.com/axisml/axisml/components/compute/internal/metrics"
	"github.com/axisml/axisml/components/compute/internal/quota"
	"github.com/axisml/axisml/components/compute/internal/resourcepool"
)

// Reconciler is the leader-only PG-polling loop that syncs tenants → CRs.
type Reconciler struct {
	db        *gorm.DB
	repo      *Repository
	k8sClient client.Client
	pools     *resourcepool.Service
	quotas    *quota.Service
	log       logr.Logger
	interval  time.Duration
}

// NewReconciler builds a Reconciler with shared module deps.
func NewReconciler(
	db *gorm.DB,
	cl client.Client,
	pools *resourcepool.Service,
	log logr.Logger,
	interval time.Duration,
) *Reconciler {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Reconciler{
		db:        db,
		repo:      NewRepository(db),
		k8sClient: cl,
		pools:     pools,
		log:       log,
		interval:  interval,
	}
}

// SetQuotas allows late wiring (avoids constructor circular dep).
func (r *Reconciler) SetQuotas(q *quota.Service) { r.quotas = q }

// NeedLeaderElection ensures only the leader runs reconcile loops.
func (r *Reconciler) NeedLeaderElection() bool { return true }

// Start blocks until ctx is cancelled.
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

func (r *Reconciler) handleCreate(ctx context.Context, t *Tenant) {
	cr := r.toCR(ctx, t)
	if cr == nil {
		return
	}
	if err := r.k8sClient.Create(ctx, cr); err != nil {
		if apierrors.IsAlreadyExists(err) {
			r.markApplied(ctx, t.ID, t.DesiredSpecHash)
			metrics.ReconcilerActions.WithLabelValues("tenant", "creating", "noop").Inc()
			return
		}
		r.log.Error(err, "create tenant CR", "name", t.Name)
		metrics.ReconcilerActions.WithLabelValues("tenant", "creating", "error").Inc()
		_ = r.repo.Update(ctx, nil, t.ID, map[string]any{"message": err.Error()})
		return
	}
	r.markApplied(ctx, t.ID, t.DesiredSpecHash)
	metrics.ReconcilerActions.WithLabelValues("tenant", "creating", "success").Inc()
}

func (r *Reconciler) handleDelete(ctx context.Context, t *Tenant) {
	cr := &tenantv1alpha1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: t.Name}}
	if err := r.k8sClient.Delete(ctx, cr); err != nil && !apierrors.IsNotFound(err) {
		r.log.Error(err, "delete tenant CR", "name", t.Name)
		metrics.ReconcilerActions.WithLabelValues("tenant", "deleting", "error").Inc()
		return
	}
	metrics.ReconcilerActions.WithLabelValues("tenant", "deleting", "success").Inc()
}

func (r *Reconciler) handleSpecSync(ctx context.Context, t *Tenant) {
	current := &tenantv1alpha1.Tenant{}
	if err := r.k8sClient.Get(ctx, client.ObjectKey{Name: t.Name}, current); err != nil {
		if apierrors.IsNotFound(err) {
			// Treat missing CR as "needs (re)create".
			r.handleCreate(ctx, t)
			return
		}
		r.log.Error(err, "get tenant CR", "name", t.Name)
		return
	}
	desired := r.toCR(ctx, t)
	if desired == nil {
		return
	}
	if reflect.DeepEqual(current.Spec, desired.Spec) {
		// CR already matches desired (e.g. previous Update succeeded but
		// markApplied failed). Idempotently catch up applied_spec_hash and
		// skip the API patch.
		r.markApplied(ctx, t.ID, t.DesiredSpecHash)
		metrics.ReconcilerActions.WithLabelValues("tenant", "spec_sync", "noop").Inc()
		return
	}
	current.Spec = desired.Spec
	if err := r.k8sClient.Update(ctx, current); err != nil {
		r.log.Error(err, "patch tenant CR", "name", t.Name)
		metrics.ReconcilerActions.WithLabelValues("tenant", "spec_sync", "error").Inc()
		return
	}
	r.markApplied(ctx, t.ID, t.DesiredSpecHash)
	metrics.ReconcilerActions.WithLabelValues("tenant", "spec_sync", "success").Inc()
}

func (r *Reconciler) markApplied(ctx context.Context, id uuid.UUID, hash string) {
	if err := r.repo.Update(ctx, nil, id, map[string]any{"applied_spec_hash": hash, "message": ""}); err != nil {
		r.log.Error(err, "mark applied")
	}
}

// toCR builds the desired tenant CR from a PG row, including a fresh quota
// rendering. Returns nil and logs if rendering fails.
func (r *Reconciler) toCR(ctx context.Context, t *Tenant) *tenantv1alpha1.Tenant {
	var snap SpecSnapshot
	if len(t.Spec) > 0 {
		if err := json.Unmarshal(t.Spec, &snap); err != nil {
			r.log.Error(err, "decode tenant spec snapshot", "id", t.ID)
			return nil
		}
	}
	quotas, err := RenderQuotas(ctx, nil, r.quotas, r.pools, t.ID)
	if err != nil {
		r.log.Error(err, "render quotas", "tenant", t.Name)
		return nil
	}
	cr := &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: t.Name,
			Labels: map[string]string{
				tenantv1alpha1.LabelTenantID:  t.ID.String(),
				tenantv1alpha1.LabelManagedBy: "compute",
			},
		},
		Spec: tenantv1alpha1.TenantSpec{
			DisplayName:   snap.DisplayName,
			Annotations:   snap.Annotations,
			Namespace:     snap.Namespace,
			Quotas:        quotas,
			InitResources: snap.InitResources,
			Suspended:     snap.Suspended,
		},
	}
	return cr
}
