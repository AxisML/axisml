// Package v1alpha1 holds the in-repo Go types for the upstream
// scheduler-plugins ElasticQuota CRD (API group scheduling.x-k8s.io),
// consumed by the AxisML self-built scheduler (axisml-scheduler).
//
// These are wire-compatible copies of the scheduler-plugins types — the
// scheduler binary lives in its own module pinned to a compatible k8s line and
// interoperates with the operators via the CRD + labels, not Go types — so the
// operators carry their own minimal copy here rather than depend on a
// scheduler-plugins release (which lags the k8s version this module targets).
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName is the upstream scheduler-plugins API group.
const GroupName = "scheduling.x-k8s.io"

var (
	// GroupVersion is the group/version for the ElasticQuota CRD.
	GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

	// SchemeBuilder registers the types with a runtime.Scheme. Uses the
	// apimachinery builder (not controller-runtime's) so this api package keeps
	// minimal dependencies, per the canonical k8s api-package pattern.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the registered types to a scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion, &ElasticQuota{}, &ElasticQuotaList{})
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}
