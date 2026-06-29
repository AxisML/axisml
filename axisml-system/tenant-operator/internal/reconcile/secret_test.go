package reconcile

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	axisml "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

func TestImagePullSecrets_CreatesDockerConfigJSON(t *testing.T) {
	scheme := newFakeScheme(t)
	src := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: "axisml-system"},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{".dockerconfigjson": []byte(`{}`)},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(src).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.ImagePullSecrets = []axisml.ImagePullSecretSpec{{
		Name:            "registry",
		SourceSecretRef: axisml.SourceSecretRef{Namespace: "axisml-system", Name: "src"},
	}}

	statuses, err := ImagePullSecrets(context.Background(), c, c, scheme, tnt)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !statuses[0].Ready {
		t.Fatalf("not ready: %+v", statuses)
	}

	got := &corev1.Secret{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: PerTenantResourceName("team-a", "registry"),
	}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Type != corev1.SecretTypeDockerConfigJson {
		t.Errorf("type = %s; want dockerconfigjson", got.Type)
	}
	if got.Labels[labelSecretRole] != secretRoleImagePull {
		t.Errorf("role label missing/wrong: %s", got.Labels[labelSecretRole])
	}
	if got.Labels[axisml.LabelTenantID] != "team-a-id" {
		t.Errorf("tenant labels missing")
	}
}

func TestSecrets_DefaultsTypeToOpaque(t *testing.T) {
	scheme := newFakeScheme(t)
	src := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: "axisml-system"},
		Data:       map[string][]byte{"k": []byte("v")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(src).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.Secrets = []axisml.SecretSpec{{
		Name:            "creds",
		SourceSecretRef: axisml.SourceSecretRef{Namespace: "axisml-system", Name: "src"},
	}}

	statuses, err := Secrets(context.Background(), c, c, scheme, tnt)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !statuses[0].Ready {
		t.Fatalf("not ready: %+v", statuses)
	}

	got := &corev1.Secret{}
	_ = c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: PerTenantResourceName("team-a", "creds"),
	}, got)
	if got.Type != corev1.SecretTypeOpaque {
		t.Errorf("type = %s; want Opaque", got.Type)
	}
}

func TestSecrets_TypeMismatchWarning(t *testing.T) {
	scheme := newFakeScheme(t)
	src := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: "axisml-system"},
		Type:       corev1.SecretTypeBasicAuth,
		Data:       map[string][]byte{"username": []byte("u")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(src).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.Secrets = []axisml.SecretSpec{{
		Name:            "creds",
		Type:            corev1.SecretTypeOpaque,
		SourceSecretRef: axisml.SourceSecretRef{Namespace: "axisml-system", Name: "src"},
	}}

	statuses, _ := Secrets(context.Background(), c, c, scheme, tnt)
	if statuses[0].Message == "" {
		t.Error("expected mismatch warning message")
	}
}

func TestSecrets_TypeChangeRecreates(t *testing.T) {
	scheme := newFakeScheme(t)
	tnt := newTenant("team-a")
	src := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: "axisml-system"},
		Data:       map[string][]byte{"k": []byte("v")},
	}
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PerTenantResourceName("team-a", "creds"),
			Namespace: "team-a",
			Labels:    ApplyTenantLabels(tnt, map[string]string{labelSecretRole: secretRoleGeneric}),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(),
				Kind:       "Tenant",
				Name:       tnt.Name,
				UID:        tnt.UID,
				Controller: ptrTrue(),
			}},
		},
		Type: corev1.SecretTypeBasicAuth,
		Data: map[string][]byte{"x": []byte("old")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(src, existing).Build()

	tnt.Spec.InitResources.Secrets = []axisml.SecretSpec{{
		Name:            "creds",
		Type:            corev1.SecretTypeOpaque,
		SourceSecretRef: axisml.SourceSecretRef{Namespace: "axisml-system", Name: "src"},
	}}

	if _, err := Secrets(context.Background(), c, c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	got := &corev1.Secret{}
	_ = c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: PerTenantResourceName("team-a", "creds"),
	}, got)
	if got.Type != corev1.SecretTypeOpaque {
		t.Errorf("type after recreate = %s; want Opaque", got.Type)
	}
	if string(got.Data["k"]) != "v" {
		t.Errorf("data = %v; want from src", got.Data)
	}
}

func TestSecrets_GCsOrphans_ScopedToRole(t *testing.T) {
	scheme := newFakeScheme(t)
	tnt := newTenant("team-a")
	// Two tenant-owned secrets, one of each role.
	pull := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PerTenantResourceName("team-a", "old-pull"),
			Namespace: "team-a",
			Labels:    ApplyTenantLabels(tnt, map[string]string{labelSecretRole: secretRoleImagePull}),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(), Kind: "Tenant",
				Name: tnt.Name, UID: tnt.UID, Controller: ptrTrue(),
			}},
		},
	}
	gen := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PerTenantResourceName("team-a", "old-gen"),
			Namespace: "team-a",
			Labels:    ApplyTenantLabels(tnt, map[string]string{labelSecretRole: secretRoleGeneric}),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(), Kind: "Tenant",
				Name: tnt.Name, UID: tnt.UID, Controller: ptrTrue(),
			}},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pull, gen).Build()

	// Spec has no entries → both should be GC'd, but only when their respective
	// reconcilers run. Run only Secrets() and verify pull is preserved.
	if _, err := Secrets(context.Background(), c, c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	stillPull := &corev1.Secret{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: pull.Name,
	}, stillPull); err != nil {
		t.Errorf("imagepull-role secret should NOT be GC'd by Secrets(): %v", err)
	}

	stillGen := &corev1.Secret{}
	err := c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: gen.Name,
	}, stillGen)
	if err == nil {
		t.Errorf("generic-role orphan should have been GC'd")
	}
}

func TestStripPrefix(t *testing.T) {
	cases := []struct {
		in, prefix, want string
	}{
		{"axisml-tenant-team-a-x", "axisml-tenant-team-a-", "x"},
		{"unrelated", "axisml-tenant-team-a-", "unrelated"},
		{"abc", "", "abc"},
	}
	for _, tc := range cases {
		if got := stripPrefix(tc.in, tc.prefix); got != tc.want {
			t.Errorf("stripPrefix(%q, %q) = %q; want %q", tc.in, tc.prefix, got, tc.want)
		}
	}
}

func TestNameSet(t *testing.T) {
	got := nameSet([]string{"a", "b", "a"})
	if len(got) != 2 {
		t.Errorf("len = %d; want 2 (dedup)", len(got))
	}
	if _, ok := got["a"]; !ok {
		t.Errorf("missing a")
	}
}
