package server

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// ToTenantSpec converts the API request shape into the CRD spec.
// Resource maps come in as map[string]string and are parsed to
// resource.Quantity; malformed values produce zero quantities (the
// tenant-operator's Validate will catch them on reconcile).
func (r CreateTenantRequest) ToTenantSpec() tenantv1alpha1.TenantSpec {
	return tenantv1alpha1.TenantSpec{
		DisplayName:   r.DisplayName,
		Annotations:   r.Annotations,
		Namespace:     toNamespaceSpec(r.Namespace),
		Quotas:        toQuotaSpecs(r.Quotas),
		InitResources: toInitResources(r.Init),
		Suspended:     r.Suspended,
	}
}

func toNamespaceSpec(n NamespaceSpec) tenantv1alpha1.NamespaceSpec {
	return tenantv1alpha1.NamespaceSpec{
		Name:        n.Name,
		Labels:      n.Labels,
		Annotations: n.Annotations,
	}
}

func toQuotaSpecs(qs []QuotaSpec) []tenantv1alpha1.QuotaSpec {
	if len(qs) == 0 {
		return nil
	}
	out := make([]tenantv1alpha1.QuotaSpec, 0, len(qs))
	for _, q := range qs {
		out = append(out, tenantv1alpha1.QuotaSpec{
			Pool: q.Pool,
			Name: q.Name,
			Min:  ParseResourceList(q.Min),
			Max:  ParseResourceList(q.Max),
		})
	}
	return out
}

func toInitResources(in *InitResources) tenantv1alpha1.InitResources {
	if in == nil {
		return tenantv1alpha1.InitResources{}
	}
	out := tenantv1alpha1.InitResources{}
	for _, x := range in.ImagePullSecrets {
		out.ImagePullSecrets = append(out.ImagePullSecrets, tenantv1alpha1.ImagePullSecretSpec{
			Name:            x.Name,
			SourceSecretRef: tenantv1alpha1.SourceSecretRef{Namespace: x.SourceSecretRef.Namespace, Name: x.SourceSecretRef.Name},
		})
	}
	for _, x := range in.Secrets {
		out.Secrets = append(out.Secrets, tenantv1alpha1.SecretSpec{
			Name:            x.Name,
			Type:            corev1.SecretType(x.Type),
			SourceSecretRef: tenantv1alpha1.SourceSecretRef{Namespace: x.SourceSecretRef.Namespace, Name: x.SourceSecretRef.Name},
		})
	}
	for _, x := range in.ConfigMaps {
		out.ConfigMaps = append(out.ConfigMaps, tenantv1alpha1.ConfigMapSpec{
			Name:               x.Name,
			SourceConfigMapRef: tenantv1alpha1.SourceConfigMapRef{Namespace: x.SourceConfigMapRef.Namespace, Name: x.SourceConfigMapRef.Name},
		})
	}
	for _, x := range in.ServiceAccounts {
		var rbac *tenantv1alpha1.RBACSpec
		if x.RBAC != nil {
			rbac = &tenantv1alpha1.RBACSpec{Rules: toPolicyRules(x.RBAC.Rules)}
			if x.RBAC.RoleRef != nil {
				rbac.RoleRef = &tenantv1alpha1.RBACRoleRef{Kind: x.RBAC.RoleRef.Kind, Name: x.RBAC.RoleRef.Name}
			}
		}
		out.ServiceAccounts = append(out.ServiceAccounts, tenantv1alpha1.ServiceAccountSpec{
			Name:             x.Name,
			ImagePullSecrets: x.ImagePullSecrets,
			RBAC:             rbac,
		})
	}
	return out
}

func toPolicyRules(in []map[string]any) []rbacv1.PolicyRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]rbacv1.PolicyRule, 0, len(in))
	for _, raw := range in {
		var pr rbacv1.PolicyRule
		pr.Verbs = strSlice(raw["verbs"])
		pr.APIGroups = strSlice(raw["apiGroups"])
		pr.Resources = strSlice(raw["resources"])
		pr.ResourceNames = strSlice(raw["resourceNames"])
		pr.NonResourceURLs = strSlice(raw["nonResourceURLs"])
		out = append(out, pr)
	}
	return out
}

func strSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if str, ok := s.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

// ParseResourceList turns the API map[string]string shape into a
// corev1.ResourceList. Unparseable values are dropped — tenant-operator's
// Validate surfaces the error on reconcile.
func ParseResourceList(in map[string]string) corev1.ResourceList {
	if len(in) == 0 {
		return nil
	}
	out := make(corev1.ResourceList, len(in))
	for k, v := range in {
		q, err := resource.ParseQuantity(v)
		if err != nil {
			continue
		}
		out[corev1.ResourceName(k)] = q
	}
	return out
}

// FromTenant builds the API response from a CRD object.
func FromTenant(t *tenantv1alpha1.Tenant) TenantResponse {
	return TenantResponse{
		ID:          t.Labels[tenantv1alpha1.LabelTenantID],
		Name:        t.Name,
		DisplayName: t.Spec.DisplayName,
		Namespace: NamespaceSpec{
			Name:        t.Spec.Namespace.Name,
			Labels:      t.Spec.Namespace.Labels,
			Annotations: t.Spec.Namespace.Annotations,
		},
		Quotas:    fromQuotaSpecs(t.Spec.Quotas),
		Init:      fromInitResources(t.Spec.InitResources),
		Suspended: t.Spec.Suspended,
		Status:    fromTenantStatus(t.Status),
		CreatedAt: t.CreationTimestamp.Time,
	}
}

func fromQuotaSpecs(qs []tenantv1alpha1.QuotaSpec) []QuotaSpec {
	if len(qs) == 0 {
		return nil
	}
	out := make([]QuotaSpec, 0, len(qs))
	for _, q := range qs {
		out = append(out, QuotaSpec{
			Pool: q.Pool,
			Name: q.Name,
			Min:  fromResourceList(q.Min),
			Max:  fromResourceList(q.Max),
		})
	}
	return out
}

func fromInitResources(in tenantv1alpha1.InitResources) *InitResources {
	out := &InitResources{}
	for _, x := range in.ImagePullSecrets {
		out.ImagePullSecrets = append(out.ImagePullSecrets, ImagePullSecretSpec{
			Name:            x.Name,
			SourceSecretRef: ObjectRef{Namespace: x.SourceSecretRef.Namespace, Name: x.SourceSecretRef.Name},
		})
	}
	for _, x := range in.Secrets {
		out.Secrets = append(out.Secrets, SecretSpec{
			Name:            x.Name,
			Type:            string(x.Type),
			SourceSecretRef: ObjectRef{Namespace: x.SourceSecretRef.Namespace, Name: x.SourceSecretRef.Name},
		})
	}
	for _, x := range in.ConfigMaps {
		out.ConfigMaps = append(out.ConfigMaps, ConfigMapSpec{
			Name:               x.Name,
			SourceConfigMapRef: ObjectRef{Namespace: x.SourceConfigMapRef.Namespace, Name: x.SourceConfigMapRef.Name},
		})
	}
	for _, x := range in.ServiceAccounts {
		var rbac *RBAC
		if x.RBAC != nil {
			rbac = &RBAC{Rules: fromPolicyRules(x.RBAC.Rules)}
			if x.RBAC.RoleRef != nil {
				rbac.RoleRef = &RoleRef{Kind: x.RBAC.RoleRef.Kind, Name: x.RBAC.RoleRef.Name}
			}
		}
		out.ServiceAccounts = append(out.ServiceAccounts, ServiceAccountSpec{
			Name:             x.Name,
			ImagePullSecrets: x.ImagePullSecrets,
			RBAC:             rbac,
		})
	}
	if len(out.ImagePullSecrets) == 0 && len(out.Secrets) == 0 && len(out.ConfigMaps) == 0 && len(out.ServiceAccounts) == 0 {
		return nil
	}
	return out
}

func fromPolicyRules(in []rbacv1.PolicyRule) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, r := range in {
		out = append(out, map[string]any{
			"verbs":           r.Verbs,
			"apiGroups":       r.APIGroups,
			"resources":       r.Resources,
			"resourceNames":   r.ResourceNames,
			"nonResourceURLs": r.NonResourceURLs,
		})
	}
	return out
}

func fromResourceList(in corev1.ResourceList) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[string(k)] = v.String()
	}
	return out
}

func fromTenantStatus(s tenantv1alpha1.TenantStatus) TenantStatus {
	out := TenantStatus{
		Phase:   string(s.Phase),
		Message: s.Message,
	}
	for _, q := range s.Quotas {
		out.Quotas = append(out.Quotas, QuotaStatus{
			Pool:    q.Pool,
			Name:    q.Name,
			Ready:   q.Ready,
			Used:    fromResourceList(q.Used),
			Message: q.Message,
		})
	}
	return out
}

// ApplyPatchToTenant applies a PatchTenantRequest to a fetched Tenant.
func ApplyPatchToTenant(t *tenantv1alpha1.Tenant, p PatchTenantRequest) {
	if p.DisplayName != nil {
		t.Spec.DisplayName = *p.DisplayName
	}
	if p.Annotations != nil {
		t.Spec.Annotations = p.Annotations
	}
	if p.Init != nil {
		t.Spec.InitResources = toInitResources(p.Init)
	}
}

// EnsureMetadata sets metadata.name and the tenant-id label, generating
// a new UUID if one isn't present on the input request.
func EnsureMetadata(t *tenantv1alpha1.Tenant, name, tenantID string) {
	t.Name = name
	if t.Labels == nil {
		t.Labels = map[string]string{}
	}
	t.Labels[tenantv1alpha1.LabelTenantID] = tenantID
}

// NewTenantList returns a typed empty list (used when client-go can't
// deserialize an empty result back to our Go type without one).
func NewTenantList() *tenantv1alpha1.TenantList {
	return &tenantv1alpha1.TenantList{
		TypeMeta: metav1.TypeMeta{Kind: "TenantList", APIVersion: tenantv1alpha1.GroupVersion.String()},
	}
}
