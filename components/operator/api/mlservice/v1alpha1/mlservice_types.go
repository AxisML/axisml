package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Phase enumerates the four states declared in mlservice-operator.md §4.
type Phase string

const (
	PhasePending  Phase = "Pending"
	PhaseReady    Phase = "Ready"
	PhaseDegraded Phase = "Degraded"
	PhaseFailed   Phase = "Failed"
)

// Backend selects the (name, engine) tuple that routes to a Handler.
// Mutability: immutable after creation (mlservice-operator.md §3.3).
type Backend struct {
	Name   string                `json:"name"`
	Engine string                `json:"engine"`
	Config *runtime.RawExtension `json:"config,omitempty"`
}

// Scheduling carries the scheduling domain Compute synthesises from
// Quota / ResourcePool / ResourceUnit.
type Scheduling struct {
	Quota         string              `json:"quota"`
	PriorityClass string              `json:"priorityClass,omitempty"`
	NodeSelector  map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations   []corev1.Toleration `json:"tolerations,omitempty"`
}

// ModelRef points at an Artifacts model version.
type ModelRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// PodPort mirrors the small subset of corev1.ContainerPort the spec exposes.
type PodPort struct {
	Name          string          `json:"name"`
	ContainerPort int32           `json:"containerPort"`
	Protocol      corev1.Protocol `json:"protocol,omitempty"`
}

// PodTemplate exposes the limited PodSpec subset declared in §3.2.
// We deliberately do NOT embed corev1.PodSpec: hiding the full PodSpec is a
// design constraint (§3.1).
type PodTemplate struct {
	Image           string                      `json:"image"`
	ImagePullPolicy corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	Command         []string                    `json:"command,omitempty"`
	Args            []string                    `json:"args,omitempty"`
	Env             []corev1.EnvVar             `json:"env,omitempty"`
	EnvFrom         []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	WorkingDir      string                      `json:"workingDir,omitempty"`
	Ports           []PodPort                   `json:"ports,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
}

// RoleSpec is one execution role (predictor / transformer / explainer / ...).
// Single-role handlers (native deployment / statefulset) only allow name=predictor.
type RoleSpec struct {
	Name     string      `json:"name"`
	Replicas int32       `json:"replicas"`
	Template PodTemplate `json:"template"`
}

// RunPolicy lifecycle controls. Service has no suspend / activeDeadline /
// ttlSecondsAfterFinished / backoffLimit (§3.1).
type RunPolicy struct {
	ProgressDeadlineSeconds *int32 `json:"progressDeadlineSeconds,omitempty"`
}

// RouteAuthType controls SecurityPolicy generation.
type RouteAuthType string

const (
	RouteAuthNone   RouteAuthType = "none"
	RouteAuthJWT    RouteAuthType = "jwt"
	RouteAuthAPIKey RouteAuthType = "apiKey"
)

// RouteAuthJWTConfig is required when type=jwt.
type RouteAuthJWTConfig struct {
	Issuer  string `json:"issuer"`
	JwksURI string `json:"jwksUri"`
}

// RouteAuthAPIKeyConfig is required when type=apiKey.
type RouteAuthAPIKeyConfig struct {
	SecretRef corev1.LocalObjectReference `json:"secretRef"`
}

// RouteAuth maps to gateway.envoyproxy.io SecurityPolicy.
type RouteAuth struct {
	Type   RouteAuthType          `json:"type,omitempty"`
	JWT    *RouteAuthJWTConfig    `json:"jwt,omitempty"`
	APIKey *RouteAuthAPIKeyConfig `json:"apiKey,omitempty"`
}

// RouteRateLimit maps to gateway.envoyproxy.io BackendTrafficPolicy.rateLimit.
type RouteRateLimit struct {
	RequestsPerSecond int32 `json:"requestsPerSecond"`
	Burst             int32 `json:"burst,omitempty"`
}

// Route describes the optional external entrypoint (HTTPRoute + policies).
// Mutability: the whole block is immutable after creation (§3.3).
type Route struct {
	Enabled    bool            `json:"enabled"`
	TargetRole string          `json:"targetRole,omitempty"`
	PortName   string          `json:"portName,omitempty"`
	Hostname   string          `json:"hostname,omitempty"`
	Path       string          `json:"path,omitempty"`
	Auth       *RouteAuth      `json:"auth,omitempty"`
	RateLimit  *RouteRateLimit `json:"rateLimit,omitempty"`
	Timeout    string          `json:"timeout,omitempty"`
}

// MLServiceSpec is the complete spec contract. Mutability rules are enforced
// in the dispatcher reconciler (only roles[*].replicas may change after create).
type MLServiceSpec struct {
	Backend    Backend    `json:"backend"`
	Scheduling Scheduling `json:"scheduling"`
	ModelRef   ModelRef   `json:"modelRef"`
	Roles      []RoleSpec `json:"roles"`
	RunPolicy  RunPolicy  `json:"runPolicy,omitempty"`
	Route      *Route     `json:"route,omitempty"`
}

// RoleStatus aggregates per-role replica counts for observability (§4).
// Compute does not consume this field.
type RoleStatus struct {
	Name          string `json:"name"`
	Replicas      int32  `json:"replicas"`
	ReadyReplicas int32  `json:"readyReplicas"`
}

// MLServiceStatus is operator-owned. Compute may only read this subresource.
type MLServiceStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Phase              Phase              `json:"phase,omitempty"`
	Message            string             `json:"message,omitempty"`
	Endpoint           string             `json:"endpoint,omitempty"`
	ReadyReplicas      int32              `json:"readyReplicas,omitempty"`
	Selector           string             `json:"selector,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	Roles              []RoleStatus       `json:"roles,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=mls

// MLService is the Schema for the mlservices API.
type MLService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MLServiceSpec   `json:"spec,omitempty"`
	Status MLServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MLServiceList contains a list of MLService.
type MLServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MLService `json:"items"`
}

// LabelServiceID is the stable orphan-detection anchor written by Compute.
const (
	LabelServiceID = "axisml.io/service-id"
	LabelTenant    = "axisml.io/tenant"
	LabelQuota     = "axisml.io/quota"
	LabelRole      = "axisml.io/role"

	LabelKoordQuotaName = "quota.scheduling.koordinator.sh/name"

	SchedulerName = "koord-scheduler"

	DefaultRoleName = "predictor"
)

func init() {
	SchemeBuilder.Register(&MLService{}, &MLServiceList{})
}
