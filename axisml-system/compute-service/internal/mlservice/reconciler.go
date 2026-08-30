package mlservice

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/metrics"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/serviceadmission"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// Reconciler implements the service Outbox loop. Namespace is read from
// the row directly; workloads are driven through the ComputeRuntime contract.
type Reconciler struct {
	db           *gorm.DB
	repo         *Repository
	runtime      extensions.ComputeRuntime
	log          logr.Logger
	interval     time.Duration
	tenantPrefix bool
}

func NewReconciler(db *gorm.DB, rt extensions.ComputeRuntime, log logr.Logger, interval time.Duration, tenantPrefix bool) *Reconciler {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Reconciler{db: db, repo: NewRepository(db), runtime: rt, log: log, interval: interval, tenantPrefix: tenantPrefix}
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
	for i := range ws.DispatchDirty {
		r.handleAdmissionSync(ctx, &ws.DispatchDirty[i])
	}
}

// handleCreate submits the already-admitted minimum serving set. Queued rows
// never reach this reconciler; the admission controller first reserves one
// complete serving unit and atomically advances the row to Creating.
func (r *Reconciler) handleCreate(ctx context.Context, s *store.MLService) {
	cr, err := ToCR(s, r.tenantPrefix)
	if err != nil {
		r.log.Error(err, "render service CR")
		return
	}
	if err := r.runtime.ApplyMLService(ctx, cr); err != nil {
		if extensions.IsResourceUnavailable(err) {
			// Capacity changed after the admission snapshot. The runtime contract
			// guarantees no instance was created, so release this reservation and
			// return a first dispatch to Queued.
			r.rollbackAdmission(ctx, s, err.Error())
			metrics.ReconcilerActions.WithLabelValues("mlservice", "creating", "queued").Inc()
			return
		}
		r.setStatusMessage(ctx, s, err.Error())
		r.log.Error(err, "apply MLService")
		metrics.ReconcilerActions.WithLabelValues("mlservice", "creating", "error").Inc()
		return
	}
	r.markDispatched(ctx, s, true)
	metrics.ReconcilerActions.WithLabelValues("mlservice", "creating", "success").Inc()
}

// setStatusMessage merges a message into the row's status jsonb without touching
// the other status fields.
func (r *Reconciler) setStatusMessage(ctx context.Context, s *store.MLService, msg string) {
	var sf server.MLServiceStatus
	if len(s.StatusJSON) > 0 {
		_ = json.Unmarshal(s.StatusJSON, &sf)
	}
	sf.Message = msg
	b, _ := json.Marshal(sf)
	_ = r.repo.Update(ctx, s.ID, map[string]any{"status": b})
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

// handleAdmissionSync converges the runtime onto the admitted replica vector.
// Desired scale-up never reaches the runtime until the admission controller has
// reserved its incremental capacity and quota.
func (r *Reconciler) handleAdmissionSync(ctx context.Context, s *store.MLService) {
	cr, err := ToCR(s, r.tenantPrefix)
	if err != nil {
		r.log.Error(err, "render service CR")
		return
	}
	if err := r.runtime.ApplyMLService(ctx, cr); err != nil {
		if extensions.IsResourceUnavailable(err) {
			// Keep already-dispatched replicas serving and release only the
			// unmaterialised increment.
			r.rollbackAdmission(ctx, s, err.Error())
			metrics.ReconcilerActions.WithLabelValues("mlservice", "admission_sync", "pending").Inc()
			return
		}
		r.log.Error(err, "apply MLService spec")
		metrics.ReconcilerActions.WithLabelValues("mlservice", "admission_sync", "error").Inc()
		return
	}
	r.markDispatched(ctx, s, false)
	metrics.ReconcilerActions.WithLabelValues("mlservice", "admission_sync", "success").Inc()
}

func (r *Reconciler) markDispatched(ctx context.Context, s *store.MLService, initial bool) {
	updates := map[string]any{"dispatched_replicas": datatypes.JSON(s.AdmittedReplicas)}
	q := r.db.WithContext(ctx).Model(&store.MLService{}).
		Where("id = ? AND deleted_at IS NULL AND phase NOT IN ?", s.ID, []string{string(StatusDeleting), string(StatusDeleted)})
	if initial {
		updates["phase"] = string(StatusPending)
		q = q.Where("phase = ?", string(StatusCreating))
	}
	result := q.Updates(updates)
	if result.Error != nil || result.RowsAffected == 0 {
		return
	}
	if fullyAdmitted(s) {
		// A concurrent Scale bumps generation, so this condition cannot falsely
		// report that a newer desired generation was fully dispatched.
		_ = r.db.WithContext(ctx).Model(&store.MLService{}).
			Where("id = ? AND generation = ?", s.ID, s.Generation).
			Update("observed_generation", s.Generation).Error
	}
}

func (r *Reconciler) rollbackAdmission(ctx context.Context, snapshot *store.MLService, message string) {
	_ = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current store.MLService
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", snapshot.ID).First(&current).Error; err != nil {
			return err
		}
		if Status(current.Phase) == StatusDeleting || Status(current.Phase) == StatusDeleted || current.DeletedAt != nil {
			return nil
		}
		var spec mlservicev1alpha1.MLServiceSpec
		if err := json.Unmarshal(current.Spec, &spec); err != nil {
			return err
		}
		desired := serviceadmission.Desired(spec)
		currentAdmitted := serviceadmission.Decode(current.AdmittedReplicas, len(spec.Roles), desired)
		snapshotAdmitted := serviceadmission.Decode(snapshot.AdmittedReplicas, len(spec.Roles), desired)
		if !serviceadmission.Equal(currentAdmitted, snapshotAdmitted) {
			// Admission advanced while Apply was in flight; let the newer snapshot
			// take its own dispatch attempt instead of rolling it back blindly.
			return nil
		}
		dispatched := serviceadmission.Decode(current.DispatchedReplicas, len(spec.Roles), serviceadmission.Zero(spec))
		var status server.MLServiceStatus
		_ = json.Unmarshal(current.StatusJSON, &status)
		status.AdmissionReason = "InsufficientResources"
		status.AdmissionMessage = message
		status.AdmittedReplicas = 0
		statusJSON, _ := json.Marshal(status)
		updates := map[string]any{
			"admitted_replicas": datatypes.JSON(serviceadmission.Encode(dispatched)),
			"status":            datatypes.JSON(statusJSON),
		}
		if allZero(dispatched) {
			updates["phase"] = string(StatusQueued)
		}
		return tx.Model(&store.MLService{}).Where("id = ?", current.ID).Updates(updates).Error
	})
}

func fullyAdmitted(s *store.MLService) bool {
	var spec mlservicev1alpha1.MLServiceSpec
	if err := json.Unmarshal(s.Spec, &spec); err != nil {
		return false
	}
	desired := serviceadmission.Desired(spec)
	admitted := serviceadmission.Decode(s.AdmittedReplicas, len(spec.Roles), desired)
	return serviceadmission.Equal(admitted, desired)
}

func allZero(values []int32) bool {
	for _, value := range values {
		if value != 0 {
			return false
		}
	}
	return true
}
