package tenant

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	tenantv1alpha1 "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"

	"github.com/axisml/axisml/components/compute/internal/quota"
	"github.com/axisml/axisml/components/compute/internal/resourcepool"
	"github.com/axisml/axisml/components/compute/internal/spechash"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
	"github.com/axisml/axisml/components/compute/pkg/strutil"
)

// Service is the tenant business layer.
type Service struct {
	repo   *Repository
	db     *gorm.DB
	quotas *quota.Service
	pools  *resourcepool.Service
}

// NewService constructs the service and wires the quota → tenant dirty hook.
func NewService(db *gorm.DB, quotas *quota.Service, pools *resourcepool.Service) *Service {
	s := &Service{
		repo:   NewRepository(db),
		db:     db,
		quotas: quotas,
		pools:  pools,
	}
	if quotas != nil {
		quotas.SetTenantDirtyHook(s.markSpecDirty)
	}
	return s
}

// CreateInput is the API request body.
type CreateInput struct {
	Name          string                       `json:"name" binding:"required,axisml_name"`
	DisplayName   string                       `json:"displayName"`
	Namespace     tenantv1alpha1.NamespaceSpec `json:"namespace" binding:"required"`
	Annotations   map[string]string            `json:"annotations"`
	InitResources tenantv1alpha1.InitResources `json:"initResources"`
}

// UpdateInput patches mutable fields. Name and namespace.name are immutable.
type UpdateInput struct {
	DisplayName   *string                       `json:"displayName"`
	Annotations   *map[string]string            `json:"annotations"`
	NamespaceMeta *NamespaceMetaPatch           `json:"namespace"`
	InitResources *tenantv1alpha1.InitResources `json:"initResources"`
}

// NamespaceMetaPatch lets callers update labels/annotations of the target NS.
type NamespaceMetaPatch struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// View is the HTTP response payload.
type View struct {
	ID          uuid.UUID                    `json:"id"`
	Name        string                       `json:"name"`
	DisplayName string                       `json:"displayName,omitempty"`
	Namespace   tenantv1alpha1.NamespaceSpec `json:"namespace"`
	Annotations map[string]string            `json:"annotations,omitempty"`
	Status      string                       `json:"status"`
	Message     string                       `json:"message,omitempty"`
	CreatedAt   time.Time                    `json:"createdAt"`
	UpdatedAt   time.Time                    `json:"updatedAt"`
}

// Create inserts a tenant in `Creating` and persists the desired spec snapshot.
func (s *Service) Create(ctx context.Context, in CreateInput) (*View, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid tenant name")
	}
	if in.Namespace.Name == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "namespace.name is required")
	}
	if existing, err := s.repo.GetActiveByName(ctx, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "tenant already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}

	snap := SpecSnapshot{
		DisplayName:   in.DisplayName,
		Annotations:   in.Annotations,
		Namespace:     in.Namespace,
		InitResources: in.InitResources,
	}
	specJSON, err := EncodeSpec(snap)
	if err != nil {
		return nil, err
	}
	hash, err := spechash.Compute(snap)
	if err != nil {
		return nil, err
	}
	annJSON, err := json.Marshal(orEmptyMap(in.Annotations))
	if err != nil {
		return nil, err
	}

	t := &Tenant{
		ID:              uuid.New(),
		Name:            in.Name,
		Namespace:       in.Namespace.Name,
		DisplayName:     in.DisplayName,
		Spec:            datatypes.JSON(specJSON),
		DesiredSpecHash: hash,
		Status:          string(StatusCreating),
		Annotations:     datatypes.JSON(annJSON),
	}
	if err := s.repo.Create(ctx, nil, t); err != nil {
		return nil, err
	}
	return s.toView(t), nil
}

// EnsureDefault is the bootstrap helper. Idempotent + converges namespace
// to the configured value if a previous bootstrap row drifted (e.g. after
// changing helm values like compute.bootstrap.defaultTenantNamespace). The
// `spec.namespace.name` immutability rule applies to API-driven Updates;
// bootstrap is the single writer of the system default tenant and may
// overwrite. The compute reconciler then patches the Tenant CR via spec_sync.
func (s *Service) EnsureDefault(ctx context.Context, name, namespace string) (*Tenant, error) {
	if existing, err := s.repo.GetActiveByName(ctx, name); err == nil {
		if existing.Namespace == namespace {
			return existing, nil
		}
		snap := SpecSnapshot{
			DisplayName: "Default tenant",
			Namespace:   tenantv1alpha1.NamespaceSpec{Name: namespace},
		}
		specJSON, err := EncodeSpec(snap)
		if err != nil {
			return nil, err
		}
		hash, err := spechash.Compute(snap)
		if err != nil {
			return nil, err
		}
		if err := s.repo.Update(ctx, nil, existing.ID, map[string]any{
			"namespace":         namespace,
			"spec":              datatypes.JSON(specJSON),
			"desired_spec_hash": hash,
			"message":           "",
		}); err != nil {
			return nil, err
		}
		existing.Namespace = namespace
		existing.Spec = datatypes.JSON(specJSON)
		existing.DesiredSpecHash = hash
		return existing, nil
	} else if !IsNotFound(err) {
		return nil, err
	}
	snap := SpecSnapshot{
		DisplayName: "Default tenant",
		Namespace:   tenantv1alpha1.NamespaceSpec{Name: namespace},
	}
	specJSON, err := EncodeSpec(snap)
	if err != nil {
		return nil, err
	}
	hash, err := spechash.Compute(snap)
	if err != nil {
		return nil, err
	}
	t := &Tenant{
		ID:              uuid.New(),
		Name:            name,
		Namespace:       namespace,
		DisplayName:     "Default tenant",
		Spec:            datatypes.JSON(specJSON),
		DesiredSpecHash: hash,
		Status:          string(StatusCreating),
		Annotations:     datatypes.JSON([]byte(`{}`)),
	}
	if err := s.repo.Create(ctx, nil, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Get returns the tenant view by name.
func (s *Service) Get(ctx context.Context, name string) (*View, error) {
	t, err := s.repo.GetActiveByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "tenant not found")
		}
		return nil, err
	}
	return s.toView(t), nil
}

// GetByID returns the view by tenant UUID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*View, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "tenant not found")
		}
		return nil, err
	}
	return s.toView(t), nil
}

// GetID resolves a tenant name to UUID. Used by middleware.
func (s *Service) GetID(ctx context.Context, name string) (uuid.UUID, error) {
	t, err := s.repo.GetActiveByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return uuid.Nil, apperrors.New(apperrors.CodeNotFound, "tenant not found")
		}
		return uuid.Nil, err
	}
	return t.ID, nil
}

// List returns paginated tenants.
func (s *Service) List(ctx context.Context, limit, offset int) ([]View, int64, error) {
	rows, total, err := s.repo.ListActive(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, 0, len(rows))
	for i := range rows {
		out = append(out, *s.toView(&rows[i]))
	}
	return out, total, nil
}

// Update applies a partial patch and re-hashes the snapshot.
func (s *Service) Update(ctx context.Context, name string, in UpdateInput) (*View, error) {
	t, err := s.repo.GetActiveByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "tenant not found")
		}
		return nil, err
	}
	snap, err := DecodeSpec(t.Spec)
	if err != nil {
		return nil, err
	}
	if in.DisplayName != nil {
		snap.DisplayName = *in.DisplayName
		t.DisplayName = *in.DisplayName
	}
	if in.Annotations != nil {
		snap.Annotations = *in.Annotations
	}
	if in.NamespaceMeta != nil {
		if in.NamespaceMeta.Labels != nil {
			snap.Namespace.Labels = in.NamespaceMeta.Labels
		}
		if in.NamespaceMeta.Annotations != nil {
			snap.Namespace.Annotations = in.NamespaceMeta.Annotations
		}
	}
	if in.InitResources != nil {
		snap.InitResources = *in.InitResources
	}
	return s.persistSnapshot(ctx, t, snap)
}

// Suspend flips spec.suspended=true. Reconciler will patch the CR.
func (s *Service) Suspend(ctx context.Context, name string) (*View, error) {
	return s.toggleSuspend(ctx, name, true)
}

// Unsuspend flips spec.suspended=false.
func (s *Service) Unsuspend(ctx context.Context, name string) (*View, error) {
	return s.toggleSuspend(ctx, name, false)
}

func (s *Service) toggleSuspend(ctx context.Context, name string, suspended bool) (*View, error) {
	t, err := s.repo.GetActiveByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "tenant not found")
		}
		return nil, err
	}
	snap, err := DecodeSpec(t.Spec)
	if err != nil {
		return nil, err
	}
	snap.Suspended = suspended
	if suspended {
		t.Status = string(StatusSuspended)
	}
	return s.persistSnapshot(ctx, t, snap)
}

// Delete moves the tenant to `Deleting` and cascades quota soft-delete.
func (s *Service) Delete(ctx context.Context, name string) error {
	t, err := s.repo.GetActiveByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.MarkDeleting(ctx, tx, t.ID); err != nil {
			return err
		}
		if s.quotas != nil {
			if err := s.quotas.SoftDeleteAllByTenant(ctx, tx, t.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// markSpecDirty is invoked by the quota service inside its write transaction.
// We re-render the snapshot (excluding quotas which are joined at patch-time)
// to refresh the desired hash so the reconciler picks up the change.
func (s *Service) markSpecDirty(tx *gorm.DB, tenantID uuid.UUID) error {
	var t Tenant
	if err := tx.Where("id = ?", tenantID).First(&t).Error; err != nil {
		return err
	}
	snap, err := DecodeSpec(t.Spec)
	if err != nil {
		return err
	}
	// Render quotas inline so the desired hash includes them.
	rendered, err := RenderQuotas(context.Background(), tx, s.quotas, s.pools, tenantID)
	if err != nil {
		return err
	}
	snap.Quotas = rendered
	hash, err := spechash.Compute(snap)
	if err != nil {
		return err
	}
	specJSON, err := EncodeSpec(snap)
	if err != nil {
		return err
	}
	return tx.Model(&Tenant{}).Where("id = ?", tenantID).Updates(map[string]any{
		"spec":              datatypes.JSON(specJSON),
		"desired_spec_hash": hash,
	}).Error
}

func (s *Service) persistSnapshot(ctx context.Context, t *Tenant, snap SpecSnapshot) (*View, error) {
	rendered, err := RenderQuotas(ctx, nil, s.quotas, s.pools, t.ID)
	if err != nil {
		return nil, err
	}
	snap.Quotas = rendered
	hash, err := spechash.Compute(snap)
	if err != nil {
		return nil, err
	}
	specJSON, err := EncodeSpec(snap)
	if err != nil {
		return nil, err
	}
	annJSON, err := json.Marshal(orEmptyMap(snap.Annotations))
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"spec":              datatypes.JSON(specJSON),
		"desired_spec_hash": hash,
		"display_name":      snap.DisplayName,
		"annotations":       datatypes.JSON(annJSON),
	}
	if t.Status == string(StatusSuspended) {
		updates["status"] = t.Status
	}
	if err := s.repo.Update(ctx, nil, t.ID, updates); err != nil {
		return nil, err
	}
	t, err = s.repo.Get(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	return s.toView(t), nil
}

func (s *Service) toView(t *Tenant) *View {
	v := &View{
		ID:          t.ID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		Namespace:   tenantv1alpha1.NamespaceSpec{Name: t.Namespace},
		Status:      t.Status,
		Message:     t.Message,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if len(t.Spec) > 0 {
		var snap SpecSnapshot
		if err := json.Unmarshal(t.Spec, &snap); err == nil {
			v.Annotations = snap.Annotations
			if snap.Namespace.Name != "" {
				v.Namespace = snap.Namespace
			}
		}
	}
	return v
}

func orEmptyMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return map[K]V{}
	}
	return m
}
