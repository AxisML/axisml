package tenant

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

// Service implements the tenant business logic on top of Repository.
type Service struct{ repo *Repository }

func NewService(db *gorm.DB) *Service { return &Service{repo: NewRepository(db)} }

// Create inserts a new tenant (phase=Creating, generation=1).
func (s *Service) Create(ctx context.Context, in CreateInput, lastModifiedBy string) (Response, error) {
	if in.Namespace.Name == "" {
		return Response{}, apperrors.New(apperrors.CodeValidation, "spec.namespace.name is required")
	}
	if err := validateQuotas(in.Quotas); err != nil {
		return Response{}, err
	}
	specBytes, err := toSpecBytes(in)
	if err != nil {
		return Response{}, apperrors.Wrap(apperrors.CodeInternal, "marshal spec", err)
	}

	t := &Tenant{
		ID:             uuid.New(),
		Name:           in.Name,
		DisplayName:    in.DisplayName,
		Description:    in.Description,
		Owner:          lastModifiedBy,
		Labels:         mapToJSON(in.Labels),
		Annotations:    mapToJSON(in.Annotations),
		Spec:           specBytes,
		Generation:     1,
		Phase:          PhaseCreating,
		Status:         []byte("{}"),
		LastModifiedBy: lastModifiedBy,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		if isUniqueViolation(err) {
			return Response{}, apperrors.Newf(apperrors.CodeConflict,
				"tenant %q already exists", in.Name)
		}
		return Response{}, apperrors.Wrap(apperrors.CodeInternal, "create tenant", err)
	}
	return toResponse(t)
}

// Get fetches one tenant by name.
func (s *Service) Get(ctx context.Context, name string) (Response, error) {
	t, err := s.repo.GetByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return Response{}, apperrors.Newf(apperrors.CodeNotFound, "tenant %q not found", name)
		}
		return Response{}, apperrors.Wrap(apperrors.CodeInternal, "get tenant", err)
	}
	return toResponse(t)
}

// List returns all active tenants, optionally filtered by labelSelector
// (K8s grammar; see server.JSONLabelsSQL for SQL projection).
func (s *Service) List(ctx context.Context, limit, offset int, labelClause string, labelArgs []any) (ListResponse, error) {
	rows, total, err := s.repo.List(ctx, limit, offset, labelClause, labelArgs)
	if err != nil {
		return ListResponse{}, apperrors.Wrap(apperrors.CodeInternal, "list tenants", err)
	}
	out := ListResponse{Items: make([]Response, 0, len(rows)), Total: total}
	for i := range rows {
		r, err := toResponse(&rows[i])
		if err != nil {
			return ListResponse{}, apperrors.Wrap(apperrors.CodeInternal, "marshal tenant", err)
		}
		out.Items = append(out.Items, r)
	}
	return out, nil
}

// Patch updates the mutable spec fields (quotas / initResources) + display
// fields. Bumps generation for spec mutations only.
func (s *Service) Patch(ctx context.Context, name string, in PatchInput, lastModifiedBy string) (Response, error) {
	t, err := s.repo.GetByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return Response{}, apperrors.Newf(apperrors.CodeNotFound, "tenant %q not found", name)
		}
		return Response{}, apperrors.Wrap(apperrors.CodeInternal, "load tenant", err)
	}

	updates := map[string]any{"last_modified_by": lastModifiedBy}
	specMutated := false

	if in.DisplayName != nil {
		updates["display_name"] = *in.DisplayName
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.Labels != nil {
		updates["labels"] = mapToJSON(in.Labels)
	}
	if in.Annotations != nil {
		updates["annotations"] = mapToJSON(in.Annotations)
	}

	if in.Quotas != nil || in.InitResources != nil {
		var spec SpecJSON
		if err := json.Unmarshal(t.Spec, &spec); err != nil {
			return Response{}, apperrors.Wrap(apperrors.CodeInternal, "unmarshal spec", err)
		}
		if in.Quotas != nil {
			if err := validateQuotas(*in.Quotas); err != nil {
				return Response{}, err
			}
			// (pool, name) is the immutable identity anchor per design §4.1
			// / §4.2: rename quota = delete-then-add via the sub-routes.
			// PATCH at this top level must NOT silently rewrite an entry's
			// identity tuple. Reject if any incoming (pool, name) doesn't
			// already exist on the tenant; min/max changes are allowed.
			if err := guardQuotaIdentity(spec.Quotas, *in.Quotas); err != nil {
				return Response{}, err
			}
			spec.Quotas = *in.Quotas
		}
		if in.InitResources != nil {
			spec.InitResources = in.InitResources
		}
		b, err := json.Marshal(spec)
		if err != nil {
			return Response{}, apperrors.Wrap(apperrors.CodeInternal, "marshal spec", err)
		}
		updates["spec"] = b
		updates["generation"] = gorm.Expr("generation + 1")
		specMutated = true
	}
	_ = specMutated

	if err := s.repo.Update(ctx, t.ID, updates); err != nil {
		return Response{}, apperrors.Wrap(apperrors.CodeInternal, "update tenant", err)
	}
	return s.Get(ctx, name)
}

// Restore reverses a soft-delete on a tenant. Per design §4.1: only rows
// in phase=Deleted (i.e. the operator has already torn down the CR) are
// eligible; the row's deleted_at is cleared, phase flips back to
// Creating, generation bumps so the reconciler re-creates the CR.
func (s *Service) Restore(ctx context.Context, name string, lastModifiedBy string) (Response, error) {
	var t Tenant
	if err := s.repo.db.WithContext(ctx).Unscoped().
		Where("name = ?", name).Order("created_at DESC").First(&t).Error; err != nil {
		if IsNotFound(err) {
			return Response{}, apperrors.Newf(apperrors.CodeNotFound, "tenant %q not found", name)
		}
		return Response{}, apperrors.Wrap(apperrors.CodeInternal, "load tenant", err)
	}
	if t.Phase != PhaseDeleted {
		return Response{}, apperrors.Newf(apperrors.CodePrecondition,
			"tenant %q phase=%s; only Deleted tenants can be restored", name, t.Phase)
	}
	if err := s.repo.db.WithContext(ctx).Unscoped().Model(&Tenant{}).
		Where("id = ?", t.ID).
		Updates(map[string]any{
			"phase":            PhaseCreating,
			"deleted_at":       gorm.Expr("NULL"),
			"generation":       gorm.Expr("generation + 1"),
			"last_modified_by": lastModifiedBy,
		}).Error; err != nil {
		return Response{}, apperrors.Wrap(apperrors.CodeInternal, "restore tenant", err)
	}
	return s.Get(ctx, name)
}

// GetIncludingDeleted returns the tenant row even if soft-deleted (used by
// the DELETE handler so it can return the tombstone).
func (s *Service) GetIncludingDeleted(ctx context.Context, name string) (Response, error) {
	t, err := s.repo.FindByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return Response{}, apperrors.Newf(apperrors.CodeNotFound, "tenant %q not found", name)
		}
		return Response{}, apperrors.Wrap(apperrors.CodeInternal, "load tenant", err)
	}
	return toResponse(t)
}

// Delete soft-deletes the tenant.
func (s *Service) Delete(ctx context.Context, name string) error {
	t, err := s.repo.GetByName(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return nil // idempotent
		}
		return apperrors.Wrap(apperrors.CodeInternal, "load tenant", err)
	}
	if err := s.repo.SoftDelete(ctx, t.ID); err != nil {
		return apperrors.Wrap(apperrors.CodeInternal, "soft delete tenant", err)
	}
	return nil
}

// guardQuotaIdentity refuses a PATCH whose incoming quotas[] adds a tuple
// that doesn't already exist on the row OR drops a tuple that does. Both
// are identity mutations and the design routes them through the quota sub-
// resource endpoints (POST / DELETE), not top-level PATCH. Reorderings,
// min/max edits on existing tuples — fine.
func guardQuotaIdentity(existing []QuotaSpec, incoming []QuotaSpec) error {
	have := map[string]struct{}{}
	for _, q := range existing {
		have[q.Pool+"/"+q.Name] = struct{}{}
	}
	want := map[string]struct{}{}
	for _, q := range incoming {
		want[q.Pool+"/"+q.Name] = struct{}{}
	}
	for k := range want {
		if _, ok := have[k]; !ok {
			return apperrors.Newf(apperrors.CodeValidation,
				"quota %s does not exist on this tenant; use POST /quotas to add", k)
		}
	}
	for k := range have {
		if _, ok := want[k]; !ok {
			return apperrors.Newf(apperrors.CodeValidation,
				"quota %s missing from PATCH body; use DELETE /quotas/{pool}/{name} to remove", k)
		}
	}
	return nil
}

func validateQuotas(qs []QuotaSpec) error {
	seen := map[string]struct{}{}
	for _, q := range qs {
		if q.Pool == "" || q.Name == "" {
			return apperrors.New(apperrors.CodeValidation, "quotas[].pool and quotas[].name are required")
		}
		k := q.Pool + "/" + q.Name
		if _, ok := seen[k]; ok {
			return apperrors.Newf(apperrors.CodeValidation, "duplicate quota (pool=%s,name=%s)", q.Pool, q.Name)
		}
		seen[k] = struct{}{}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
