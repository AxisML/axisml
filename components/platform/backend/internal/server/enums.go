package server

// Enumerated string types. The generator (cmd/openapi-gen) renders each one of
// these either as a referenced component schema (the "named" group, listed in
// the source spec under its own schema name) or as an inline `enum` on the
// field that uses it (the "inline" group). The value slices below are the
// single source of truth for the allowed values.

// ---- Named enum components (referenced via $ref in the source spec) ----

// RoleName is a platform RBAC role.
type RoleName string

// RoleNameValues enumerates RoleName.
var RoleNameValues = []string{"system-admin", "tenant-admin", "user"}

// TenantPhase is the lifecycle phase of a Tenant.
type TenantPhase string

// TenantPhaseValues enumerates TenantPhase.
var TenantPhaseValues = []string{"Creating", "Active", "Failed", "Deleting", "Deleted"}

// WorkspacePhase is derived from compute service phase + replicas.
type WorkspacePhase string

// WorkspacePhaseValues enumerates WorkspacePhase.
var WorkspacePhaseValues = []string{
	"Creating", "Starting", "Running", "Degraded", "Failed", "Stopped", "Deleting", "Deleted", "Pending",
}

// WorkspaceDesiredState is the user-requested run state of a Workspace.
type WorkspaceDesiredState string

// WorkspaceDesiredStateValues enumerates WorkspaceDesiredState.
var WorkspaceDesiredStateValues = []string{"Running", "Stopped"}

// RunPhase is the phase of a Run (compute MLRun).
type RunPhase string

// RunPhaseValues enumerates RunPhase.
var RunPhaseValues = []string{
	"Creating", "Pending", "Running", "Succeeded", "Failed", "Canceling", "Cancelled", "Deleting", "Deleted",
}

// MLServicePhase is the lifecycle phase of an MLService.
type MLServicePhase string

// MLServicePhaseValues enumerates MLServicePhase.
var MLServicePhaseValues = []string{
	"Creating", "Pending", "Ready", "Degraded", "Failed", "Stopped", "Deleting", "Deleted",
}

// MLServiceRouteAuthType selects the auth mode for an MLService route.
type MLServiceRouteAuthType string

// MLServiceRouteAuthTypeValues enumerates MLServiceRouteAuthType.
var MLServiceRouteAuthTypeValues = []string{"none", "jwt", "apiKey"}

// MLServiceMetricName names a queryable MLService metric.
type MLServiceMetricName string

// MLServiceMetricNameValues enumerates MLServiceMetricName.
var MLServiceMetricNameValues = []string{
	"request_rate", "latency", "error_rate", "cpu_util", "mem_util", "gpu_util",
}

// ModelStatus mirrors artifacts ArtifactStatus for kind=model.
type ModelStatus string

// ImageStatus mirrors artifacts ArtifactStatus for kind=image.
type ImageStatus string

// DatasetStatus mirrors artifacts ArtifactStatus for kind=dataset.
type DatasetStatus string

// ArtifactStatusValues enumerates the shared Model/Image/Dataset status set.
var ArtifactStatusValues = []string{"Uploading", "Ready", "Failed", "Deleting", "Deleted"}

// ---- Inline enums (rendered as `enum` on the using field) ----

// HealthState is a per-dependency / overall health value.
type HealthState string

// HealthStateValues enumerates HealthState.
var HealthStateValues = []string{"ok", "degraded", "unavailable"}

// ConditionStatus is a Kubernetes-style condition status.
type ConditionStatus string

// ConditionStatusValues enumerates ConditionStatus.
var ConditionStatusValues = []string{"True", "False", "Unknown"}

// MemberRoleName is the bindable subset of RoleName for tenant membership.
type MemberRoleName string

// MemberRoleNameValues enumerates MemberRoleName (system-admin is excluded).
var MemberRoleNameValues = []string{"tenant-admin", "user"}

// BackendName names a compute backend.
type BackendName string

// BackendNameValues enumerates BackendName.
var BackendNameValues = []string{"native", "kubeflow-trainer", "kserve", "custom"}

// RestartPolicy is a role's pod restart policy.
type RestartPolicy string

// RestartPolicyValues enumerates RestartPolicy.
var RestartPolicyValues = []string{"Never", "OnFailure", "Always"}

// ArtifactKind names the kind of an artifact reference.
type ArtifactKind string

// ArtifactKindValues enumerates ArtifactKind.
var ArtifactKindValues = []string{"image", "model", "dataset"}

// StorageKind names the backing store of an artifact version.
type StorageKind string

// StorageKindValues enumerates StorageKind.
var StorageKindValues = []string{"oci", "s3"}

// DefinitionKind names the kind of a Platform artifact definition.
type DefinitionKind string

// DefinitionKindValues enumerates DefinitionKind.
var DefinitionKindValues = []string{"model", "image", "dataset"}

// Visibility is an artifact definition's visibility scope.
type Visibility string

// VisibilityValues enumerates Visibility.
var VisibilityValues = []string{"tenant", "public"}

// PodPhase is a pod's lifecycle phase.
type PodPhase string

// PodPhaseValues enumerates PodPhase.
var PodPhaseValues = []string{"Pending", "Running", "Succeeded", "Failed", "Unknown"}

// EventType is a Kubernetes event type.
type EventType string

// EventTypeValues enumerates EventType.
var EventTypeValues = []string{"Normal", "Warning"}

// AuditResult is the outcome of an audited action.
type AuditResult string

// AuditResultValues enumerates AuditResult.
var AuditResultValues = []string{"success", "failure"}
