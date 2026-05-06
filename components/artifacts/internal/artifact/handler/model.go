package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/axisml/axisml/components/artifacts/internal/storage/oci"
	apperrors "github.com/axisml/axisml/components/artifacts/pkg/errors"
)

// validFrameworks is the closed set design §5.1 calls out for the model Kind.
var validFrameworks = map[string]struct{}{
	"pytorch":     {},
	"tensorflow":  {},
	"onnx":        {},
	"safetensors": {},
	"gguf":        {},
	"custom":      {},
}

// ModelHandler implements Handler for OCI-backed model artifacts.
type ModelHandler struct {
	oci *oci.Client
}

// NewModelHandler returns a Handler implementation. Caller must pass a
// configured OCI client.
func NewModelHandler(client *oci.Client) *ModelHandler {
	return &ModelHandler{oci: client}
}

// Kind reports "model".
func (h *ModelHandler) Kind() string { return "model" }

// StorageKind reports OCI.
func (h *ModelHandler) StorageKind() StorageKind { return StorageOCI }

// BuildStorageURI constructs the OCI image reference per design §5.1:
//
//	<oci-host>/<scope>/models/<repo>:<version>
//
// `scope` is "tenants/<tenant>" for tenant-private repos.
func (h *ModelHandler) BuildStorageURI(scope, repo, version string) string {
	return fmt.Sprintf("%s/%s/models/%s:%s", h.oci.Endpoint(), scope, repo, version)
}

// repoPath returns the OCI repository path (no host, no tag) used in v2
// API URLs.
func (h *ModelHandler) repoPath(scope, repo string) string {
	return fmt.Sprintf("%s/models/%s", scope, repo)
}

// ValidateSpec enforces the design §5.1 required fields.
func (h *ModelHandler) ValidateSpec(_ context.Context, spec Spec) error {
	framework, ok := stringField(spec, "framework")
	if !ok {
		return apperrors.New(apperrors.CodeValidation, "spec.framework is required")
	}
	if _, valid := validFrameworks[framework]; !valid {
		return apperrors.Newf(apperrors.CodeValidation,
			"spec.framework %q is not in {pytorch,tensorflow,onnx,safetensors,gguf,custom}", framework)
	}
	if format, ok := stringField(spec, "format"); !ok || format == "" {
		return apperrors.New(apperrors.CodeValidation, "spec.format is required")
	}
	return nil
}

// InitiateUpload signs push credentials. MVP: admin htpasswd passthrough.
func (h *ModelHandler) InitiateUpload(ctx context.Context, a Artifact, ttl time.Duration) (Credentials, error) {
	return h.oci.IssueUploadCredentials(ctx, h.repoPath(a.Scope, a.RepoName), ttl)
}

// IssuePullCredentials signs pull credentials. MVP: admin passthrough.
func (h *ModelHandler) IssuePullCredentials(ctx context.Context, a Artifact, ttl time.Duration) (Credentials, error) {
	return h.oci.IssuePullCredentials(ctx, h.repoPath(a.Scope, a.RepoName), ttl)
}

// VerifyComplete HEADs the manifest at <repoPath>:<version> and checks the
// digest matches what the cli reported.
func (h *ModelHandler) VerifyComplete(ctx context.Context, a Artifact, claim CompleteClaim) (string, error) {
	digest, err := h.oci.HeadManifest(ctx, h.repoPath(a.Scope, a.RepoName), a.Version)
	if err != nil {
		if errors.Is(err, oci.ErrNotFound) {
			return "", apperrors.Newf(apperrors.CodePrecondition,
				"manifest %s:%s not found in registry", h.repoPath(a.Scope, a.RepoName), a.Version)
		}
		return "", apperrors.Wrap(apperrors.CodeUnavailable, "head manifest", err)
	}
	if claim.Digest == "" {
		return "", apperrors.New(apperrors.CodeValidation, "complete: digest is required")
	}
	if digest != claim.Digest {
		return "", apperrors.Newf(apperrors.CodeConflict,
			"digest mismatch: registry has %s, claim is %s", digest, claim.Digest)
	}
	return digest, nil
}

// GCBackend deletes the manifest by digest. NotFound is treated as success
// inside the OCI client.
func (h *ModelHandler) GCBackend(ctx context.Context, a Artifact) error {
	if a.Digest == "" {
		// No digest means the artifact never reached Ready; nothing to clean
		// up beyond what zot's GC handles for orphan blobs.
		return nil
	}
	return h.oci.DeleteManifest(ctx, h.repoPath(a.Scope, a.RepoName), a.Digest)
}

func stringField(s Spec, key string) (string, bool) {
	v, ok := s[key]
	if !ok {
		return "", false
	}
	str, ok := v.(string)
	return str, ok
}
