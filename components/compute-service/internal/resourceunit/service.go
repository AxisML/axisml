package resourceunit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"

	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/components/compute-service/pkg/strutil"
)

type Service struct {
	repo *Repository
	db   *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{repo: NewRepository(db), db: db} }

type CreateInput struct {
	Name         string              `json:"name" binding:"required,axisml_resource_unit"`
	Description  string              `json:"description"`
	Requests     corev1.ResourceList `json:"requests" binding:"required"`
	Limits       corev1.ResourceList `json:"limits"`
	NodeSelector map[string]string   `json:"nodeSelector"`
}

type UpdateInput struct {
	Description  *string              `json:"description"`
	Requests     *corev1.ResourceList `json:"requests"`
	Limits       *corev1.ResourceList `json:"limits"`
	NodeSelector *map[string]string   `json:"nodeSelector"`
}

type View struct {
	ID           uuid.UUID           `json:"id"`
	PoolID       uuid.UUID           `json:"poolId"`
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	Requests     corev1.ResourceList `json:"requests"`
	Limits       corev1.ResourceList `json:"limits,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
}

func (s *Service) Create(ctx context.Context, poolID uuid.UUID, in CreateInput) (*View, error) {
	if !strutil.IsValidResourceUnitName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid resource unit name")
	}
	if existing, err := s.repo.GetByPoolName(ctx, poolID, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "resource unit already exists in pool")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}
	if len(in.Requests) == 0 {
		return nil, apperrors.New(apperrors.CodeValidation, "requests must not be empty")
	}
	req, err := jsonResources(in.Requests)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "encode requests", err)
	}
	lim, err := jsonResources(in.Limits)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "encode limits", err)
	}
	ns, err := jsonOrEmpty(in.NodeSelector)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "encode nodeSelector", err)
	}
	u := &ResourceUnit{
		ID:           uuid.New(),
		PoolID:       poolID,
		Name:         in.Name,
		Description:  in.Description,
		Requests:     req,
		Limits:       lim,
		NodeSelector: ns,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return toView(u)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ResourceUnit, error) {
	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, translateNotFound(err)
	}
	return u, nil
}

func (s *Service) Get(ctx context.Context, poolID uuid.UUID, name string) (*View, error) {
	u, err := s.repo.GetByPoolName(ctx, poolID, name)
	if err != nil {
		return nil, translateNotFound(err)
	}
	return toView(u)
}

func (s *Service) ListByPool(ctx context.Context, poolID uuid.UUID, limit, offset int) ([]View, int64, error) {
	units, total, err := s.repo.ListByPool(ctx, poolID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, 0, len(units))
	for i := range units {
		v, err := toView(&units[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, nil
}

func (s *Service) Update(ctx context.Context, poolID uuid.UUID, name string, in UpdateInput) (*View, error) {
	u, err := s.repo.GetByPoolName(ctx, poolID, name)
	if err != nil {
		return nil, translateNotFound(err)
	}
	updates := map[string]any{}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.Requests != nil {
		j, err := jsonResources(*in.Requests)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeValidation, "encode requests", err)
		}
		updates["requests"] = j
	}
	if in.Limits != nil {
		j, err := jsonResources(*in.Limits)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeValidation, "encode limits", err)
		}
		updates["limits"] = j
	}
	if in.NodeSelector != nil {
		j, err := jsonOrEmpty(*in.NodeSelector)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeValidation, "encode nodeSelector", err)
		}
		updates["node_selector"] = j
	}
	if len(updates) == 0 {
		return toView(u)
	}
	if err := s.repo.Update(ctx, u.ID, updates); err != nil {
		return nil, err
	}
	u, err = s.repo.Get(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return toView(u)
}

func (s *Service) Delete(ctx context.Context, poolID uuid.UUID, name string) error {
	u, err := s.repo.GetByPoolName(ctx, poolID, name)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	return s.repo.SoftDelete(ctx, u.ID)
}

// Decoded turns a stored row into the strongly typed values needed by the
// scheduling-injection helper.
type Decoded struct {
	Requests     corev1.ResourceList
	Limits       corev1.ResourceList
	NodeSelector map[string]string
}

func Decode(u *ResourceUnit) (*Decoded, error) {
	d := &Decoded{}
	if len(u.Requests) > 0 {
		if err := json.Unmarshal(u.Requests, &d.Requests); err != nil {
			return nil, err
		}
	}
	if len(u.Limits) > 0 {
		if err := json.Unmarshal(u.Limits, &d.Limits); err != nil {
			return nil, err
		}
	}
	if len(u.NodeSelector) > 0 {
		if err := json.Unmarshal(u.NodeSelector, &d.NodeSelector); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func toView(u *ResourceUnit) (*View, error) {
	d, err := Decode(u)
	if err != nil {
		return nil, err
	}
	return &View{
		ID:           u.ID,
		PoolID:       u.PoolID,
		Name:         u.Name,
		Description:  u.Description,
		Requests:     d.Requests,
		Limits:       d.Limits,
		NodeSelector: d.NodeSelector,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}, nil
}

func jsonResources(rl corev1.ResourceList) (datatypes.JSON, error) {
	if rl == nil {
		return datatypes.JSON([]byte(`{}`)), nil
	}
	b, err := json.Marshal(rl)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
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

func translateNotFound(err error) error {
	if IsNotFound(err) {
		return apperrors.New(apperrors.CodeNotFound, "resource unit not found")
	}
	return err
}

func timeNow() *time.Time {
	t := time.Now().UTC()
	return &t
}
