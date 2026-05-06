package job

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"

	"github.com/axisml/axisml/components/compute/internal/auth"
	"github.com/axisml/axisml/components/compute/internal/resourcepool"
	"github.com/axisml/axisml/components/compute/internal/resourceunit"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
	"github.com/axisml/axisml/components/compute/pkg/strutil"
)

// Service is the job business layer. After de-tenant rewrite the service
// is keyed on bare namespace strings; tenant + quota lookups are gone.
// `spec.scheduling.quota` is whatever the caller passes through.
type Service struct {
	repo  *Repository
	db    *gorm.DB
	pools *resourcepool.Service
	units *resourceunit.Service
}

// NewService constructs the job service.
func NewService(
	db *gorm.DB,
	pools *resourcepool.Service,
	units *resourceunit.Service,
) *Service {
	return &Service{
		repo:  NewRepository(db),
		db:    db,
		pools: pools,
		units: units,
	}
}

// CreateInput is the API request body. Caller selects pool/unit/quota by
// ID; we resolve unit + pool at write time. `Quota` is the ElasticQuota CR
// name (cluster-unique string) that gets stamped onto Pod labels — Compute
// treats it as opaque and does no existence check.
type CreateInput struct {
	Name           string                       `json:"name" binding:"required,axisml_name"`
	DisplayName    string                       `json:"displayName"`
	Description    string                       `json:"description"`
	ResourceUnitID uuid.UUID                    `json:"resourceUnitId" binding:"required"`
	Quota          string                       `json:"quota" binding:"required"`
	Backend        *mljobv1alpha1.BackendSpec   `json:"backend"`
	Roles          []mljobv1alpha1.RoleSpec     `json:"roles" binding:"required,min=1"`
	RunPolicy      *mljobv1alpha1.RunPolicySpec `json:"runPolicy"`
}

// View is the HTTP response payload.
type View struct {
	ID          uuid.UUID               `json:"id"`
	Namespace   string                  `json:"namespace"`
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

// Create writes a new job row. CR is reconciled asynchronously.
func (s *Service) Create(ctx context.Context, namespace string, in CreateInput) (*View, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid job name")
	}
	if in.Quota == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "quota is required")
	}
	if existing, err := s.repo.GetByNamespaceName(ctx, namespace, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "job already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}

	unit, err := s.units.GetByID(ctx, in.ResourceUnitID)
	if err != nil {
		return nil, err
	}
	pool, err := s.pools.GetByID(ctx, unit.PoolID)
	if err != nil {
		return nil, err
	}

	decoded, err := resourceunit.Decode(unit)
	if err != nil {
		return nil, err
	}

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

	roles := make([]mljobv1alpha1.RoleSpec, len(in.Roles))
	for i, r := range in.Roles {
		role := r
		role.Template.Resources = resourceunit.BuildResources(decoded.Requests, decoded.Limits)
		roles[i] = role
	}

	runPolicy := mljobv1alpha1.RunPolicySpec{}
	if in.RunPolicy != nil {
		runPolicy = *in.RunPolicy
		runPolicy.Suspend = false
	}

	spec := mljobv1alpha1.MLJobSpec{
		Backend: backend,
		Scheduling: mljobv1alpha1.SchedulingSpec{
			Quota:        in.Quota,
			NodeSelector: resourceunit.MergeNodeSelector(poolSel, decoded.NodeSelector),
			Tolerations:  poolTols,
		},
		Roles:     roles,
		RunPolicy: runPolicy,
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
		Namespace:          namespace,
		PoolID:             pool.ID,
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

func (s *Service) Get(ctx context.Context, namespace, name string) (*View, error) {
	j, err := s.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "job not found")
		}
		return nil, err
	}
	return s.toView(j)
}

func (s *Service) List(ctx context.Context, namespace string, limit, offset int) ([]View, int64, error) {
	rows, total, err := s.repo.ListByNamespace(ctx, namespace, limit, offset)
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

func (s *Service) Cancel(ctx context.Context, namespace, name string) (*View, error) {
	j, err := s.repo.GetByNamespaceName(ctx, namespace, name)
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

func (s *Service) Delete(ctx context.Context, namespace, name string) error {
	j, err := s.repo.GetByNamespaceName(ctx, namespace, name)
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
		Namespace:   j.Namespace,
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
