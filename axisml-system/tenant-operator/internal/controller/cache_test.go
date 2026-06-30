package controller

import (
	"testing"

	schedulingv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/scheduling/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

func TestCacheByObject_RestrictsToManagedBy(t *testing.T) {
	got := CacheByObject()
	wantTypes := []client.Object{
		&corev1.Secret{},
		&corev1.ConfigMap{},
		&corev1.ServiceAccount{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
		&schedulingv1alpha1.ElasticQuota{},
	}
	if len(got) != len(wantTypes) {
		t.Fatalf("len = %d; want %d", len(got), len(wantTypes))
	}

	matchAll := labels.Set{tenantv1alpha1.LabelManagedBy: tenantv1alpha1.ManagedByValue}
	mismatch := labels.Set{tenantv1alpha1.LabelManagedBy: "someone-else"}
	for k, by := range got {
		if by.Label == nil {
			t.Errorf("%T has nil Label selector — would pull all of these into cache", k)
			continue
		}
		if !by.Label.Matches(matchAll) {
			t.Errorf("%T selector should match managed-by=tenant-operator", k)
		}
		if by.Label.Matches(mismatch) {
			t.Errorf("%T selector should reject managed-by=someone-else", k)
		}
	}
}
