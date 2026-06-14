// Package v1alpha1 contains the MLTrafficPolicy CRD types served by
// axisml-operator. Contract: docs/system_design/components/compute-operator.md
// §4.3 (MLTrafficPolicy controller).
// +kubebuilder:object:generate=true
// +groupName=axisml.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupVersion = schema.GroupVersion{Group: "axisml.io", Version: "v1alpha1"}

	SchemeBuilder = &runtime.SchemeBuilder{}

	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes registers the given Kinds with this group/version.
func addKnownTypes(objs ...runtime.Object) {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, objs...)
		metav1.AddToGroupVersion(s, GroupVersion)
		return nil
	})
}
