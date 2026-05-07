package dispatcher

import (
	"context"
	"errors"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	axisv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
	axishandler "github.com/axisml/axisml/components/compute-operator/internal/mljob/handler"
	axislabels "github.com/axisml/axisml/components/compute-operator/internal/mljob/labels"
)

// recordingHandler is a minimal Handler that records whether Reconcile
// was invoked. By default it Validates clean and returns Pending; tests
// can inject validate errors or a fixed MapStatus phase.
type recordingHandler struct {
	reconcileCalled bool
	sweepCalled     bool
	sweepRequeue    int32
	sweepErr        error
	validateErrs    field.ErrorList
	mapStatusPhase  axisv1alpha1.MLJobPhase
}

func (h *recordingHandler) Key() axishandler.Key {
	return axishandler.Key{Backend: "test", Engine: "engine"}
}

func (h *recordingHandler) Validate(*axisv1alpha1.MLJob) field.ErrorList { return h.validateErrs }

func (h *recordingHandler) Reconcile(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob) (any, axishandler.ReconcileResult, error) {
	h.reconcileCalled = true
	return nil, axishandler.ReconcileResult{}, nil
}

func (h *recordingHandler) MapStatus(any) axishandler.MapStatusResult {
	if h.mapStatusPhase != "" {
		return axishandler.MapStatusResult{Phase: h.mapStatusPhase}
	}
	return axishandler.MapStatusResult{Phase: axisv1alpha1.PhasePending}
}

func (h *recordingHandler) WatchTargets() []client.Object { return nil }

func (h *recordingHandler) Sweep(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob) (int32, error) {
	h.sweepCalled = true
	return h.sweepRequeue, h.sweepErr
}

func newReconcilerScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := axisv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func newTerminalMLJob() *axisv1alpha1.MLJob {
	mlj := &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "term",
			Namespace:  "tnt",
			Generation: 1,
		},
		Spec: axisv1alpha1.MLJobSpec{
			Backend: axisv1alpha1.BackendSpec{Name: "test", Engine: "engine"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:     axisv1alpha1.DefaultRoleName,
				Replicas: 1,
				Template: axisv1alpha1.PodTemplateSubset{Image: "x"},
			}},
		},
		Status: axisv1alpha1.MLJobStatus{
			Phase:              axisv1alpha1.PhaseSucceeded,
			ObservedGeneration: 1,
		},
	}
	// Anchor the applied-spec annotation so tests that exercise the
	// terminal short-circuit don't trip the immutability check (which
	// runs after the short-circuit, but having a self-consistent
	// fingerprint is the realistic on-cluster shape).
	mlj.Annotations = map[string]string{axislabels.AppliedSpecAnnotation: specFingerprint(mlj)}
	return mlj
}

func TestReconcile_TerminalShortCircuits(t *testing.T) {
	// A terminal MLJob must not enter handler.Reconcile (otherwise
	// owned-resource delete events from TTL GC or manual cleanup would
	// trigger a rerun).
	s := newReconcilerScheme(t)
	mlj := newTerminalMLJob()
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(mlj).WithStatusSubresource(mlj).Build()

	h := &recordingHandler{}
	reg := NewRegistry()
	reg.Register(h)

	r := &MLJobReconciler{Client: c, Registry: reg}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if h.reconcileCalled {
		t.Fatalf("handler.Reconcile must NOT be called when MLJob is already terminal")
	}
	if !h.sweepCalled {
		t.Fatalf("handler.Sweep must be called on terminal short-circuit")
	}
}

func TestReconcile_TerminalSyncsObservedGeneration(t *testing.T) {
	// User mutates spec post-terminal: generation moves ahead of
	// observedGeneration. Dispatcher must catch up the field while
	// keeping the terminal phase locked.
	s := newReconcilerScheme(t)
	mlj := newTerminalMLJob()
	mlj.Generation = 5
	mlj.Status.ObservedGeneration = 1
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(mlj).WithStatusSubresource(mlj).Build()

	h := &recordingHandler{}
	reg := NewRegistry()
	reg.Register(h)

	r := &MLJobReconciler{Client: c, Registry: reg}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var got axisv1alpha1.MLJob
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.ObservedGeneration != 5 {
		t.Fatalf("observedGeneration: want 5, got %d", got.Status.ObservedGeneration)
	}
	if got.Status.Phase != axisv1alpha1.PhaseSucceeded {
		t.Fatalf("phase regressed from terminal: got %q", got.Status.Phase)
	}
}

func TestReconcile_TerminalPropagatesSweepError(t *testing.T) {
	s := newReconcilerScheme(t)
	mlj := newTerminalMLJob()
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(mlj).WithStatusSubresource(mlj).Build()

	wantErr := errors.New("sweep boom")
	h := &recordingHandler{sweepErr: wantErr}
	reg := NewRegistry()
	reg.Register(h)

	r := &MLJobReconciler{Client: c, Registry: reg}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected sweep error to surface, got %v", err)
	}
}

func newPendingMLJob() *axisv1alpha1.MLJob {
	return &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "fresh",
			Namespace:  "tnt",
			Generation: 1,
		},
		Spec: axisv1alpha1.MLJobSpec{
			Backend: axisv1alpha1.BackendSpec{Name: "test", Engine: "engine"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:     axisv1alpha1.DefaultRoleName,
				Replicas: 2,
				Template: axisv1alpha1.PodTemplateSubset{Image: "x"},
			}},
		},
	}
}

func TestReconcile_AnchorsAppliedSpecAnnotationOnFirstObservation(t *testing.T) {
	// First reconcile of a CR that has no axisml.io/applied-spec
	// annotation must persist the fingerprint so subsequent mutations
	// are detectable.
	s := newReconcilerScheme(t)
	mlj := newPendingMLJob()
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(mlj).WithStatusSubresource(mlj).Build()

	reg := NewRegistry()
	reg.Register(&recordingHandler{})

	r := &MLJobReconciler{Client: c, Registry: reg}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var got axisv1alpha1.MLJob
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	want := specFingerprint(mlj)
	if gotAnno := got.Annotations[axislabels.AppliedSpecAnnotation]; gotAnno != want {
		t.Fatalf("applied-spec annotation: want %q, got %q", want, gotAnno)
	}
}

func TestReconcile_UnknownBackendFails(t *testing.T) {
	// Design §5: dispatcher must surface Phase=Failed with a clear
	// message when no handler claims the (backend, engine) tuple.
	s := newReconcilerScheme(t)
	mlj := newPendingMLJob()
	mlj.Spec.Backend = axisv1alpha1.BackendSpec{Name: "unknown", Engine: "missing"}
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(mlj).WithStatusSubresource(mlj).Build()

	r := &MLJobReconciler{Client: c, Registry: NewRegistry()}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var got axisv1alpha1.MLJob
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.Phase != axisv1alpha1.PhaseFailed {
		t.Fatalf("phase: want Failed, got %q", got.Status.Phase)
	}
	if !strings.Contains(got.Status.Message, "unknown") {
		t.Fatalf("message must name the missing backend, got %q", got.Status.Message)
	}
}

func TestReconcile_ValidateFailureFails(t *testing.T) {
	// Handler.Validate errors must surface as Phase=Failed; the
	// dispatcher must not call Reconcile when Validate is non-empty.
	s := newReconcilerScheme(t)
	mlj := newPendingMLJob()
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(mlj).WithStatusSubresource(mlj).Build()

	h := &recordingHandler{
		validateErrs: field.ErrorList{
			field.Required(field.NewPath("spec", "scheduling", "quota"), "missing quota"),
		},
	}
	reg := NewRegistry()
	reg.Register(h)

	r := &MLJobReconciler{Client: c, Registry: reg}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if h.reconcileCalled {
		t.Fatalf("handler.Reconcile must not run when Validate fails")
	}

	var got axisv1alpha1.MLJob
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: mlj.Namespace, Name: mlj.Name}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.Phase != axisv1alpha1.PhaseFailed {
		t.Fatalf("phase: want Failed, got %q", got.Status.Phase)
	}
}

func TestSpecFingerprint(t *testing.T) {
	// Property tests: the fingerprint must be deterministic, free of
	// RunPolicy.Suspend (the documented mutable cancel knob), and must
	// change whenever a locked field changes — backend, scheduling, role
	// topology, role template, and non-Suspend runPolicy.
	base := func() *axisv1alpha1.MLJob {
		return &axisv1alpha1.MLJob{Spec: axisv1alpha1.MLJobSpec{
			Backend: axisv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Scheduling: axisv1alpha1.SchedulingSpec{
				Quota:        "axisml-tnt-default-training",
				NodeSelector: map[string]string{"gpu": "h100"},
			},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:     axisv1alpha1.DefaultRoleName,
				Replicas: 4,
				Template: axisv1alpha1.PodTemplateSubset{Image: "img:v1"},
			}},
		}}
	}

	t.Run("deterministic", func(t *testing.T) {
		if a, b := specFingerprint(base()), specFingerprint(base()); a != b {
			t.Fatalf("non-deterministic: %q vs %q", a, b)
		}
	})

	t.Run("suspend toggle leaves fingerprint unchanged", func(t *testing.T) {
		a := specFingerprint(base())
		m := base()
		m.Spec.RunPolicy.Suspend = true
		if b := specFingerprint(m); a != b {
			t.Fatalf("Suspend must be free; fingerprint flipped %q -> %q", a, b)
		}
	})

	mutations := []struct {
		name   string
		mutate func(*axisv1alpha1.MLJob)
	}{
		{"backend.name", func(m *axisv1alpha1.MLJob) { m.Spec.Backend.Name = "other" }},
		{"backend.engine", func(m *axisv1alpha1.MLJob) { m.Spec.Backend.Engine = "podgroup" }},
		{"role replicas", func(m *axisv1alpha1.MLJob) { m.Spec.Roles[0].Replicas = 8 }},
		{"role name", func(m *axisv1alpha1.MLJob) { m.Spec.Roles[0].Name = "ps" }},
		{"template image", func(m *axisv1alpha1.MLJob) { m.Spec.Roles[0].Template.Image = "img:v2" }},
		{"scheduling quota", func(m *axisv1alpha1.MLJob) { m.Spec.Scheduling.Quota = "other-quota" }},
		{"runPolicy.activeDeadline", func(m *axisv1alpha1.MLJob) {
			d := int64(60)
			m.Spec.RunPolicy.ActiveDeadlineSeconds = &d
		}},
	}
	a := specFingerprint(base())
	for _, mc := range mutations {
		t.Run(mc.name+" changes fingerprint", func(t *testing.T) {
			m := base()
			mc.mutate(m)
			if b := specFingerprint(m); a == b {
				t.Fatalf("mutation to %s did not change fingerprint", mc.name)
			}
		})
	}
}

// reconcileImmutabilityRejection seeds an MLJob whose applied-spec
// annotation reflects `prior` and whose live spec is `current`, then
// runs Reconcile. Returns the post-reconcile MLJob.
func reconcileImmutabilityRejection(t *testing.T, prior, current *axisv1alpha1.MLJob) axisv1alpha1.MLJob {
	t.Helper()
	s := newReconcilerScheme(t)
	current.Annotations = map[string]string{axislabels.AppliedSpecAnnotation: specFingerprint(prior)}
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(current).WithStatusSubresource(current).Build()
	reg := NewRegistry()
	reg.Register(&recordingHandler{})
	r := &MLJobReconciler{Client: c, Registry: reg}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: current.Namespace, Name: current.Name},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	var got axisv1alpha1.MLJob
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: current.Namespace, Name: current.Name}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	return got
}

func TestReconcile_RejectsRoleTopologyMutation(t *testing.T) {
	// Once a CR has been observed (annotation anchored) the role
	// topology becomes immutable: handlers do not implement scale-down
	// or rename. A mutation must flip phase to Failed with a clear
	// message, not silently re-reconcile.
	prior := newPendingMLJob() // replicas=2
	current := newPendingMLJob()
	current.Generation = 2
	current.Spec.Roles[0].Replicas = 5 // attempted scale-up

	got := reconcileImmutabilityRejection(t, prior, current)
	if got.Status.Phase != axisv1alpha1.PhaseFailed {
		t.Fatalf("phase: want Failed, got %q (msg=%q)", got.Status.Phase, got.Status.Message)
	}
	if got.Status.Message == "" {
		t.Fatalf("expected status.message to explain the immutability violation")
	}
}

func TestReconcile_RejectsTemplateMutation(t *testing.T) {
	// Handlers don't reconcile template drift on the live underlying
	// resource (Job/PodGroup is built once at creation). A user editing
	// the image after the CR is anchored must see Failed, not silent
	// no-op.
	prior := newPendingMLJob()
	prior.Spec.Roles[0].Template.Image = "old-image"
	current := newPendingMLJob()
	current.Generation = 2
	current.Spec.Roles[0].Template.Image = "new-image"

	got := reconcileImmutabilityRejection(t, prior, current)
	if got.Status.Phase != axisv1alpha1.PhaseFailed {
		t.Fatalf("template mutation must produce Failed, got %q", got.Status.Phase)
	}
}

func TestReconcile_RejectsRunPolicyDeadlineMutation(t *testing.T) {
	// Non-Suspend RunPolicy fields are baked into the underlying Job at
	// create time and cannot be reconciled. Mutating them post-anchor
	// must Fail.
	prior := newPendingMLJob()
	current := newPendingMLJob()
	current.Generation = 2
	d := int64(120)
	current.Spec.RunPolicy.ActiveDeadlineSeconds = &d

	got := reconcileImmutabilityRejection(t, prior, current)
	if got.Status.Phase != axisv1alpha1.PhaseFailed {
		t.Fatalf("runPolicy mutation must produce Failed, got %q", got.Status.Phase)
	}
}

func TestReconcile_AcceptsSuspendMutation(t *testing.T) {
	// Suspend is the documented cancel mechanism — mutating it must
	// NOT trigger immutability rejection.
	prior := newPendingMLJob()
	current := newPendingMLJob()
	current.Generation = 2
	current.Spec.RunPolicy.Suspend = true

	got := reconcileImmutabilityRejection(t, prior, current)
	if got.Status.Phase == axisv1alpha1.PhaseFailed {
		t.Fatalf("Suspend toggle must be accepted; got Failed (msg=%q)", got.Status.Message)
	}
}

// Compile-time assertion that recordingHandler satisfies both the
// Handler contract and the optional Sweeper extension.
var _ axishandler.Handler = (*recordingHandler)(nil)
var _ axishandler.Sweeper = (*recordingHandler)(nil)
