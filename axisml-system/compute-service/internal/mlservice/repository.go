package mlservice

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

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*store.MLService, error) {
	var s store.MLService
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetByNamespaceName(ctx context.Context, namespace, name string) (*store.MLService, error) {
	var s store.MLService
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND name = ? AND deleted_at IS NULL", namespace, name).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListByNamespace(ctx context.Context, namespace, kind string, limit, offset int, labelClause string, labelArgs []any) ([]store.MLService, int64, error) {
	var rows []store.MLService
	var total int64
	q := r.db.WithContext(ctx).Model(&store.MLService{}).Where("namespace = ? AND deleted_at IS NULL", namespace)
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
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
var phaseColumns = []string{"namespace", "name", "phase", "status", "generation", "observed_generation"}

// ListPhasesByNames returns the phase columns for the given service names in
// the namespace (soft-deleted excluded). Missing names are simply absent from
// the result; the caller does not paginate — the set is bounded by the name
// list.
func (r *Repository) ListPhasesByNames(ctx context.Context, namespace string, names []string) ([]store.MLService, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var rows []store.MLService
	if err := r.db.WithContext(ctx).
		Select(phaseColumns).
		Where("namespace = ? AND name IN ? AND deleted_at IS NULL", namespace, names).
		Order("name ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListPhasesBySelector returns the phase columns for rows matching the kind and
// label selector, paginated exactly like ListByNamespace.
func (r *Repository) ListPhasesBySelector(ctx context.Context, namespace, kind string, limit, offset int, labelClause string, labelArgs []any) ([]store.MLService, int64, error) {
	var rows []store.MLService
	var total int64
	q := r.db.WithContext(ctx).Model(&store.MLService{}).Where("namespace = ? AND deleted_at IS NULL", namespace)
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
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

func (r *Repository) Create(ctx context.Context, s *store.MLService) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&store.MLService{}).Where("id = ?", id).Updates(fields).Error
}

func (r *Repository) MarkDeleting(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&store.MLService{}).Where("id = ?", id).Updates(map[string]any{
		"phase":      string(StatusDeleting),
		"deleted_at": time.Now().UTC(),
	}).Error
}

// FindObservable returns the live rows whose underlying workload may still
// change state and therefore need a runtime Observe poll. Used by the standalone
// status poller (the Kubernetes form reflows via informer events instead).
func (r *Repository) FindObservable(ctx context.Context) ([]store.MLService, error) {
	var rows []store.MLService
	phases := []string{
		string(StatusCreating), string(StatusPending), string(StatusReady),
		string(StatusDegraded), string(StatusFailed), string(StatusDeleting),
	}
	if err := r.db.WithContext(ctx).
		Where("phase IN ?", phases).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

type WorkSet struct {
	Creating  []store.MLService
	Deleting  []store.MLService
	SpecDirty []store.MLService
}

const workSetBatch = 100

func (r *Repository) FindWorkSet(ctx context.Context) (WorkSet, error) {
	var ws WorkSet
	if err := r.db.WithContext(ctx).
		Where("phase = ? AND deleted_at IS NULL", string(StatusCreating)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Creating).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("phase = ?", string(StatusDeleting)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Deleting).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("generation <> observed_generation AND deleted_at IS NULL").
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.SpecDirty).Error; err != nil {
		return ws, err
	}
	return ws, nil
}
