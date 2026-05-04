package quota

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/axisml/axisml/components/compute/internal/resourcepool"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
	"github.com/axisml/axisml/components/compute/pkg/strutil"
)

// Spec mirrors the inline quota body persisted in jsonb.
type Spec struct {
	Min corev1.ResourceList `json:"min,omitempty"`
	Max corev1.ResourceList `json:"max"`
}

// Service is the quota business layer. Note: Service does not Patch the
// Tenant CR directly — it only writes PG and asks the tenant Service to
// rehydrate the desired spec hash so the tenant reconciler picks up the
// change on its next tick.
type Service struct {
	repo  *Repository
	db    *gorm.DB
	pools *resourcepool.Service

	// markTenantDirty is wired by tenant.NewService once both modules are
	// constructed (avoids an import cycle). Must be called from inside the
	// quota write transaction.
	markTenantDirty func(tx *gorm.DB, tenantID uuid.UUID) error
}

func NewService(db *gorm.DB, pools *resourcepool.Service) *Service {
	return &Service{repo: NewRepository(db), db: db, pools: pools}
}

// SetTenantDirtyHook wires the dirty-marking callback (called by tenant.NewService).
func (s *Service) SetTenantDirtyHook(fn func(tx *gorm.DB, tenantID uuid.UUID) error) {
	s.markTenantDirty = fn
}

// CreateInput is the API request body.
type CreateInput struct {
	Pool string              `json:"pool" binding:"required,axisml_name"`
	Name string              `json:"name" binding:"required,axisml_name"`
	Min  corev1.ResourceList `json:"min"`
	Max  corev1.ResourceList `json:"max" binding:"required"`
}

// UpdateInput patches mutable fields. (pool, name) are immutable.
type UpdateInput struct {
	Min *corev1.ResourceList `json:"min"`
	Max *corev1.ResourceList `json:"max"`
}

// View is the API response.
type View struct {
	ID        uuid.UUID           `json:"id"`
	TenantID  uuid.UUID           `json:"tenantId"`
	Pool      string              `json:"pool"`
	PoolID    uuid.UUID           `json:"poolId"`
	Name      string              `json:"name"`
	Min       corev1.ResourceList `json:"min,omitempty"`
	Max       corev1.ResourceList `json:"max"`
	Used      corev1.ResourceList `json:"used,omitempty"`
	Status    string              `json:"status"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, in CreateInput) (*View, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid quota name")
	}
	if len(in.Max) == 0 {
		return nil, apperrors.New(apperrors.CodeValidation, "max must be set")
	}
	if err := validateMinMax(in.Min, in.Max); err != nil {
		return nil, err
	}

	pool, err := s.pools.Get(ctx, in.Pool)
	if err != nil {
		return nil, err
	}

	if existing, err := s.repo.GetByTenantPoolName(ctx, tenantID, pool.ID, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "quota already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}

	specJSON, err := encodeSpec(Spec{Min: in.Min, Max: in.Max})
	if err != nil {
		return nil, err
	}

	q := &Quota{
		ID:       uuid.New(),
		TenantID: tenantID,
		PoolID:   pool.ID,
		Name:     in.Name,
		Spec:     specJSON,
		Status:   string(StatusCreating),
		Used:     datatypes.JSON([]byte(`{}`)),
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Create(ctx, tx, q); err != nil {
			return err
		}
		if s.markTenantDirty != nil {
			if err := s.markTenantDirty(tx, tenantID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.toView(ctx, q)
}

// EnsureDefault is the bootstrap helper. Idempotent; returns the existing
// row if it already exists.
func (s *Service) EnsureDefault(ctx context.Context, tenantID, poolID uuid.UUID, max map[string]string) (*Quota, error) {
	if existing, err := s.repo.GetByTenantPoolName(ctx, tenantID, poolID, "default"); err == nil {
		return existing, nil
	} else if !IsNotFound(err) {
		return nil, err
	}
	rl := corev1.ResourceList{}
	for k, v := range max {
		q, err := resource.ParseQuantity(v)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeValidation, "parse default quota max", err)
		}
		rl[corev1.ResourceName(k)] = q
	}
	specJSON, err := encodeSpec(Spec{Max: rl})
	if err != nil {
		return nil, err
	}
	q := &Quota{
		ID:       uuid.New(),
		TenantID: tenantID,
		PoolID:   poolID,
		Name:     "default",
		Spec:     specJSON,
		Status:   string(StatusCreating),
		Used:     datatypes.JSON([]byte(`{}`)),
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Create(ctx, tx, q); err != nil {
			return err
		}
		if s.markTenantDirty != nil {
			return s.markTenantDirty(tx, tenantID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (s *Service) Get(ctx context.Context, tenantID uuid.UUID, name string) (*View, error) {
	q, err := s.findByTenantAndName(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}
	return s.toView(ctx, q)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Quota, error) {
	q, err := s.repo.Get(ctx, id)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "quota not found")
		}
		return nil, err
	}
	return q, nil
}

func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]View, int64, error) {
	rows, total, err := s.repo.ListByTenantPaginated(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, 0, len(rows))
	for i := range rows {
		v, err := s.toView(ctx, &rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, nil
}

func (s *Service) Update(ctx context.Context, tenantID uuid.UUID, name string, in UpdateInput) (*View, error) {
	q, err := s.findByTenantAndName(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}
	current, err := decodeSpec(q.Spec)
	if err != nil {
		return nil, err
	}
	if in.Min != nil {
		current.Min = *in.Min
	}
	if in.Max != nil {
		current.Max = *in.Max
	}
	if err := validateMinMax(current.Min, current.Max); err != nil {
		return nil, err
	}
	specJSON, err := encodeSpec(current)
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Update(ctx, tx, q.ID, map[string]any{"spec": specJSON}); err != nil {
			return err
		}
		if s.markTenantDirty != nil {
			return s.markTenantDirty(tx, tenantID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	q, err = s.repo.Get(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	return s.toView(ctx, q)
}

func (s *Service) Delete(ctx context.Context, tenantID uuid.UUID, name string) error {
	q, err := s.findByTenantAndName(ctx, tenantID, name)
	if err != nil {
		if e, ok := apperrors.As(err); ok && e.Code == apperrors.CodeNotFound {
			return nil // idempotent
		}
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.UpdateStatus(ctx, tx, q.ID, StatusDeleting); err != nil {
			return err
		}
		if s.markTenantDirty != nil {
			return s.markTenantDirty(tx, tenantID)
		}
		return nil
	})
}

// SoftDeleteAllByTenant cascades when the tenant itself is deleted.
func (s *Service) SoftDeleteAllByTenant(ctx context.Context, tx *gorm.DB, tenantID uuid.UUID) error {
	if tx == nil {
		tx = s.db.WithContext(ctx)
	}
	return tx.Model(&Quota{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Updates(map[string]any{
			"status":     string(StatusDeleting),
			"deleted_at": time.Now().UTC(),
		}).Error
}

// ListByTenantTx is used inside the tenant render path to materialise quota rows.
func (s *Service) ListByTenantTx(ctx context.Context, tx *gorm.DB, tenantID uuid.UUID) ([]Quota, error) {
	if tx == nil {
		return s.repo.ListByTenant(ctx, tenantID)
	}
	return s.repo.ListAllActiveByTenant(ctx, tx, tenantID)
}

// SyncFromTenantStatus is invoked by the tenant Informer with the latest
// `Tenant.status.quotas[]`. It looks up the matching PG row and updates
// status / used.
type StatusObservation struct {
	Pool  string
	Name  string
	Ready bool
	Used  corev1.ResourceList
}

func (s *Service) SyncFromTenantStatus(ctx context.Context, tenantID uuid.UUID, obs []StatusObservation) error {
	for _, o := range obs {
		pool, err := s.pools.Get(ctx, o.Pool)
		if err != nil {
			continue
		}
		q, err := s.repo.GetByTenantPoolName(ctx, tenantID, pool.ID, o.Name)
		if err != nil {
			if IsNotFound(err) {
				continue
			}
			return err
		}
		// Status transitions: Creating → Active when Ready=true.
		if Status(q.Status) == StatusCreating && o.Ready {
			if err := s.repo.UpdateStatus(ctx, nil, q.ID, StatusActive); err != nil {
				return err
			}
		}
		// Used cache.
		if len(o.Used) > 0 {
			b, err := json.Marshal(o.Used)
			if err == nil {
				_ = s.repo.UpdateUsed(ctx, q.ID, datatypes.JSON(b))
			}
		}
	}
	return nil
}

// MarkDeletedByMissingFromTenant is invoked when the Tenant CR is gone or
// the quota disappeared from spec.quotas[].
func (s *Service) MarkDeletedByMissingFromTenant(ctx context.Context, tenantID uuid.UUID, presentNames map[string]struct{}) error {
	rows, err := s.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, q := range rows {
		if Status(q.Status) != StatusDeleting {
			continue
		}
		key := q.PoolID.String() + "/" + q.Name
		if _, present := presentNames[key]; !present {
			if err := s.repo.SoftDelete(ctx, nil, q.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) findByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (*Quota, error) {
	rows, err := s.repo.FindByTenantName(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}
	switch len(rows) {
	case 0:
		return nil, apperrors.New(apperrors.CodeNotFound, "quota not found")
	case 1:
		return &rows[0], nil
	default:
		// Schema permits same name across pools; the API path doesn't carry
		// pool, so refuse rather than picking arbitrarily.
		return nil, apperrors.New(apperrors.CodeConflict, "quota name is ambiguous across pools; specify pool explicitly")
	}
}

func (s *Service) toView(ctx context.Context, q *Quota) (*View, error) {
	pool, err := s.pools.GetByID(ctx, q.PoolID)
	if err != nil {
		return nil, err
	}
	spec, err := decodeSpec(q.Spec)
	if err != nil {
		return nil, err
	}
	used := corev1.ResourceList{}
	if len(q.Used) > 0 {
		_ = json.Unmarshal(q.Used, &used)
	}
	return &View{
		ID:        q.ID,
		TenantID:  q.TenantID,
		Pool:      pool.Name,
		PoolID:    q.PoolID,
		Name:      q.Name,
		Min:       spec.Min,
		Max:       spec.Max,
		Used:      used,
		Status:    q.Status,
		CreatedAt: q.CreatedAt,
		UpdatedAt: q.UpdatedAt,
	}, nil
}

func encodeSpec(s Spec) (datatypes.JSON, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

func decodeSpec(b datatypes.JSON) (Spec, error) {
	var s Spec
	if len(b) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	return s, nil
}

func validateMinMax(min, max corev1.ResourceList) error {
	for k, m := range min {
		mx, ok := max[k]
		if !ok {
			continue
		}
		if m.Cmp(mx) > 0 {
			return apperrors.Newf(apperrors.CodeValidation, "min[%s] (%s) > max[%s] (%s)", k, m.String(), k, mx.String())
		}
	}
	for _, v := range max {
		if v.Sign() < 0 {
			return apperrors.New(apperrors.CodeValidation, "max values must be non-negative")
		}
	}
	for _, v := range min {
		if v.Sign() < 0 {
			return apperrors.New(apperrors.CodeValidation, "min values must be non-negative")
		}
	}
	return nil
}

// PrecheckRequest is the call site signature for jobs/services.
type PrecheckRequest struct {
	QuotaID  uuid.UUID
	Resource corev1.ResourceList // requested for this submit
}

// Precheck performs a best-effort used+request <= max comparison and returns
// CodeQuotaExceeded on rejection. Non-fatal errors are surfaced to the caller.
func (s *Service) Precheck(ctx context.Context, req PrecheckRequest) error {
	q, err := s.GetByID(ctx, req.QuotaID)
	if err != nil {
		return err
	}
	spec, err := decodeSpec(q.Spec)
	if err != nil {
		return err
	}
	used := corev1.ResourceList{}
	_ = json.Unmarshal(q.Used, &used)
	for name, want := range req.Resource {
		max, ok := spec.Max[name]
		if !ok {
			continue
		}
		current := used[name]
		current.Add(want)
		if current.Cmp(max) > 0 {
			return apperrors.Newf(apperrors.CodeQuotaExceeded,
				"quota %s exceeded for %s: used+request=%s > max=%s",
				q.Name, name, current.String(), max.String())
		}
	}
	return nil
}
