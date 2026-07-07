package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/storage/oci"
	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

// contentDigestPrefix marks a real content-addressable digest. Only such a
// value can be pinned into an OCI pull reference; a digest column holding a
// non-digest (e.g. an external artifact's remote URI) must not be pinned.
const contentDigestPrefix = "sha256:"

// ociBacked holds the OCI-backed mechanics shared by the model and image kinds,
// parameterized by the kind string and the repository sub-path segment
// ("models" / "images") so models and images don't collide. Concrete handlers
// embed it and add only their kind-specific ValidateSpec.
type ociBacked struct {
	oci     *oci.Client
	kind    string
	subpath string
}

func (h *ociBacked) Kind() string             { return h.kind }
func (h *ociBacked) StorageKind() StorageKind { return StorageOCI }

// BuildStorageURI constructs the OCI image reference:
// <oci-host>/namespaces/<namespace>/<subpath>/<name>:<version>.
func (h *ociBacked) BuildStorageURI(namespace, name, version string) string {
	return fmt.Sprintf("%s/namespaces/%s/%s/%s:%s", h.oci.Endpoint(), namespace, h.subpath, name, version)
}

// BuildPullURI pins the resolved reference to digest so consumers fetch
// immutable content (<oci-host>/namespaces/<namespace>/<subpath>/<name>@<digest>).
// Falls back to the tag form when there is no content digest to pin — before the
// artifact is Ready (empty digest) or when the digest column holds a non-digest
// value (an external artifact's remote URI).
func (h *ociBacked) BuildPullURI(namespace, name, version, digest string) string {
	if !strings.HasPrefix(digest, contentDigestPrefix) {
		return h.BuildStorageURI(namespace, name, version)
	}
	return fmt.Sprintf("%s/namespaces/%s/%s/%s@%s", h.oci.Endpoint(), namespace, h.subpath, name, digest)
}

// repoPath returns the OCI repository path (no host, no tag) used in v2 API URLs.
func (h *ociBacked) repoPath(namespace, name string) string {
	return fmt.Sprintf("namespaces/%s/%s/%s", namespace, h.subpath, name)
}

// InitiateUpload signs push credentials. MVP: admin htpasswd passthrough.
func (h *ociBacked) InitiateUpload(ctx context.Context, a Artifact, ttl time.Duration) (Credentials, error) {
	return h.oci.IssueUploadCredentials(ctx, h.repoPath(a.Namespace, a.Name), ttl)
}

// IssuePullCredentials signs pull credentials. MVP: admin passthrough.
func (h *ociBacked) IssuePullCredentials(ctx context.Context, a Artifact, ttl time.Duration) (Credentials, error) {
	return h.oci.IssuePullCredentials(ctx, h.repoPath(a.Namespace, a.Name), ttl)
}

// VerifyComplete HEADs the manifest at <repoPath>:<version> and checks the
// digest matches what the cli reported.
func (h *ociBacked) VerifyComplete(ctx context.Context, a Artifact, claim CompleteClaim) (string, error) {
	digest, err := h.oci.HeadManifest(ctx, h.repoPath(a.Namespace, a.Name), a.Version)
	if err != nil {
		if errors.Is(err, oci.ErrNotFound) {
			return "", apperrors.Newf(apperrors.CodePrecondition,
				"%s manifest %s:%s not found in registry", h.kind, h.repoPath(a.Namespace, a.Name), a.Version)
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
func (h *ociBacked) GCBackend(ctx context.Context, a Artifact) error {
	if a.Digest == "" {
		// No digest means the artifact never reached Ready; nothing to clean
		// up beyond what zot's GC handles for orphan blobs.
		return nil
	}
	return h.oci.DeleteManifest(ctx, h.repoPath(a.Namespace, a.Name), a.Digest)
}

func stringField(s Spec, key string) (string, bool) {
	v, ok := s[key]
	if !ok {
		return "", false
	}
	str, ok := v.(string)
	return str, ok
}
