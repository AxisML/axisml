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

// GetByRepoVersion loads an artifact by its (repo_id, version) natural key.
// Soft-deleted rows are excluded.
func (r *Repository) GetByRepoVersion(ctx context.Context, repoID uuid.UUID, version string) (*Artifact, error) {
	var row Artifact
	if err := r.db.WithContext(ctx).
		Where("repo_id = ? AND version = ? AND deleted_at IS NULL", repoID, version).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListByRepo returns artifacts for a repo. Status filter is optional.
func (r *Repository) ListByRepo(ctx context.Context, repoID uuid.UUID, status string, limit, offset int) ([]Artifact, int64, error) {
	var rows []Artifact
	var total int64
	q := r.db.WithContext(ctx).Model(&Artifact{}).
		Where("repo_id = ? AND deleted_at IS NULL", repoID)
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
// Returns a map keyed by the parent repo's kind.
func (r *Repository) CountUploadingByKind(ctx context.Context) (map[string]int64, error) {
	type row struct {
		Kind string
		N    int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("artifacts").
		Select("artifact_repos.kind AS kind, COUNT(artifacts.id) AS n").
		Joins("JOIN artifact_repos ON artifact_repos.id = artifacts.repo_id").
		Where("artifacts.status = ?", StatusUploading).
		Where("artifacts.deleted_at IS NULL").
		Group("artifact_repos.kind").
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

// GCRow bundles an Artifact with the parent repo fields the GC handler
// needs (kind, scope segments, name). Joined in a single query so the
// worker doesn't fan out a parent lookup per row.
type GCRow struct {
	ID         uuid.UUID
	Version    string
	Digest     string
	RepoID     uuid.UUID
	Kind       string
	RepoName   string
	TenantName *string
}

// Scope mirrors *repomod.Repo.Scope so the GC worker stays in artifact-land.
func (g GCRow) Scope() string {
	if g.TenantName == nil {
		return "system"
	}
	return "tenants/" + *g.TenantName
}

// FindStaleUploading returns Uploading artifacts older than the cutoff,
// joined to their parent repo so the caller has everything it needs to
// dispatch a GCBackend call without a follow-up query (design §3.4 first
// predicate).
func (r *Repository) FindStaleUploading(ctx context.Context, cutoff time.Time) ([]GCRow, error) {
	var rows []GCRow
	err := r.db.WithContext(ctx).
		Table("artifacts").
		Select("artifacts.id, artifacts.version, artifacts.digest, artifacts.repo_id, "+
			"artifact_repos.kind AS kind, artifact_repos.name AS repo_name, artifact_repos.tenant_name AS tenant_name").
		Joins("JOIN artifact_repos ON artifact_repos.id = artifacts.repo_id").
		Where("artifacts.status = ? AND artifacts.created_at < ? AND artifacts.deleted_at IS NULL", StatusUploading, cutoff).
		Scan(&rows).Error
	return rows, err
}

// FindDeleting returns artifacts in the Deleting state, joined to their
// parent repo (design §3.4 third predicate).
func (r *Repository) FindDeleting(ctx context.Context) ([]GCRow, error) {
	var rows []GCRow
	err := r.db.WithContext(ctx).
		Table("artifacts").
		Select("artifacts.id, artifacts.version, artifacts.digest, artifacts.repo_id, "+
			"artifact_repos.kind AS kind, artifact_repos.name AS repo_name, artifact_repos.tenant_name AS tenant_name").
		Joins("JOIN artifact_repos ON artifact_repos.id = artifacts.repo_id").
		Where("artifacts.status = ?", StatusDeleting).
		Scan(&rows).Error
	return rows, err
}

// CascadeFromDeletingRepos moves any non-Deleted/non-Deleting artifact whose
// repo is Deleting into Deleting. Returns the number of rows affected.
func (r *Repository) CascadeFromDeletingRepos(ctx context.Context, tx *gorm.DB) (int64, error) {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	res := tx.Exec(`
		UPDATE artifacts
		SET status = 'Deleting', updated_at = now()
		WHERE deleted_at IS NULL
		  AND status NOT IN ('Deleting', 'Deleted')
		  AND repo_id IN (SELECT id FROM artifact_repos WHERE status = 'Deleting')
	`)
	return res.RowsAffected, res.Error
}

// FinalizeEmptyDeletingRepos flips Deleting repos with no remaining live
// artifacts to Deleted in a single set-based UPDATE.
func (r *Repository) FinalizeEmptyDeletingRepos(ctx context.Context, tx *gorm.DB) (int64, error) {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	res := tx.Exec(`
		UPDATE artifact_repos
		SET status = 'Deleted', deleted_at = now(), updated_at = now()
		WHERE status = 'Deleting'
		  AND NOT EXISTS (
		    SELECT 1 FROM artifacts
		    WHERE artifacts.repo_id = artifact_repos.id
		      AND artifacts.status <> 'Deleted'
		  )
	`)
	return res.RowsAffected, res.Error
}
