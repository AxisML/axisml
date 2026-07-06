package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/storage/s3"
	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

// datasetManifestObject is the canonical object whose SHA256 is the dataset
// digest (design §4.2). The client uploads it to the version prefix alongside
// the payload; complete verifies the stored bytes hash to the claimed digest.
const datasetManifestObject = "artifact-manifest.json"

// validDatasetFormats is the closed set per design §4.2.
var validDatasetFormats = map[string]struct{}{
	"parquet":    {},
	"jsonl":      {},
	"csv":        {},
	"webdataset": {},
	"tfrecord":   {},
	"custom":     {},
}

// DatasetHandler implements Handler for S3-backed (RustFS) dataset
// artifacts. Datasets are directory-shaped: the digest is the SHA256 of
// the `artifact-manifest.json` the client uploads alongside the dataset
// payload (design §4.2).
//
// s3c is optional: when non-nil, complete verifies the claimed digest against
// the stored manifest bytes and GC deletes the version prefix. When nil (the
// single-host dev form ships without object storage), the handler records the
// client-supplied digest unverified and GC is a no-op.
//
// The MVP DatasetHandler uses Credentials' bare-token fields to surface
// the S3 prefix + an opaque session marker; a real STS / IAM integration
// is a follow-up.
type DatasetHandler struct {
	bucket string
	s3c    *s3.Client
}

// NewDatasetHandler builds the handler with a configured bucket and an optional
// S3 client (nil disables live digest verification).
func NewDatasetHandler(bucket string, s3c *s3.Client) *DatasetHandler {
	if bucket == "" {
		bucket = "axisml-artifact-hub"
	}
	return &DatasetHandler{bucket: bucket, s3c: s3c}
}

func (h *DatasetHandler) Kind() string             { return "dataset" }
func (h *DatasetHandler) StorageKind() StorageKind { return StorageS3 }

// BuildStorageURI constructs the s3:// prefix per design §4.2:
// s3://<bucket>/namespaces/<namespace>/datasets/<name>/<version>/.
func (h *DatasetHandler) BuildStorageURI(namespace, name, version string) string {
	return fmt.Sprintf("s3://%s/namespaces/%s/datasets/%s/%s/", h.bucket, namespace, name, version)
}

// BuildPullURI returns the version-scoped S3 prefix unchanged: RustFS has no
// content-addressable path, so the prefix is already immutable per version and
// the digest is returned alongside for client-side verification.
func (h *DatasetHandler) BuildPullURI(namespace, name, version, _ string) string {
	return h.BuildStorageURI(namespace, name, version)
}

func (h *DatasetHandler) ValidateSpec(_ context.Context, spec Spec) error {
	format, ok := stringField(spec, "format")
	if !ok || format == "" {
		return apperrors.New(apperrors.CodeValidation, "spec.format is required")
	}
	if _, valid := validDatasetFormats[format]; !valid {
		return apperrors.Newf(apperrors.CodeValidation,
			"spec.format %q is not in {parquet,jsonl,csv,webdataset,tfrecord,custom}", format)
	}
	return nil
}

// InitiateUpload surfaces a placeholder credential structure carrying the
// dataset's S3 prefix so the cli knows where to PutObject. A real STS
// session is a follow-up — for now we return the prefix and a fixed TTL
// expiry so the contract is well-formed.
func (h *DatasetHandler) InitiateUpload(_ context.Context, a Artifact, ttl time.Duration) (Credentials, error) {
	return Credentials{
		Username:  "axisml-dataset-upload",
		Password:  fmt.Sprintf("prefix:%s", h.prefix(a.Namespace, a.Name, a.Version)),
		ExpiresAt: time.Now().Add(ttl).UTC(),
	}, nil
}

func (h *DatasetHandler) IssuePullCredentials(_ context.Context, a Artifact, ttl time.Duration) (Credentials, error) {
	return Credentials{
		Username:  "axisml-dataset-pull",
		Password:  fmt.Sprintf("prefix:%s", h.prefix(a.Namespace, a.Name, a.Version)),
		ExpiresAt: time.Now().Add(ttl).UTC(),
	}, nil
}

// VerifyComplete verifies the claimed digest against the stored dataset
// manifest. When an S3 backend is configured it GETs <prefix>artifact-manifest.json,
// computes its SHA256, and rejects a mismatch (409) or a missing manifest (412);
// without a backend it records the claimed digest unverified (dev form).
func (h *DatasetHandler) VerifyComplete(ctx context.Context, a Artifact, claim CompleteClaim) (string, error) {
	if claim.Digest == "" {
		return "", apperrors.New(apperrors.CodeValidation, "complete: digest is required")
	}
	if h.s3c == nil {
		// No object store configured — trust the claim (dev form).
		return claim.Digest, nil
	}

	key := h.prefix(a.Namespace, a.Name, a.Version) + datasetManifestObject
	body, err := h.s3c.GetObject(ctx, key)
	if err != nil {
		if errors.Is(err, s3.ErrNotFound) {
			return "", apperrors.Newf(apperrors.CodePrecondition,
				"dataset manifest %s not found in object store", key)
		}
		return "", apperrors.Wrap(apperrors.CodeUnavailable, "get dataset manifest", err)
	}

	sum := sha256.Sum256(body)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if digest != claim.Digest {
		return "", apperrors.Newf(apperrors.CodeConflict,
			"digest mismatch: manifest hashes to %s, claim is %s", digest, claim.Digest)
	}
	return digest, nil
}

// GCBackend deletes the dataset version prefix. No-op without an S3 backend;
// NotFound is treated as success inside the client.
func (h *DatasetHandler) GCBackend(ctx context.Context, a Artifact) error {
	if h.s3c == nil {
		return nil
	}
	return h.s3c.DeletePrefix(ctx, h.prefix(a.Namespace, a.Name, a.Version))
}

func (h *DatasetHandler) prefix(namespace, name, version string) string {
	return fmt.Sprintf("namespaces/%s/datasets/%s/%s/", namespace, name, version)
}
