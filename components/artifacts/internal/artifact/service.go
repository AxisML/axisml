package artifact

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/axisml/axisml/components/artifacts/internal/artifact/handler"
	"github.com/axisml/axisml/components/artifacts/internal/config"
	"github.com/axisml/axisml/components/artifacts/internal/dbjson"
	apperrors "github.com/axisml/axisml/components/artifacts/pkg/errors"
	"github.com/axisml/axisml/components/artifacts/pkg/strutil"
)

// InitiateInput is the API request body for the two-phase write step 1.
type InitiateInput struct {
	Version     string            `json:"version" binding:"required,axisml_version"`
	Spec        map[string]any    `json:"spec" binding:"required"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// InitiateResult is what we return to the cli after step 1.
type InitiateResult struct {
	ArtifactID        uuid.UUID           `json:"artifact_id"`
	StorageKind       string              `json:"storage_kind"`
	URI               string              `json:"uri"`
	UploadCredentials handler.Credentials `json:"upload_credentials"`
	ExpiresAt         time.Time           `json:"expires_at"`
}

// CompleteInput is the API request body for the two-phase write step 2.
type CompleteInput struct {
	Digest string `json:"digest" binding:"required"`
}

// ResolveResult is what we return on /resolve.
type ResolveResult struct {
	StorageKind     string               `json:"storage_kind"`
	URI             string               `json:"uri"`
	Digest          string               `json:"digest,omitempty"`
	PullCredentials *handler.Credentials `json:"pull_credentials,omitempty"`
	ExpiresAt       *time.Time           `json:"expires_at,omitempty"`
}

// Service holds Artifact CRUD + state-machine logic. After the de-repo
// rewrite there's no parent ArtifactRepo lookup; rows are addressed by
// (namespace, kind, name, version) directly.
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

	// Idempotency: if (namespace, kind, name, version) already exists, refuse.
	existing, err := s.rows.GetByCoord(ctx, namespace, kind, name, in.Version)
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
		ArtifactID:        row.ID,
		StorageKind:       string(h.StorageKind()),
		URI:               h.BuildStorageURI(namespace, name, in.Version),
		UploadCredentials: creds,
		ExpiresAt:         creds.ExpiresAt,
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

	return s.rows.GetByID(ctx, row.ID)
}

// Get returns a single artifact by coord.
func (s *Service) Get(ctx context.Context, namespace, kind, name, version string) (*Artifact, error) {
	return s.loadRow(ctx, namespace, kind, name, version)
}

// List returns artifacts under (namespace, kind, name). Pass an empty `name`
// to list every artifact name+version under (namespace, kind).
func (s *Service) List(ctx context.Context, namespace, kind, name, status string, limit, offset int) ([]Artifact, int64, error) {
	rows, total, err := s.rows.ListByCoord(ctx, namespace, kind, name, status, limit, offset)
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

// loadRow returns the artifact row by coord, or NotFound.
func (s *Service) loadRow(ctx context.Context, namespace, kind, name, version string) (*Artifact, error) {
	row, err := s.rows.GetByCoord(ctx, namespace, kind, name, version)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.Newf(apperrors.CodeNotFound, "artifact %s/%s/%s@%s not found", namespace, kind, name, version)
		}
		return nil, apperrors.Wrap(apperrors.CodeInternal, "lookup artifact", err)
	}
	return row, nil
}
