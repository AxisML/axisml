package workloadconfig

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mlrun "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	configapi "github.com/axisml/axisml/axisml-system/apis/pkg/workloadconfig"
)

func testOwner() *mlrun.MLRun {
	return &mlrun.MLRun{ObjectMeta: metav1.ObjectMeta{
		Name: "run", Namespace: "tenant-a", UID: types.UID("run-uid"),
	}}
}

func testClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := mlrun.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func TestValidate(t *testing.T) {
	if errs := Validate([]configapi.ConfigMap{{Name: "trainer.config"}}, field.NewPath("spec", "configMaps")); len(errs) != 0 {
		t.Fatalf("valid DNS-1123 subdomain rejected: %v", errs)
	}
	errs := Validate([]configapi.ConfigMap{
		{Name: "bad_name", Data: map[string]string{"bad key": "x"}},
		{Name: "bad_name"},
	}, field.NewPath("spec", "configMaps"))
	if len(errs) < 3 {
		t.Fatalf("expected name, key, and duplicate errors; got %v", errs)
	}
}

func TestReconcileCreatesAndCorrectsDrift(t *testing.T) {
	ctx := context.Background()
	owner := testOwner()
	c := testClient(t, owner)
	specs := []configapi.ConfigMap{{Name: "run-config", Data: map[string]string{"mode": "train"}}}

	if err := Reconcile(ctx, c, owner, mlrun.GroupVersion.WithKind("MLRun"), specs,
		map[string]string{mlrun.LabelRunID: "run-id"}); err != nil {
		t.Fatal(err)
	}
	got := &corev1.ConfigMap{}
	key := client.ObjectKey{Namespace: owner.Namespace, Name: "run-config"}
	if err := c.Get(ctx, key, got); err != nil {
		t.Fatal(err)
	}
	if got.Data["mode"] != "train" || got.Labels[mlrun.LabelRunID] != "run-id" {
		t.Fatalf("created ConfigMap mismatch: %+v", got)
	}
	if !metav1.IsControlledBy(got, owner) {
		t.Fatal("created ConfigMap is not controlled by the MLRun")
	}

	got.Data["mode"] = "manual-drift"
	if err := c.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	if err := Reconcile(ctx, c, owner, mlrun.GroupVersion.WithKind("MLRun"), specs, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, key, got); err != nil {
		t.Fatal(err)
	}
	if got.Data["mode"] != "train" {
		t.Fatalf("drift was not corrected: %+v", got.Data)
	}
}

func TestReconcileRejectsForeignConfigMap(t *testing.T) {
	owner := testOwner()
	foreign := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: owner.Namespace}}
	c := testClient(t, owner, foreign)
	err := Reconcile(context.Background(), c, owner, mlrun.GroupVersion.WithKind("MLRun"),
		[]configapi.ConfigMap{{Name: "shared"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("expected ownership collision, got %v", err)
	}
	if !IsOwnershipConflict(err) {
		t.Fatalf("IsOwnershipConflict(%v) = false", err)
	}
}

func TestReconcileDeletesObsoleteOwnedConfigMap(t *testing.T) {
	ctx := context.Background()
	owner := testOwner()
	obsolete := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      "obsolete",
		Namespace: owner.Namespace,
		Labels: map[string]string{
			LabelManagedConfig: ManagedValue,
			LabelOwnerUID:      string(owner.UID),
		},
		OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(owner, mlrun.GroupVersion.WithKind("MLRun"))},
	}}
	c := testClient(t, owner, obsolete)
	if err := Reconcile(ctx, c, owner, mlrun.GroupVersion.WithKind("MLRun"), nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(obsolete), &corev1.ConfigMap{}); err == nil {
		t.Fatal("obsolete owned ConfigMap was not deleted")
	}
}
