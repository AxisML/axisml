package trafficpolicy

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*store.TrafficPolicy, error) {
	var p store.TrafficPolicy
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) GetByNamespaceName(ctx context.Context, namespace, name string) (*store.TrafficPolicy, error) {
	var p store.TrafficPolicy
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND name = ? AND deleted_at IS NULL", namespace, name).
		First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) ListByNamespace(ctx context.Context, namespace string, limit, offset int, labelClause string, labelArgs []any) ([]store.TrafficPolicy, int64, error) {
	var rows []store.TrafficPolicy
	var total int64
	q := r.db.WithContext(ctx).Model(&store.TrafficPolicy{}).Where("namespace = ? AND deleted_at IS NULL", namespace)
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

func (r *Repository) Create(ctx context.Context, p *store.TrafficPolicy) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// CreateGuarded inserts the policy while holding a transaction-scoped advisory
// lock per member service and re-checking the "at most one active policy per
// service" occupancy rule inside the transaction. This closes the TOCTOU
// between validateMembers' occupancy check and the insert: two concurrent
// creates that name the same member serialize on the advisory lock, so the
// second sees the first's committed row and is rejected. A DB unique constraint
// is impossible here because the membership lives in a jsonb array.
func (r *Repository) CreateGuarded(ctx context.Context, p *store.TrafficPolicy, memberNames []string) error {
	sorted := append([]string(nil), memberNames...)
	sort.Strings(sorted) // deterministic lock order avoids deadlocks between creates sharing members
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Take all member locks first (namespaced so two namespaces don't contend),
		// released automatically at end of transaction.
		for _, name := range sorted {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?), hashtext(?))",
				p.Namespace, name).Error; err != nil {
				return err
			}
		}
		// Re-check occupancy under the lock against committed rows.
		for _, name := range sorted {
			var n int64
			if err := tx.Model(&store.TrafficPolicy{}).
				Where("namespace = ? AND deleted_at IS NULL AND EXISTS ("+
					"SELECT 1 FROM jsonb_array_elements(spec->'backends') AS b "+
					"WHERE b->>'serviceName' = ?)", p.Namespace, name).
				Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return apperrors.Newf(apperrors.CodeConflict,
					"member %q is already used by an active policy", name)
			}
		}
		return tx.Create(p).Error
	})
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&store.TrafficPolicy{}).Where("id = ?", id).Updates(fields).Error
}

func (r *Repository) MarkDeleting(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&store.TrafficPolicy{}).Where("id = ?", id).Updates(map[string]any{
		"phase":      string(StatusDeleting),
		"deleted_at": time.Now().UTC(),
	}).Error
}

// FindActiveReferencing returns active (non-deleted) policies in the namespace
// whose spec.backends[] references the given service name. Used to enforce the
// "a service is referenced by at most one active policy" occupancy rule
// (compute-service.md §4.5) — there is no DB constraint for it because the
// membership lives inside a jsonb array.
func (r *Repository) FindActiveReferencing(ctx context.Context, namespace, serviceName string) ([]store.TrafficPolicy, error) {
	var rows []store.TrafficPolicy
	err := r.db.WithContext(ctx).
		Where("namespace = ? AND deleted_at IS NULL AND EXISTS ("+
			"SELECT 1 FROM jsonb_array_elements(spec->'backends') AS b "+
			"WHERE b->>'serviceName' = ?)", namespace, serviceName).
		Find(&rows).Error
	return rows, err
}

// ActiveReferenceName returns one active policy referencing serviceName, or an
// empty string when the service is not in use.
func (r *Repository) ActiveReferenceName(ctx context.Context, namespace, serviceName string) (string, error) {
	rows, err := r.FindActiveReferencing(ctx, namespace, serviceName)
	if err != nil || len(rows) == 0 {
		return "", err
	}
	return rows[0].Name, nil
}

// FindObservable returns the live rows whose underlying route may still change
// state and therefore need a runtime Observe poll. Used by the Lite status
// poller (the Kubernetes form reflows via informer events instead).
func (r *Repository) FindObservable(ctx context.Context) ([]store.TrafficPolicy, error) {
	var rows []store.TrafficPolicy
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
	Creating  []store.TrafficPolicy
	Deleting  []store.TrafficPolicy
	SpecDirty []store.TrafficPolicy
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
