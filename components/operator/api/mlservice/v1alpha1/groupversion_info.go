// Package v1alpha1 contains the MLService CRD types served by axisml-operator.
// Contract: docs/system_design/operator/operator.md (MLService controller).
// +kubebuilder:object:generate=true
// +groupName=axisml.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	GroupVersion = schema.GroupVersion{Group: "axisml.io", Version: "v1alpha1"}

	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	AddToScheme = SchemeBuilder.AddToScheme
)
