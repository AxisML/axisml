package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/storage/oci"
	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

// validImagePurposes is the closed set per design §4.3.
var validImagePurposes = map[string]struct{}{
	"training":  {},
	"inference": {},
	"dev":       {},
}

// ImageHandler implements Handler for OCI-backed container image artifacts.
// Mirrors ModelHandler's flow against the same zot endpoint but with a
// distinct repository sub-path so models and images don't collide.
type ImageHandler struct {
	oci *oci.Client
}

// NewImageHandler returns a Handler implementation.
func NewImageHandler(client *oci.Client) *ImageHandler {
	return &ImageHandler{oci: client}
}

func (h *ImageHandler) Kind() string             { return "image" }
func (h *ImageHandler) StorageKind() StorageKind { return StorageOCI }

// BuildStorageURI constructs the OCI image reference per design §4.3:
// <oci-host>/namespaces/<namespace>/images/<name>:<version>.
func (h *ImageHandler) BuildStorageURI(namespace, name, version string) string {
	return fmt.Sprintf("%s/namespaces/%s/images/%s:%s", h.oci.Endpoint(), namespace, name, version)
}

// BuildPullURI pins the resolved reference to digest so consumers fetch
// immutable content (<oci-host>/namespaces/<namespace>/images/<name>@<digest>).
// Falls back to the tag form before the artifact is Ready (empty digest).
func (h *ImageHandler) BuildPullURI(namespace, name, version, digest string) string {
	if digest == "" {
		return h.BuildStorageURI(namespace, name, version)
	}
	return fmt.Sprintf("%s/namespaces/%s/images/%s@%s", h.oci.Endpoint(), namespace, name, digest)
}

func (h *ImageHandler) repoPath(namespace, name string) string {
	return fmt.Sprintf("namespaces/%s/images/%s", namespace, name)
}

// ValidateSpec enforces the design §4.3 required field (purpose).
func (h *ImageHandler) ValidateSpec(_ context.Context, spec Spec) error {
	purpose, ok := stringField(spec, "purpose")
	if !ok || purpose == "" {
		return apperrors.New(apperrors.CodeValidation, "spec.purpose is required")
	}
	if _, valid := validImagePurposes[purpose]; !valid {
		return apperrors.Newf(apperrors.CodeValidation,
			"spec.purpose %q is not in {training,inference,dev}", purpose)
	}
	return nil
}

func (h *ImageHandler) InitiateUpload(ctx context.Context, a Artifact, ttl time.Duration) (Credentials, error) {
	return h.oci.IssueUploadCredentials(ctx, h.repoPath(a.Namespace, a.Name), ttl)
}

func (h *ImageHandler) IssuePullCredentials(ctx context.Context, a Artifact, ttl time.Duration) (Credentials, error) {
	return h.oci.IssuePullCredentials(ctx, h.repoPath(a.Namespace, a.Name), ttl)
}

func (h *ImageHandler) VerifyComplete(ctx context.Context, a Artifact, claim CompleteClaim) (string, error) {
	digest, err := h.oci.HeadManifest(ctx, h.repoPath(a.Namespace, a.Name), a.Version)
	if err != nil {
		if errors.Is(err, oci.ErrNotFound) {
			return "", apperrors.Newf(apperrors.CodePrecondition,
				"image manifest %s:%s not found in registry",
				h.repoPath(a.Namespace, a.Name), a.Version)
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

func (h *ImageHandler) GCBackend(ctx context.Context, a Artifact) error {
	if a.Digest == "" {
		return nil
	}
	return h.oci.DeleteManifest(ctx, h.repoPath(a.Namespace, a.Name), a.Digest)
}
