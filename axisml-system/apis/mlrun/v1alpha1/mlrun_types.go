package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/axisml/axisml/axisml-system/apis/pkg/workloadconfig"
)

// MLRunPhase enumerates the four-state phase per design §4. Frozen.
// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
type MLRunPhase string

const (
	PhasePending   MLRunPhase = "Pending"
	PhaseRunning   MLRunPhase = "Running"
	PhaseSucceeded MLRunPhase = "Succeeded"
	PhaseFailed    MLRunPhase = "Failed"
)

// BackendSpec selects the (backend, engine) handler. Both fields are
// immutable after creation; dispatcher rejects mutations and writes
// status.message.
type BackendSpec struct {
	// Name selects the backend family.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Engine selects the concrete handler within the backend.
	// +kubebuilder:validation:MinLength=1
	Engine string `json:"engine"`
	// Config carries handler-specific schemaless configuration.
	// +optional
	Config *runtime.RawExtension `json:"config,omitempty"`
}

// SchedulingSpec is filled by Compute from Quota/Pool/Unit.
type SchedulingSpec struct {
	// Quota is the derived ElasticQuota CR name (axisml-<tenant>-<pool>).
	// +kubebuilder:validation:MinLength=1
	Quota string `json:"quota"`
	// PriorityClass is an optional Kubernetes PriorityClass name.
	// +optional
	PriorityClass string `json:"priorityClass,omitempty"`
	// NodeSelector is merged from ResourcePool + ResourceUnit by Compute.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Tolerations come from the ResourcePool.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// PodTemplateSubset exposes a curated subset of corev1.PodSpec / Container.
type PodTemplateSubset struct {
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`
	// +optional
	// +kubebuilder:validation:Enum=IfNotPresent;Always;Never
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// +optional
	Command []string `json:"command,omitempty"`
	// +optional
	Args []string `json:"args,omitempty"`
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`
	// +optional
	WorkingDir string `json:"workingDir,omitempty"`
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Volumes the role's containers can mount (e.g. a PVC-backed data volume
	// holding the training dataset). Mirrors MLService's template; a mount
	// referencing a volume not declared here is a capability error.
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`
	// VolumeMounts binds declared Volumes into the container filesystem.
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

// RoleSpec carries one role within the job's role topology.
type RoleSpec struct {
	// Name is the role identifier (unique within a single MLRun).
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Replicas is the desired count for this role; 0 disables the role.
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`
	// RestartPolicy applies to the underlying Pod template.
	// +kubebuilder:validation:Enum=OnFailure;Never
	// +kubebuilder:default=OnFailure
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty"`
	// Template is the role's Pod template subset.
	Template PodTemplateSubset `json:"template"`
}

// RunPolicySpec controls lifecycle hooks shared across handlers.
type RunPolicySpec struct {
	// Suspend signals cancellation; handler must pause or clean up.
	// +optional
	Suspend bool `json:"suspend,omitempty"`
	// ActiveDeadlineSeconds is a hard timeout enforced by handlers.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
	// TTLSecondsAfterFinished triggers GC of underlying resources after
	// the job reaches a terminal phase.
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
	// BackoffLimit is the retry budget; semantics vary per handler.
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`
}

// MLRunSpec defines the desired state.
type MLRunSpec struct {
	Backend    BackendSpec    `json:"backend"`
	Scheduling SchedulingSpec `json:"scheduling"`
	// ConfigMaps are created in the MLRun namespace and owned by the MLRun.
	// +optional
	ConfigMaps []workloadconfig.ConfigMap `json:"configMaps,omitempty"`
	// +kubebuilder:validation:MinItems=1
	Roles     []RoleSpec    `json:"roles"`
	RunPolicy RunPolicySpec `json:"runPolicy,omitempty"`
}

// RoleStatus is per-role replica aggregation surfaced for UI / observability.
// Compute does not consume this; it reads only status.phase.
type RoleStatus struct {
	Name              string `json:"name"`
	Replicas          int32  `json:"replicas"`
	ActiveReplicas    int32  `json:"activeReplicas,omitempty"`
	ReadyReplicas     int32  `json:"readyReplicas,omitempty"`
	SucceededReplicas int32  `json:"succeededReplicas,omitempty"`
	FailedReplicas    int32  `json:"failedReplicas,omitempty"`
}

// MLRunStatus is single-writer: only the operator writes; Compute reads
// only `phase` (and the Suspended condition for cancel propagation).
type MLRunStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	Phase MLRunPhase `json:"phase,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// +optional
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	Roles []RoleStatus `json:"roles,omitempty"`
}

// Standard condition types per design §4.
const (
	ConditionInitialized = "Initialized"
	ConditionScheduled   = "Scheduled"
	ConditionSuspended   = "Suspended"
	ConditionFailed      = "Failed"
)

// CancelRequested is the agreed reason placed on the Suspended condition
// when a handler completes the cancel path. Compute consumes this signal.
const ReasonCancelRequested = "CancelRequested"

// Public label/annotation keys and naming constants. Exported here (rather
// than in internal/labels) so external clients — Compute, e2e tests — can
// reference them without poking through internal/. Mirrors the layout used
// by mlservice-operator's api/v1alpha1.
const (
	LabelRunID          = "compute.axisml.io/run-id"
	LabelTenant         = "tenant.axisml.io/name"
	LabelQuota          = "compute.axisml.io/quota"
	LabelRole           = "compute.axisml.io/role"
	LabelResourcePool   = "resource.axisml.io/pool"
	LabelResourceUnit   = "resource.axisml.io/unit"
	LabelSchedulerQuota = "scheduling.axisml.io/quota"

	AnnotationAppliedSpec = "compute.axisml.io/applied-spec"
	// AnnotationPriority is the public queue priority contract. Compute parses
	// it as a signed int32 when the MLRun is submitted; larger values are
	// considered before smaller values.
	AnnotationPriority = "scheduling.axisml.io/priority"

	SchedulerName = "axisml-scheduler"

	// DefaultRoleName is the role name required by the (native, *) handlers.
	DefaultRoleName = "worker"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=mlj
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Backend",type="string",JSONPath=".spec.backend.name"
// +kubebuilder:printcolumn:name="Engine",type="string",JSONPath=".spec.backend.engine"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// MLRun is the Schema for axisml.io/v1alpha1 MLRuns.
type MLRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MLRunSpec   `json:"spec,omitempty"`
	Status MLRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MLRunList contains a list of MLRun.
type MLRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MLRun `json:"items"`
}

func init() {
	addKnownTypes(&MLRun{}, &MLRunList{})
}
