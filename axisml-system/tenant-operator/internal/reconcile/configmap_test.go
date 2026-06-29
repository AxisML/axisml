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

func TestConfigMaps_CopiesFromSource(t *testing.T) {
	scheme := newFakeScheme(t)
	src := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: "axisml-system"},
		Data:       map[string]string{"k": "v"},
		BinaryData: map[string][]byte{"b": {0x01}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(src).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.ConfigMaps = []axisml.ConfigMapSpec{{
		Name: "envs",
		SourceConfigMapRef: axisml.SourceConfigMapRef{
			Namespace: "axisml-system", Name: "src",
		},
	}}

	statuses, err := ConfigMaps(context.Background(), c, c, scheme, tnt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 1 || !statuses[0].Ready {
		t.Fatalf("expected one ready status, got %+v", statuses)
	}

	got := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: PerTenantResourceName("team-a", "envs"),
	}, got); err != nil {
		t.Fatalf("expected configmap to be created: %v", err)
	}
	if got.Data["k"] != "v" {
		t.Errorf("data not copied: %v", got.Data)
	}
	if string(got.BinaryData["b"]) != "\x01" {
		t.Errorf("binary data not copied: %v", got.BinaryData)
	}
	if got.Labels[axisml.LabelTenantID] != "team-a-id" {
		t.Errorf("tenant labels not applied: %v", got.Labels)
	}
	if !hasOwner(got.OwnerReferences, tnt) {
		t.Errorf("owner ref missing: %v", got.OwnerReferences)
	}
}

func TestConfigMaps_MissingSource(t *testing.T) {
	scheme := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.ConfigMaps = []axisml.ConfigMapSpec{{
		Name: "envs",
		SourceConfigMapRef: axisml.SourceConfigMapRef{
			Namespace: "axisml-system", Name: "missing",
		},
	}}

	statuses, err := ConfigMaps(context.Background(), c, c, scheme, tnt)
	if err != nil {
		t.Fatalf("expected nil err for not-found source, got %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Ready {
		t.Errorf("Ready should be false when source missing")
	}
	if statuses[0].Message == "" {
		t.Errorf("expected message")
	}
}

func TestConfigMaps_BlankSourceRef(t *testing.T) {
	scheme := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.ConfigMaps = []axisml.ConfigMapSpec{{
		Name:               "envs",
		SourceConfigMapRef: axisml.SourceConfigMapRef{},
	}}
	statuses, err := ConfigMaps(context.Background(), c, c, scheme, tnt)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if statuses[0].Ready {
		t.Error("Ready should be false when sourceRef blank")
	}
}

func TestConfigMaps_GCsOrphans(t *testing.T) {
	scheme := newFakeScheme(t)
	tnt := newTenant("team-a")

	// Pre-existing tenant-owned configmap that's no longer in spec.
	orphan := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PerTenantResourceName("team-a", "old"),
			Namespace: "team-a",
			Labels:    TenantLabels(tnt),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(),
				Kind:       "Tenant",
				Name:       tnt.Name,
				UID:        tnt.UID,
				Controller: ptrTrue(),
			}},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(orphan).Build()

	tnt.Spec.InitResources.ConfigMaps = nil // empty desired set
	if _, err := ConfigMaps(context.Background(), c, c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	got := &corev1.ConfigMap{}
	err := c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: orphan.Name,
	}, got)
	if err == nil {
		t.Fatalf("expected orphan to be deleted")
	}
}

func TestConfigMaps_SkipsForeignOwnedDuringGC(t *testing.T) {
	scheme := newFakeScheme(t)
	tnt := newTenant("team-a")

	foreign := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PerTenantResourceName("team-a", "outsider"),
			Namespace: "team-a",
			Labels:    TenantLabels(tnt),
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(foreign).Build()

	if _, err := ConfigMaps(context.Background(), c, c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	got := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: foreign.Name,
	}, got); err != nil {
		t.Fatalf("foreign-owned configmap should NOT have been deleted: %v", err)
	}
}

func TestConfigMaps_PatchesOnSourceDrift(t *testing.T) {
	scheme := newFakeScheme(t)
	tnt := newTenant("team-a")

	src := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: "axisml-system"},
		Data:       map[string]string{"k": "new"},
	}
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PerTenantResourceName("team-a", "envs"),
			Namespace: "team-a",
			Labels:    TenantLabels(tnt),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: axisml.GroupVersion.String(),
				Kind:       "Tenant",
				Name:       tnt.Name,
				UID:        tnt.UID,
				Controller: ptrTrue(),
			}},
		},
		Data: map[string]string{"k": "old"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(src, existing).Build()

	tnt.Spec.InitResources.ConfigMaps = []axisml.ConfigMapSpec{{
		Name: "envs",
		SourceConfigMapRef: axisml.SourceConfigMapRef{
			Namespace: "axisml-system", Name: "src",
		},
	}}

	if _, err := ConfigMaps(context.Background(), c, c, scheme, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	got := &corev1.ConfigMap{}
	_ = c.Get(context.Background(), types.NamespacedName{
		Namespace: "team-a", Name: existing.Name,
	}, got)
	if got.Data["k"] != "new" {
		t.Errorf("drift not corrected: %v", got.Data)
	}
}

func ptrTrue() *bool { b := true; return &b }
