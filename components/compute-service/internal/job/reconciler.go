package job

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mljobv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/metrics"
)

// Reconciler implements the job Outbox loop. Reads namespace directly off
// the row.
type Reconciler struct {
	db        *gorm.DB
	repo      *Repository
	k8sClient client.Client
	log       logr.Logger
	interval  time.Duration
}

func NewReconciler(
	db *gorm.DB,
	cl client.Client,
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
		log:       log,
		interval:  interval,
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

func (r *Reconciler) handleCreate(ctx context.Context, j *Job) {
	cr, err := ToCR(j)
	if err != nil {
		r.log.Error(err, "render job CR")
		return
	}
	if err := r.k8sClient.Create(ctx, cr); err != nil {
		if apierrors.IsAlreadyExists(err) {
			metrics.ReconcilerActions.WithLabelValues("job", "creating", "noop").Inc()
			return
		}
		r.log.Error(err, "create MLJob", "name", j.Name)
		_ = r.repo.Update(ctx, j.ID, map[string]any{"message": err.Error()})
		metrics.ReconcilerActions.WithLabelValues("job", "creating", "error").Inc()
		return
	}
	metrics.ReconcilerActions.WithLabelValues("job", "creating", "success").Inc()
}

func (r *Reconciler) handleCancel(ctx context.Context, j *Job) {
	cr := &mljobv1alpha1.MLJob{ObjectMeta: metav1.ObjectMeta{Name: j.Name, Namespace: j.Namespace}}
	patch := client.RawPatch(client.Merge.Type(), []byte(`{"spec":{"runPolicy":{"suspend":true}}}`))
	if err := r.k8sClient.Patch(ctx, cr, patch); err != nil {
		if apierrors.IsNotFound(err) {
			metrics.ReconcilerActions.WithLabelValues("job", "canceling", "noop").Inc()
			return
		}
		r.log.Error(err, "patch suspend", "job", j.Name)
		metrics.ReconcilerActions.WithLabelValues("job", "canceling", "error").Inc()
		return
	}
	metrics.ReconcilerActions.WithLabelValues("job", "canceling", "success").Inc()
}

func (r *Reconciler) handleDelete(ctx context.Context, j *Job) {
	cr := &mljobv1alpha1.MLJob{ObjectMeta: metav1.ObjectMeta{Name: j.Name, Namespace: j.Namespace}}
	err := r.k8sClient.Delete(ctx, cr)
	if err == nil {
		metrics.ReconcilerActions.WithLabelValues("job", "deleting", "success").Inc()
		return
	}
	if apierrors.IsNotFound(err) {
		_ = r.repo.Update(ctx, j.ID, map[string]any{"status": string(StatusDeleted)})
		metrics.ReconcilerActions.WithLabelValues("job", "deleting", "noop").Inc()
		return
	}
	r.log.Error(err, "delete MLJob", "name", j.Name)
	metrics.ReconcilerActions.WithLabelValues("job", "deleting", "error").Inc()
}
