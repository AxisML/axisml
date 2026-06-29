package reconcile

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	axisml "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

func newFakeScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(axisml.AddToScheme(s))
	return s
}

func newTenant(name string) *axisml.Tenant {
	return &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			UID:    types.UID(name + "-uid"),
			Labels: map[string]string{axisml.LabelTenantID: name + "-id"},
		},
		Spec: axisml.TenantSpec{
			Namespace: axisml.NamespaceSpec{Name: name},
		},
	}
}

func TestNamespace_EmptyNameReturnsError(t *testing.T) {
	s := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(s).Build()
	tnt := newTenant("team-a")
	tnt.Spec.Namespace.Name = ""

	ready, msg, err := Namespace(context.Background(), c, tnt)
	if err == nil {
		t.Fatal("expected error for empty namespace")
	}
	if ready {
		t.Error("ready should be false")
	}
	if msg == "" {
		t.Error("msg should be non-empty")
	}
}

func TestNamespace_CreatesIfMissing(t *testing.T) {
	s := newFakeScheme(t)
	c := fake.NewClientBuilder().WithScheme(s).Build()
	tnt := newTenant("team-a")
	tnt.Spec.Namespace.Labels = map[string]string{"env": "test"}
	tnt.Spec.Namespace.Annotations = map[string]string{"team": "a"}

	// Fake client treats a freshly created Namespace as having empty status.Phase,
	// so the function returns ready=false but creates the resource. Verify the
	// post-state.
	_, _, _ = Namespace(context.Background(), c, tnt)

	got := &corev1.Namespace{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "team-a"}, got); err != nil {
		t.Fatalf("get namespace: %v", err)
	}
	if got.Labels[axisml.LabelManagedBy] != axisml.ManagedByValue {
		t.Errorf("missing managed-by label: %v", got.Labels)
	}
	if got.Labels["env"] != "test" {
		t.Errorf("user labels not propagated: %v", got.Labels)
	}
	if got.Annotations["team"] != "a" {
		t.Errorf("user annotations not propagated: %v", got.Annotations)
	}
}

func TestNamespace_StampsLabelOnExistingShared(t *testing.T) {
	s := newFakeScheme(t)
	existing := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "shared",
			Labels: map[string]string{"foreign": "yes"},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(existing).Build()
	tnt := newTenant("shared")
	tnt.Spec.Namespace.Labels = map[string]string{"new": "value"}

	ready, _, err := Namespace(context.Background(), c, tnt)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !ready {
		t.Error("expected ready=true for active existing namespace")
	}
	got := &corev1.Namespace{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "shared"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Labels[axisml.LabelManagedBy] != axisml.ManagedByValue {
		t.Errorf("managed-by not stamped: %v", got.Labels)
	}
	if got.Labels["foreign"] != "yes" {
		t.Errorf("foreign label clobbered: %v", got.Labels)
	}
	if _, present := got.Labels["new"]; present {
		t.Errorf("user labels should NOT overwrite existing namespace; got %v", got.Labels)
	}
}

func TestNamespace_DoesNotOverwriteForeignManagedBy(t *testing.T) {
	s := newFakeScheme(t)
	existing := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ns",
			Labels: map[string]string{axisml.LabelManagedBy: "someone-else"},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(existing).Build()
	tnt := newTenant("ns")

	if _, _, err := Namespace(context.Background(), c, tnt); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	got := &corev1.Namespace{}
	_ = c.Get(context.Background(), types.NamespacedName{Name: "ns"}, got)
	if got.Labels[axisml.LabelManagedBy] != "someone-else" {
		t.Errorf("foreign managed-by overwritten: %v", got.Labels)
	}
}

func TestNamespace_NotReadyOnTerminating(t *testing.T) {
	s := newFakeScheme(t)
	existing := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "ns"},
		Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating},
	}
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(existing).Build()
	tnt := newTenant("ns")
	ready, msg, err := Namespace(context.Background(), c, tnt)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if ready {
		t.Error("expected ready=false on Terminating namespace")
	}
	if msg == "" {
		t.Error("expected message")
	}
}

func TestNamespace_CopyMapHelper(t *testing.T) {
	if got := copyMap(nil); got != nil {
		t.Errorf("copyMap(nil) = %v; want nil", got)
	}
	in := map[string]string{"a": "1"}
	got := copyMap(in)
	got["a"] = "2"
	if in["a"] != "1" {
		t.Errorf("copyMap should be a deep copy; mutation leaked")
	}
}

// silence unused-client warnings if helper functions are added later
var _ client.Client = (client.Client)(nil)
