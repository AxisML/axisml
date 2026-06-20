package artifact

// Status enumerates the Artifact lifecycle (design §3.5).
const (
	StatusUploading = "Uploading"
	StatusReady     = "Ready"
	StatusFailed    = "Failed"
	StatusDeleting  = "Deleting"
	StatusDeleted   = "Deleted"
)

// Visibility enumerates per-artifact visibility (design §3).
const (
	VisibilityTenant = "tenant" // default; visible only within the row's tenant scope
	VisibilityPublic = "public" // global; only allowed in the default tenant scope
)

// Source enumerates how a version was added (database.md §3). The first three
// follow the two-phase upload; external registers a remote artifact (no upload).
const (
	SourceWebUpload  = "webUpload"
	SourceORAS       = "oras"
	SourceDockerPush = "dockerPush"
	SourceExternal   = "external"
)

// PublicVisibilityNamespace is the only logical tenant scope where
// visibility='public' is accepted; design §3 + database.md §3.1.
const PublicVisibilityNamespace = "default"
