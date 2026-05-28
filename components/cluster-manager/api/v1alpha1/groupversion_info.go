// Package v1alpha1 contains the ResourcePool CRD types (axisml.io/v1alpha1).
// The ResourcePool is owned by cluster-manager (REST writer) and consumed
// by compute-service via a SharedInformer cache (read-only) per the
// docs/system_design/components/cluster-manager.md design.
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

func addKnownTypes(objs ...runtime.Object) {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, objs...)
		metav1.AddToGroupVersion(s, GroupVersion)
		return nil
	})
}
