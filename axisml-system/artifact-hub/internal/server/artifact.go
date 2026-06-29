package server

import (
	"time"

	"github.com/google/uuid"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/storage/oci"
)

// Artifact is the JSON projection of an Artifact returned by the API. JSON
// field names follow the design yaml (camelCase) for parity with the
// OpenAPI contract clients consume.
type Artifact struct {
	ID          uuid.UUID         `json:"id" desc:"Stable artifact identifier (UUID)."`
	Namespace   string            `json:"namespace" desc:"Tenant namespace the artifact belongs to (= compute tenants.name)."`
	Kind        string            `json:"kind" desc:"Artifact kind (model, dataset, image)."`
	Name        string            `json:"name" desc:"Artifact name, unique within (namespace, kind)."`
	Version     string            `json:"version" desc:"Artifact version (free-form string)."`
	Visibility  string            `json:"visibility" desc:"Access scope of the artifact (tenant or public)."`
	DisplayName string            `json:"displayName,omitempty" desc:"Human-readable name for display."`
	Description string            `json:"description,omitempty" desc:"Free-form description of the artifact."`
	Labels      map[string]string `json:"labels,omitempty" desc:"K8s-style labels used for selector filtering."`
	Annotations map[string]string `json:"annotations,omitempty" desc:"Non-identifying metadata annotations."`
	Owner       string            `json:"owner,omitempty" desc:"Identity that owns the artifact version."`
	Spec        map[string]any    `json:"spec" desc:"Kind-specific free-form specification of the artifact."`
	Status      string            `json:"status" desc:"Lifecycle status (Uploading, Ready, Failed, Deleting, Deleted)."`
	Source      string            `json:"source,omitempty" desc:"Provenance of the version (webUpload, oras, dockerPush, external)."`
	Message     string            `json:"message,omitempty" desc:"Human-readable detail for the current status (e.g. failure reason)."`
	Digest      string            `json:"digest,omitempty" desc:"Content digest of the stored artifact, set once the upload completes."`
	ReadyAt     *time.Time        `json:"readyAt,omitempty" desc:"Timestamp the artifact became Ready (RFC3339)."`
	CreatedAt   time.Time         `json:"createdAt" desc:"Creation timestamp (RFC3339)."`
	UpdatedAt   time.Time         `json:"updatedAt" desc:"Last-update timestamp (RFC3339)."`
	DeletedAt   *time.Time        `json:"deletedAt,omitempty" desc:"Soft-delete timestamp, set when the artifact is being removed (RFC3339)."`
}

// ArtifactInitiateRequest is the API request body for the two-phase write step 1.
//
// Source selects the version's provenance: webUpload / oras / dockerPush all
// return push credentials and follow the two-phase write; external registers a
// remote artifact with NO upload (SourceURI is required) and is born Ready.
// Empty Source defaults to webUpload.
type ArtifactInitiateRequest struct {
	Version     string            `json:"version" binding:"required,axisml_version" desc:"Version string to create for the artifact name."`
	Spec        map[string]any    `json:"spec" binding:"required" desc:"Kind-specific free-form specification of the artifact."`
	Source      string            `json:"source,omitempty" binding:"omitempty,oneof=webUpload oras dockerPush external" desc:"Provenance of the version (webUpload, oras, dockerPush, external); external registers a remote artifact with no upload."`
	SourceURI   string            `json:"sourceUri,omitempty" desc:"Remote URI of the artifact; required when source is external."`
	Visibility  string            `json:"visibility,omitempty" desc:"Access scope of the artifact (tenant or public); defaults to tenant."`
	DisplayName string            `json:"displayName,omitempty" desc:"Human-readable name for display."`
	Description string            `json:"description,omitempty" desc:"Free-form description of the artifact."`
	Labels      map[string]string `json:"labels,omitempty" desc:"K8s-style labels used for selector filtering."`
	Annotations map[string]string `json:"annotations,omitempty" desc:"Non-identifying metadata annotations."`
}

// ArtifactPatchRequest is the body for PATCH .../{kindPlural}/{name}/{version}.
// Only the four "displayable" fields are mutable post-Ready (design §6).
type ArtifactPatchRequest struct {
	DisplayName *string           `json:"displayName,omitempty" desc:"New human-readable display name; omit to leave unchanged."`
	Description *string           `json:"description,omitempty" desc:"New free-form description; omit to leave unchanged."`
	Labels      map[string]string `json:"labels,omitempty" desc:"Replacement K8s-style labels; omit to leave unchanged."`
	Annotations map[string]string `json:"annotations,omitempty" desc:"Replacement metadata annotations; omit to leave unchanged."`
}

// UploadCredentials wraps the storage-backend credential + storage URI
// returned alongside the artifact row on Initiate (design yaml).
type UploadCredentials struct {
	StorageKind string          `json:"storageKind" desc:"Storage backend kind backing the upload (e.g. oci)."`
	URI         string          `json:"uri" desc:"Target storage URI the client pushes the artifact content to."`
	Credentials oci.Credentials `json:"credentials" desc:"Push-capable credentials for the upload target."`
	ExpiresAt   time.Time       `json:"expiresAt" desc:"Expiry of the upload credentials (RFC3339)."`
}

// ArtifactInitiateResponse is what we return to the cli after step 1. Per design
// yaml, it bundles the newly persisted Artifact view with the upload
// credentials so the caller has a one-stop reply.
type ArtifactInitiateResponse struct {
	Artifact Artifact          `json:"artifact" desc:"The newly persisted artifact version (status Uploading until upload completes)."`
	Upload   UploadCredentials `json:"upload" desc:"Upload target and credentials for pushing the artifact content."`
}

// ArtifactCompleteRequest is the API request body for the two-phase write step 2.
type ArtifactCompleteRequest struct {
	Digest string `json:"digest" binding:"required" desc:"Content digest of the pushed artifact, finalizing the two-phase write."`
}

// ArtifactResolveResponse is what we return on /resolve. CamelCase per design yaml;
// `visibility` is the artifact's persisted visibility (tenant|public) so
// callers don't need a second GET.
type ArtifactResolveResponse struct {
	StorageKind     string           `json:"storageKind" desc:"Storage backend kind serving the artifact (e.g. oci)."`
	URI             string           `json:"uri" desc:"Storage URI the client pulls the artifact content from."`
	Digest          string           `json:"digest,omitempty" desc:"Content digest of the resolved artifact."`
	Visibility      string           `json:"visibility,omitempty" desc:"Persisted visibility of the artifact (tenant or public)."`
	PullCredentials *oci.Credentials `json:"pullCredentials,omitempty" desc:"Pull credentials, omitted for public artifacts that need none."`
	ExpiresAt       *time.Time       `json:"expiresAt,omitempty" desc:"Expiry of the pull credentials (RFC3339)."`
}
