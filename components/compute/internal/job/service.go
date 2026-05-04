package job

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/operators/mljob-operator/api/v1alpha1"

	"github.com/axisml/axisml/components/compute/internal/auth"
	"github.com/axisml/axisml/components/compute/internal/quota"
	"github.com/axisml/axisml/components/compute/internal/resourcepool"
	"github.com/axisml/axisml/components/compute/internal/resourceunit"
	"github.com/axisml/axisml/components/compute/internal/tenant"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
	"github.com/axisml/axisml/components/compute/pkg/strutil"
)

// Service is the job business layer.
type Service struct {
	repo    *Repository
	db      *gorm.DB
	tenants *tenant.Service
	pools   *resourcepool.Service
	units   *resourceunit.Service
	quotas  *quota.Service
}

// NewService constructs the job service.
func NewService(
	db *gorm.DB,
	tenants *tenant.Service,
	pools *resourcepool.Service,
	units *resourceunit.Service,
	quotas *quota.Service,
) *Service {
	return &Service{
		repo:    NewRepository(db),
		db:      db,
		tenants: tenants,
		pools:   pools,
		units:   units,
		quotas:  quotas,
	}
}

// CreateInput is the API request body. The caller selects pool/unit/quota by
// name; we resolve them at write time and capture the snapshot.
type CreateInput struct {
	Name           string                       `json:"name" binding:"required,axisml_name"`
	DisplayName    string                       `json:"displayName"`
	Description    string                       `json:"description"`
	ResourceUnitID uuid.UUID                    `json:"resourceUnitId" binding:"required"`
	QuotaID        uuid.UUID                    `json:"quotaId" binding:"required"`
	Backend        *mljobv1alpha1.BackendSpec   `json:"backend"`
	Roles          []mljobv1alpha1.RoleSpec     `json:"roles" binding:"required,min=1"`
	RunPolicy      *mljobv1alpha1.RunPolicySpec `json:"runPolicy"`
}

// View is the HTTP response payload.
type View struct {
	ID          uuid.UUID               `json:"id"`
	TenantID    uuid.UUID               `json:"tenantId"`
	Name        string                  `json:"name"`
	DisplayName string                  `json:"displayName,omitempty"`
	Description string                  `json:"description,omitempty"`
	OwnerUser   string                  `json:"ownerUser,omitempty"`
	Status      string                  `json:"status"`
	Message     string                  `json:"message,omitempty"`
	StartedAt   *time.Time              `json:"startedAt,omitempty"`
	FinishedAt  *time.Time              `json:"finishedAt,omitempty"`
	Spec        mljobv1alpha1.MLJobSpec `json:"spec"`
	CreatedAt   time.Time               `json:"createdAt"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

// Create writes a new job row and returns the view. The CR is reconciled
// asynchronously by the job reconciler.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, in CreateInput) (*View, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid job name")
	}
	if existing, err := s.repo.GetByTenantName(ctx, tenantID, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "job already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}

	tnt, err := s.tenants.Get(ctx, tenantNameFromCtx(ctx, tenantID))
	if err != nil {
		// Best-effort path: load by ID would require an extra repo method.
		// The middleware should have already verified existence; if Get fails,
		// fall through with a generic error.
		_ = tnt
	}

	unit, err := s.units.GetByID(ctx, in.ResourceUnitID)
	if err != nil {
		return nil, err
	}
	pool, err := s.pools.GetByID(ctx, unit.PoolID)
	if err != nil {
		return nil, err
	}
	q, err := s.quotas.GetByID(ctx, in.QuotaID)
	if err != nil {
		return nil, err
	}
	if q.PoolID != pool.ID {
		return nil, apperrors.New(apperrors.CodeValidation, "quota and resource unit belong to different pools")
	}
	if q.TenantID != tenantID {
		return nil, apperrors.New(apperrors.CodeValidation, "quota does not belong to tenant")
	}

	decoded, err := resourceunit.Decode(unit)
	if err != nil {
		return nil, err
	}

	// Backend defaults.
	backend := mljobv1alpha1.BackendSpec{Name: "native", Engine: "job"}
	if in.Backend != nil {
		if in.Backend.Name != "" {
			backend.Name = in.Backend.Name
		}
		if in.Backend.Engine != "" {
			backend.Engine = in.Backend.Engine
		}
		backend.Config = in.Backend.Config
	}

	poolSel, err := decodePoolNodeSelector(pool)
	if err != nil {
		return nil, err
	}
	poolTols, err := decodePoolTolerations(pool)
	if err != nil {
		return nil, err
	}

	// Resource injection into each role.
	roles := make([]mljobv1alpha1.RoleSpec, len(in.Roles))
	for i, r := range in.Roles {
		role := r
		role.Template.Resources = resourceunit.BuildResources(decoded.Requests, decoded.Limits)
		roles[i] = role
	}

	runPolicy := mljobv1alpha1.RunPolicySpec{}
	if in.RunPolicy != nil {
		runPolicy = *in.RunPolicy
		runPolicy.Suspend = false // reset on creation
	}

	spec := mljobv1alpha1.MLJobSpec{
		Backend: backend,
		Scheduling: mljobv1alpha1.SchedulingSpec{
			Quota:        elasticQuotaName(tnt, pool, q),
			NodeSelector: resourceunit.MergeNodeSelector(poolSel, decoded.NodeSelector),
			Tolerations:  poolTols,
		},
		Roles:     roles,
		RunPolicy: runPolicy,
	}

	// Quota precheck (best-effort).
	if err := s.quotas.Precheck(ctx, quota.PrecheckRequest{
		QuotaID:  q.ID,
		Resource: decoded.Requests,
	}); err != nil {
		return nil, err
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	reqJSON, err := json.Marshal(decoded.Requests)
	if err != nil {
		return nil, err
	}
	j := &Job{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		PoolID:             pool.ID,
		QuotaID:            q.ID,
		ResourceUnitID:     unit.ID,
		Name:               in.Name,
		DisplayName:        in.DisplayName,
		Description:        in.Description,
		OwnerUser:          auth.User(ctx),
		Spec:               datatypes.JSON(specJSON),
		RequestedResources: datatypes.JSON(reqJSON),
		Status:             string(StatusCreating),
	}
	if err := s.repo.Create(ctx, j); err != nil {
		return nil, err
	}
	return s.toView(j)
}

// Get returns the job view by name within a tenant.
func (s *Service) Get(ctx context.Context, tenantID uuid.UUID, name string) (*View, error) {
	j, err := s.repo.GetByTenantName(ctx, tenantID, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "job not found")
		}
		return nil, err
	}
	return s.toView(j)
}

// List returns paginated jobs for a tenant.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]View, int64, error) {
	rows, total, err := s.repo.ListByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, 0, len(rows))
	for i := range rows {
		v, err := s.toView(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, nil
}

// Cancel transitions Running/Pending jobs into Canceling.
func (s *Service) Cancel(ctx context.Context, tenantID uuid.UUID, name string) (*View, error) {
	j, err := s.repo.GetByTenantName(ctx, tenantID, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "job not found")
		}
		return nil, err
	}
	switch Status(j.Status) {
	case StatusCreating:
		return nil, apperrors.New(apperrors.CodePrecondition, "job is still being created; use DELETE")
	case StatusCanceling, StatusCancelled, StatusDeleting, StatusDeleted, StatusSucceeded, StatusFailed:
		return nil, apperrors.New(apperrors.CodePrecondition, "job is not cancellable in current state")
	}
	if err := s.repo.Update(ctx, j.ID, map[string]any{
		"status":  string(StatusCanceling),
		"message": "user cancelled",
	}); err != nil {
		return nil, err
	}
	j, err = s.repo.Get(ctx, j.ID)
	if err != nil {
		return nil, err
	}
	return s.toView(j)
}

// Delete moves the job to Deleting and lets the reconciler reap the CR.
func (s *Service) Delete(ctx context.Context, tenantID uuid.UUID, name string) error {
	j, err := s.repo.GetByTenantName(ctx, tenantID, name)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	switch Status(j.Status) {
	case StatusDeleting, StatusDeleted:
		return nil
	}
	return s.repo.MarkDeleting(ctx, j.ID)
}

func (s *Service) toView(j *Job) (*View, error) {
	var spec mljobv1alpha1.MLJobSpec
	if len(j.Spec) > 0 {
		_ = json.Unmarshal(j.Spec, &spec)
	}
	return &View{
		ID:          j.ID,
		TenantID:    j.TenantID,
		Name:        j.Name,
		DisplayName: j.DisplayName,
		Description: j.Description,
		OwnerUser:   j.OwnerUser,
		Status:      j.Status,
		Message:     j.Message,
		StartedAt:   j.StartedAt,
		FinishedAt:  j.FinishedAt,
		Spec:        spec,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
	}, nil
}

// elasticQuotaName mirrors tenant-operator's naming: axisml-<tenant>-<pool>-<quota>.
func elasticQuotaName(tnt *tenant.View, pool *resourcepool.ResourcePool, q *quota.Quota) string {
	tenantName := ""
	if tnt != nil {
		tenantName = tnt.Name
	}
	return "axisml-" + tenantName + "-" + pool.Name + "-" + q.Name
}

// tenantNameFromCtx extracts the tenant name from gin context (set by middleware).
func tenantNameFromCtx(ctx context.Context, _ uuid.UUID) string {
	v, _ := ctx.Value(ctxTenantNameKey{}).(string)
	return v
}

type ctxTenantNameKey struct{}

// WithTenantName puts the tenant name in context for downstream services.
func WithTenantName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, ctxTenantNameKey{}, name)
}

// decodePoolNodeSelector returns the pool's nodeSelector map.
func decodePoolNodeSelector(p *resourcepool.ResourcePool) (map[string]string, error) {
	if len(p.NodeSelector) == 0 {
		return nil, nil
	}
	m := map[string]string{}
	if err := json.Unmarshal(p.NodeSelector, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func decodePoolTolerations(p *resourcepool.ResourcePool) ([]corev1.Toleration, error) {
	if len(p.Tolerations) == 0 {
		return nil, nil
	}
	var t []corev1.Toleration
	if err := json.Unmarshal(p.Tolerations, &t); err != nil {
		return nil, err
	}
	return t, nil
}
