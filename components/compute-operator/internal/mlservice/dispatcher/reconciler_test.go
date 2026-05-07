package dispatcher

import (
	"context"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	axisml "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	hpkg "github.com/axisml/axisml/components/compute-operator/internal/mlservice/handler"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(axisml.AddToScheme(s))
	return s
}

func TestMergeConditions_PreservesLastTransitionWhenStatusUnchanged(t *testing.T) {
	old := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Second)
	existing := []metav1.Condition{{
		Type:               "Available",
		Status:             metav1.ConditionTrue,
		Reason:             "AllReplicasReady",
		LastTransitionTime: metav1.Time{Time: old},
	}}
	updates := []metav1.Condition{{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "AllReplicasReady",
		Message: "still ready",
	}}
	merged := mergeConditions(existing, updates, 7)
	if len(merged) != 1 {
		t.Fatalf("expected 1 condition; got %d", len(merged))
	}
	if !merged[0].LastTransitionTime.Time.Equal(old) {
		t.Errorf("LastTransitionTime advanced on a no-status-change update: got %v want %v",
			merged[0].LastTransitionTime.Time, old)
	}
	if merged[0].ObservedGeneration != 7 {
		t.Errorf("ObservedGeneration = %d; want 7", merged[0].ObservedGeneration)
	}
}

func TestMergeConditions_BumpsLastTransitionOnStatusFlip(t *testing.T) {
	old := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Second)
	existing := []metav1.Condition{{
		Type:               "Available",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Time{Time: old},
	}}
	updates := []metav1.Condition{{
		Type:   "Available",
		Status: metav1.ConditionTrue,
		Reason: "AllReplicasReady",
	}}
	merged := mergeConditions(existing, updates, 1)
	if !merged[0].LastTransitionTime.After(old) {
		t.Errorf("LastTransitionTime did not advance on status flip: got %v", merged[0].LastTransitionTime.Time)
	}
}

func TestMergeConditions_SortedByType(t *testing.T) {
	updates := []metav1.Condition{
		{Type: "Progressing", Status: metav1.ConditionTrue},
		{Type: "Available", Status: metav1.ConditionTrue},
		{Type: "Degraded", Status: metav1.ConditionFalse},
	}
	merged := mergeConditions(nil, updates, 1)
	want := []string{"Available", "Degraded", "Progressing"}
	for i, c := range merged {
		if c.Type != want[i] {
			t.Errorf("merged[%d].Type = %s; want %s", i, c.Type, want[i])
		}
	}
}

// newReconcilerWithMLS builds a Reconciler against a fake client seeded with
// the supplied MLService. It does NOT register any handler factories — tests
// supply their own handler map directly.
func newReconcilerWithMLS(t *testing.T, mls *axisml.MLService, handlers map[hpkg.Key]hpkg.Handler) (*Reconciler, client.Client) {
	t.Helper()
	scheme := newScheme(t)
	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(mls).
		WithStatusSubresource(&axisml.MLService{}).
		Build()
	r := &Reconciler{client: cli, scheme: scheme, handlers: handlers}
	return r, cli
}

func TestReconcile_NoHandler_WritesFailedStatus(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "smoke",
			Namespace: "tenant-demo",
			Labels: map[string]string{
				axisml.LabelServiceID: "uuid-1",
			},
		},
		Spec: axisml.MLServiceSpec{
			Backend: axisml.Backend{Name: "unregistered", Engine: "missing"},
		},
	}
	r, cli := newReconcilerWithMLS(t, mls, map[hpkg.Key]hpkg.Handler{})

	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: mls.Name, Namespace: mls.Namespace},
	}); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	got := &axisml.MLService{}
	if err := cli.Get(context.Background(), types.NamespacedName{Name: mls.Name, Namespace: mls.Namespace}, got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != axisml.PhaseFailed {
		t.Errorf("phase = %s; want Failed", got.Status.Phase)
	}
	if !strings.Contains(got.Status.Message, "no handler for backend=unregistered") {
		t.Errorf("status.message = %q; want it to mention the missing handler tuple", got.Status.Message)
	}
}

func TestReconcile_MissingServiceIDLabel_WritesFailedStatus(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "smoke",
			Namespace: "tenant-demo",
			// No axisml.io/service-id — the dispatcher must reject this.
		},
		Spec: axisml.MLServiceSpec{
			Backend: axisml.Backend{Name: "native", Engine: "deployment"},
		},
	}
	r, cli := newReconcilerWithMLS(t, mls, map[hpkg.Key]hpkg.Handler{})

	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: mls.Name, Namespace: mls.Namespace},
	}); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	got := &axisml.MLService{}
	if err := cli.Get(context.Background(), types.NamespacedName{Name: mls.Name, Namespace: mls.Namespace}, got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != axisml.PhaseFailed {
		t.Errorf("phase = %s; want Failed", got.Status.Phase)
	}
	if !strings.Contains(got.Status.Message, axisml.LabelServiceID) {
		t.Errorf("status.message = %q; want it to mention the missing service-id label", got.Status.Message)
	}
}

func TestWriteStatus_NoOpWhenStatusUnchanged(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "smoke",
			Namespace:  "tenant-demo",
			Generation: 3,
		},
		Status: axisml.MLServiceStatus{
			ObservedGeneration: 3,
			Phase:              axisml.PhaseReady,
		},
	}
	r, cli := newReconcilerWithMLS(t, mls, nil)

	// Read back to capture the apiserver-assigned ResourceVersion.
	live := &axisml.MLService{}
	if err := cli.Get(context.Background(), types.NamespacedName{Name: mls.Name, Namespace: mls.Namespace}, live); err != nil {
		t.Fatal(err)
	}
	rvBefore := live.ResourceVersion

	upd := hpkg.StatusUpdate{Phase: axisml.PhaseReady}
	if err := r.writeStatus(context.Background(), live, upd); err != nil {
		t.Fatalf("writeStatus error: %v", err)
	}

	live2 := &axisml.MLService{}
	if err := cli.Get(context.Background(), types.NamespacedName{Name: mls.Name, Namespace: mls.Namespace}, live2); err != nil {
		t.Fatal(err)
	}
	if live2.ResourceVersion != rvBefore {
		t.Errorf("writeStatus issued a patch on a no-op update: ResourceVersion %s -> %s", rvBefore, live2.ResourceVersion)
	}
}
