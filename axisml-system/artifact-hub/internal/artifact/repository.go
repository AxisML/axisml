package artifact

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/store"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// IsNotFound surfaces the GORM "no row" sentinel.
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

// GetByID loads an artifact by primary key.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*store.Artifact, error) {
	var row store.Artifact
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetByCoordIncludingDeleted loads an artifact by its (namespace, name, version)
// natural key, soft-deleted rows included. kind is not part of the key — a name
// is unique within a namespace across all kinds. Read paths (Get / Resolve /
// Initiate's idempotency check) use this so a client can still observe a row's
// terminal Deleted status after GC has finalised it, and so a Deleted tombstone
// still occupies the coordinate (which never recycles — database.md §1.2).
func (r *Repository) GetByCoordIncludingDeleted(ctx context.Context, namespace, name, version string) (*store.Artifact, error) {
	var row store.Artifact
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND name = ? AND version = ?",
			namespace, name, version).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListByCoord returns artifacts under a namespace, optionally narrowed by kind
// and/or name. All three of kind / name / status filters are optional: pass an
// empty kind to list across every kind, an empty name to list across all names,
// and an empty status for no status filter.
func (r *Repository) ListByCoord(ctx context.Context, namespace, kind, name, status string, limit, offset int, labelClause string, labelArgs []any) ([]store.Artifact, int64, error) {
	var rows []store.Artifact
	var total int64
	q := r.db.WithContext(ctx).Model(&store.Artifact{}).
		Where("namespace = ? AND deleted_at IS NULL", namespace)
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if name != "" {
		q = q.Where("name = ?", name)
	}
	if status != "" {
		q = q.Where("status = ?", status)
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

// Create persists a new artifact row.
func (r *Repository) Create(ctx context.Context, tx *gorm.DB, row *store.Artifact) error {
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
	return tx.Model(&store.Artifact{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateIfStatus applies fields only when the row is still in expectStatus,
// returning whether a row was actually updated. This is a compare-and-set used
// to avoid clobbering a row that transitioned concurrently (e.g. GC marking an
// Uploading row Failed after the client's Complete already flipped it Ready).
func (r *Repository) UpdateIfStatus(ctx context.Context, tx *gorm.DB, id uuid.UUID, expectStatus string, fields map[string]any) (bool, error) {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	res := tx.Model(&store.Artifact{}).
		Where("id = ? AND status = ?", id, expectStatus).
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
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
