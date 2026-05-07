package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/components/compute/internal/auth"
	"github.com/axisml/axisml/components/compute/internal/resourcepool"
	"github.com/axisml/axisml/components/compute/internal/resourceunit"
	"github.com/axisml/axisml/components/compute/internal/spechash"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
	"github.com/axisml/axisml/components/compute/pkg/strutil"
)

// Module wraps the service business layer. Keyed on bare namespace strings.
type Module struct {
	repo  *Repository
	db    *gorm.DB
	pools *resourcepool.Service
	units *resourceunit.Service
}

// NewService builds the service module wiring.
func NewService(
	db *gorm.DB,
	pools *resourcepool.Service,
	units *resourceunit.Service,
) *Module {
	return &Module{
		repo:  NewRepository(db),
		db:    db,
		pools: pools,
		units: units,
	}
}

// CreateInput is the API request body. `Quota` is the ElasticQuota CR name
// (cluster-unique string) Compute stamps onto Pod labels — opaque, no
// existence check.
type CreateInput struct {
	Name           string                       `json:"name" binding:"required,axisml_name"`
	DisplayName    string                       `json:"displayName"`
	Description    string                       `json:"description"`
	ResourceUnitID uuid.UUID                    `json:"resourceUnitId" binding:"required"`
	Quota          string                       `json:"quota" binding:"required"`
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
	Namespace     string                          `json:"namespace"`
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

func (m *Module) Create(ctx context.Context, namespace string, in CreateInput) (*View, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid service name")
	}
	if in.Quota == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "quota is required")
	}
	if existing, err := m.repo.GetByNamespaceName(ctx, namespace, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "service already exists")
	} else if err != nil && !IsNotFound(err) {
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

	poolSel, err := pool.DecodeNodeSelector()
	if err != nil {
		return nil, err
	}
	poolTols, err := pool.DecodeTolerations()
	if err != nil {
		return nil, err
	}

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
			Quota:        in.Quota,
			NodeSelector: resourceunit.MergeNodeSelector(poolSel, decoded.NodeSelector),
			Tolerations:  poolTols,
		},
		ModelRef:  in.ModelRef,
		Roles:     roles,
		RunPolicy: runPolicy,
		Route:     in.Route,
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
		Namespace:          namespace,
		PoolID:             pool.ID,
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

func (m *Module) Get(ctx context.Context, namespace, name string) (*View, error) {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "service not found")
		}
		return nil, err
	}
	return m.toView(row)
}

func (m *Module) List(ctx context.Context, namespace string, limit, offset int) ([]View, int64, error) {
	rows, total, err := m.repo.ListByNamespace(ctx, namespace, limit, offset)
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

func (m *Module) Scale(ctx context.Context, namespace, name string, in ScaleInput) (*View, error) {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
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
	row.Spec = datatypes.JSON(specJSON)
	row.DesiredSpecHash = hash
	row.Replicas = in.Replicas
	row.UpdatedAt = time.Now().UTC()
	return m.toView(row)
}

func (m *Module) Delete(ctx context.Context, namespace, name string) error {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
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
		Namespace:     s.Namespace,
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
