package tenant

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// ToCR materialises a Tenant CR (cluster-scoped) from a PG row. compute is
// the sole writer of the CR spec; tenant-operator writes status (read back
// via the Informer). Per design §6, display_name / description / labels /
// annotations stay PG-only and are NOT propagated to the CR.
func ToCR(t *Tenant) (*tenantv1alpha1.Tenant, error) {
	var spec SpecJSON
	if len(t.Spec) > 0 {
		if err := json.Unmarshal(t.Spec, &spec); err != nil {
			return nil, err
		}
	}
	cr := &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: t.Name,
			Labels: map[string]string{
				tenantv1alpha1.LabelTenantID: t.ID.String(),
			},
		},
		Spec: tenantv1alpha1.TenantSpec{
			Namespace: tenantv1alpha1.NamespaceSpec{
				Name:        spec.Namespace.Name,
				Labels:      spec.Namespace.Labels,
				Annotations: spec.Namespace.Annotations,
			},
		},
	}
	for _, q := range spec.Quotas {
		cr.Spec.Quotas = append(cr.Spec.Quotas, tenantv1alpha1.QuotaSpec{
			Pool: q.Pool,
			Name: q.Name,
			Min:  toResourceList(q.Min),
			Max:  toResourceList(q.Max),
		})
	}
	if spec.InitResources != nil {
		cr.Spec.InitResources = toInitResources(*spec.InitResources)
	}
	return cr, nil
}

func toResourceList(in map[string]string) corev1.ResourceList {
	if len(in) == 0 {
		return nil
	}
	out := corev1.ResourceList{}
	for k, v := range in {
		q, err := resource.ParseQuantity(v)
		if err != nil {
			// Skip malformed values; the API layer should validate before
			// commit, so a malformed entry here is a hard bug, not user data.
			continue
		}
		out[corev1.ResourceName(k)] = q
	}
	return out
}

func toInitResources(in InitResources) tenantv1alpha1.InitResources {
	out := tenantv1alpha1.InitResources{}
	for _, ips := range in.ImagePullSecrets {
		out.ImagePullSecrets = append(out.ImagePullSecrets, tenantv1alpha1.ImagePullSecretSpec{
			Name: ips.Name,
			SourceSecretRef: tenantv1alpha1.SourceSecretRef{
				Namespace: ips.SourceSecretRef.Namespace,
				Name:      ips.SourceSecretRef.Name,
			},
		})
	}
	for _, sec := range in.Secrets {
		secret := tenantv1alpha1.SecretSpec{
			Name: sec.Name,
			SourceSecretRef: tenantv1alpha1.SourceSecretRef{
				Namespace: sec.SourceSecretRef.Namespace,
				Name:      sec.SourceSecretRef.Name,
			},
		}
		if sec.Type != "" {
			secret.Type = corev1.SecretType(sec.Type)
		}
		out.Secrets = append(out.Secrets, secret)
	}
	for _, cm := range in.ConfigMaps {
		out.ConfigMaps = append(out.ConfigMaps, tenantv1alpha1.ConfigMapSpec{
			Name: cm.Name,
			SourceConfigMapRef: tenantv1alpha1.SourceConfigMapRef{
				Namespace: cm.SourceConfigMapRef.Namespace,
				Name:      cm.SourceConfigMapRef.Name,
			},
		})
	}
	for _, sa := range in.ServiceAccounts {
		out.ServiceAccounts = append(out.ServiceAccounts, tenantv1alpha1.ServiceAccountSpec{
			Name:             sa.Name,
			ImagePullSecrets: sa.ImagePullSecrets,
		})
	}
	return out
}
