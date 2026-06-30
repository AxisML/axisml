package reconcile

import (
	"context"
	"testing"

	schedv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/scheduling/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	axisml "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

func newSchemeWithEQ(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(axisml.AddToScheme(s))
	utilruntime.Must(schedv1alpha1.AddToScheme(s))
	return s
}

func TestElasticQuotas_CreatesPerSpec(t *testing.T) {
	scheme := newSchemeWithEQ(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	tnt := newTenant("team-a")
	tnt.Spec.Quotas = []axisml.QuotaSpec{
		{Pool: "gpu", Name: "default",
			Min: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
			Max: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")}},
	}

	statuses, err := ElasticQuotas(context.Background(), c, scheme, tnt)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(statuses) != 1 || !statuses[0].Ready {
		t.Fatalf("expected ready quota status, got %+v", statuses)
	}

	got := &schedv1alpha1.ElasticQuota{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: ElasticQuotaName("team-a", "gpu", "default"),
	}, got); err != nil {
		t.Fatalf("EQ missing: %v", err)
	}
	if got.Spec.Max.Cpu().String() != "4" {
		t.Errorf("max.cpu = %s; want 4", got.Spec.Max.Cpu())
	}
	if got.Labels[axisml.LabelTenantID] == "" {
		t.Errorf("tenant labels missing on EQ")
	}
	if !hasOwner(got.OwnerReferences, tnt) {
		t.Errorf("owner ref missing")
	}
}

func TestElasticQuotas_PatchesOnDrift(t *testing.T) {
	scheme := newSchemeWithEQ(t)
	tnt := newTenant("team-a")
	eqName := ElasticQuotaName("team-a", "gpu", "default")
	existing := &schedv1alpha1.ElasticQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: eqName, Namespace: "team-a",
			Labels: TenantLabels(tnt),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(), Kind: "Tenant",
				Name: tnt.Name, UID: tnt.UID, Controller: ptrTrue(),
			}},
		},
		Spec: schedv1alpha1.ElasticQuotaSpec{
			Max: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")}, // drift
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

	tnt.Spec.Quotas = []axisml.QuotaSpec{
		{Pool: "gpu", Name: "default",
			Max: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("8")}},
	}
	if _, err := ElasticQuotas(context.Background(), c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	got := &schedv1alpha1.ElasticQuota{}
	_ = c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: eqName}, got)
	if got.Spec.Max.Cpu().String() != "8" {
		t.Errorf("Max.Cpu post-patch = %s; want 8", got.Spec.Max.Cpu())
	}
}

func TestElasticQuotas_GCsOrphans(t *testing.T) {
	scheme := newSchemeWithEQ(t)
	tnt := newTenant("team-a")
	stale := &schedv1alpha1.ElasticQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ElasticQuotaName("team-a", "gpu", "old"),
			Namespace: "team-a",
			Labels:    TenantLabels(tnt),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(), Kind: "Tenant",
				Name: tnt.Name, UID: tnt.UID, Controller: ptrTrue(),
			}},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stale).Build()

	tnt.Spec.Quotas = nil
	if _, err := ElasticQuotas(context.Background(), c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	got := &schedv1alpha1.ElasticQuota{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: stale.Name}, got); !apierrors.IsNotFound(err) {
		t.Errorf("orphan EQ not deleted: %v", err)
	}
}

func TestElasticQuotas_SkipsForeignOwnedDuringGC(t *testing.T) {
	scheme := newSchemeWithEQ(t)
	tnt := newTenant("team-a")

	// Tenant labels but no ownerRef → not ours.
	foreign := &schedv1alpha1.ElasticQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "axisml-team-a-foo-bar",
			Namespace: "team-a",
			Labels:    TenantLabels(tnt),
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(foreign).Build()

	tnt.Spec.Quotas = nil
	if _, err := ElasticQuotas(context.Background(), c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	got := &schedv1alpha1.ElasticQuota{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: foreign.Name}, got); err != nil {
		t.Errorf("foreign-owned EQ should NOT be GC'd: %v", err)
	}
}

func TestHasOwner(t *testing.T) {
	tnt := newTenant("team-a")
	refs := []metav1.OwnerReference{
		{UID: "other"},
		{UID: tnt.UID},
	}
	if !hasOwner(refs, tnt) {
		t.Error("hasOwner should match by UID")
	}
	if hasOwner([]metav1.OwnerReference{{UID: "other"}}, tnt) {
		t.Error("hasOwner false positive")
	}
}
