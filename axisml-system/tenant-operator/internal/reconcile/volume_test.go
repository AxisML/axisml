package reconcile

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	axisml "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

func TestVolumes_CreatesManagedPVC(t *testing.T) {
	scheme := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.Volumes = []axisml.VolumeSpec{{
		Name:        "dataset",
		Size:        "50Gi",
		Description: "shared training data",
	}}

	statuses, err := Volumes(context.Background(), c, c, tnt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 1 || !statuses[0].Ready {
		t.Fatalf("expected one ready status, got %+v", statuses)
	}

	got := &corev1.PersistentVolumeClaim{}
	// The PVC name is the raw volume name (the claim name a workload mounts),
	// NOT the per-tenant-prefixed name the credential resources use.
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "dataset"}, got); err != nil {
		t.Fatalf("expected pvc to be created: %v", err)
	}
	if got.Labels[volumeManagedByLabel] != volumeManagedByValue {
		t.Errorf("catalog managed-by label missing: %v", got.Labels)
	}
	if got.Labels[axisml.LabelTenantID] != "team-a-id" {
		t.Errorf("tenant id label missing: %v", got.Labels)
	}
	if got.Annotations[volumeDescriptionAnnotation] != "shared training data" {
		t.Errorf("description annotation missing: %v", got.Annotations)
	}
	if q := got.Spec.Resources.Requests[corev1.ResourceStorage]; q.Cmp(resource.MustParse("50Gi")) != 0 {
		t.Errorf("storage request not set: %v", got.Spec.Resources.Requests)
	}
	// Non-destructive: the operator must NOT own the PVC, so tenant/namespace
	// teardown can never cascade-delete the data volume.
	if hasOwner(got.OwnerReferences, tnt) {
		t.Errorf("pvc must not carry an owner reference: %v", got.OwnerReferences)
	}
}

func TestVolumes_IdempotentAndPreservesExistingSize(t *testing.T) {
	scheme := newFakeScheme(t)
	// A pre-existing managed volume already expanded (via the catalog) beyond its
	// declared size, and holding data. Re-reconcile must never shrink it.
	existing := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dataset",
			Namespace: "team-a",
			Labels:    map[string]string{volumeManagedByLabel: volumeManagedByValue},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("100Gi")},
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.Volumes = []axisml.VolumeSpec{{Name: "dataset", Size: "50Gi"}}

	statuses, err := Volumes(context.Background(), c, c, tnt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 1 || !statuses[0].Ready {
		t.Fatalf("expected one ready status, got %+v", statuses)
	}

	got := &corev1.PersistentVolumeClaim{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "dataset"}, got); err != nil {
		t.Fatalf("get pvc: %v", err)
	}
	if q := got.Spec.Resources.Requests[corev1.ResourceStorage]; q.Cmp(resource.MustParse("100Gi")) != 0 {
		t.Errorf("existing size must be preserved, got %v", q.String())
	}
	// The catalog label is stamped so the existing volume becomes catalog-visible.
	if got.Labels[volumeManagedByLabel] != volumeManagedByValue {
		t.Errorf("catalog label not synced onto existing pvc: %v", got.Labels)
	}
}

func TestVolumes_RefusesForeignPVC(t *testing.T) {
	scheme := newFakeScheme(t)
	// A PVC of the same name exists but is NOT AxisML-managed (no managed-by
	// label) — created by some other actor. The operator must refuse to adopt or
	// relabel it, surfacing a not-ready status instead of silently hijacking it.
	foreign := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dataset",
			Namespace: "team-a",
			Labels:    map[string]string{"app.kubernetes.io/managed-by": "someone-else"},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(foreign).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.Volumes = []axisml.VolumeSpec{{Name: "dataset", Size: "50Gi"}}

	statuses, err := Volumes(context.Background(), c, c, tnt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Ready {
		t.Fatalf("expected one not-ready status, got %+v", statuses)
	}
	if !strings.Contains(statuses[0].Message, "not managed by AxisML") {
		t.Errorf("expected refuse-to-adopt message, got %q", statuses[0].Message)
	}

	got := &corev1.PersistentVolumeClaim{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "dataset"}, got); err != nil {
		t.Fatalf("get pvc: %v", err)
	}
	// The foreign PVC must be left untouched — no catalog / tenant labels stamped.
	if got.Labels[volumeManagedByLabel] == volumeManagedByValue {
		t.Errorf("foreign pvc was relabeled: %v", got.Labels)
	}
	if _, ok := got.Labels[axisml.LabelTenantID]; ok {
		t.Errorf("foreign pvc got a tenant-id label: %v", got.Labels)
	}
}

func TestVolumes_RejectsHostPath(t *testing.T) {
	scheme := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	tnt := newTenant("team-a")
	tnt.Spec.InitResources.Volumes = []axisml.VolumeSpec{{Name: "hostdata", HostPath: "/data/ds"}}

	statuses, err := Volumes(context.Background(), c, c, tnt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Ready {
		t.Fatalf("expected one not-ready status, got %+v", statuses)
	}
	if !strings.Contains(statuses[0].Message, "hostPath") {
		t.Errorf("expected hostPath-not-supported message, got %q", statuses[0].Message)
	}
	// No PVC should have been created for a hostPath volume.
	got := &corev1.PersistentVolumeClaim{}
	err = c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "hostdata"}, got)
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected no pvc created for hostPath volume, got err=%v", err)
	}
}

func TestVolumes_Empty(t *testing.T) {
	scheme := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	statuses, err := Volumes(context.Background(), c, c, newTenant("team-a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 0 {
		t.Fatalf("expected no statuses, got %+v", statuses)
	}
}
