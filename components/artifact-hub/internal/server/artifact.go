package server

import (
	"time"

	"github.com/google/uuid"

	"github.com/axisml/axisml/components/artifact-hub/internal/storage/oci"
)

// Artifact is the JSON projection of an Artifact returned by the API. JSON
// field names follow the design yaml (camelCase) for parity with the
// OpenAPI contract clients consume.
type Artifact struct {
	ID          uuid.UUID         `json:"id"`
	Namespace   string            `json:"namespace"`
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Visibility  string            `json:"visibility"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Spec        map[string]any    `json:"spec"`
	Status      string            `json:"status"`
	Source      string            `json:"source,omitempty"`
	Message     string            `json:"message,omitempty"`
	Digest      string            `json:"digest,omitempty"`
	ReadyAt     *time.Time        `json:"readyAt,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	DeletedAt   *time.Time        `json:"deletedAt,omitempty"`
}

// ArtifactInitiateRequest is the API request body for the two-phase write step 1.
//
// Source selects the version's provenance: webUpload / oras / dockerPush all
// return push credentials and follow the two-phase write; external registers a
// remote artifact with NO upload (SourceURI is required) and is born Ready.
// Empty Source defaults to webUpload.
type ArtifactInitiateRequest struct {
	Version     string            `json:"version" binding:"required,axisml_version"`
	Spec        map[string]any    `json:"spec" binding:"required"`
	Source      string            `json:"source,omitempty" binding:"omitempty,oneof=webUpload oras dockerPush external"`
	SourceURI   string            `json:"sourceUri,omitempty"`
	Visibility  string            `json:"visibility,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ArtifactPatchRequest is the body for PATCH .../{kindPlural}/{name}/{version}.
// Only the four "displayable" fields are mutable post-Ready (design §6).
type ArtifactPatchRequest struct {
	DisplayName *string           `json:"displayName,omitempty"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// UploadCredentials wraps the storage-backend credential + storage URI
// returned alongside the artifact row on Initiate (design yaml).
type UploadCredentials struct {
	StorageKind string          `json:"storageKind"`
	URI         string          `json:"uri"`
	Credentials oci.Credentials `json:"credentials"`
	ExpiresAt   time.Time       `json:"expiresAt"`
}

// ArtifactInitiateResponse is what we return to the cli after step 1. Per design
// yaml, it bundles the newly persisted Artifact view with the upload
// credentials so the caller has a one-stop reply.
type ArtifactInitiateResponse struct {
	Artifact Artifact          `json:"artifact"`
	Upload   UploadCredentials `json:"upload"`
}

// ArtifactCompleteRequest is the API request body for the two-phase write step 2.
type ArtifactCompleteRequest struct {
	Digest string `json:"digest" binding:"required"`
}

// ArtifactResolveResponse is what we return on /resolve. CamelCase per design yaml;
// `visibility` is the artifact's persisted visibility (tenant|public) so
// callers don't need a second GET.
type ArtifactResolveResponse struct {
	StorageKind     string           `json:"storageKind"`
	URI             string           `json:"uri"`
	Digest          string           `json:"digest,omitempty"`
	Visibility      string           `json:"visibility,omitempty"`
	PullCredentials *oci.Credentials `json:"pullCredentials,omitempty"`
	ExpiresAt       *time.Time       `json:"expiresAt,omitempty"`
}
