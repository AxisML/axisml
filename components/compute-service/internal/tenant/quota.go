package tenant

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

// QuotaPatchInput is the body for PATCH/POST on a quota. POST requires
// pool / name; PATCH treats them as path params and lets min/max change.
type QuotaPatchInput struct {
	Pool string            `json:"pool,omitempty"`
	Name string            `json:"name,omitempty"`
	Min  map[string]string `json:"min,omitempty"`
	Max  map[string]string `json:"max,omitempty"`
}

// ListQuotas returns the tenant's spec.quotas[] (no separate table).
func (s *Service) ListQuotas(ctx context.Context, tenantName string) ([]QuotaSpec, error) {
	t, err := s.repo.GetByName(ctx, tenantName)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.Newf(apperrors.CodeNotFound, "tenant %q not found", tenantName)
		}
		return nil, apperrors.Wrap(apperrors.CodeInternal, "load tenant", err)
	}
	spec, err := unmarshalSpec(t.Spec)
	if err != nil {
		return nil, err
	}
	return spec.Quotas, nil
}

// AddQuota appends a quota to spec.quotas[]. The (pool, name) tuple must
// be unique within the tenant.
func (s *Service) AddQuota(ctx context.Context, tenantName string, in QuotaPatchInput, lastModifiedBy string) (QuotaSpec, error) {
	if in.Pool == "" || in.Name == "" {
		return QuotaSpec{}, apperrors.New(apperrors.CodeValidation, "pool and name are required")
	}
	if len(in.Max) == 0 {
		return QuotaSpec{}, apperrors.New(apperrors.CodeValidation, "max is required")
	}

	q := QuotaSpec(in)
	if err := s.mutateQuotas(ctx, tenantName, lastModifiedBy, func(qs []QuotaSpec) ([]QuotaSpec, error) {
		for _, existing := range qs {
			if existing.Pool == q.Pool && existing.Name == q.Name {
				return nil, apperrors.Newf(apperrors.CodeConflict,
					"quota (pool=%s,name=%s) already exists", q.Pool, q.Name)
			}
		}
		return append(qs, q), nil
	}); err != nil {
		return QuotaSpec{}, err
	}
	return q, nil
}

// PatchQuota updates min/max on a (pool, name) entry. (pool, name) is
// immutable; changes require delete-then-add.
func (s *Service) PatchQuota(ctx context.Context, tenantName, pool, name string, in QuotaPatchInput, lastModifiedBy string) (QuotaSpec, error) {
	var out QuotaSpec
	err := s.mutateQuotas(ctx, tenantName, lastModifiedBy, func(qs []QuotaSpec) ([]QuotaSpec, error) {
		idx := -1
		for i, existing := range qs {
			if existing.Pool == pool && existing.Name == name {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, apperrors.Newf(apperrors.CodeNotFound,
				"quota (pool=%s,name=%s) not found", pool, name)
		}
		if in.Min != nil {
			qs[idx].Min = in.Min
		}
		if in.Max != nil {
			qs[idx].Max = in.Max
		}
		out = qs[idx]
		return qs, nil
	})
	return out, err
}

// DeleteQuota removes a (pool, name) entry; idempotent.
func (s *Service) DeleteQuota(ctx context.Context, tenantName, pool, name, lastModifiedBy string) error {
	return s.mutateQuotas(ctx, tenantName, lastModifiedBy, func(qs []QuotaSpec) ([]QuotaSpec, error) {
		out := qs[:0]
		for _, existing := range qs {
			if existing.Pool == pool && existing.Name == name {
				continue
			}
			out = append(out, existing)
		}
		return out, nil
	})
}

// mutateQuotas loads the tenant's spec.quotas[], runs the supplied
// transform, marshals it back, and bumps generation so the reconciler
// notices.
func (s *Service) mutateQuotas(ctx context.Context, tenantName string, lastModifiedBy string, mutate func([]QuotaSpec) ([]QuotaSpec, error)) error {
	t, err := s.repo.GetByName(ctx, tenantName)
	if err != nil {
		if IsNotFound(err) {
			return apperrors.Newf(apperrors.CodeNotFound, "tenant %q not found", tenantName)
		}
		return apperrors.Wrap(apperrors.CodeInternal, "load tenant", err)
	}
	spec, err := unmarshalSpec(t.Spec)
	if err != nil {
		return err
	}
	updated, err := mutate(spec.Quotas)
	if err != nil {
		return err
	}
	spec.Quotas = updated
	b, err := json.Marshal(spec)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternal, "marshal spec", err)
	}
	updates := map[string]any{
		"spec":             b,
		"generation":       gorm.Expr("generation + 1"),
		"last_modified_by": lastModifiedBy,
	}
	if err := s.repo.Update(ctx, t.ID, updates); err != nil {
		return apperrors.Wrap(apperrors.CodeInternal, "update tenant spec", err)
	}
	return nil
}

func unmarshalSpec(raw []byte) (SpecJSON, error) {
	var spec SpecJSON
	if len(raw) == 0 {
		return spec, nil
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return spec, apperrors.Wrap(apperrors.CodeInternal, "unmarshal tenant spec", err)
	}
	return spec, nil
}
