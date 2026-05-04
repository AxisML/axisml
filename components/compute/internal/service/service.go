package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	mlservicev1alpha1 "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/components/compute/internal/auth"
	"github.com/axisml/axisml/components/compute/internal/quota"
	"github.com/axisml/axisml/components/compute/internal/resourcepool"
	"github.com/axisml/axisml/components/compute/internal/resourceunit"
	"github.com/axisml/axisml/components/compute/internal/spechash"
	"github.com/axisml/axisml/components/compute/internal/tenant"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
	"github.com/axisml/axisml/components/compute/pkg/strutil"
)

// Module wraps the service business layer. We name the type Module rather
// than Service to avoid shadowing the package's data model type.
type Module struct {
	repo    *Repository
	db      *gorm.DB
	tenants *tenant.Service
	pools   *resourcepool.Service
	units   *resourceunit.Service
	quotas  *quota.Service
}

// NewService builds the service module wiring.
func NewService(
	db *gorm.DB,
	tenants *tenant.Service,
	pools *resourcepool.Service,
	units *resourceunit.Service,
	quotas *quota.Service,
) *Module {
	return &Module{
		repo:    NewRepository(db),
		db:      db,
		tenants: tenants,
		pools:   pools,
		units:   units,
		quotas:  quotas,
	}
}

// CreateInput is the API request body.
type CreateInput struct {
	Name           string                       `json:"name" binding:"required,axisml_name"`
	DisplayName    string                       `json:"displayName"`
	Description    string                       `json:"description"`
	ResourceUnitID uuid.UUID                    `json:"resourceUnitId" binding:"required"`
	QuotaID        uuid.UUID                    `json:"quotaId" binding:"required"`
	Backend        *mlservicev1alpha1.Backend   `json:"backend"`
	ModelRef       mlservicev1alpha1.ModelRef   `json:"modelRef" binding:"required"`
	Roles          []mlservicev1alpha1.RoleSpec `json:"roles" binding:"required,min=1"`
	RunPolicy      *mlservicev1alpha1.RunPolicy `json:"runPolicy"`
	Route          *mlservicev1alpha1.Route     `json:"route"`
}

// ScaleInput is the body for /:scale.
type ScaleInput struct {
	Replicas int32 `json:"replicas" binding:"required,gte=0"`
}

// View is the HTTP response.
type View struct {
	ID            uuid.UUID                       `json:"id"`
	TenantID      uuid.UUID                       `json:"tenantId"`
	Name          string                          `json:"name"`
	DisplayName   string                          `json:"displayName,omitempty"`
	Description   string                          `json:"description,omitempty"`
	OwnerUser     string                          `json:"ownerUser,omitempty"`
	Replicas      int32                           `json:"replicas"`
	ReadyReplicas int32                           `json:"readyReplicas"`
	Endpoint      string                          `json:"endpoint,omitempty"`
	Status        string                          `json:"status"`
	Message       string                          `json:"message,omitempty"`
	Spec          mlservicev1alpha1.MLServiceSpec `json:"spec"`
	CreatedAt     time.Time                       `json:"createdAt"`
	UpdatedAt     time.Time                       `json:"updatedAt"`
}

func (m *Module) Create(ctx context.Context, tenantID uuid.UUID, in CreateInput) (*View, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid service name")
	}
	if existing, err := m.repo.GetByTenantName(ctx, tenantID, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "service already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}
	tnt, err := m.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	unit, err := m.units.GetByID(ctx, in.ResourceUnitID)
	if err != nil {
		return nil, err
	}
	pool, err := m.pools.GetByID(ctx, unit.PoolID)
	if err != nil {
		return nil, err
	}
	q, err := m.quotas.GetByID(ctx, in.QuotaID)
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

	backend := mlservicev1alpha1.Backend{Name: "native", Engine: "deployment"}
	if in.Backend != nil {
		if in.Backend.Name != "" {
			backend.Name = in.Backend.Name
		}
		if in.Backend.Engine != "" {
			backend.Engine = in.Backend.Engine
		}
		backend.Config = in.Backend.Config
	}

	poolSel := decodePoolSelector(pool)
	poolTols := decodePoolTolerations(pool)

	// Inject resources into each role.
	roles := make([]mlservicev1alpha1.RoleSpec, len(in.Roles))
	replicas := int32(0)
	for i, r := range in.Roles {
		role := r
		role.Template.Resources = resourceunit.BuildResources(decoded.Requests, decoded.Limits)
		roles[i] = role
		if i == 0 {
			replicas = role.Replicas
		}
	}

	runPolicy := mlservicev1alpha1.RunPolicy{}
	if in.RunPolicy != nil {
		runPolicy = *in.RunPolicy
	}

	spec := mlservicev1alpha1.MLServiceSpec{
		Backend: backend,
		Scheduling: mlservicev1alpha1.Scheduling{
			Quota:        elasticQuotaName(tnt, pool, q),
			NodeSelector: resourceunit.MergeNodeSelector(poolSel, decoded.NodeSelector),
			Tolerations:  poolTols,
		},
		ModelRef:  in.ModelRef,
		Roles:     roles,
		RunPolicy: runPolicy,
		Route:     in.Route,
	}

	if err := m.quotas.Precheck(ctx, quota.PrecheckRequest{
		QuotaID:  q.ID,
		Resource: scaledRequests(decoded.Requests, replicas),
	}); err != nil {
		return nil, err
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	hash, err := spechash.Compute(spec)
	if err != nil {
		return nil, err
	}
	reqJSON, err := json.Marshal(decoded.Requests)
	if err != nil {
		return nil, err
	}

	row := &Service{
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
		DesiredSpecHash:    hash,
		RequestedResources: datatypes.JSON(reqJSON),
		Replicas:           replicas,
		Status:             string(StatusCreating),
	}
	if err := m.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	return m.toView(row)
}

func (m *Module) Get(ctx context.Context, tenantID uuid.UUID, name string) (*View, error) {
	row, err := m.repo.GetByTenantName(ctx, tenantID, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "service not found")
		}
		return nil, err
	}
	return m.toView(row)
}

func (m *Module) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]View, int64, error) {
	rows, total, err := m.repo.ListByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, 0, len(rows))
	for i := range rows {
		v, err := m.toView(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, nil
}

func (m *Module) Scale(ctx context.Context, tenantID uuid.UUID, name string, in ScaleInput) (*View, error) {
	row, err := m.repo.GetByTenantName(ctx, tenantID, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "service not found")
		}
		return nil, err
	}
	var spec mlservicev1alpha1.MLServiceSpec
	if err := json.Unmarshal(row.Spec, &spec); err != nil {
		return nil, err
	}
	if len(spec.Roles) == 0 {
		return nil, apperrors.New(apperrors.CodePrecondition, "service has no roles")
	}
	spec.Roles[0].Replicas = in.Replicas

	hash, err := spechash.Compute(spec)
	if err != nil {
		return nil, err
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	if err := m.repo.Update(ctx, row.ID, map[string]any{
		"spec":              datatypes.JSON(specJSON),
		"desired_spec_hash": hash,
		"replicas":          in.Replicas,
	}); err != nil {
		return nil, err
	}
	row, err = m.repo.Get(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return m.toView(row)
}

func (m *Module) Delete(ctx context.Context, tenantID uuid.UUID, name string) error {
	row, err := m.repo.GetByTenantName(ctx, tenantID, name)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	switch Status(row.Status) {
	case StatusDeleting, StatusDeleted:
		return nil
	}
	return m.repo.MarkDeleting(ctx, row.ID)
}

func (m *Module) toView(s *Service) (*View, error) {
	var spec mlservicev1alpha1.MLServiceSpec
	if len(s.Spec) > 0 {
		_ = json.Unmarshal(s.Spec, &spec)
	}
	return &View{
		ID:            s.ID,
		TenantID:      s.TenantID,
		Name:          s.Name,
		DisplayName:   s.DisplayName,
		Description:   s.Description,
		OwnerUser:     s.OwnerUser,
		Replicas:      s.Replicas,
		ReadyReplicas: s.ReadyReplicas,
		Endpoint:      s.Endpoint,
		Status:        s.Status,
		Message:       s.Message,
		Spec:          spec,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}, nil
}

// elasticQuotaName mirrors tenant-operator naming.
func elasticQuotaName(tnt *tenant.View, pool *resourcepool.ResourcePool, q *quota.Quota) string {
	tenantName := ""
	if tnt != nil {
		tenantName = tnt.Name
	}
	return "axisml-" + tenantName + "-" + pool.Name + "-" + q.Name
}

func decodePoolSelector(p *resourcepool.ResourcePool) map[string]string {
	if len(p.NodeSelector) == 0 {
		return nil
	}
	m := map[string]string{}
	if err := json.Unmarshal(p.NodeSelector, &m); err != nil {
		return nil
	}
	return m
}

func decodePoolTolerations(p *resourcepool.ResourcePool) []corev1.Toleration {
	if len(p.Tolerations) == 0 {
		return nil
	}
	var t []corev1.Toleration
	if err := json.Unmarshal(p.Tolerations, &t); err != nil {
		return nil
	}
	return t
}

func scaledRequests(req corev1.ResourceList, replicas int32) corev1.ResourceList {
	if replicas <= 0 || len(req) == 0 {
		return nil
	}
	out := corev1.ResourceList{}
	for k, v := range req {
		// Preserve precision via MilliValue (0.001 unit) so fractional CPU
		// requests scale correctly. Format mirrors the canonical resource
		// shape used elsewhere ("100m" / "1Mi" etc.).
		scaled := resource.NewMilliQuantity(v.MilliValue()*int64(replicas), v.Format)
		out[k] = *scaled
	}
	return out
}
