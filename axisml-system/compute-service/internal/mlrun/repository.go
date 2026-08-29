package mlrun

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*store.MLRun, error) {
	var j store.MLRun
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&j).Error; err != nil {
		return nil, err
	}
	return &j, nil
}

// GetByNamespaceName looks up the live row for (namespace, name). Soft-
// deleted rows (deleted_at IS NOT NULL) are excluded so the name becomes
// reusable.
func (r *Repository) GetByNamespaceName(ctx context.Context, namespace, name string) (*store.MLRun, error) {
	var j store.MLRun
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND name = ? AND deleted_at IS NULL", namespace, name).
		First(&j).Error; err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *Repository) ListByNamespace(ctx context.Context, namespace string, limit, offset int, labelClause string, labelArgs []any) ([]store.MLRun, int64, error) {
	var rows []store.MLRun
	var total int64
	q := r.db.WithContext(ctx).Model(&store.MLRun{}).Where("namespace = ? AND deleted_at IS NULL", namespace)
	if labelClause != "" {
		q = q.Where(labelClause, labelArgs...)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// phaseColumns are the only columns the batch/single phase probes read — the
// heavy spec jsonb is never selected.
var phaseColumns = []string{"namespace", "name", "phase", "status", "scheduled_at"}

// ListPhasesByNames returns the phase columns for the given run names in the
// namespace (soft-deleted excluded). Missing names are simply absent from the
// result; the caller does not paginate — the set is bounded by the name list.
func (r *Repository) ListPhasesByNames(ctx context.Context, namespace string, names []string) ([]store.MLRun, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var rows []store.MLRun
	if err := r.db.WithContext(ctx).
		Select(phaseColumns).
		Where("namespace = ? AND name IN ? AND deleted_at IS NULL", namespace, names).
		Order("name ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListPhasesBySelector returns the phase columns for rows matching the label
// selector, paginated exactly like ListByNamespace.
func (r *Repository) ListPhasesBySelector(ctx context.Context, namespace string, limit, offset int, labelClause string, labelArgs []any) ([]store.MLRun, int64, error) {
	var rows []store.MLRun
	var total int64
	q := r.db.WithContext(ctx).Model(&store.MLRun{}).Where("namespace = ? AND deleted_at IS NULL", namespace)
	if labelClause != "" {
		q = q.Where(labelClause, labelArgs...)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Select(phaseColumns).
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) Create(ctx context.Context, j *store.MLRun) error {
	return r.db.WithContext(ctx).Create(j).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&store.MLRun{}).Where("id = ?", id).Updates(fields).Error
}

// UpdatePhase applies fields only while the row is still in the expected
// phase. Recovery observes the runtime outside PostgreSQL, so this CAS keeps a
// concurrent informer write from being overwritten by a stale observation.
func (r *Repository) UpdatePhase(ctx context.Context, id uuid.UUID, phase Status, fields map[string]any) (bool, error) {
	result := r.db.WithContext(ctx).Model(&store.MLRun{}).
		Where("id = ? AND phase = ? AND deleted_at IS NULL", id, string(phase)).
		Updates(fields)
	return result.RowsAffected == 1, result.Error
}

func (r *Repository) MarkDeleting(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&store.MLRun{}).Where("id = ?", id).Updates(map[string]any{
		"phase":      string(StatusDeleting),
		"deleted_at": time.Now().UTC(),
	}).Error
}

// FindObservable returns the live rows whose underlying workload may still
// change state and therefore need a runtime Observe poll: everything that has
// been (or is being) applied and is not yet terminal. Used by the standalone status
// poller (the Kubernetes form reflows via informer events instead).
func (r *Repository) FindObservable(ctx context.Context) ([]store.MLRun, error) {
	var rows []store.MLRun
	phases := []string{
		string(StatusCreating), string(StatusPending), string(StatusRunning),
		string(StatusCanceling), string(StatusDeleting),
	}
	if err := r.db.WithContext(ctx).
		Where("phase IN ?", phases).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// WorkSet groups rows that match reconciler predicates.
type WorkSet struct {
	Creating  []store.MLRun
	Canceling []store.MLRun
	Deleting  []store.MLRun
}

const workSetBatch = 100

// FindDispatchRecoverySet returns admitted rows whose dispatch has not
// converged within the controller's timeout. Pending is included so an
// in-place upgrade can recover the old standalone resource-wait state where no
// runtime object had been created yet.
func (r *Repository) FindDispatchRecoverySet(ctx context.Context, updatedBefore time.Time) ([]store.MLRun, error) {
	var rows []store.MLRun
	if err := r.db.WithContext(ctx).
		Where("phase IN ? AND updated_at <= ? AND deleted_at IS NULL", []string{string(StatusCreating), string(StatusPending)}, updatedBefore).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repository) FindWorkSet(ctx context.Context) (WorkSet, error) {
	var ws WorkSet
	// Only admitted Creating rows may reach the runtime. Queued rows are owned by
	// the admission controller and Pending rows already have a runtime object.
	if err := r.db.WithContext(ctx).
		Where("phase = ? AND deleted_at IS NULL", string(StatusCreating)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Creating).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("phase = ?", string(StatusCanceling)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Canceling).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("phase = ?", string(StatusDeleting)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Deleting).Error; err != nil {
		return ws, err
	}
	return ws, nil
}
