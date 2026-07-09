package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=rp
//
// ResourcePool is a cluster-scoped admin vocabulary CR maintained by
// cluster-manager. compute-service consumes it via an Informer cache and
// expands (poolName, unitName) into Pod primitives at Job/Service creation
// time. See axisml-system/docs/cluster-manager.md §3.
type ResourcePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ResourcePoolSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type ResourcePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourcePool `json:"items"`
}

// ResourcePoolSpec mirrors the YAML shape in cluster-manager.md §3.1.
// `units` is embedded — no separate CR; it shares the pool's lifecycle.
type ResourcePoolSpec struct {
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
	Units        []ResourceUnit      `json:"units,omitempty"`
}

// ResourceUnit is one entry of spec.units[]. Identified by (poolName, name).
type ResourceUnit struct {
	Name         string              `json:"name"`
	Requests     corev1.ResourceList `json:"requests,omitempty"`
	Limits       corev1.ResourceList `json:"limits,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Annotations  map[string]string   `json:"annotations,omitempty"`
}

// DeepCopyInto / DeepCopyObject are hand-written (mirrors what
// controller-tools would generate).
func (in *ResourcePool) DeepCopyInto(out *ResourcePool) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

func (in *ResourcePool) DeepCopy() *ResourcePool {
	if in == nil {
		return nil
	}
	out := new(ResourcePool)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourcePool) DeepCopyObject() runtime.Object { return in.DeepCopy() }

func (in *ResourcePoolList) DeepCopyInto(out *ResourcePoolList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		l := make([]ResourcePool, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&l[i])
		}
		out.Items = l
	}
}

func (in *ResourcePoolList) DeepCopy() *ResourcePoolList {
	if in == nil {
		return nil
	}
	out := new(ResourcePoolList)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourcePoolList) DeepCopyObject() runtime.Object { return in.DeepCopy() }

func (in *ResourcePoolSpec) DeepCopyInto(out *ResourcePoolSpec) {
	*out = *in
	if in.NodeSelector != nil {
		m := make(map[string]string, len(in.NodeSelector))
		for k, v := range in.NodeSelector {
			m[k] = v
		}
		out.NodeSelector = m
	}
	if in.Tolerations != nil {
		l := make([]corev1.Toleration, len(in.Tolerations))
		for i := range in.Tolerations {
			in.Tolerations[i].DeepCopyInto(&l[i])
		}
		out.Tolerations = l
	}
	if in.Units != nil {
		l := make([]ResourceUnit, len(in.Units))
		for i := range in.Units {
			in.Units[i].DeepCopyInto(&l[i])
		}
		out.Units = l
	}
}

func (in *ResourceUnit) DeepCopyInto(out *ResourceUnit) {
	*out = *in
	if in.Requests != nil {
		m := make(corev1.ResourceList, len(in.Requests))
		for k, v := range in.Requests {
			m[k] = v.DeepCopy()
		}
		out.Requests = m
	}
	if in.Limits != nil {
		m := make(corev1.ResourceList, len(in.Limits))
		for k, v := range in.Limits {
			m[k] = v.DeepCopy()
		}
		out.Limits = m
	}
	if in.NodeSelector != nil {
		m := make(map[string]string, len(in.NodeSelector))
		for k, v := range in.NodeSelector {
			m[k] = v
		}
		out.NodeSelector = m
	}
	if in.Annotations != nil {
		m := make(map[string]string, len(in.Annotations))
		for k, v := range in.Annotations {
			m[k] = v
		}
		out.Annotations = m
	}
}

func init() {
	addKnownTypes(&ResourcePool{}, &ResourcePoolList{})
}
