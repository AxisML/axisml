package artifact

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/axisml/axisml/components/artifact-hub/internal/artifact/handler"
	"github.com/axisml/axisml/components/artifact-hub/internal/config"
	"github.com/axisml/axisml/components/artifact-hub/internal/dbjson"
	apperrors "github.com/axisml/axisml/components/artifact-hub/pkg/errors"
	"github.com/axisml/axisml/components/artifact-hub/pkg/strutil"
)

// InitiateInput is the API request body for the two-phase write step 1.
type InitiateInput struct {
	Version     string            `json:"version" binding:"required,axisml_version"`
	Spec        map[string]any    `json:"spec" binding:"required"`
	Visibility  string            `json:"visibility,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PatchInput is the body for PATCH .../{kindPlural}/{name}/{version}.
// Only the four "displayable" fields are mutable post-Ready (design §6).
type PatchInput struct {
	DisplayName *string           `json:"displayName,omitempty"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// UploadCredentials wraps the storage-backend credential + storage URI
// returned alongside the artifact row on Initiate (design yaml).
type UploadCredentials struct {
	StorageKind string              `json:"storageKind"`
	URI         string              `json:"uri"`
	Credentials handler.Credentials `json:"credentials"`
	ExpiresAt   time.Time           `json:"expiresAt"`
}

// InitiateResult is what we return to the cli after step 1. Per design
// yaml, it bundles the newly persisted Artifact view with the upload
// credentials so the caller has a one-stop reply.
type InitiateResult struct {
	Artifact View              `json:"artifact"`
	Upload   UploadCredentials `json:"upload"`
}

// CompleteInput is the API request body for the two-phase write step 2.
type CompleteInput struct {
	Digest string `json:"digest" binding:"required"`
}

// ResolveResult is what we return on /resolve. CamelCase per design yaml;
// `visibility` is the artifact's persisted visibility (tenant|public) so
// callers don't need a second GET.
type ResolveResult struct {
	StorageKind     string               `json:"storageKind"`
	URI             string               `json:"uri"`
	Digest          string               `json:"digest,omitempty"`
	Visibility      string               `json:"visibility,omitempty"`
	PullCredentials *handler.Credentials `json:"pullCredentials,omitempty"`
	ExpiresAt       *time.Time           `json:"expiresAt,omitempty"`
}

// Service holds Artifact CRUD + state-machine logic. Rows are addressed
// by (namespace, kind, name, version) directly — there's no parent
// ArtifactRepo.
type Service struct {
	cfg  config.Config
	rows *Repository
}

// NewService constructs a Service.
func NewService(cfg config.Config, db *gorm.DB) *Service {
	return &Service{cfg: cfg, rows: NewRepository(db)}
}

// Initiate is two-phase write step 1. Validates, inserts an Uploading row,
// and asks the Kind handler to mint upload credentials.
func (s *Service) Initiate(ctx context.Context, namespace, kind, name, ownerUser string, in InitiateInput) (*InitiateResult, error) {
	if !strutil.IsValidVersion(in.Version) {
		return nil, apperrors.New(apperrors.CodeValidation, "version does not satisfy OCI tag-safe charset (A-Za-z0-9_.-) or length 1..128")
	}

	h, ok := handler.Get(kind)
	if !ok {
		return nil, apperrors.Newf(apperrors.CodeValidation, "kind %q has no registered handler", kind)
	}

	if err := h.ValidateSpec(ctx, in.Spec); err != nil {
		return nil, err
	}

	visibility := in.Visibility
	if visibility == "" {
		visibility = VisibilityTenant
	}
	if visibility != VisibilityTenant && visibility != VisibilityPublic {
		return nil, apperrors.Newf(apperrors.CodeValidation,
			"visibility must be %q or %q", VisibilityTenant, VisibilityPublic)
	}
	if visibility == VisibilityPublic && namespace != PublicVisibilityNamespace {
		return nil, apperrors.Newf(apperrors.CodeForbidden,
			"visibility=%q is only allowed under the %q namespace",
			VisibilityPublic, PublicVisibilityNamespace)
	}

	// Idempotency: artifact (namespace, kind, name, version) never recycles
	// (database.md §1.2). Check the deleted_at-inclusive lookup so a
	// tombstone is treated as occupying the coord — clients bump version
	// instead of replaying the same tuple.
	existing, err := s.rows.GetByCoordIncludingDeleted(ctx, namespace, kind, name, in.Version)
	if err != nil && !IsNotFound(err) {
		return nil, apperrors.Wrap(apperrors.CodeInternal, "lookup artifact", err)
	}
	if existing != nil {
		return nil, apperrors.Newf(apperrors.CodeConflict,
			"version %s already exists for %s/%s/%s (status=%s)", in.Version, namespace, kind, name, existing.Status)
	}

	specJSON, err := json.Marshal(in.Spec)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "marshal spec", err)
	}
	labels, _ := dbjson.MapToJSON(in.Labels)
	annotations, _ := dbjson.MapToJSON(in.Annotations)

	row := &Artifact{
		ID:          uuid.New(),
		Namespace:   namespace,
		Kind:        kind,
		Name:        name,
		Version:     in.Version,
		Visibility:  visibility,
		DisplayName: in.DisplayName,
		Description: in.Description,
		Labels:      labels,
		Annotations: annotations,
		OwnerUser:   ownerUser,
		Spec:        datatypes.JSON(specJSON),
		Status:      StatusUploading,
	}

	if err := s.rows.Create(ctx, nil, row); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternal, "insert artifact", err)
	}

	hArt := handler.Artifact{
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
		Version:   in.Version,
		Spec:      in.Spec,
	}

	creds, err := h.InitiateUpload(ctx, hArt, s.cfg.UploadTokenTTL)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeUnavailable, "issue upload credentials", err)
	}

	return &InitiateResult{
		Artifact: toView(row),
		Upload: UploadCredentials{
			StorageKind: string(h.StorageKind()),
			URI:         h.BuildStorageURI(namespace, name, in.Version),
			Credentials: creds,
			ExpiresAt:   creds.ExpiresAt,
		},
	}, nil
}

// Complete is two-phase write step 2.
func (s *Service) Complete(ctx context.Context, namespace, kind, name, version string, in CompleteInput) (*Artifact, error) {
	row, err := s.loadRow(ctx, namespace, kind, name, version)
	if err != nil {
		return nil, err
	}

	switch row.Status {
	case StatusUploading:
	case StatusReady:
		if row.Digest == in.Digest {
			return row, nil
		}
		return nil, apperrors.Newf(apperrors.CodeConflict,
			"digest mismatch: artifact already Ready with digest %s", row.Digest)
	default:
		return nil, apperrors.Newf(apperrors.CodePrecondition,
			"artifact is %s, cannot complete", row.Status)
	}

	h, ok := handler.Get(kind)
	if !ok {
		return nil, apperrors.Newf(apperrors.CodeValidation, "kind %q has no registered handler", kind)
	}

	hArt := handler.Artifact{Kind: kind, Namespace: namespace, Name: name, Version: version}
	digest, err := h.VerifyComplete(ctx, hArt, handler.CompleteClaim{Digest: in.Digest})
	if err != nil {
		if e, ok := apperrors.As(err); ok && (e.Code == apperrors.CodeConflict || e.Code == apperrors.CodePrecondition || e.Code == apperrors.CodeValidation) {
			_ = s.rows.Update(ctx, nil, row.ID, map[string]any{
				"status":  StatusFailed,
				"message": e.Message,
			})
		}
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.rows.Update(ctx, nil, row.ID, map[string]any{
		"status":   StatusReady,
		"digest":   digest,
		"ready_at": now,
		"message":  "",
	}); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternal, "mark ready", err)
	}
	row.Status = StatusReady
	row.Digest = digest
	row.ReadyAt = &now
	row.Message = ""
	row.UpdatedAt = now
	return row, nil
}

// Get returns a single artifact by coord.
func (s *Service) Get(ctx context.Context, namespace, kind, name, version string) (*Artifact, error) {
	return s.loadRow(ctx, namespace, kind, name, version)
}

// List returns artifacts under (namespace, kind, name). Pass an empty `name`
// to list every artifact name+version under (namespace, kind). labelClause
// and labelArgs come from server.JSONLabelsSQL applied to the ?labelSelector
// query — empty for no filtering.
func (s *Service) List(ctx context.Context, namespace, kind, name, status string, limit, offset int, labelClause string, labelArgs []any) ([]Artifact, int64, error) {
	rows, total, err := s.rows.ListByCoord(ctx, namespace, kind, name, status, limit, offset, labelClause, labelArgs)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternal, "list artifacts", err)
	}
	return rows, total, nil
}

// Resolve returns storage coordinates and (optionally) pull credentials.
func (s *Service) Resolve(ctx context.Context, namespace, kind, name, version, usage string) (*ResolveResult, error) {
	row, err := s.loadRow(ctx, namespace, kind, name, version)
	if err != nil {
		return nil, err
	}

	switch row.Status {
	case StatusReady:
	case StatusUploading, StatusFailed:
		return nil, apperrors.Newf(apperrors.CodePrecondition, "artifact is %s, not Ready", row.Status)
	case StatusDeleting, StatusDeleted:
		return nil, apperrors.New(apperrors.CodeGone, "artifact has been deleted")
	}

	h, ok := handler.Get(kind)
	if !ok {
		return nil, apperrors.Newf(apperrors.CodeValidation, "kind %q has no registered handler", kind)
	}

	uri := h.BuildStorageURI(namespace, name, version)
	res := &ResolveResult{
		StorageKind: string(h.StorageKind()),
		URI:         uri,
		Digest:      row.Digest,
		Visibility:  row.Visibility,
	}

	if usage == "" {
		usage = "inspect"
	}
	switch usage {
	case "inspect":
	case "download":
		hArt := handler.Artifact{Kind: kind, Namespace: namespace, Name: name, Version: version, Digest: row.Digest}
		creds, err := h.IssuePullCredentials(ctx, hArt, s.cfg.UploadTokenTTL)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeUnavailable, "issue pull credentials", err)
		}
		res.PullCredentials = &creds
		expires := creds.ExpiresAt
		res.ExpiresAt = &expires
	default:
		return nil, apperrors.Newf(apperrors.CodeValidation, "usage must be inspect or download (got %q)", usage)
	}
	return res, nil
}

// MarkDeleting transitions the artifact to Deleting.
// Patch updates the four mutable display-tier fields on an artifact row
// (display_name / description / labels / annotations). Per design §6:
//   - spec / digest / visibility / kind / name / version / namespace are
//     immutable;
//   - rows in Deleting / Deleted return 409.
//   - labels / annotations follow whole-map replace semantics; if a key
//     is absent from the patch, the column is left as-is unless the
//     payload supplies a non-nil map (then it replaces).
func (s *Service) Patch(ctx context.Context, namespace, kind, name, version string, in PatchInput) (*Artifact, error) {
	row, err := s.loadRow(ctx, namespace, kind, name, version)
	if err != nil {
		return nil, err
	}
	if row.Status == StatusDeleting || row.Status == StatusDeleted {
		return nil, apperrors.Newf(apperrors.CodeConflict,
			"artifact is %s; patch rejected", row.Status)
	}
	updates := map[string]any{}
	if in.DisplayName != nil {
		updates["display_name"] = *in.DisplayName
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.Labels != nil {
		labels, _ := dbjson.MapToJSON(in.Labels)
		updates["labels"] = labels
	}
	if in.Annotations != nil {
		anns, _ := dbjson.MapToJSON(in.Annotations)
		updates["annotations"] = anns
	}
	if len(updates) == 0 {
		return row, nil
	}
	if err := s.rows.Update(ctx, nil, row.ID, updates); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternal, "update artifact", err)
	}
	return s.loadRow(ctx, namespace, kind, name, version)
}

func (s *Service) MarkDeleting(ctx context.Context, namespace, kind, name, version string) error {
	row, err := s.loadRow(ctx, namespace, kind, name, version)
	if err != nil {
		return err
	}
	switch row.Status {
	case StatusDeleting, StatusDeleted:
		return nil
	}
	return s.rows.Update(ctx, nil, row.ID, map[string]any{
		"status":  StatusDeleting,
		"message": "",
	})
}

// loadRow returns the artifact row by coord, or NotFound. Read paths use
// this so a client can still observe a row's terminal Deleted status after
// GC has finalised it (Initiate keeps using the deleted_at-filtered
// GetByCoord so a tombstone doesn't block a re-create on the same coord).
func (s *Service) loadRow(ctx context.Context, namespace, kind, name, version string) (*Artifact, error) {
	row, err := s.rows.GetByCoordIncludingDeleted(ctx, namespace, kind, name, version)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.Newf(apperrors.CodeNotFound, "artifact %s/%s/%s@%s not found", namespace, kind, name, version)
		}
		return nil, apperrors.Wrap(apperrors.CodeInternal, "lookup artifact", err)
	}
	return row, nil
}
