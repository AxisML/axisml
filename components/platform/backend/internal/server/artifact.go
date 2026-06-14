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
	// DatasetSpec is the artifact-side spec for kind=dataset.
	DatasetSpec map[string]any
)

// ---- Models ----

// Model is a model artifact version (Platform view).
type Model struct {
	ID          UUID        `json:"id"`
	Namespace   string      `json:"namespace"`
	TenantName  string      `json:"tenantName"`
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	DisplayName string      `json:"displayName,omitempty"`
	Description string      `json:"description,omitempty"`
	Status      ModelStatus `json:"status"`
	Digest      string      `json:"digest,omitempty"`
	Spec        ModelSpec   `json:"spec,omitempty"`
	Owner       string      `json:"owner,omitempty"`
	OwnerID     UUID        `json:"ownerId,omitempty"`
	URI         string      `json:"uri,omitempty"`
	SizeBytes   int64       `json:"sizeBytes,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	ReadyAt     *time.Time  `json:"readyAt,omitempty"`
	UpdatedAt   time.Time   `json:"updatedAt,omitempty"`
}

// ModelList is a page of Model.
type ModelList struct {
	Items         []Model `json:"items"`
	Count         int     `json:"count" binding:"min=0"`
	ContinueToken string  `json:"continueToken,omitempty"`
	Partial       bool    `json:"partial,omitempty"`
}

// ModelInitiateRequest starts a new model version upload.
type ModelInitiateRequest struct {
	Version     string    `json:"version" binding:"required,min=1,max=64"`
	DisplayName string    `json:"displayName,omitempty"`
	Description string    `json:"description,omitempty"`
	Spec        ModelSpec `json:"spec,omitempty"`
}

// ModelInitiateResponse returns the push target for a model upload.
type ModelInitiateResponse struct {
	ID                UUID           `json:"id"`
	URI               string         `json:"uri"`
	StorageKind       StorageKind    `json:"storageKind,omitempty"`
	UploadCredentials map[string]any `json:"uploadCredentials,omitempty"`
	ExpiresAt         time.Time      `json:"expiresAt,omitempty"`
}

// ModelCompleteRequest finalizes a model version upload.
type ModelCompleteRequest struct {
	Digest string `json:"digest" binding:"required"`
}

// ---- Artifacts (shared) ----

// ArtifactUpdateRequest is the partial-replace body for per-kind PATCH endpoints.
type ArtifactUpdateRequest struct {
	DisplayName string    `json:"displayName,omitempty" binding:"max=256"`
	Description string    `json:"description,omitempty" binding:"max=4096"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
}

// ArtifactDefinitionCreateInput creates a Platform-owned artifact definition.
type ArtifactDefinitionCreateInput struct {
	Name        string         `json:"name" binding:"required,artifactname,min=1,max=128"`
	DisplayName string         `json:"displayName,omitempty" binding:"max=256"`
	Description string         `json:"description,omitempty" binding:"max=4096"`
	Labels      StringMap      `json:"labels,omitempty"`
	Annotations StringMap      `json:"annotations,omitempty"`
	Spec        map[string]any `json:"spec,omitempty"`
}

// ArtifactDefinitionPatchInput patches an artifact definition.
type ArtifactDefinitionPatchInput struct {
	DisplayName string         `json:"displayName,omitempty" binding:"max=256"`
	Description string         `json:"description,omitempty" binding:"max=4096"`
	Labels      StringMap      `json:"labels,omitempty"`
	Annotations StringMap      `json:"annotations,omitempty"`
	Spec        map[string]any `json:"spec,omitempty"`
}

// ArtifactDefinitionView is a Platform-owned artifact definition (name-level).
type ArtifactDefinitionView struct {
	ID          UUID           `json:"id"`
	Namespace   string         `json:"namespace"`
	TenantName  string         `json:"tenantName"`
	Name        string         `json:"name"`
	Kind        DefinitionKind `json:"kind"`
	DisplayName string         `json:"displayName,omitempty"`
	Description string         `json:"description,omitempty"`
	Owner       string         `json:"owner,omitempty"`
	OwnerID     UUID           `json:"ownerId,omitempty"`
	Labels      StringMap      `json:"labels,omitempty"`
	Annotations StringMap      `json:"annotations,omitempty"`
	Spec        map[string]any `json:"spec,omitempty"`
	Visibility  Visibility     `json:"visibility,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// ArtifactDefinitionList is a page of ArtifactDefinitionView.
type ArtifactDefinitionList struct {
	Items         []ArtifactDefinitionView `json:"items"`
	Count         int                      `json:"count" binding:"min=0"`
	ContinueToken string                   `json:"continueToken,omitempty"`
	Partial       bool                     `json:"partial,omitempty"`
}

// ArtifactResolveResponse returns a download target for an artifact version.
type ArtifactResolveResponse struct {
	StorageKind     StorageKind    `json:"storageKind"`
	URI             string         `json:"uri"`
	Digest          string         `json:"digest,omitempty"`
	PullCredentials map[string]any `json:"pullCredentials,omitempty"`
	ExpiresAt       time.Time      `json:"expiresAt,omitempty"`
}

// ---- Images ----

// Image is an image artifact version (Platform view).
type Image struct {
	ID          UUID        `json:"id"`
	Namespace   string      `json:"namespace"`
	TenantName  string      `json:"tenantName"`
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	DisplayName string      `json:"displayName,omitempty"`
	Description string      `json:"description,omitempty"`
	Status      ImageStatus `json:"status"`
	Digest      string      `json:"digest,omitempty"`
	Spec        ImageSpec   `json:"spec,omitempty"`
	Owner       string      `json:"owner,omitempty"`
	OwnerID     UUID        `json:"ownerId,omitempty"`
	URI         string      `json:"uri,omitempty"`
	SizeBytes   int64       `json:"sizeBytes,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	ReadyAt     *time.Time  `json:"readyAt,omitempty"`
	UpdatedAt   time.Time   `json:"updatedAt,omitempty"`
}

// ImageList is a page of Image.
type ImageList struct {
	Items         []Image `json:"items"`
	Count         int     `json:"count" binding:"min=0"`
	ContinueToken string  `json:"continueToken,omitempty"`
	Partial       bool    `json:"partial,omitempty"`
}

// ImageInitiateRequest starts a new image version upload.
type ImageInitiateRequest struct {
	Version     string    `json:"version" binding:"required,min=1,max=64"`
	DisplayName string    `json:"displayName,omitempty"`
	Description string    `json:"description,omitempty"`
	Spec        ImageSpec `json:"spec" binding:"required"`
}

// ImageInitiateResponse returns the push target for an image upload.
type ImageInitiateResponse struct {
	ID                UUID           `json:"id"`
	URI               string         `json:"uri"`
	StorageKind       StorageKind    `json:"storageKind,omitempty"`
	UploadCredentials map[string]any `json:"uploadCredentials,omitempty"`
	ExpiresAt         time.Time      `json:"expiresAt,omitempty"`
}

// ImageCompleteRequest finalizes an image version upload.
type ImageCompleteRequest struct {
	Digest string `json:"digest" binding:"required"`
}

// ---- Datasets ----

// Dataset is a dataset artifact version (Platform view).
type Dataset struct {
	ID          UUID          `json:"id"`
	Namespace   string        `json:"namespace"`
	TenantName  string        `json:"tenantName"`
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	DisplayName string        `json:"displayName,omitempty"`
	Description string        `json:"description,omitempty"`
	Status      DatasetStatus `json:"status"`
	Digest      string        `json:"digest,omitempty"`
	Spec        DatasetSpec   `json:"spec,omitempty"`
	Owner       string        `json:"owner,omitempty"`
	OwnerID     UUID          `json:"ownerId,omitempty"`
	URI         string        `json:"uri,omitempty"`
	SizeBytes   int64         `json:"sizeBytes,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	ReadyAt     *time.Time    `json:"readyAt,omitempty"`
	UpdatedAt   time.Time     `json:"updatedAt,omitempty"`
}

// DatasetList is a page of Dataset.
type DatasetList struct {
	Items         []Dataset `json:"items"`
	Count         int       `json:"count" binding:"min=0"`
	ContinueToken string    `json:"continueToken,omitempty"`
	Partial       bool      `json:"partial,omitempty"`
}

// DatasetInitiateRequest starts a new dataset version upload.
type DatasetInitiateRequest struct {
	Version     string      `json:"version" binding:"required,min=1,max=64"`
	DisplayName string      `json:"displayName,omitempty"`
	Description string      `json:"description,omitempty"`
	Spec        DatasetSpec `json:"spec" binding:"required"`
}

// DatasetInitiateResponse returns the push target for a dataset upload.
type DatasetInitiateResponse struct {
	ID                UUID           `json:"id"`
	URI               string         `json:"uri"`
	StorageKind       StorageKind    `json:"storageKind,omitempty"`
	UploadCredentials map[string]any `json:"uploadCredentials,omitempty"`
	ExpiresAt         time.Time      `json:"expiresAt,omitempty"`
}

// DatasetCompleteRequest finalizes a dataset version upload.
type DatasetCompleteRequest struct {
	Digest string `json:"digest" binding:"required"`
}
