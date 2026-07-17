package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantPhase enumerates the operator-side high-level state of a Tenant.
// Per design §5.2, phase is Active when all sub-reconcilers are ready,
// Failed on a non-transient critical failure, otherwise the previous phase
// is preserved (initial state is empty until first successful pass).
type TenantPhase string

const (
	TenantPhaseActive TenantPhase = "Active"
	TenantPhaseFailed TenantPhase = "Failed"
)

// ElasticQuotaName returns the scheduler-facing ElasticQuota name for one
// tenant and resource pool. There is exactly one quota per (tenant, pool), so
// callers never select or supply a separate quota name.
func ElasticQuotaName(tenantName, poolName string) string {
	return fmt.Sprintf("axisml-%s-%s", tenantName, poolName)
}

// Well-known label keys applied by tenant-operator.
const (
	LabelTenantID               = "tenant.axisml.io/id"
	LabelManagedBy              = "tenant.axisml.io/managed-by"
	ManagedByValue              = "tenant-operator"
	ConditionNamespaceReady     = "NamespaceReady"
	ConditionQuotasReady        = "QuotasReady"
	ConditionInitResourcesReady = "InitResourcesReady"
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

// TenantSpec mirrors the new tenant-operator design (§3 / §6 — namespace +
// quotas[] + initResources). Per the design, display_name / description /
// labels / annotations are PG-only and never propagated to the CR.
type TenantSpec struct {
	Namespace     NamespaceSpec `json:"namespace"`
	Quotas        []QuotaSpec   `json:"quotas,omitempty"`
	InitResources InitResources `json:"initResources,omitempty"`
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
	Min  corev1.ResourceList `json:"min,omitempty"`
	Max  corev1.ResourceList `json:"max"`
}

// InitResources groups the per-tenant initialization resources the operator
// seeds into the tenant namespace: credential/RBAC objects copied from a source,
// plus predefined data Volumes ensured to exist for workloads to mount.
type InitResources struct {
	ImagePullSecrets []ImagePullSecretSpec `json:"imagePullSecrets,omitempty"`
	Secrets          []SecretSpec          `json:"secrets,omitempty"`
	ConfigMaps       []ConfigMapSpec       `json:"configMaps,omitempty"`
	ServiceAccounts  []ServiceAccountSpec  `json:"serviceAccounts,omitempty"`
	Volumes          []VolumeSpec          `json:"volumes,omitempty"`
}

// VolumeSpec declares a predefined data volume the tenant guarantees exists.
// The operator ensures it as a managed PersistentVolumeClaim (a managed Docker
// volume in a single-host deployment) in the tenant namespace, named exactly Name, so a workload
// can mount it by claim name without a separate provisioning step. Ensure is
// idempotent and non-destructive: the PVC is created if absent and never
// shrunk, relabeled away, or deleted by the operator — Size is the initial
// request only, and ongoing lifecycle (expand/delete) belongs to the
// data-volume catalog (cluster-manager). Removing a VolumeSpec merely stops the
// existence guarantee; it never deletes data.
type VolumeSpec struct {
	Name         string                              `json:"name"`
	Size         string                              `json:"size,omitempty"`
	StorageClass string                              `json:"storageClass,omitempty"`
	AccessModes  []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
	Description  string                              `json:"description,omitempty"`
	// HostPath, when set, makes this a host-backed volume instead of a managed
	// PVC: a workload that mounts it by name gets the host directory bind-mounted.
	// Supported ONLY in the single-host standalone runtime (rendered as a Docker bind
	// mount); the multi-tenant Standard operator REJECTS it — a hostPath breaks
	// tenant isolation, pins the workload to a node, and has no cluster-wide
	// "ensure exists" semantics. Mutually exclusive with size/storageClass.
	HostPath string `json:"hostPath,omitempty"`
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
	Ready   bool                `json:"ready"`
	Used    corev1.ResourceList `json:"used,omitempty"`
	Message string              `json:"message,omitempty"`
}

type InitResourcesStatus struct {
	ImagePullSecrets []InitResourceItemStatus `json:"imagePullSecrets,omitempty"`
	Secrets          []InitResourceItemStatus `json:"secrets,omitempty"`
	ConfigMaps       []InitResourceItemStatus `json:"configMaps,omitempty"`
	ServiceAccounts  []InitResourceItemStatus `json:"serviceAccounts,omitempty"`
	Volumes          []InitResourceItemStatus `json:"volumes,omitempty"`
}

type InitResourceItemStatus struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

func init() {
	addKnownTypes(&Tenant{}, &TenantList{})
}
