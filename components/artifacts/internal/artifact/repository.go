package artifact

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// IsNotFound surfaces the GORM "no row" sentinel.
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

// GetByID loads an artifact by primary key.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Artifact, error) {
	var row Artifact
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetByCoord loads an artifact by its (namespace, kind, name, version)
// natural key. Soft-deleted rows are excluded.
func (r *Repository) GetByCoord(ctx context.Context, namespace, kind, name, version string) (*Artifact, error) {
	var row Artifact
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND kind = ? AND name = ? AND version = ? AND deleted_at IS NULL",
			namespace, kind, name, version).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListByCoord returns artifacts under (namespace, kind, name). Status filter
// is optional. To list across versions of a single artifact name, callers
// pass the name; to list across all names within a kind, pass an empty
// name.
func (r *Repository) ListByCoord(ctx context.Context, namespace, kind, name, status string, limit, offset int) ([]Artifact, int64, error) {
	var rows []Artifact
	var total int64
	q := r.db.WithContext(ctx).Model(&Artifact{}).
		Where("namespace = ? AND kind = ? AND deleted_at IS NULL", namespace, kind)
	if name != "" {
		q = q.Where("name = ?", name)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Create persists a new artifact row.
func (r *Repository) Create(ctx context.Context, tx *gorm.DB, row *Artifact) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Create(row).Error
}

// Update applies the given column map to an artifact row.
func (r *Repository) Update(ctx context.Context, tx *gorm.DB, id uuid.UUID, fields map[string]any) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Model(&Artifact{}).Where("id = ?", id).Updates(fields).Error
}

// CountUploadingByKind powers the axisml_artifacts_uploading_count metric.
func (r *Repository) CountUploadingByKind(ctx context.Context) (map[string]int64, error) {
	type row struct {
		Kind string
		N    int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("artifacts").
		Select("kind, COUNT(id) AS n").
		Where("status = ?", StatusUploading).
		Where("deleted_at IS NULL").
		Group("kind").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Kind] = r.N
	}
	return out, nil
}

// GCRow bundles an Artifact with the fields the GC handler needs to dispatch
// a GCBackend call without follow-up queries.
type GCRow struct {
	ID        uuid.UUID
	Namespace string
	Kind      string
	Name      string
	Version   string
	Digest    string
}

// FindStaleUploading returns Uploading artifacts older than the cutoff.
func (r *Repository) FindStaleUploading(ctx context.Context, cutoff time.Time) ([]GCRow, error) {
	var rows []GCRow
	err := r.db.WithContext(ctx).
		Table("artifacts").
		Select("id, namespace, kind, name, version, digest").
		Where("status = ? AND created_at < ? AND deleted_at IS NULL", StatusUploading, cutoff).
		Scan(&rows).Error
	return rows, err
}

// FindDeleting returns artifacts in the Deleting state.
func (r *Repository) FindDeleting(ctx context.Context) ([]GCRow, error) {
	var rows []GCRow
	err := r.db.WithContext(ctx).
		Table("artifacts").
		Select("id, namespace, kind, name, version, digest").
		Where("status = ?", StatusDeleting).
		Scan(&rows).Error
	return rows, err
}
