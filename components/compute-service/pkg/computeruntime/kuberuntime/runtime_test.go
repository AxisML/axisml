package kuberuntime

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltrafficpolicyv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, mlrunv1alpha1.AddToScheme(s))
	require.NoError(t, mlservicev1alpha1.AddToScheme(s))
	require.NoError(t, mltrafficpolicyv1alpha1.AddToScheme(s))
	return s
}

func newRuntime(t *testing.T, objs ...client.Object) *KubernetesRuntime {
	t.Helper()
	cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(objs...).Build()
	return New(cl, fakeclientset.NewSimpleClientset())
}

func runKey() types.NamespacedName { return types.NamespacedName{Namespace: "default", Name: "job-a"} }

func TestApplyMLRun_CreatesThenIdempotent(t *testing.T) {
	ctx := context.Background()
	rt := newRuntime(t)

	desired := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default", Name: "job-a",
			Labels: map[string]string{mlrunv1alpha1.LabelRunID: "run-1"},
		},
	}
	require.NoError(t, rt.ApplyMLRun(ctx, desired))

	got := &mlrunv1alpha1.MLRun{}
	require.NoError(t, rt.ctrl.Get(ctx, runKey(), got))
	assert.Equal(t, "run-1", got.Labels[mlrunv1alpha1.LabelRunID])

	// Second apply is a no-op (Run spec is immutable post-create).
	require.NoError(t, rt.ApplyMLRun(ctx, desired))
}

func TestObserveMLRun(t *testing.T) {
	ctx := context.Background()
	seed := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "job-a"},
		Status:     mlrunv1alpha1.MLRunStatus{Phase: mlrunv1alpha1.PhaseRunning},
	}
	rt := newRuntime(t, seed)

	st, err := rt.ObserveMLRun(ctx, runKey())
	require.NoError(t, err)
	assert.Equal(t, mlrunv1alpha1.PhaseRunning, st.Phase)

	_, err = rt.ObserveMLRun(ctx, types.NamespacedName{Namespace: "default", Name: "missing"})
	assert.True(t, apierrors.IsNotFound(err), "missing CR must surface NotFound")
}

func TestCancelMLRun_PatchesSuspend(t *testing.T) {
	ctx := context.Background()
	seed := &mlrunv1alpha1.MLRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "job-a"}}
	rt := newRuntime(t, seed)

	require.NoError(t, rt.CancelMLRun(ctx, runKey()))
	got := &mlrunv1alpha1.MLRun{}
	require.NoError(t, rt.ctrl.Get(ctx, runKey(), got))
	assert.True(t, got.Spec.RunPolicy.Suspend)

	// Idempotent on a missing CR.
	require.NoError(t, rt.CancelMLRun(ctx, types.NamespacedName{Namespace: "default", Name: "missing"}))
}

func TestDeleteMLRun_Idempotent(t *testing.T) {
	ctx := context.Background()
	seed := &mlrunv1alpha1.MLRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "job-a"}}
	rt := newRuntime(t, seed)

	require.NoError(t, rt.DeleteMLRun(ctx, runKey()))
	err := rt.ctrl.Get(ctx, runKey(), &mlrunv1alpha1.MLRun{})
	assert.True(t, apierrors.IsNotFound(err))

	// Deleting an already-gone Run is a no-op.
	require.NoError(t, rt.DeleteMLRun(ctx, runKey()))
}

func TestListMLRunInstances_FiltersByLabel(t *testing.T) {
	ctx := context.Background()
	run := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default", Name: "job-a",
			Labels: map[string]string{mlrunv1alpha1.LabelRunID: "run-1"},
		},
	}
	mine := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "default", Name: "p-mine",
		Labels: map[string]string{mlrunv1alpha1.LabelRunID: "run-1"},
	}}
	other := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "default", Name: "p-other",
		Labels: map[string]string{mlrunv1alpha1.LabelRunID: "run-2"},
	}}
	rt := newRuntime(t, run, mine, other)

	pods, err := rt.ListMLRunInstances(ctx, runKey())
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	assert.Equal(t, "p-mine", pods.Items[0].Name)
}

func TestGetMLRunInstanceLogs_VerifiesOwnership(t *testing.T) {
	ctx := context.Background()
	run := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default", Name: "job-a",
			Labels: map[string]string{mlrunv1alpha1.LabelRunID: "run-1"},
		},
	}
	mine := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "default", Name: "p-mine",
		Labels: map[string]string{mlrunv1alpha1.LabelRunID: "run-1"},
	}}
	foreign := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "default", Name: "p-foreign",
		Labels: map[string]string{mlrunv1alpha1.LabelRunID: "run-9"},
	}}
	rt := newRuntime(t, run, mine, foreign)

	rc, err := rt.GetMLRunInstanceLogs(ctx, runKey(), "p-mine", nil)
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()
	body, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.NotEmpty(t, body)

	// A pod that doesn't belong to the Run must be rejected before streaming.
	_, err = rt.GetMLRunInstanceLogs(ctx, runKey(), "p-foreign", nil)
	require.Error(t, err)
}

func TestGetMLRunEvents_FiltersRegarding(t *testing.T) {
	ctx := context.Background()
	run := &mlrunv1alpha1.MLRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "job-a"}}
	evRun := &eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "e1"},
		Regarding:  corev1.ObjectReference{Kind: "MLRun", Name: "job-a"},
		Reason:     "Created",
	}
	evOther := &eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "e2"},
		Regarding:  corev1.ObjectReference{Kind: "MLRun", Name: "job-b"},
		Reason:     "Created",
	}
	rt := newRuntime(t, run, evRun, evOther)

	list, err := rt.GetMLRunEvents(ctx, runKey())
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "e1", list.Items[0].Name)
}

func TestApplyMLService_CreateThenConverge(t *testing.T) {
	ctx := context.Background()
	rt := newRuntime(t)
	key := types.NamespacedName{Namespace: "default", Name: "svc-a"}

	desired := &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default", Name: "svc-a",
			Labels: map[string]string{mlservicev1alpha1.LabelServiceID: "svc-1"},
		},
		Spec: mlservicev1alpha1.MLServiceSpec{
			Roles: []mlservicev1alpha1.RoleSpec{{Name: "main", Replicas: 1}},
		},
	}
	require.NoError(t, rt.ApplyMLService(ctx, desired))

	// Converge: bump replicas and re-apply.
	desired.Spec.Roles[0].Replicas = 3
	require.NoError(t, rt.ApplyMLService(ctx, desired))

	got := &mlservicev1alpha1.MLService{}
	require.NoError(t, rt.ctrl.Get(ctx, key, got))
	require.Len(t, got.Spec.Roles, 1)
	assert.Equal(t, int32(3), got.Spec.Roles[0].Replicas)
}

func TestApplyMLTrafficPolicy_AndEvents(t *testing.T) {
	ctx := context.Background()
	rt := newRuntime(t)
	key := types.NamespacedName{Namespace: "default", Name: "tp-a"}

	desired := &mltrafficpolicyv1alpha1.MLTrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default", Name: "tp-a",
			Labels: map[string]string{mltrafficpolicyv1alpha1.LabelTrafficPolicyID: "tp-1"},
		},
	}
	require.NoError(t, rt.ApplyMLTrafficPolicy(ctx, desired))

	got := &mltrafficpolicyv1alpha1.MLTrafficPolicy{}
	require.NoError(t, rt.ctrl.Get(ctx, key, got))
	assert.Equal(t, "tp-1", got.Labels[mltrafficpolicyv1alpha1.LabelTrafficPolicyID])

	require.NoError(t, rt.DeleteMLTrafficPolicy(ctx, key))
	assert.True(t, apierrors.IsNotFound(rt.ctrl.Get(ctx, key, &mltrafficpolicyv1alpha1.MLTrafficPolicy{})))

	// Events filter on the policy CR; an empty namespace yields an empty list.
	list, err := rt.GetMLTrafficPolicyEvents(ctx, key)
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}
