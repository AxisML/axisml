package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/axisml/axisml/components/artifact-hub/internal/storage/oci"
	apperrors "github.com/axisml/axisml/components/artifact-hub/pkg/errors"
)

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
// the canonical-JSON `artifact-manifest.json` the client uploads alongside
// the dataset payload (design §4.2).
//
// The MVP DatasetHandler uses Credentials' bare-token fields to surface
// the S3 prefix + an opaque session marker; a real STS / IAM integration
// is a follow-up. ValidateSpec / BuildStorageURI / GCBackend match the
// design contract today.
type DatasetHandler struct {
	bucket   string
	endpoint string
}

// NewDatasetHandler builds the handler with a configured bucket and
// public endpoint (used for the s3:// URI).
func NewDatasetHandler(bucket, endpoint string) *DatasetHandler {
	if bucket == "" {
		bucket = "axisml-artifact-hub"
	}
	return &DatasetHandler{bucket: bucket, endpoint: endpoint}
}

func (h *DatasetHandler) Kind() string             { return "dataset" }
func (h *DatasetHandler) StorageKind() StorageKind { return StorageS3 }

// BuildStorageURI constructs the s3:// prefix per design §4.2:
// s3://<bucket>/namespaces/<namespace>/datasets/<name>/<version>/.
func (h *DatasetHandler) BuildStorageURI(namespace, name, version string) string {
	return fmt.Sprintf("s3://%s/namespaces/%s/datasets/%s/%s/", h.bucket, namespace, name, version)
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

// VerifyComplete accepts the cli's claimed digest verbatim (no live HEAD
// against the S3 manifest yet). Real S3 verification — HEAD
// <prefix>artifact-manifest.json — is a follow-up.
func (h *DatasetHandler) VerifyComplete(_ context.Context, _ Artifact, claim CompleteClaim) (string, error) {
	if claim.Digest == "" {
		return "", apperrors.New(apperrors.CodeValidation, "complete: digest is required")
	}
	return claim.Digest, nil
}

// GCBackend is a no-op in the MVP; a real implementation would issue a
// DeletePrefix against RustFS. Keep idempotent semantics for the GC worker.
func (h *DatasetHandler) GCBackend(_ context.Context, _ Artifact) error {
	return nil
}

func (h *DatasetHandler) prefix(namespace, name, version string) string {
	return fmt.Sprintf("namespaces/%s/datasets/%s/%s/", namespace, name, version)
}

// silence unused import (we only reference oci for shared types).
var _ = oci.ErrNotFound
