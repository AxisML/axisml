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
	repomod "github.com/axisml/axisml/components/artifacts/internal/repo"
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

// Service holds Artifact CRUD + state-machine logic.
type Service struct {
	cfg   config.Config
	rows  *Repository
	repos *repomod.Service
}

// NewService constructs a Service. The repo service is required so
// initiate/resolve can resolve the parent repo and assemble the OCI scope.
func NewService(cfg config.Config, db *gorm.DB, repos *repomod.Service) *Service {
	return &Service{
		cfg:   cfg,
		rows:  NewRepository(db),
		repos: repos,
	}
}

// Initiate is two-phase write step 1 (design §3.2). Validates, inserts an
// Uploading row, and asks the Kind handler to mint upload credentials.
func (s *Service) Initiate(ctx context.Context, tenant, kind, repoName, ownerUser string, in InitiateInput) (*InitiateResult, error) {
	if !strutil.IsValidVersion(in.Version) {
		return nil, apperrors.New(apperrors.CodeValidation, "version does not satisfy OCI tag-safe charset (A-Za-z0-9_.-) or length 1..128")
	}

	parent, err := s.repos.Get(ctx, tenant, kind, repoName)
	if err != nil {
		return nil, err
	}
	if parent.Status != repomod.StatusActive {
		return nil, apperrors.Newf(apperrors.CodePrecondition, "repo is %s, not Active", parent.Status)
	}

	h, ok := handler.Get(parent.Kind)
	if !ok {
		return nil, apperrors.Newf(apperrors.CodeValidation, "kind %q has no registered handler", parent.Kind)
	}

	if err := h.ValidateSpec(ctx, in.Spec); err != nil {
		return nil, err
	}

	// Idempotency: if (repo_id, version) already exists, refuse with a
	// status-aware message (design §3.2). MVP rejects all states; Phase 2
	// lets Failed retry on the same row.
	existing, err := s.rows.GetByRepoVersion(ctx, parent.ID, in.Version)
	if err != nil && !IsNotFound(err) {
		return nil, apperrors.Wrap(apperrors.CodeInternal, "lookup artifact", err)
	}
	if existing != nil {
		return nil, apperrors.Newf(apperrors.CodeConflict,
			"version %s already exists in this repo (status=%s)", in.Version, existing.Status)
	}

	specJSON, err := json.Marshal(in.Spec)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeValidation, "marshal spec", err)
	}
	labels, _ := dbjson.MapToJSON(in.Labels)
	annotations, _ := dbjson.MapToJSON(in.Annotations)

	row := &Artifact{
		ID:          uuid.New(),
		RepoID:      parent.ID,
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

	scope := parent.Scope()
	hArt := handler.Artifact{
		Kind:     parent.Kind,
		Scope:    scope,
		RepoName: parent.Name,
		Version:  in.Version,
		Spec:     in.Spec,
	}

	creds, err := h.InitiateUpload(ctx, hArt, s.cfg.UploadTokenTTL)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeUnavailable, "issue upload credentials", err)
	}

	return &InitiateResult{
		ArtifactID:        row.ID,
		StorageKind:       string(h.StorageKind()),
		URI:               h.BuildStorageURI(scope, parent.Name, in.Version),
		UploadCredentials: creds,
		ExpiresAt:         creds.ExpiresAt,
	}, nil
}

// Complete is two-phase write step 2 (design §3.2). Verifies the manifest
// at the registry matches the cli-supplied digest, then transitions to Ready.
func (s *Service) Complete(ctx context.Context, tenant, kind, repoName, version string, in CompleteInput) (*Artifact, error) {
	parent, row, err := s.loadPair(ctx, tenant, kind, repoName, version)
	if err != nil {
		return nil, err
	}

	switch row.Status {
	case StatusUploading:
		// proceed
	case StatusReady:
		// design §3.2: digest match → 200; mismatch → 409 DigestMismatch.
		if row.Digest == in.Digest {
			return row, nil
		}
		return nil, apperrors.Newf(apperrors.CodeConflict,
			"digest mismatch: artifact already Ready with digest %s", row.Digest)
	default:
		return nil, apperrors.Newf(apperrors.CodePrecondition,
			"artifact is %s, cannot complete", row.Status)
	}

	h, ok := handler.Get(parent.Kind)
	if !ok {
		return nil, apperrors.Newf(apperrors.CodeValidation, "kind %q has no registered handler", parent.Kind)
	}

	hArt := handler.Artifact{
		Kind:     parent.Kind,
		Scope:    parent.Scope(),
		RepoName: parent.Name,
		Version:  version,
	}
	digest, err := h.VerifyComplete(ctx, hArt, handler.CompleteClaim{Digest: in.Digest})
	if err != nil {
		// Determinate failure (digest mismatch / manifest missing): persist
		// Failed for diagnostics, then surface the error.
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

	// Best-effort latest pointer maintenance. MVP defers full latest tracking
	// to Phase 2 (design §8.3 #8); we set it here when it's still null so the
	// happy path produces a non-empty pointer.
	if parent.LatestArtifactID == nil {
		_ = s.repos.SetLatestArtifact(ctx, parent.ID, row.ID)
	}

	return s.rows.GetByID(ctx, row.ID)
}

// Get returns a single artifact by repo + version.
func (s *Service) Get(ctx context.Context, tenant, kind, repoName, version string) (*Artifact, error) {
	_, row, err := s.loadPair(ctx, tenant, kind, repoName, version)
	return row, err
}

// List returns artifacts under a repo.
func (s *Service) List(ctx context.Context, tenant, kind, repoName, status string, limit, offset int) ([]Artifact, int64, error) {
	parent, err := s.repos.Get(ctx, tenant, kind, repoName)
	if err != nil {
		return nil, 0, err
	}
	rows, total, err := s.rows.ListByRepo(ctx, parent.ID, status, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternal, "list artifacts", err)
	}
	return rows, total, nil
}

// Resolve returns storage coordinates and (optionally) pull credentials.
func (s *Service) Resolve(ctx context.Context, tenant, kind, repoName, version, usage string) (*ResolveResult, error) {
	parent, row, err := s.loadPair(ctx, tenant, kind, repoName, version)
	if err != nil {
		return nil, err
	}

	switch row.Status {
	case StatusReady:
		// proceed
	case StatusUploading, StatusFailed:
		return nil, apperrors.Newf(apperrors.CodePrecondition,
			"artifact is %s, not Ready", row.Status)
	case StatusDeleting, StatusDeleted:
		return nil, apperrors.New(apperrors.CodeGone, "artifact has been deleted")
	}

	h, ok := handler.Get(parent.Kind)
	if !ok {
		return nil, apperrors.Newf(apperrors.CodeValidation, "kind %q has no registered handler", parent.Kind)
	}

	scope := parent.Scope()
	uri := h.BuildStorageURI(scope, parent.Name, version)
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
		// MVP: no auth_hint (design §8.2). Operator uses convention-named
		// imagePullSecret directly.
	case "download":
		hArt := handler.Artifact{
			Kind:     parent.Kind,
			Scope:    scope,
			RepoName: parent.Name,
			Version:  version,
			Digest:   row.Digest,
		}
		creds, err := h.IssuePullCredentials(ctx, hArt, s.cfg.UploadTokenTTL)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeUnavailable, "issue pull credentials", err)
		}
		res.PullCredentials = &creds
		expires := creds.ExpiresAt
		res.ExpiresAt = &expires
	default:
		return nil, apperrors.Newf(apperrors.CodeValidation,
			"usage must be inspect or download (got %q)", usage)
	}
	return res, nil
}

// MarkDeleting transitions the artifact to Deleting; the GC worker drives
// the rest of the lifecycle (design §3.4).
func (s *Service) MarkDeleting(ctx context.Context, tenant, kind, repoName, version string) error {
	_, row, err := s.loadPair(ctx, tenant, kind, repoName, version)
	if err != nil {
		return err
	}
	switch row.Status {
	case StatusDeleting, StatusDeleted:
		return nil
	case StatusReady, StatusFailed, StatusUploading:
		// design §3.5: Ready/Failed transition to Deleting. We also accept
		// Uploading here so DELETE works mid-upload; GC will clean up any
		// partially uploaded blobs.
	}
	return s.rows.Update(ctx, nil, row.ID, map[string]any{
		"status":  StatusDeleting,
		"message": "",
	})
}

// loadPair loads parent repo + artifact row (or returns NotFound).
func (s *Service) loadPair(ctx context.Context, tenant, kind, repoName, version string) (*repomod.Repo, *Artifact, error) {
	parent, err := s.repos.Get(ctx, tenant, kind, repoName)
	if err != nil {
		return nil, nil, err
	}
	row, err := s.rows.GetByRepoVersion(ctx, parent.ID, version)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil, apperrors.Newf(apperrors.CodeNotFound,
				"artifact %s@%s not found", repoName, version)
		}
		return nil, nil, apperrors.Wrap(apperrors.CodeInternal, "lookup artifact", err)
	}
	return parent, row, nil
}
