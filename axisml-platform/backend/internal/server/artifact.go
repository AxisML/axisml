package server

import "time"

// ModelSpec / ImageSpec / DatasetSpec are artifact-side specs, pass-through to
// the artifacts service. The canonical field shapes live in artifact-hub's
// spec; here they are opaque, free-form objects.
type (
	// ModelSpec is the artifact-side spec for kind=model.
	ModelSpec map[string]any
	// ImageSpec is the artifact-side spec for kind=image.
	ImageSpec map[string]any
)

// ---- Models ----

// Model is a model artifact version (Platform view).
type Model struct {
	ID          UUID           `json:"id" desc:"Stable model version identifier."`
	Namespace   string         `json:"namespace" desc:"Platform tenant namespace the model belongs to."`
	TenantName  string         `json:"tenantName" desc:"Tenant identifier owning the model."`
	Name        string         `json:"name" desc:"Model definition name (unique within the tenant)."`
	Version     string         `json:"version" desc:"Model version label."`
	DisplayName string         `json:"displayName,omitempty" desc:"Human-readable version label."`
	Description string         `json:"description,omitempty" desc:"Free-text version description."`
	Status      ModelStatus    `json:"status" desc:"Version lifecycle status (Uploading, Ready, Failed, Deleting, Deleted)."`
	Source      ArtifactSource `json:"source,omitempty" desc:"How this version was added (webUpload, oras, dockerPush, external)."`
	Digest      string         `json:"digest,omitempty" desc:"Content digest of the uploaded artifact."`
	Spec        ModelSpec      `json:"spec,omitempty" desc:"Pass-through artifact-side spec (free-form)."`
	Owner       string         `json:"owner,omitempty" desc:"Username of the version owner."`
	OwnerID     UUID           `json:"ownerId,omitempty" desc:"User ID of the version owner."`
	URI         string         `json:"uri,omitempty" desc:"Artifact registry URI of the version."`
	SizeBytes   int64          `json:"sizeBytes,omitempty" desc:"Total size of the version content in bytes."`
	CreatedAt   time.Time      `json:"createdAt" desc:"Time the version was created."`
	ReadyAt     *time.Time     `json:"readyAt,omitempty" desc:"Time the version became Ready."`
	UpdatedAt   time.Time      `json:"updatedAt,omitempty" desc:"Time the version was last updated."`
}

// ModelList is a page of Model.
type ModelList struct {
	Items         []Model `json:"items" desc:"Model versions in this page."`
	Count         int     `json:"count" binding:"min=0" desc:"Number of model versions in this page."`
	ContinueToken string  `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool    `json:"partial,omitempty" desc:"True if the list was truncated by an upstream limit."`
}

// ModelInitiateRequest starts a new model version. Source selects the method:
// empty/webUpload = direct upload, oras = CLI push (both return upload
// credentials), external = register a remote URI with no upload.
type ModelInitiateRequest struct {
	Version          string           `json:"version" binding:"required,min=1,max=64" desc:"Model version label to create."`
	DisplayName      string           `json:"displayName,omitempty" desc:"Human-readable version label."`
	Description      string           `json:"description,omitempty" desc:"Free-text version description."`
	Source           ArtifactSource   `json:"source,omitempty" desc:"Upload method (webUpload, oras, external); empty defaults to webUpload."`
	RemoteURI        string           `json:"remoteUri,omitempty" desc:"Remote artifact URI to register (required when source=external)."`
	RemoteSourceKind RemoteSourceKind `json:"remoteSourceKind,omitempty" desc:"Backing store of the remote URI (s3, oci, http, hf, custom); required when source=external."`
	Spec             ModelSpec        `json:"spec,omitempty" desc:"Pass-through artifact-side spec (free-form)."`
}

// ModelInitiateResponse returns the push target for a model upload.
type ModelInitiateResponse struct {
	ID                UUID           `json:"id" desc:"Identifier of the newly initiated model version."`
	URI               string         `json:"uri" desc:"Upload target URI for the version content."`
	StorageKind       StorageKind    `json:"storageKind,omitempty" desc:"Backing store of the upload target (oci, s3)."`
	UploadCredentials map[string]any `json:"uploadCredentials,omitempty" desc:"Short-lived credentials for pushing the content."`
	ExpiresAt         time.Time      `json:"expiresAt,omitempty" desc:"Expiry time of the upload credentials."`
}

// ModelCompleteRequest finalizes a model version upload.
type ModelCompleteRequest struct {
	Digest string `json:"digest" binding:"required" desc:"Content digest of the uploaded artifact (finalizes the version)."`
}

// ---- Artifacts (shared) ----

// ArtifactUpdateRequest is the partial-replace body for per-kind PATCH endpoints.
type ArtifactUpdateRequest struct {
	DisplayName string    `json:"displayName,omitempty" binding:"max=256" desc:"Updated human-readable label."`
	Description string    `json:"description,omitempty" binding:"max=4096" desc:"Updated free-text description."`
	Labels      StringMap `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations StringMap `json:"annotations,omitempty" desc:"Replacement annotation set."`
}

// ArtifactDefinitionCreateRequest creates a Platform-owned artifact definition.
type ArtifactDefinitionCreateRequest struct {
	Name        string         `json:"name" binding:"required,artifactname,min=1,max=128" desc:"Artifact definition name (unique within the tenant)."`
	DisplayName string         `json:"displayName,omitempty" binding:"max=256" desc:"Human-readable definition label."`
	Description string         `json:"description,omitempty" binding:"max=4096" desc:"Free-text definition description."`
	Labels      StringMap      `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations StringMap      `json:"annotations,omitempty" desc:"User-defined annotations."`
	Spec        map[string]any `json:"spec,omitempty" desc:"Pass-through definition spec (free-form)."`
}

// ArtifactDefinitionPatchRequest patches an artifact definition.
type ArtifactDefinitionPatchRequest struct {
	DisplayName string         `json:"displayName,omitempty" binding:"max=256" desc:"Updated human-readable definition label."`
	Description string         `json:"description,omitempty" binding:"max=4096" desc:"Updated free-text definition description."`
	Labels      StringMap      `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations StringMap      `json:"annotations,omitempty" desc:"Replacement annotation set."`
	Spec        map[string]any `json:"spec,omitempty" desc:"Replacement definition spec (free-form)."`
}

// ArtifactDefinition is a Platform-owned artifact definition (name-level).
type ArtifactDefinition struct {
	ID          UUID           `json:"id" desc:"Stable definition identifier."`
	Namespace   string         `json:"namespace" desc:"Platform tenant namespace the definition belongs to."`
	TenantName  string         `json:"tenantName" desc:"Tenant identifier owning the definition."`
	Name        string         `json:"name" desc:"Definition name (unique within the tenant)."`
	Kind        DefinitionKind `json:"kind" desc:"Definition kind (model or image)."`
	DisplayName string         `json:"displayName,omitempty" desc:"Human-readable definition label."`
	Description string         `json:"description,omitempty" desc:"Free-text definition description."`
	Owner       string         `json:"owner,omitempty" desc:"Username of the definition owner."`
	OwnerID     UUID           `json:"ownerId,omitempty" desc:"User ID of the definition owner."`
	Labels      StringMap      `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations StringMap      `json:"annotations,omitempty" desc:"User-defined annotations."`
	Spec        map[string]any `json:"spec,omitempty" desc:"Pass-through definition spec (free-form)."`
	CreatedAt   time.Time      `json:"createdAt" desc:"Time the definition was created."`
	UpdatedAt   time.Time      `json:"updatedAt" desc:"Time the definition was last updated."`
}

// ArtifactDefinitionList is a page of ArtifactDefinition.
type ArtifactDefinitionList struct {
	Items         []ArtifactDefinition `json:"items" desc:"Artifact definitions in this page."`
	Count         int                  `json:"count" binding:"min=0" desc:"Number of definitions in this page."`
	ContinueToken string               `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool                 `json:"partial,omitempty" desc:"True if the list was truncated by an upstream limit."`
}

// ArtifactResolveResponse returns a download target for an artifact version.
type ArtifactResolveResponse struct {
	StorageKind     StorageKind    `json:"storageKind" desc:"Backing store of the download target (oci, s3)."`
	URI             string         `json:"uri" desc:"Download URI for the version content."`
	Digest          string         `json:"digest,omitempty" desc:"Content digest of the version."`
	PullCredentials map[string]any `json:"pullCredentials,omitempty" desc:"Short-lived credentials for pulling the content."`
	ExpiresAt       time.Time      `json:"expiresAt,omitempty" desc:"Expiry time of the pull credentials."`
}

// ---- Images ----

// Image is an image artifact version (Platform view).
type Image struct {
	ID          UUID           `json:"id" desc:"Stable image version identifier."`
	Namespace   string         `json:"namespace" desc:"Platform tenant namespace the image belongs to."`
	TenantName  string         `json:"tenantName" desc:"Tenant identifier owning the image."`
	Name        string         `json:"name" desc:"Image definition name (unique within the tenant)."`
	Version     string         `json:"version" desc:"Image version (tag)."`
	DisplayName string         `json:"displayName,omitempty" desc:"Human-readable version label."`
	Description string         `json:"description,omitempty" desc:"Free-text version description."`
	Status      ImageStatus    `json:"status" desc:"Version lifecycle status (Uploading, Ready, Failed, Deleting, Deleted)."`
	Source      ArtifactSource `json:"source,omitempty" desc:"How this version was added (webUpload, oras, dockerPush, external)."`
	Digest      string         `json:"digest,omitempty" desc:"Content digest of the image."`
	Spec        ImageSpec      `json:"spec,omitempty" desc:"Pass-through artifact-side spec (free-form)."`
	Owner       string         `json:"owner,omitempty" desc:"Username of the version owner."`
	OwnerID     UUID           `json:"ownerId,omitempty" desc:"User ID of the version owner."`
	URI         string         `json:"uri,omitempty" desc:"Artifact registry URI of the version."`
	SizeBytes   int64          `json:"sizeBytes,omitempty" desc:"Total size of the image content in bytes."`
	CreatedAt   time.Time      `json:"createdAt" desc:"Time the version was created."`
	ReadyAt     *time.Time     `json:"readyAt,omitempty" desc:"Time the version became Ready."`
	UpdatedAt   time.Time      `json:"updatedAt,omitempty" desc:"Time the version was last updated."`
}

// ImageList is a page of Image.
type ImageList struct {
	Items         []Image `json:"items" desc:"Image versions in this page."`
	Count         int     `json:"count" binding:"min=0" desc:"Number of image versions in this page."`
	ContinueToken string  `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool    `json:"partial,omitempty" desc:"True if the list was truncated by an upstream limit."`
}

// ImageInitiateRequest starts a new image version. Source selects the method:
// dockerPush returns push credentials; external syncs a remote image ref.
type ImageInitiateRequest struct {
	Version        string         `json:"version" binding:"required,min=1,max=64" desc:"Image version (tag) to create."`
	DisplayName    string         `json:"displayName,omitempty" desc:"Human-readable version label."`
	Description    string         `json:"description,omitempty" desc:"Free-text version description."`
	Source         ArtifactSource `json:"source,omitempty" desc:"Upload method (dockerPush, external); empty defaults to dockerPush."`
	SourceImageRef string         `json:"sourceImageRef,omitempty" desc:"Remote image reference to sync (required when source=external)."`
	Spec           ImageSpec      `json:"spec" binding:"required" desc:"Pass-through artifact-side spec (free-form)."`
}

// ImageInitiateResponse returns the push target for an image upload.
type ImageInitiateResponse struct {
	ID                UUID           `json:"id" desc:"Identifier of the newly initiated image version."`
	URI               string         `json:"uri" desc:"Push target URI for the image content."`
	StorageKind       StorageKind    `json:"storageKind,omitempty" desc:"Backing store of the push target (oci, s3)."`
	UploadCredentials map[string]any `json:"uploadCredentials,omitempty" desc:"Short-lived credentials for pushing the image."`
	ExpiresAt         time.Time      `json:"expiresAt,omitempty" desc:"Expiry time of the upload credentials."`
}

// ImageCompleteRequest finalizes an image version upload.
type ImageCompleteRequest struct {
	Digest string `json:"digest" binding:"required" desc:"Content digest of the pushed image (finalizes the version)."`
}
