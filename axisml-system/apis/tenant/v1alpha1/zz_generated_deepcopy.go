// Code generated manually; equivalent to controller-gen output for the
// types declared in this package. Keep in sync with tenant_types.go.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (in *Tenant) DeepCopyInto(out *Tenant) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *Tenant) DeepCopy() *Tenant {
	if in == nil {
		return nil
	}
	out := new(Tenant)
	in.DeepCopyInto(out)
	return out
}

func (in *Tenant) DeepCopyObject() runtime.Object { return in.DeepCopy() }

func (in *TenantList) DeepCopyInto(out *TenantList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]Tenant, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *TenantList) DeepCopy() *TenantList {
	if in == nil {
		return nil
	}
	out := new(TenantList)
	in.DeepCopyInto(out)
	return out
}

func (in *TenantList) DeepCopyObject() runtime.Object { return in.DeepCopy() }

func (in *TenantSpec) DeepCopyInto(out *TenantSpec) {
	*out = *in
	in.Namespace.DeepCopyInto(&out.Namespace)
	if in.Quotas != nil {
		out.Quotas = make([]QuotaSpec, len(in.Quotas))
		for i := range in.Quotas {
			in.Quotas[i].DeepCopyInto(&out.Quotas[i])
		}
	}
	in.InitResources.DeepCopyInto(&out.InitResources)
}

func (in *TenantSpec) DeepCopy() *TenantSpec {
	if in == nil {
		return nil
	}
	out := new(TenantSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *NamespaceSpec) DeepCopyInto(out *NamespaceSpec) {
	*out = *in
	if in.Labels != nil {
		out.Labels = make(map[string]string, len(in.Labels))
		for k, v := range in.Labels {
			out.Labels[k] = v
		}
	}
	if in.Annotations != nil {
		out.Annotations = make(map[string]string, len(in.Annotations))
		for k, v := range in.Annotations {
			out.Annotations[k] = v
		}
	}
}

func (in *NamespaceSpec) DeepCopy() *NamespaceSpec {
	if in == nil {
		return nil
	}
	out := new(NamespaceSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *QuotaSpec) DeepCopyInto(out *QuotaSpec) {
	*out = *in
	if in.Min != nil {
		out.Min = in.Min.DeepCopy()
	}
	if in.Max != nil {
		out.Max = in.Max.DeepCopy()
	}
}

func (in *QuotaSpec) DeepCopy() *QuotaSpec {
	if in == nil {
		return nil
	}
	out := new(QuotaSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *InitResources) DeepCopyInto(out *InitResources) {
	*out = *in
	if in.ImagePullSecrets != nil {
		out.ImagePullSecrets = append([]ImagePullSecretSpec(nil), in.ImagePullSecrets...)
	}
	if in.Secrets != nil {
		out.Secrets = append([]SecretSpec(nil), in.Secrets...)
	}
	if in.ConfigMaps != nil {
		out.ConfigMaps = append([]ConfigMapSpec(nil), in.ConfigMaps...)
	}
	if in.ServiceAccounts != nil {
		out.ServiceAccounts = make([]ServiceAccountSpec, len(in.ServiceAccounts))
		for i := range in.ServiceAccounts {
			in.ServiceAccounts[i].DeepCopyInto(&out.ServiceAccounts[i])
		}
	}
	if in.Volumes != nil {
		out.Volumes = make([]VolumeSpec, len(in.Volumes))
		for i := range in.Volumes {
			in.Volumes[i].DeepCopyInto(&out.Volumes[i])
		}
	}
}

func (in *InitResources) DeepCopy() *InitResources {
	if in == nil {
		return nil
	}
	out := new(InitResources)
	in.DeepCopyInto(out)
	return out
}

func (in *VolumeSpec) DeepCopyInto(out *VolumeSpec) {
	*out = *in
	if in.AccessModes != nil {
		out.AccessModes = append([]corev1.PersistentVolumeAccessMode(nil), in.AccessModes...)
	}
}

func (in *VolumeSpec) DeepCopy() *VolumeSpec {
	if in == nil {
		return nil
	}
	out := new(VolumeSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *ServiceAccountSpec) DeepCopyInto(out *ServiceAccountSpec) {
	*out = *in
	if in.ImagePullSecrets != nil {
		out.ImagePullSecrets = append([]string(nil), in.ImagePullSecrets...)
	}
	if in.RBAC != nil {
		out.RBAC = new(RBACSpec)
		in.RBAC.DeepCopyInto(out.RBAC)
	}
}

func (in *ServiceAccountSpec) DeepCopy() *ServiceAccountSpec {
	if in == nil {
		return nil
	}
	out := new(ServiceAccountSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *RBACSpec) DeepCopyInto(out *RBACSpec) {
	*out = *in
	if in.Rules != nil {
		out.Rules = make([]rbacv1.PolicyRule, len(in.Rules))
		for i := range in.Rules {
			in.Rules[i].DeepCopyInto(&out.Rules[i])
		}
	}
	if in.RoleRef != nil {
		out.RoleRef = new(RBACRoleRef)
		*out.RoleRef = *in.RoleRef
	}
}

func (in *RBACSpec) DeepCopy() *RBACSpec {
	if in == nil {
		return nil
	}
	out := new(RBACSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *TenantStatus) DeepCopyInto(out *TenantStatus) {
	*out = *in
	if in.Quotas != nil {
		out.Quotas = make([]QuotaStatus, len(in.Quotas))
		for i := range in.Quotas {
			in.Quotas[i].DeepCopyInto(&out.Quotas[i])
		}
	}
	in.InitResources.DeepCopyInto(&out.InitResources)
	if in.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(in.Conditions))
		for i := range in.Conditions {
			in.Conditions[i].DeepCopyInto(&out.Conditions[i])
		}
	}
}

func (in *TenantStatus) DeepCopy() *TenantStatus {
	if in == nil {
		return nil
	}
	out := new(TenantStatus)
	in.DeepCopyInto(out)
	return out
}

func (in *QuotaStatus) DeepCopyInto(out *QuotaStatus) {
	*out = *in
	if in.Used != nil {
		out.Used = in.Used.DeepCopy()
	}
}

func (in *QuotaStatus) DeepCopy() *QuotaStatus {
	if in == nil {
		return nil
	}
	out := new(QuotaStatus)
	in.DeepCopyInto(out)
	return out
}

func (in *InitResourcesStatus) DeepCopyInto(out *InitResourcesStatus) {
	*out = *in
	if in.ImagePullSecrets != nil {
		out.ImagePullSecrets = append([]InitResourceItemStatus(nil), in.ImagePullSecrets...)
	}
	if in.Secrets != nil {
		out.Secrets = append([]InitResourceItemStatus(nil), in.Secrets...)
	}
	if in.ConfigMaps != nil {
		out.ConfigMaps = append([]InitResourceItemStatus(nil), in.ConfigMaps...)
	}
	if in.ServiceAccounts != nil {
		out.ServiceAccounts = append([]InitResourceItemStatus(nil), in.ServiceAccounts...)
	}
	if in.Volumes != nil {
		out.Volumes = append([]InitResourceItemStatus(nil), in.Volumes...)
	}
}

func (in *InitResourcesStatus) DeepCopy() *InitResourcesStatus {
	if in == nil {
		return nil
	}
	out := new(InitResourcesStatus)
	in.DeepCopyInto(out)
	return out
}
