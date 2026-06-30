package validate_test

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisml "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
	"github.com/axisml/axisml/axisml-system/tenant-operator/internal/validate"
)

func validSpec() *axisml.TenantSpec {
	return &axisml.TenantSpec{
		Namespace: axisml.NamespaceSpec{Name: "tenant-a"},
		Quotas: []axisml.QuotaSpec{
			{
				Pool: "gpu",
				Name: "default",
				Min:  corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
				Max:  corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")},
			},
		},
		InitResources: axisml.InitResources{
			ImagePullSecrets: []axisml.ImagePullSecretSpec{
				{Name: "registry", SourceSecretRef: axisml.SourceSecretRef{Namespace: "axisml-system", Name: "src"}},
			},
			Secrets:    []axisml.SecretSpec{{Name: "creds", SourceSecretRef: axisml.SourceSecretRef{Namespace: "axisml-system", Name: "src"}}},
			ConfigMaps: []axisml.ConfigMapSpec{{Name: "envs", SourceConfigMapRef: axisml.SourceConfigMapRef{Namespace: "axisml-system", Name: "src"}}},
			ServiceAccounts: []axisml.ServiceAccountSpec{
				{
					Name:             "default",
					ImagePullSecrets: []string{"registry"},
					RBAC: &axisml.RBACSpec{
						Rules: []rbacv1.PolicyRule{{Verbs: []string{"get"}, Resources: []string{"pods"}, APIGroups: []string{""}}},
					},
				},
			},
		},
	}
}

func defaultOpts() validate.Options {
	return validate.Options{NamespaceDenylist: validate.DefaultNamespaceDenylist()}
}

func TestValidate_Happy(t *testing.T) {
	if err := validate.Validate(validSpec(), defaultOpts()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidate_RBACRulesGuardrail(t *testing.T) {
	rejected := map[string][]rbacv1.PolicyRule{
		"escalate verb":         {{Verbs: []string{"escalate"}, APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"roles"}}},
		"bind verb":             {{Verbs: []string{"bind"}, APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"clusterroles"}}},
		"impersonate":           {{Verbs: []string{"impersonate"}, APIGroups: []string{""}, Resources: []string{"users"}}},
		"rbac write":            {{Verbs: []string{"create"}, APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"rolebindings"}}},
		"wildcard rbac":         {{Verbs: []string{"*"}, APIGroups: []string{"*"}, Resources: []string{"*"}}},
		"wildcard verb on rbac": {{Verbs: []string{"*"}, APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"roles"}}},
	}
	for name, rules := range rejected {
		t.Run("rejected/"+name, func(t *testing.T) {
			s := validSpec()
			s.InitResources.ServiceAccounts[0].RBAC.Rules = rules
			if err := validate.Validate(s, defaultOpts()); err == nil {
				t.Fatalf("expected rejection for %s, got nil", name)
			}
		})
	}

	allowed := map[string][]rbacv1.PolicyRule{
		"read pods":            {{Verbs: []string{"get", "list", "watch"}, APIGroups: []string{""}, Resources: []string{"pods"}}},
		"write configmaps":     {{Verbs: []string{"create", "update", "delete"}, APIGroups: []string{""}, Resources: []string{"configmaps"}}},
		"read rbac (no write)": {{Verbs: []string{"get", "list"}, APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"roles"}}},
	}
	for name, rules := range allowed {
		t.Run("allowed/"+name, func(t *testing.T) {
			s := validSpec()
			s.InitResources.ServiceAccounts[0].RBAC.Rules = rules
			if err := validate.Validate(s, defaultOpts()); err != nil {
				t.Fatalf("expected %s to pass, got %v", name, err)
			}
		})
	}
}

func TestValidateMeta(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		meta := &metav1.ObjectMeta{
			Name:   "team-a",
			Labels: map[string]string{axisml.LabelTenantID: "uuid-1"},
		}
		if err := validate.ValidateMeta(meta); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
	t.Run("missing tenant-id label", func(t *testing.T) {
		meta := &metav1.ObjectMeta{Name: "team-a"}
		err := validate.ValidateMeta(meta)
		if err == nil || !strings.Contains(err.Error(), "tenant.axisml.io/id") {
			t.Fatalf("expected tenant id label error, got %v", err)
		}
	})
	t.Run("empty tenant-id label", func(t *testing.T) {
		meta := &metav1.ObjectMeta{
			Name:   "team-a",
			Labels: map[string]string{axisml.LabelTenantID: ""},
		}
		err := validate.ValidateMeta(meta)
		if err == nil {
			t.Fatal("expected error for empty tenant-id label")
		}
	})
	t.Run("nil meta", func(t *testing.T) {
		if err := validate.ValidateMeta(nil); err == nil {
			t.Fatal("expected error for nil meta")
		}
	})
}

func TestValidate_NilSpec(t *testing.T) {
	if err := validate.Validate(nil, defaultOpts()); err == nil {
		t.Fatal("expected error for nil spec")
	}
}

func TestValidate_Namespace(t *testing.T) {
	cases := []struct {
		name    string
		ns      string
		wantSub string
	}{
		{"empty", "", "required"},
		{"upper case", "TenantA", "DNS-1123"},
		{"underscore", "tenant_a", "DNS-1123"},
		{"too long", strings.Repeat("a", 64), "exceeds 63"},
		{"system kube-system", "kube-system", "denylist"},
		{"system default", "default", "denylist"},
		{"system axisml-system", "axisml-system", "denylist"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validSpec()
			s.Namespace.Name = tc.ns
			err := validate.Validate(s, defaultOpts())
			if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("expected error containing %q, got %v", tc.wantSub, err)
			}
		})
	}
}

func TestValidate_Namespace_AllowsSharedTenantNamespace(t *testing.T) {
	s := validSpec()
	s.Namespace.Name = "axisml-tenant"
	if err := validate.Validate(s, defaultOpts()); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_Quotas(t *testing.T) {
	t.Run("duplicate pool/name", func(t *testing.T) {
		s := validSpec()
		s.Quotas = append(s.Quotas, s.Quotas[0])
		if err := validate.Validate(s, defaultOpts()); err == nil ||
			!strings.Contains(err.Error(), "duplicates") {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})

	t.Run("missing max", func(t *testing.T) {
		s := validSpec()
		s.Quotas[0].Max = nil
		if err := validate.Validate(s, defaultOpts()); err == nil ||
			!strings.Contains(err.Error(), "max is required") {
			t.Fatalf("expected max-required error, got %v", err)
		}
	})

	t.Run("min exceeds max", func(t *testing.T) {
		s := validSpec()
		s.Quotas[0].Min = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("8")}
		s.Quotas[0].Max = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")}
		if err := validate.Validate(s, defaultOpts()); err == nil ||
			!strings.Contains(err.Error(), "exceeds max") {
			t.Fatalf("expected min>max error, got %v", err)
		}
	})

	t.Run("min has key absent from max", func(t *testing.T) {
		s := validSpec()
		s.Quotas[0].Min = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		}
		s.Quotas[0].Max = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")}
		if err := validate.Validate(s, defaultOpts()); err == nil ||
			!strings.Contains(err.Error(), "missing key") {
			t.Fatalf("expected missing-key error, got %v", err)
		}
	})

	t.Run("negative max", func(t *testing.T) {
		s := validSpec()
		s.Quotas[0].Max = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("-1")}
		if err := validate.Validate(s, defaultOpts()); err == nil ||
			!strings.Contains(err.Error(), "negative") {
			t.Fatalf("expected negative error, got %v", err)
		}
	})

	t.Run("empty pool", func(t *testing.T) {
		s := validSpec()
		s.Quotas[0].Pool = ""
		if err := validate.Validate(s, defaultOpts()); err == nil ||
			!strings.Contains(err.Error(), "pool is required") {
			t.Fatalf("expected pool-required error, got %v", err)
		}
	})
}

func TestValidate_InitResources(t *testing.T) {
	t.Run("duplicate imagePullSecret name", func(t *testing.T) {
		s := validSpec()
		s.InitResources.ImagePullSecrets = append(s.InitResources.ImagePullSecrets,
			axisml.ImagePullSecretSpec{Name: "registry", SourceSecretRef: axisml.SourceSecretRef{Namespace: "ns", Name: "x"}})
		err := validate.Validate(s, defaultOpts())
		if err == nil || !strings.Contains(err.Error(), "duplicated") {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})

	t.Run("imagePullSecret/secret name collision", func(t *testing.T) {
		s := validSpec()
		s.InitResources.Secrets = append(s.InitResources.Secrets,
			axisml.SecretSpec{Name: "registry", SourceSecretRef: axisml.SourceSecretRef{Namespace: "axisml-system", Name: "src"}})
		err := validate.Validate(s, defaultOpts())
		if err == nil || !strings.Contains(err.Error(), "collides with spec.initResources.imagePullSecrets") {
			t.Fatalf("expected collision error, got %v", err)
		}
	})

	t.Run("dangling SA→imagePullSecret reference", func(t *testing.T) {
		s := validSpec()
		s.InitResources.ServiceAccounts[0].ImagePullSecrets = []string{"missing"}
		err := validate.Validate(s, defaultOpts())
		if err == nil || !strings.Contains(err.Error(), "not declared") {
			t.Fatalf("expected dangling-reference error, got %v", err)
		}
	})

	t.Run("invalid roleRef.kind", func(t *testing.T) {
		s := validSpec()
		s.InitResources.ServiceAccounts[0].RBAC.RoleRef = &axisml.RBACRoleRef{Kind: "Random", Name: "x"}
		err := validate.Validate(s, defaultOpts())
		if err == nil || !strings.Contains(err.Error(), "Role or ClusterRole") {
			t.Fatalf("expected roleRef.kind error, got %v", err)
		}
	})

	t.Run("missing roleRef.name", func(t *testing.T) {
		s := validSpec()
		s.InitResources.ServiceAccounts[0].RBAC.RoleRef = &axisml.RBACRoleRef{Kind: "ClusterRole"}
		err := validate.Validate(s, defaultOpts())
		if err == nil || !strings.Contains(err.Error(), "roleRef.name is required") {
			t.Fatalf("expected roleRef.name error, got %v", err)
		}
	})

	t.Run("ClusterRole binding without inline rules is fine", func(t *testing.T) {
		s := validSpec()
		s.InitResources.ServiceAccounts[0].RBAC = &axisml.RBACSpec{
			RoleRef: &axisml.RBACRoleRef{Kind: "ClusterRole", Name: "platform-default"},
		}
		if err := validate.Validate(s, defaultOpts()); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})
}
