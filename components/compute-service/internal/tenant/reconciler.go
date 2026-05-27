package tenant

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/metrics"
)

// Reconciler is the Tenant Outbox loop. It scans the `tenants` PG table on
// a fixed interval and drives CR mutations to converge:
//
//   - phase=Creating               → Create() the Tenant CR
//   - generation != observed       → Patch() spec, update observed_generation
//   - phase=Deleting               → Delete() the Tenant CR (Informer will
//     finalise the PG row to Deleted on the resulting CR DELETE event)
//
// Leader-only (NeedLeaderElection=true) so multi-replica compute deployments
// don't double-write CRs.
type Reconciler struct {
	db       *gorm.DB
	repo     *Repository
	k8s      client.Client
	log      logr.Logger
	interval time.Duration
}

func NewReconciler(db *gorm.DB, cl client.Client, log logr.Logger, interval time.Duration) *Reconciler {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Reconciler{
		db:       db,
		repo:     NewRepository(db),
		k8s:      cl,
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
		r.log.Error(err, "tenant find work set")
		return
	}
	for i := range ws.Creating {
		r.handleCreate(ctx, &ws.Creating[i])
	}
	for i := range ws.Patching {
		r.handlePatch(ctx, &ws.Patching[i])
	}
	for i := range ws.Deleting {
		r.handleDelete(ctx, &ws.Deleting[i])
	}
}

func (r *Reconciler) handleCreate(ctx context.Context, t *Tenant) {
	cr, err := ToCR(t)
	if err != nil {
		r.log.Error(err, "render tenant CR", "name", t.Name)
		return
	}
	if err := r.k8s.Create(ctx, cr); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// CR exists; treat as success and mark observed_generation so we
			// don't churn — the Informer will overwrite phase based on the
			// CR's own status.
			_ = r.repo.Update(ctx, t.ID, map[string]any{
				"observed_generation": t.Generation,
			})
			metrics.ReconcilerActions.WithLabelValues("tenant", "creating", "noop").Inc()
			return
		}
		r.log.Error(err, "create Tenant CR", "name", t.Name)
		_ = r.repo.Update(ctx, t.ID, map[string]any{"phase": PhaseFailed,
			"status": jsonMessage("create Tenant CR failed: " + err.Error())})
		metrics.ReconcilerActions.WithLabelValues("tenant", "creating", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, t.ID, map[string]any{
		"observed_generation": t.Generation,
	})
	metrics.ReconcilerActions.WithLabelValues("tenant", "creating", "success").Inc()
}

func (r *Reconciler) handlePatch(ctx context.Context, t *Tenant) {
	cr, err := ToCR(t)
	if err != nil {
		r.log.Error(err, "render tenant CR for patch", "name", t.Name)
		return
	}
	// MergeFrom an empty CR effectively says "replace these fields on the
	// remote object". For simplicity we send the whole desired spec.
	existing := &tenantv1alpha1.Tenant{}
	if getErr := r.k8s.Get(ctx, types.NamespacedName{Name: t.Name}, existing); getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// CR was deleted out-of-band; recreate per design §5.5.
			r.handleCreate(ctx, t)
			return
		}
		r.log.Error(getErr, "get Tenant CR before patch", "name", t.Name)
		metrics.ReconcilerActions.WithLabelValues("tenant", "patching", "error").Inc()
		return
	}
	patch := client.MergeFrom(existing.DeepCopy())
	existing.Spec = cr.Spec
	if err := r.k8s.Patch(ctx, existing, patch); err != nil {
		r.log.Error(err, "patch Tenant CR", "name", t.Name)
		metrics.ReconcilerActions.WithLabelValues("tenant", "patching", "error").Inc()
		return
	}
	_ = r.repo.Update(ctx, t.ID, map[string]any{
		"observed_generation": t.Generation,
	})
	metrics.ReconcilerActions.WithLabelValues("tenant", "patching", "success").Inc()
}

func (r *Reconciler) handleDelete(ctx context.Context, t *Tenant) {
	cr := &tenantv1alpha1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: t.Name}}
	err := r.k8s.Delete(ctx, cr)
	if err == nil {
		metrics.ReconcilerActions.WithLabelValues("tenant", "deleting", "success").Inc()
		return
	}
	if apierrors.IsNotFound(err) {
		// CR already gone — finalise the PG row.
		_ = r.repo.Update(ctx, t.ID, map[string]any{"phase": PhaseDeleted})
		metrics.ReconcilerActions.WithLabelValues("tenant", "deleting", "noop").Inc()
		return
	}
	r.log.Error(err, "delete Tenant CR", "name", t.Name)
	metrics.ReconcilerActions.WithLabelValues("tenant", "deleting", "error").Inc()
}

func jsonMessage(msg string) []byte {
	b, _ := json.Marshal(map[string]any{"message": msg})
	return b
}
