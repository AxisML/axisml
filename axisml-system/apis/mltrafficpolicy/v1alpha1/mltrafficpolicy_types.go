package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Phase enumerates the four states declared in compute-operator.md §4.3.2.
type Phase string

const (
	PhasePending  Phase = "Pending"
	PhaseReady    Phase = "Ready"
	PhaseDegraded Phase = "Degraded"
	PhaseFailed   Phase = "Failed"
)

// TrafficMode is the policy mode. Immutable after creation
// (compute-service.md §4.5).
type TrafficMode string

const (
	// TrafficModeWeighted distributes by explicit per-backend weights (Σ=100).
	TrafficModeWeighted TrafficMode = "weighted"
	// TrafficModeCanary runs one stable baseline + one canary; the canary
	// percentage is the canary backend weight, stable = 100−p.
	TrafficModeCanary TrafficMode = "canary"
	// TrafficModeBlueGreen flips all traffic between exactly two backends.
	TrafficModeBlueGreen TrafficMode = "bluegreen"
)

// BackendRole tags a member's role. For canary mode exactly one `stable`
// (current baseline) + one `canary`; for bluegreen `blue` / `green`; for
// weighted it is left empty. The current canary baseline is the member whose
// role is `stable` — there is no separate baseline pointer.
type BackendRole string

const (
	RoleStable BackendRole = "stable"
	RoleCanary BackendRole = "canary"
	RoleBlue   BackendRole = "blue"
	RoleGreen  BackendRole = "green"
)

// Backend selects the (name, engine) tuple that routes to a Handler.
// compute-service derives it from the member services' backend family:
// (native,httproute) or (kserve,inference). Immutable after creation.
type Backend struct {
	Name   string                `json:"name"`
	Engine string                `json:"engine"`
	Config *runtime.RawExtension `json:"config,omitempty"`
}

// EndpointAuthType controls SecurityPolicy generation on the derived route.
type EndpointAuthType string

const (
	EndpointAuthNone EndpointAuthType = "none"
	EndpointAuthJWT  EndpointAuthType = "jwt"
)

// EndpointAuthJWTConfig is required when type=jwt. The data-plane gateway
// verifies the access JWT (aud=axisml-inference) issued by Platform.
type EndpointAuthJWTConfig struct {
	Issuer   string `json:"issuer,omitempty"`
	JwksURI  string `json:"jwksUri,omitempty"`
	Audience string `json:"audience,omitempty"`
}

// EndpointAuth maps to gateway.envoyproxy.io SecurityPolicy.
type EndpointAuth struct {
	Type EndpointAuthType       `json:"type,omitempty"`
	JWT  *EndpointAuthJWTConfig `json:"jwt,omitempty"`
}

// Endpoint is the stable external entrypoint. Immutable after creation —
// changing it means recreating the policy.
type Endpoint struct {
	Path     string        `json:"path,omitempty"`
	Hostname string        `json:"hostname,omitempty"`
	Auth     *EndpointAuth `json:"auth,omitempty"`
}

// BackendMember references one member MLService (same namespace, kind=service)
// and its weight. `weight` is the only field mutable after creation (via
// /split, /promote, /rollback); `role` additionally flips on canary promote.
type BackendMember struct {
	ServiceName string      `json:"serviceName"`
	Role        BackendRole `json:"role,omitempty"`
	Weight      int32       `json:"weight"`
}

// MLTrafficPolicySpec is the complete spec contract. Mutability: only
// backends[*].weight (and, on canary promote, backends[*].role) may change
// after create; backend / endpoint / mode are immutable.
type MLTrafficPolicySpec struct {
	Backend  Backend         `json:"backend"`
	Mode     TrafficMode     `json:"mode"`
	Endpoint Endpoint        `json:"endpoint"`
	Backends []BackendMember `json:"backends"`
}

// BackendStatus reports the effective weight and readiness of a member,
// feeding Platform's canary-health view.
type BackendStatus struct {
	ServiceName string `json:"serviceName"`
	Weight      int32  `json:"weight"`
	Ready       bool   `json:"ready"`
}

// MLTrafficPolicyStatus is operator-owned. Compute may only read this
// subresource.
type MLTrafficPolicyStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Phase              Phase              `json:"phase,omitempty"`
	Message            string             `json:"message,omitempty"`
	Endpoint           string             `json:"endpoint,omitempty"`
	Backends           []BackendStatus    `json:"backends,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=mltp

// MLTrafficPolicy is the Schema for the mltrafficpolicies API.
type MLTrafficPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MLTrafficPolicySpec   `json:"spec,omitempty"`
	Status MLTrafficPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MLTrafficPolicyList contains a list of MLTrafficPolicy.
type MLTrafficPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MLTrafficPolicy `json:"items"`
}

const (
	// LabelTrafficPolicyID is the stable orphan-detection anchor written by
	// compute-service (the PG row UUID).
	LabelTrafficPolicyID = "compute.axisml.io/traffic-policy-id"
	LabelTenant          = "tenant.axisml.io/name"

	// BackendKindNative routes to the weighted-HTTPRoute handler;
	// BackendKindKServe routes to the InferenceService canary handler.
	BackendKindNative = "native"
	BackendKindKServe = "kserve"

	EngineHTTPRoute = "httproute"
	EngineInference = "inference"

	// InferenceAudience is the access-JWT audience the data-plane gateway
	// verifies for online-service traffic.
	InferenceAudience = "axisml-inference"
)

func init() {
	addKnownTypes(&MLTrafficPolicy{}, &MLTrafficPolicyList{})
}
