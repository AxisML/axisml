package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticQuota sets elastic quota restrictions per namespace.
type ElasticQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the Min and Max for Quota.
	// +optional
	Spec ElasticQuotaSpec `json:"spec,omitempty"`

	// Status reflects the observed usage.
	// +optional
	Status ElasticQuotaStatus `json:"status,omitempty"`
}

// ElasticQuotaSpec defines the Min and Max for Quota.
type ElasticQuotaSpec struct {
	// Min is the set of desired guaranteed limits for each named resource.
	// +optional
	Min corev1.ResourceList `json:"min,omitempty"`

	// Max is the set of desired max limits for each named resource.
	// +optional
	Max corev1.ResourceList `json:"max,omitempty"`
}

// ElasticQuotaStatus defines the observed use.
type ElasticQuotaStatus struct {
	// Used is the current observed total usage of the resource in the namespace.
	// +optional
	Used corev1.ResourceList `json:"used,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticQuotaList is a list of ElasticQuota items.
type ElasticQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of ElasticQuota objects.
	Items []ElasticQuota `json:"items"`
}
