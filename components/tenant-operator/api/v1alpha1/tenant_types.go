package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantPhase enumerates the operator-side high-level state of a Tenant.
// Compute maps Failed → Suspended (with message); see design §4.
type TenantPhase string

const (
	TenantPhaseActive    TenantPhase = "Active"
	TenantPhaseSuspended TenantPhase = "Suspended"
	TenantPhaseFailed    TenantPhase = "Failed"
)

// Well-known label keys applied by tenant-operator.
const (
	LabelTenantID               = "axisml.io/tenant-id"
	LabelManagedBy              = "axisml.io/managed-by"
	ManagedByValue              = "tenant-operator"
	ConditionNamespaceReady     = "NamespaceReady"
	ConditionQuotasReady        = "QuotasReady"
	ConditionInitResourcesReady = "InitResourcesReady"
	ConditionSuspended          = "Suspended"
	ConditionFailed             = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=tnt
// +kubebuilder:subresource:status

// Tenant is a cluster-scoped CR managed by AxisML Compute Service and reconciled by
// tenant-operator into Namespace + ElasticQuota + per-tenant init resources.
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TenantList is the API list type for Tenant.
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

// TenantSpec mirrors design §3.2.
type TenantSpec struct {
	DisplayName   string            `json:"displayName,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Namespace     NamespaceSpec     `json:"namespace"`
	Quotas        []QuotaSpec       `json:"quotas,omitempty"`
	InitResources InitResources     `json:"initResources,omitempty"`
	Suspended     bool              `json:"suspended,omitempty"`
}

// NamespaceSpec describes the target Namespace; spec.namespace.name is immutable.
type NamespaceSpec struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// QuotaSpec is rendered 1:1 to a sigs.k8s.io scheduler-plugins ElasticQuota CR.
type QuotaSpec struct {
	Pool string              `json:"pool"`
	Name string              `json:"name"`
	Min  corev1.ResourceList `json:"min,omitempty"`
	Max  corev1.ResourceList `json:"max"`
}

// InitResources groups the four kinds of per-tenant initialization resources.
type InitResources struct {
	ImagePullSecrets []ImagePullSecretSpec `json:"imagePullSecrets,omitempty"`
	Secrets          []SecretSpec          `json:"secrets,omitempty"`
	ConfigMaps       []ConfigMapSpec       `json:"configMaps,omitempty"`
	ServiceAccounts  []ServiceAccountSpec  `json:"serviceAccounts,omitempty"`
}

// SourceSecretRef references a controlled Secret to copy data from.
type SourceSecretRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// SourceConfigMapRef references a controlled ConfigMap to copy data from.
type SourceConfigMapRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type ImagePullSecretSpec struct {
	Name            string          `json:"name"`
	SourceSecretRef SourceSecretRef `json:"sourceSecretRef"`
}

type SecretSpec struct {
	Name            string            `json:"name"`
	Type            corev1.SecretType `json:"type,omitempty"`
	SourceSecretRef SourceSecretRef   `json:"sourceSecretRef"`
}

type ConfigMapSpec struct {
	Name               string             `json:"name"`
	SourceConfigMapRef SourceConfigMapRef `json:"sourceConfigMapRef"`
}

type ServiceAccountSpec struct {
	Name             string    `json:"name"`
	ImagePullSecrets []string  `json:"imagePullSecrets,omitempty"`
	RBAC             *RBACSpec `json:"rbac,omitempty"`
}

type RBACSpec struct {
	Rules   []rbacv1.PolicyRule `json:"rules,omitempty"`
	RoleRef *RBACRoleRef        `json:"roleRef,omitempty"`
}

type RBACRoleRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// TenantStatus mirrors design §4.
type TenantStatus struct {
	ObservedGeneration int64               `json:"observedGeneration,omitempty"`
	Phase              TenantPhase         `json:"phase,omitempty"`
	Message            string              `json:"message,omitempty"`
	NamespaceReady     bool                `json:"namespaceReady,omitempty"`
	Quotas             []QuotaStatus       `json:"quotas,omitempty"`
	InitResources      InitResourcesStatus `json:"initResources,omitempty"`
	Conditions         []metav1.Condition  `json:"conditions,omitempty"`
}

type QuotaStatus struct {
	Pool    string              `json:"pool"`
	Name    string              `json:"name"`
	Ready   bool                `json:"ready"`
	Used    corev1.ResourceList `json:"used,omitempty"`
	Message string              `json:"message,omitempty"`
}

type InitResourcesStatus struct {
	ImagePullSecrets []InitResourceItemStatus `json:"imagePullSecrets,omitempty"`
	Secrets          []InitResourceItemStatus `json:"secrets,omitempty"`
	ConfigMaps       []InitResourceItemStatus `json:"configMaps,omitempty"`
	ServiceAccounts  []InitResourceItemStatus `json:"serviceAccounts,omitempty"`
}

type InitResourceItemStatus struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

func init() {
	addKnownTypes(&Tenant{}, &TenantList{})
}
