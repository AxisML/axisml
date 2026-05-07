package reconcile

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	axisml "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

func TestServiceAccounts_PlainSA_NoRBAC(t *testing.T) {
	scheme := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.ServiceAccounts = []axisml.ServiceAccountSpec{{Name: "default"}}

	statuses, err := ServiceAccounts(context.Background(), c, scheme, tnt)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !statuses[0].Ready {
		t.Fatalf("not ready: %+v", statuses)
	}

	saName := PerTenantResourceName("team-a", "default")
	got := &corev1.ServiceAccount{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, got); err != nil {
		t.Fatalf("sa missing: %v", err)
	}
	if !hasOwner(got.OwnerReferences, tnt) {
		t.Errorf("missing owner ref")
	}
	// No RBAC → no Role/RoleBinding.
	rb := &rbacv1.RoleBinding{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, rb); !apierrors.IsNotFound(err) {
		t.Errorf("expected no RoleBinding, got %v", err)
	}
}

func TestServiceAccounts_ResolvesImagePullSecretRefs(t *testing.T) {
	scheme := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.ServiceAccounts = []axisml.ServiceAccountSpec{{
		Name:             "default",
		ImagePullSecrets: []string{"registry"},
	}}

	if _, err := ServiceAccounts(context.Background(), c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	got := &corev1.ServiceAccount{}
	_ = c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: PerTenantResourceName("team-a", "default"),
	}, got)
	if len(got.ImagePullSecrets) != 1 {
		t.Fatalf("expected 1 imagePullSecret, got %d", len(got.ImagePullSecrets))
	}
	want := PerTenantResourceName("team-a", "registry")
	if got.ImagePullSecrets[0].Name != want {
		t.Errorf("imagePullSecret = %s; want %s", got.ImagePullSecrets[0].Name, want)
	}
}

func TestServiceAccounts_RBACInline_CreatesRoleAndBinding(t *testing.T) {
	scheme := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.ServiceAccounts = []axisml.ServiceAccountSpec{{
		Name: "default",
		RBAC: &axisml.RBACSpec{
			Rules: []rbacv1.PolicyRule{{Verbs: []string{"get"}, APIGroups: []string{""}, Resources: []string{"pods"}}},
		},
	}}

	if _, err := ServiceAccounts(context.Background(), c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	saName := PerTenantResourceName("team-a", "default")
	r := &rbacv1.Role{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, r); err != nil {
		t.Fatalf("Role missing: %v", err)
	}
	if len(r.Rules) != 1 || r.Rules[0].Verbs[0] != "get" {
		t.Errorf("unexpected rules: %+v", r.Rules)
	}
	rb := &rbacv1.RoleBinding{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, rb); err != nil {
		t.Fatalf("RoleBinding missing: %v", err)
	}
	if rb.RoleRef.Kind != "Role" || rb.RoleRef.Name != saName {
		t.Errorf("RoleRef = %+v", rb.RoleRef)
	}
	if len(rb.Subjects) != 1 || rb.Subjects[0].Name != saName {
		t.Errorf("Subjects = %+v", rb.Subjects)
	}
}

func TestServiceAccounts_RBACClusterRole_OnlyBinding(t *testing.T) {
	scheme := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.ServiceAccounts = []axisml.ServiceAccountSpec{{
		Name: "default",
		RBAC: &axisml.RBACSpec{
			RoleRef: &axisml.RBACRoleRef{Kind: "ClusterRole", Name: "platform-default"},
		},
	}}

	if _, err := ServiceAccounts(context.Background(), c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	saName := PerTenantResourceName("team-a", "default")
	rb := &rbacv1.RoleBinding{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, rb); err != nil {
		t.Fatalf("RoleBinding missing: %v", err)
	}
	if rb.RoleRef.Kind != "ClusterRole" || rb.RoleRef.Name != "platform-default" {
		t.Errorf("RoleRef = %+v", rb.RoleRef)
	}

	r := &rbacv1.Role{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, r); !apierrors.IsNotFound(err) {
		t.Errorf("ClusterRole binding should not create per-tenant Role; got %v", err)
	}
}

func TestServiceAccounts_RBACDroppedCleansUp(t *testing.T) {
	scheme := newFakeScheme(t)
	tnt := newTenant("team-a")
	saName := PerTenantResourceName("team-a", "default")

	preRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: "team-a"},
	}
	preRB := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: "team-a"},
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: saName},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(preRole, preRB).Build()

	tnt.Spec.InitResources.ServiceAccounts = []axisml.ServiceAccountSpec{{Name: "default"}} // no RBAC
	if _, err := ServiceAccounts(context.Background(), c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	r := &rbacv1.Role{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, r); !apierrors.IsNotFound(err) {
		t.Errorf("Role should be cleaned up; got %v", err)
	}
	rb := &rbacv1.RoleBinding{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, rb); !apierrors.IsNotFound(err) {
		t.Errorf("RoleBinding should be cleaned up; got %v", err)
	}
}

func TestServiceAccounts_RoleBindingRoleRefChange_Recreates(t *testing.T) {
	scheme := newFakeScheme(t)
	tnt := newTenant("team-a")
	saName := PerTenantResourceName("team-a", "default")

	// Pre-existing Role + RoleBinding pointing at per-tenant Role.
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName, Namespace: "team-a",
			Labels: ApplyTenantLabels(tnt, nil),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(), Kind: "Tenant",
				Name: tnt.Name, UID: tnt.UID, Controller: ptrTrue(),
			}},
		},
		Rules: []rbacv1.PolicyRule{{Verbs: []string{"get"}, APIGroups: []string{""}, Resources: []string{"pods"}}},
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName, Namespace: "team-a",
			Labels: ApplyTenantLabels(tnt, nil),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(), Kind: "Tenant",
				Name: tnt.Name, UID: tnt.UID, Controller: ptrTrue(),
			}},
		},
		RoleRef:  rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: saName},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: saName, Namespace: "team-a"}},
	}
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: "team-a"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sa, role, rb).Build()

	// Switch to ClusterRole binding → Role goes away, RoleBinding RoleRef flips.
	tnt.Spec.InitResources.ServiceAccounts = []axisml.ServiceAccountSpec{{
		Name: "default",
		RBAC: &axisml.RBACSpec{
			RoleRef: &axisml.RBACRoleRef{Kind: "ClusterRole", Name: "platform-default"},
		},
	}}

	if _, err := ServiceAccounts(context.Background(), c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	gotRB := &rbacv1.RoleBinding{}
	_ = c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, gotRB)
	if gotRB.RoleRef.Kind != "ClusterRole" {
		t.Errorf("RoleBinding RoleRef.Kind = %s; want ClusterRole", gotRB.RoleRef.Kind)
	}
	gotRole := &rbacv1.Role{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, gotRole); !apierrors.IsNotFound(err) {
		t.Errorf("per-tenant Role should be removed; got %v", err)
	}
}

func TestServiceAccounts_GCsOrphans(t *testing.T) {
	scheme := newFakeScheme(t)
	tnt := newTenant("team-a")
	saName := PerTenantResourceName("team-a", "old-sa")

	orphanSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName, Namespace: "team-a",
			Labels: ApplyTenantLabels(tnt, nil),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(), Kind: "Tenant",
				Name: tnt.Name, UID: tnt.UID, Controller: ptrTrue(),
			}},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(orphanSA).Build()

	tnt.Spec.InitResources.ServiceAccounts = nil
	if _, err := ServiceAccounts(context.Background(), c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	got := &corev1.ServiceAccount{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: saName}, got); !apierrors.IsNotFound(err) {
		t.Errorf("orphan SA not GC'd: %v", err)
	}
}
