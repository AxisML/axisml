package artifactdef

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"github.com/axisml/axisml/components/platform/internal/clients/artifacthub"
	"github.com/axisml/axisml/components/platform/internal/server"
	"github.com/axisml/axisml/components/platform/internal/store"
	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

// Service holds artifact-definition logic for one kind (model | image).
type Service struct {
	defs         *store.DefinitionRepo
	artifacts    *artifacthub.Client
	kind         string                // "model" | "image"
	defKind      server.DefinitionKind // contract enum
	publicTenant string                // tenant scope hosting public artifacts
}

// NewService constructs an artifact-definition Service.
func NewService(defs *store.DefinitionRepo, artifacts *artifacthub.Client, kind string, publicTenant string) *Service {
	return &Service{defs: defs, artifacts: artifacts, kind: kind, defKind: server.DefinitionKind(kind), publicTenant: publicTenant}
}

// ---- definitions ----

// CreateDef writes a definition.
func (s *Service) CreateDef(ctx context.Context, tenant, name, owner string, in server.ArtifactDefinitionCreateRequest) (*server.ArtifactDefinition, error) {
	d := &store.Definition{
		TenantName:  tenant,
		Name:        name,
		DisplayName: in.DisplayName,
		Description: in.Description,
		OwnerUser:   owner,
		Labels:      store.StrMap(in.Labels),
		Annotations: store.StrMap(in.Annotations),
		Spec:        marshalSpec(in.Spec),
	}
	if err := s.defs.Create(ctx, d); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperrors.New(apperrors.ClassConflict, s.kind+" already exists").WithReason(s.kind + "-exists")
		}
		return nil, apperrors.Wrap(apperrors.ClassInternal, "create "+s.kind, err)
	}
	v := defView(d, s.defKind)
	return &v, nil
}

// GetDef returns a definition.
func (s *Service) GetDef(ctx context.Context, tenant, name string) (*server.ArtifactDefinition, error) {
	d, err := s.getDef(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	v := defView(d, s.defKind)
	return &v, nil
}

// ListDefs lists definitions visible to the caller, merging public ones from the
// built-in tenant.
func (s *Service) ListDefs(ctx context.Context, scope []string, q string, limit, offset int) ([]server.ArtifactDefinition, error) {
	defs, err := s.defs.List(ctx, scope, "", q, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "list "+s.kind, err)
	}
	seen := map[string]bool{}
	out := make([]server.ArtifactDefinition, 0, len(defs))
	for i := range defs {
		seen[defs[i].TenantName+"/"+defs[i].Name] = true
		out = append(out, defView(&defs[i], s.defKind))
	}
	// Merge public definitions from the default tenant (unless already in scope).
	if !contains(scope, s.publicTenant) {
		pub, err := s.defs.List(ctx, []string{s.publicTenant}, "", q, limit, offset)
		if err == nil {
			for i := range pub {
				if seen[pub[i].TenantName+"/"+pub[i].Name] {
					continue
				}
				out = append(out, defView(&pub[i], s.defKind))
			}
		}
	}
	return out, nil
}

// UpdateDef edits definition metadata.
func (s *Service) UpdateDef(ctx context.Context, tenant, name string, in server.ArtifactDefinitionPatchRequest) (*server.ArtifactDefinition, error) {
	d, err := s.getDef(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	if in.DisplayName != "" {
		d.DisplayName = in.DisplayName
	}
	if in.Description != "" {
		d.Description = in.Description
	}
	if in.Labels != nil {
		d.Labels = store.StrMap(in.Labels)
	}
	if in.Annotations != nil {
		d.Annotations = store.StrMap(in.Annotations)
	}
	if in.Spec != nil {
		d.Spec = marshalSpec(in.Spec)
	}
	if err := s.defs.Update(ctx, d); err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "update "+s.kind, err)
	}
	v := defView(d, s.defKind)
	return &v, nil
}

// DeleteDef cascade soft-deletes all versions then the definition (§4.5).
func (s *Service) DeleteDef(ctx context.Context, tenant, name string) error {
	if _, err := s.getDef(ctx, tenant, name); err != nil {
		return err
	}
	if versions, err := s.artifacts.ListVersions(ctx, tenant, s.kind, name); err == nil {
		for i := range versions {
			_ = s.artifacts.Delete(ctx, tenant, s.kind, name, versions[i].Version)
		}
	}
	if err := s.defs.SoftDelete(ctx, tenant, name); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "delete "+s.kind, err)
	}
	return nil
}

// ---- versions ----

// ListVersions lists a definition's versions (live).
func (s *Service) ListVersions(ctx context.Context, tenant, name string) ([]server.Model, error) {
	if _, err := s.getDef(ctx, tenant, name); err != nil {
		return nil, err
	}
	versions, err := s.artifacts.ListVersions(ctx, tenant, s.kind, name)
	if err != nil {
		return nil, err
	}
	out := make([]server.Model, 0, len(versions))
	for i := range versions {
		out = append(out, versionView(&versions[i], tenant))
	}
	return out, nil
}

// InitiateResult bundles the version view with the upload target.
type InitiateResult struct {
	View   server.Model
	Upload map[string]any
}

// InitiateVersion starts a new version. webUpload/oras/dockerPush return push
// credentials; external (sourceURI required) registers a remote artifact with no
// upload (born Ready). Source/external are backed by artifact-hub (push-down #4).
func (s *Service) InitiateVersion(ctx context.Context, tenant, name, version, displayName, description, source, sourceURI string, spec map[string]any) (*InitiateResult, error) {
	if _, err := s.getDef(ctx, tenant, name); err != nil {
		return nil, err
	}
	if source == "external" && sourceURI == "" {
		return nil, apperrors.New(apperrors.ClassValidation, "sourceUri is required when source=external").WithReason("source-uri-required")
	}
	in := artifacthub.InitiateRequest{Version: version, Spec: spec}
	if source != "" {
		in.Source = &source
	}
	if sourceURI != "" {
		in.SourceUri = &sourceURI
	}
	if displayName != "" {
		in.DisplayName = &displayName
	}
	if description != "" {
		in.Description = &description
	}
	res, err := s.artifacts.Initiate(ctx, tenant, s.kind, name, in)
	if err != nil {
		return nil, err
	}
	var upload map[string]any
	b, _ := json.Marshal(res.Upload)
	_ = json.Unmarshal(b, &upload)
	return &InitiateResult{View: versionView(&res.Artifact, tenant), Upload: upload}, nil
}

// CompleteVersion finalizes an upload.
func (s *Service) CompleteVersion(ctx context.Context, tenant, name, version, digest string) (*server.Model, error) {
	if digest == "" {
		return nil, apperrors.New(apperrors.ClassValidation, "digest is required").WithReason("digest-required")
	}
	v, err := s.artifacts.Complete(ctx, tenant, s.kind, name, version, digest)
	if err != nil {
		return nil, err
	}
	out := versionView(v, tenant)
	return &out, nil
}

// GetVersion returns one version.
func (s *Service) GetVersion(ctx context.Context, tenant, name, version string) (*server.Model, error) {
	v, err := s.artifacts.Get(ctx, tenant, s.kind, name, version)
	if err != nil {
		return nil, err
	}
	out := versionView(v, tenant)
	return &out, nil
}

// UpdateVersion patches mutable version metadata.
func (s *Service) UpdateVersion(ctx context.Context, tenant, name, version string, in server.ArtifactUpdateRequest) (*server.Model, error) {
	patch := artifacthub.PatchRequest{}
	if in.DisplayName != "" {
		patch.DisplayName = &in.DisplayName
	}
	if in.Description != "" {
		patch.Description = &in.Description
	}
	if in.Labels != nil {
		m := map[string]string(in.Labels)
		patch.Labels = &m
	}
	if in.Annotations != nil {
		m := map[string]string(in.Annotations)
		patch.Annotations = &m
	}
	v, err := s.artifacts.Update(ctx, tenant, s.kind, name, version, patch)
	if err != nil {
		return nil, err
	}
	out := versionView(v, tenant)
	return &out, nil
}

// DeleteVersion soft-deletes one version.
func (s *Service) DeleteVersion(ctx context.Context, tenant, name, version string) error {
	return s.artifacts.Delete(ctx, tenant, s.kind, name, version)
}

// Resolve returns a download target for a version.
func (s *Service) Resolve(ctx context.Context, tenant, name, version string) (*server.ArtifactResolveResponse, error) {
	r, err := s.artifacts.Resolve(ctx, tenant, s.kind, name, version, "download")
	if err != nil {
		return nil, err
	}
	out := &server.ArtifactResolveResponse{
		StorageKind: server.StorageKind(r.StorageKind),
		URI:         r.Uri,
	}
	if r.Digest != nil {
		out.Digest = *r.Digest
	}
	if r.ExpiresAt != nil {
		out.ExpiresAt = *r.ExpiresAt
	}
	var creds map[string]any
	b, _ := json.Marshal(r.PullCredentials)
	_ = json.Unmarshal(b, &creds)
	out.PullCredentials = creds
	return out, nil
}

func (s *Service) getDef(ctx context.Context, tenant, name string) (*store.Definition, error) {
	d, err := s.defs.GetByName(ctx, tenant, name)
	if errors.Is(err, store.ErrNotFound) {
		return nil, server.NotFound(s.kind + " not found")
	}
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "lookup "+s.kind, err)
	}
	return d, nil
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
