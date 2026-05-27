package job

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	mljobv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/auth"
	"github.com/axisml/axisml/components/compute-service/internal/poolcache"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/components/compute-service/pkg/strutil"
)

// Service is the job business layer. Keyed on bare namespace strings;
// `spec.scheduling.quota` is whatever the caller passes through.
type Service struct {
	repo  *Repository
	db    *gorm.DB
	pools *poolcache.Reader
}

// NewService constructs the job service.
func NewService(db *gorm.DB, pools *poolcache.Reader) *Service {
	return &Service{
		repo:  NewRepository(db),
		db:    db,
		pools: pools,
	}
}

// CreateInput is the API request body. Caller selects pool/unit by NAME
// (the ResourcePool CRD lives in K8s; compute reads it via Informer cache).
// `Quota` is the ElasticQuota CR name (cluster-unique string) stamped onto
// Pod labels — compute treats it as opaque.
type CreateInput struct {
	Name          string                       `json:"name" binding:"required,axisml_name"`
	DisplayName   string                       `json:"displayName"`
	Description   string                       `json:"description"`
	Labels        map[string]string            `json:"labels,omitempty"`
	Annotations   map[string]string            `json:"annotations,omitempty"`
	PoolName      string                       `json:"poolName" binding:"required"`
	UnitName      string                       `json:"unitName" binding:"required"`
	Quota         string                       `json:"quota" binding:"required"`
	PriorityClass string                       `json:"priorityClass,omitempty"`
	Backend       *mljobv1alpha1.BackendSpec   `json:"backend"`
	Roles         []mljobv1alpha1.RoleSpec     `json:"roles" binding:"required,min=1"`
	RunPolicy     *mljobv1alpha1.RunPolicySpec `json:"runPolicy"`
}

// View is the HTTP response payload.
type View struct {
	ID          uuid.UUID               `json:"id"`
	Namespace   string                  `json:"namespace"`
	Name        string                  `json:"name"`
	PoolName    string                  `json:"poolName,omitempty"`
	UnitName    string                  `json:"unitName,omitempty"`
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

	expanded, err := s.pools.Resolve(ctx, in.PoolName, in.UnitName)
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

	roles := make([]mljobv1alpha1.RoleSpec, len(in.Roles))
	for i, r := range in.Roles {
		role := r
		role.Template.Resources = poolcache.BuildResources(expanded.Requests, expanded.Limits)
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
			Quota:         in.Quota,
			PriorityClass: in.PriorityClass,
			NodeSelector:  expanded.NodeSelector,
			Tolerations:   expanded.Tolerations,
		},
		Roles:     roles,
		RunPolicy: runPolicy,
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	reqJSON, err := json.Marshal(expanded.Requests)
	if err != nil {
		return nil, err
	}
	j := &Job{
		ID:                 uuid.New(),
		Namespace:          namespace,
		PoolName:           in.PoolName,
		UnitName:           in.UnitName,
		Name:               in.Name,
		DisplayName:        in.DisplayName,
		Description:        in.Description,
		OwnerUser:          auth.User(ctx),
		Labels:             mapBytes(in.Labels),
		Annotations:        mapBytes(in.Annotations),
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

func (s *Service) List(ctx context.Context, namespace string, limit, offset int, labelClause string, labelArgs []any) ([]View, int64, error) {
	rows, total, err := s.repo.ListByNamespace(ctx, namespace, limit, offset, labelClause, labelArgs)
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
	j.Status = string(StatusCanceling)
	j.Message = "user cancelled"
	j.UpdatedAt = time.Now().UTC()
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

func mapBytes(m map[string]string) datatypes.JSON {
	if m == nil {
		m = map[string]string{}
	}
	b, _ := json.Marshal(m)
	return b
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
		PoolName:    j.PoolName,
		UnitName:    j.UnitName,
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
