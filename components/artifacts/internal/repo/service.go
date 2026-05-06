package repo

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/axisml/axisml/components/artifacts/internal/dbjson"
	apperrors "github.com/axisml/axisml/components/artifacts/pkg/errors"
	"github.com/axisml/axisml/components/artifacts/pkg/strutil"
)

// CreateInput is the API-layer request body for creating a repo.
type CreateInput struct {
	Kind        string            `json:"kind" binding:"required"`
	Name        string            `json:"name" binding:"required,axisml_name"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// UpdateInput is the API-layer request body for PATCH. Only mutable fields.
type UpdateInput struct {
	DisplayName *string            `json:"display_name,omitempty"`
	Description *string            `json:"description,omitempty"`
	Labels      *map[string]string `json:"labels,omitempty"`
	Annotations *map[string]string `json:"annotations,omitempty"`
}

// Service holds repo CRUD logic.
type Service struct {
	repos *Repository
}

// NewService constructs a Service.
func NewService(db *gorm.DB) *Service {
	return &Service{repos: NewRepository(db)}
}

// SupportedKinds reports which kinds are accepted at the API surface. MVP
// only ships `model`; the other kinds are reserved by design but rejected
// here until their handlers land.
func SupportedKinds() []string {
	return []string{KindModel}
}

func isSupportedKind(kind string) bool {
	for _, k := range SupportedKinds() {
		if k == kind {
			return true
		}
	}
	return false
}

// Create persists a new repo for the given tenant.
func (s *Service) Create(ctx context.Context, tenant, ownerUser string, in CreateInput) (*Repo, error) {
	in.Kind = strings.TrimSpace(in.Kind)
	in.Name = strings.TrimSpace(in.Name)
	if !isSupportedKind(in.Kind) {
		return nil, apperrors.Newf(apperrors.CodeValidation, "unsupported kind %q (MVP supports: %v)", in.Kind, SupportedKinds())
	}
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "name does not satisfy AxisML name policy")
	}

	labels, err := dbjson.MapToJSON(in.Labels)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "labels", err)
	}
	annotations, err := dbjson.MapToJSON(in.Annotations)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "annotations", err)
	}

	tn := tenant
	row := &Repo{
		ID:          uuid.New(),
		TenantName:  &tn,
		Kind:        in.Kind,
		Name:        in.Name,
		DisplayName: in.DisplayName,
		Description: in.Description,
		Labels:      labels,
		Annotations: annotations,
		OwnerUser:   ownerUser,
		Status:      StatusActive,
	}

	if err := s.repos.Create(ctx, nil, row); err != nil {
		if dbjson.IsUniqueViolation(err) {
			return nil, apperrors.Newf(apperrors.CodeConflict,
				"repo %s/%s already exists in tenant %s", in.Kind, in.Name, tenant)
		}
		return nil, apperrors.Wrap(apperrors.CodeInternal, "create repo", err)
	}
	return row, nil
}

// Get returns a single repo by tenant + name.
func (s *Service) Get(ctx context.Context, tenant, kind, name string) (*Repo, error) {
	row, err := s.repos.GetByTenantKindName(ctx, tenant, kind, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.Newf(apperrors.CodeNotFound, "repo %s/%s not found in tenant %s", kind, name, tenant)
		}
		return nil, apperrors.Wrap(apperrors.CodeInternal, "get repo", err)
	}
	return row, nil
}

// List returns active repos for the tenant. Optional kind filter ("" = all).
func (s *Service) List(ctx context.Context, tenant, kind string, limit, offset int) ([]Repo, int64, error) {
	rows, total, err := s.repos.ListByTenant(ctx, tenant, kind, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternal, "list repos", err)
	}
	return rows, total, nil
}

// Update applies a PATCH; only mutable fields are honored.
func (s *Service) Update(ctx context.Context, tenant, kind, name string, in UpdateInput) (*Repo, error) {
	row, err := s.Get(ctx, tenant, kind, name)
	if err != nil {
		return nil, err
	}
	if row.Status != StatusActive {
		return nil, apperrors.Newf(apperrors.CodePrecondition, "repo is %s, not Active", row.Status)
	}

	fields := map[string]any{}
	if in.DisplayName != nil {
		fields["display_name"] = *in.DisplayName
	}
	if in.Description != nil {
		fields["description"] = *in.Description
	}
	if in.Labels != nil {
		j, err := dbjson.MapToJSON(*in.Labels)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeValidation, "labels", err)
		}
		fields["labels"] = j
	}
	if in.Annotations != nil {
		j, err := dbjson.MapToJSON(*in.Annotations)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeValidation, "annotations", err)
		}
		fields["annotations"] = j
	}

	if len(fields) == 0 {
		return row, nil
	}
	if err := s.repos.Update(ctx, nil, row.ID, fields); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternal, "update repo", err)
	}
	return s.repos.GetByID(ctx, row.ID)
}

// MarkDeleting transitions the repo to Deleting. The cascade fanout to
// child artifacts is the GC worker's job (design §3.4).
func (s *Service) MarkDeleting(ctx context.Context, tenant, kind, name string) error {
	row, err := s.Get(ctx, tenant, kind, name)
	if err != nil {
		return err
	}
	if row.Status == StatusDeleting || row.Status == StatusDeleted {
		return nil
	}
	if err := s.repos.MarkDeleting(ctx, nil, row.ID); err != nil {
		return apperrors.Wrap(apperrors.CodeInternal, "mark deleting", err)
	}
	return nil
}

// SetLatestArtifact updates a repo's latest_artifact_id pointer. Used by
// the artifact service when an artifact transitions to Ready.
func (s *Service) SetLatestArtifact(ctx context.Context, repoID, artifactID uuid.UUID) error {
	return s.repos.Update(ctx, nil, repoID, map[string]any{
		"latest_artifact_id": artifactID,
	})
}
