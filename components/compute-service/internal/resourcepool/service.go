package resourcepool

import (
	"context"
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/components/compute-service/pkg/strutil"
)

// Service is the resourcepool business layer.
type Service struct {
	repo *Repository
	db   *gorm.DB
}

// NewService constructs the service.
func NewService(db *gorm.DB) *Service { return &Service{repo: NewRepository(db), db: db} }

// CreateInput is the API-layer request body.
type CreateInput struct {
	Name         string              `json:"name" binding:"required,axisml_name"`
	Description  string              `json:"description"`
	NodeSelector map[string]string   `json:"nodeSelector"`
	Tolerations  []corev1.Toleration `json:"tolerations"`
	Metadata     map[string]any      `json:"metadata"`
}

// UpdateInput patches mutable fields. Name is immutable.
type UpdateInput struct {
	Description  *string              `json:"description"`
	NodeSelector *map[string]string   `json:"nodeSelector"`
	Tolerations  *[]corev1.Toleration `json:"tolerations"`
	Metadata     *map[string]any      `json:"metadata"`
}

// View is the API-layer response.
type View struct {
	ID           uuid.UUID           `json:"id"`
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
	Metadata     map[string]any      `json:"metadata,omitempty"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*View, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid pool name")
	}
	if existing, err := s.repo.GetByName(ctx, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "resource pool already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}

	ns, err := jsonOrEmpty(in.NodeSelector)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "encode nodeSelector", err)
	}
	tols, err := jsonOrEmptyArray(in.Tolerations)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "encode tolerations", err)
	}
	meta, err := jsonOrEmpty(in.Metadata)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "encode metadata", err)
	}
	pool := &ResourcePool{
		ID:           uuid.New(),
		Name:         in.Name,
		Description:  in.Description,
		NodeSelector: ns,
		Tolerations:  tols,
		Metadata:     meta,
	}
	if err := s.repo.Create(ctx, pool); err != nil {
		return nil, err
	}
	return toView(pool)
}

// EnsureDefault is the bootstrap helper. Idempotent.
func (s *Service) EnsureDefault(ctx context.Context, name string) (*ResourcePool, error) {
	existing, err := s.repo.GetByName(ctx, name)
	if err == nil {
		return existing, nil
	}
	if !IsNotFound(err) {
		return nil, err
	}
	pool := &ResourcePool{
		ID:           uuid.New(),
		Name:         name,
		Description:  "Default resource pool",
		NodeSelector: datatypes.JSON([]byte(`{}`)),
		Tolerations:  datatypes.JSON([]byte(`[]`)),
		Metadata:     datatypes.JSON([]byte(`{}`)),
	}
	if err := s.repo.Create(ctx, pool); err != nil {
		return nil, err
	}
	return pool, nil
}

func (s *Service) Get(ctx context.Context, name string) (*View, error) {
	pool, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, translateNotFound(err, "resource pool")
	}
	return toView(pool)
}

// GetByID is used by other modules (quota / job / service) for FK validation.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ResourcePool, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, translateNotFound(err, "resource pool")
	}
	return p, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]View, int64, error) {
	pools, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, 0, len(pools))
	for i := range pools {
		v, err := toView(&pools[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, nil
}

func (s *Service) Update(ctx context.Context, name string, in UpdateInput) (*View, error) {
	pool, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, translateNotFound(err, "resource pool")
	}
	updates := map[string]any{}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.NodeSelector != nil {
		ns, err := jsonOrEmpty(*in.NodeSelector)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeValidation, "encode nodeSelector", err)
		}
		updates["node_selector"] = ns
	}
	if in.Tolerations != nil {
		t, err := jsonOrEmptyArray(*in.Tolerations)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeValidation, "encode tolerations", err)
		}
		updates["tolerations"] = t
	}
	if in.Metadata != nil {
		m, err := jsonOrEmpty(*in.Metadata)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeValidation, "encode metadata", err)
		}
		updates["metadata"] = m
	}
	if len(updates) == 0 {
		return toView(pool)
	}
	if err := s.repo.Update(ctx, pool.ID, updates); err != nil {
		return nil, err
	}
	pool, err = s.repo.Get(ctx, pool.ID)
	if err != nil {
		return nil, err
	}
	return toView(pool)
}

func (s *Service) Delete(ctx context.Context, name string) error {
	pool, err := s.repo.GetByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	return s.repo.SoftDelete(ctx, pool.ID)
}

func toView(p *ResourcePool) (*View, error) {
	v := &View{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if len(p.NodeSelector) > 0 {
		ns := map[string]string{}
		if err := json.Unmarshal(p.NodeSelector, &ns); err != nil {
			return nil, err
		}
		if len(ns) > 0 {
			v.NodeSelector = ns
		}
	}
	if len(p.Tolerations) > 0 {
		var t []corev1.Toleration
		if err := json.Unmarshal(p.Tolerations, &t); err != nil {
			return nil, err
		}
		v.Tolerations = t
	}
	if len(p.Metadata) > 0 {
		m := map[string]any{}
		if err := json.Unmarshal(p.Metadata, &m); err != nil {
			return nil, err
		}
		if len(m) > 0 {
			v.Metadata = m
		}
	}
	return v, nil
}

func jsonOrEmpty(v any) (datatypes.JSON, error) {
	if v == nil {
		return datatypes.JSON([]byte(`{}`)), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

func jsonOrEmptyArray(v any) (datatypes.JSON, error) {
	if v == nil {
		return datatypes.JSON([]byte(`[]`)), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

func translateNotFound(err error, kind string) error {
	if IsNotFound(err) {
		return apperrors.Newf(apperrors.CodeNotFound, "%s not found", kind)
	}
	return err
}

func timeNow() *time.Time {
	t := time.Now().UTC()
	return &t
}
