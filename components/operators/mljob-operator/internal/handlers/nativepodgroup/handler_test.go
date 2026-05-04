package nativepodgroup

import (
	"context"
	"testing"
	"time"

	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	axisv1alpha1 "github.com/axisml/axisml/components/operators/mljob-operator/api/v1alpha1"
	axishandler "github.com/axisml/axisml/components/operators/mljob-operator/internal/handler"
	axislabels "github.com/axisml/axisml/components/operators/mljob-operator/internal/labels"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := schedulingv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := axisv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func newMLJob(replicas int32, suspend bool) *axisv1alpha1.MLJob {
	return &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gang",
			Namespace: "tnt",
			Labels: map[string]string{
				axislabels.JobIDLabel: "11111111-2222-3333-4444-555555555555",
				axislabels.QuotaLabel: "training",
			},
		},
		Spec: axisv1alpha1.MLJobSpec{
			Backend: axisv1alpha1.BackendSpec{Name: "native", Engine: "podgroup"},
			Scheduling: axisv1alpha1.SchedulingSpec{
				Quota: "axisml-tnt-default-training",
			},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          axisv1alpha1.DefaultRoleName,
				Replicas:      replicas,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:1.36",
					Command: []string{"sh", "-c", "sleep 1"},
				},
			}},
			RunPolicy: axisv1alpha1.RunPolicySpec{Suspend: suspend},
		},
	}
}

func TestReconcile_CreatesPodGroupAndPods(t *testing.T) {
	s := newScheme(t)
	c := fakeclient.NewClientBuilder().WithScheme(s).Build()
	h := New()
	mlj := newMLJob(3, false)

	if _, _, err := h.Reconcile(context.Background(), c, mlj); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var pg schedulingv1alpha1.PodGroup
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "tnt", Name: "gang"}, &pg); err != nil {
		t.Fatalf("PodGroup not created: %v", err)
	}
	if pg.Spec.MinMember != 3 {
		t.Errorf("PodGroup minMember: want 3, got %d", pg.Spec.MinMember)
	}
	var pods corev1.PodList
	if err := c.List(context.Background(), &pods,
		client.InNamespace("tnt"),
		client.MatchingLabels{axislabels.JobIDLabel: mlj.Labels[axislabels.JobIDLabel]}); err != nil {
		t.Fatalf("list Pods: %v", err)
	}
	if len(pods.Items) != 3 {
		t.Errorf("Pods: want 3, got %d", len(pods.Items))
	}
	for _, p := range pods.Items {
		if p.Labels[axislabels.PodGroupLabel] != "gang" {
			t.Errorf("Pod %s missing pod-group label, got %q", p.Name, p.Labels[axislabels.PodGroupLabel])
		}
		if p.Spec.SchedulerName != axislabels.KoordSchedulerName {
			t.Errorf("Pod %s scheduler: want %q, got %q", p.Name, axislabels.KoordSchedulerName, p.Spec.SchedulerName)
		}
	}
}

func TestReconcile_IsIdempotent(t *testing.T) {
	// Two consecutive Reconciles with unchanged spec must not create
	// duplicate Pods or recreate the PodGroup (deterministic Pod names
	// + Get-before-Create).
	s := newScheme(t)
	c := fakeclient.NewClientBuilder().WithScheme(s).Build()
	h := New()
	mlj := newMLJob(2, false)

	for i := 0; i < 2; i++ {
		if _, _, err := h.Reconcile(context.Background(), c, mlj); err != nil {
			t.Fatalf("Reconcile #%d: %v", i+1, err)
		}
	}

	var pods corev1.PodList
	if err := c.List(context.Background(), &pods,
		client.InNamespace("tnt"),
		client.MatchingLabels{axislabels.JobIDLabel: mlj.Labels[axislabels.JobIDLabel]}); err != nil {
		t.Fatalf("list Pods: %v", err)
	}
	if len(pods.Items) != 2 {
		t.Fatalf("Pods: want 2 after idempotent reconcile, got %d", len(pods.Items))
	}

	var pgList schedulingv1alpha1.PodGroupList
	if err := c.List(context.Background(), &pgList, client.InNamespace("tnt")); err != nil {
		t.Fatalf("list PodGroups: %v", err)
	}
	if len(pgList.Items) != 1 {
		t.Fatalf("PodGroups: want 1 after idempotent reconcile, got %d", len(pgList.Items))
	}
}

func TestReconcile_SuspendOrderedShutdown(t *testing.T) {
	s := newScheme(t)
	pg := &schedulingv1alpha1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "gang", Namespace: "tnt"},
		Spec:       schedulingv1alpha1.PodGroupSpec{MinMember: 2},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gang-0", Namespace: "tnt",
			Labels: map[string]string{
				axislabels.JobIDLabel:   "11111111-2222-3333-4444-555555555555",
				axislabels.PodGroupLabel: "gang",
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(pg, pod).Build()
	h := New()
	mlj := newMLJob(2, true)

	_, recRes, err := h.Reconcile(context.Background(), c, mlj)
	if err != nil {
		t.Fatalf("Reconcile suspend: %v", err)
	}
	if !recRes.SuspendCompleted {
		t.Errorf("SuspendCompleted not set")
	}
	if recRes.SuspendReason != axisv1alpha1.ReasonCancelRequested {
		t.Errorf("SuspendReason: want CancelRequested, got %q", recRes.SuspendReason)
	}

	var got schedulingv1alpha1.PodGroup
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "tnt", Name: "gang"}, &got); err != nil {
		t.Fatalf("get PodGroup: %v", err)
	}
	if got.Spec.MinMember != 0 {
		t.Errorf("minMember after suspend: want 0, got %d", got.Spec.MinMember)
	}
	var podGot corev1.Pod
	err = c.Get(context.Background(), client.ObjectKey{Namespace: "tnt", Name: "gang-0"}, &podGot)
	if err == nil && podGot.DeletionTimestamp == nil {
		t.Errorf("Pod should be deleted (or have deletionTimestamp) after suspend")
	}
}

func TestMapStatus_AllSucceeded(t *testing.T) {
	h := New()
	now := metav1.Now()
	pods := []corev1.Pod{
		{
			Status: corev1.PodStatus{
				Phase: corev1.PodSucceeded,
				ContainerStatuses: []corev1.ContainerStatus{
					{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{FinishedAt: now}}},
				},
			},
		},
		{
			Status: corev1.PodStatus{
				Phase: corev1.PodSucceeded,
				ContainerStatuses: []corev1.ContainerStatus{
					{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{FinishedAt: now}}},
				},
			},
		},
	}
	got := h.MapStatus(underlying{Pods: pods, DesiredReplicas: 2})
	if got.Phase != axisv1alpha1.PhaseSucceeded {
		t.Fatalf("phase: want Succeeded, got %q", got.Phase)
	}
}

func TestMapStatus_AnyFailedWins(t *testing.T) {
	h := New()
	pods := []corev1.Pod{
		{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
		{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
	}
	got := h.MapStatus(underlying{Pods: pods, DesiredReplicas: 2})
	if got.Phase != axisv1alpha1.PhaseFailed {
		t.Fatalf("phase: want Failed, got %q", got.Phase)
	}
}

func TestMapStatus_RunningWhenAnyRunning(t *testing.T) {
	h := New()
	pods := []corev1.Pod{
		{Status: corev1.PodStatus{Phase: corev1.PodPending}},
		{Status: corev1.PodStatus{Phase: corev1.PodRunning}},
	}
	got := h.MapStatus(underlying{Pods: pods, DesiredReplicas: 2})
	if got.Phase != axisv1alpha1.PhaseRunning {
		t.Fatalf("phase: want Running, got %q", got.Phase)
	}
}

// TestReconcile_TTLZeroPreservesTerminalSnapshot is the regression for
// the "MapStatus loses terminal phase under TTL=0" bug: when every Pod
// is terminal and the TTL deadline has already elapsed, the same
// reconcile both deletes the Pods AND must hand the terminal snapshot
// to MapStatus. Otherwise the dispatcher would record Pending and the
// next reconcile would recreate the Pods.
func TestReconcile_TTLZeroPreservesTerminalSnapshot(t *testing.T) {
	s := newScheme(t)
	finishedAt := metav1.NewTime(time.Now().Add(-time.Minute))
	mkPod := func(name string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: "tnt",
				Labels: map[string]string{
					axislabels.JobIDLabel:    "11111111-2222-3333-4444-555555555555",
					axislabels.PodGroupLabel: "gang",
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodSucceeded,
				ContainerStatuses: []corev1.ContainerStatus{{
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{FinishedAt: finishedAt}},
				}},
			},
		}
	}
	pg := &schedulingv1alpha1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "gang", Namespace: "tnt"},
		Spec:       schedulingv1alpha1.PodGroupSpec{MinMember: 2},
	}
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(pg, mkPod("gang-0"), mkPod("gang-1")).Build()
	h := New()
	mlj := newMLJob(2, false)
	zero := int32(0)
	mlj.Spec.RunPolicy.TTLSecondsAfterFinished = &zero

	got, _, err := h.Reconcile(context.Background(), c, mlj)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	u, ok := got.(underlying)
	if !ok {
		t.Fatalf("underlying type: %T", got)
	}
	if len(u.Pods) != 2 {
		t.Fatalf("Pods snapshot must be preserved for MapStatus, got %d", len(u.Pods))
	}
	res := h.MapStatus(u)
	if res.Phase != axisv1alpha1.PhaseSucceeded {
		t.Fatalf("MapStatus must observe terminal phase, got %q", res.Phase)
	}
}

// TestSweep_DeletesPodsAfterTTL exercises the post-terminal hook: once
// dispatcher short-circuits a terminal MLJob, Sweep must still GC bare
// Pods after the TTL window.
func TestSweep_DeletesPodsAfterTTL(t *testing.T) {
	s := newScheme(t)
	finishedAt := metav1.NewTime(time.Now().Add(-time.Minute))
	mkPod := func(name string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: "tnt",
				Labels: map[string]string{
					axislabels.JobIDLabel:    "11111111-2222-3333-4444-555555555555",
					axislabels.PodGroupLabel: "gang",
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodSucceeded,
				ContainerStatuses: []corev1.ContainerStatus{{
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{FinishedAt: finishedAt}},
				}},
			},
		}
	}
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(mkPod("gang-0"), mkPod("gang-1")).Build()
	h := New()
	mlj := newMLJob(2, false)
	zero := int32(0)
	mlj.Spec.RunPolicy.TTLSecondsAfterFinished = &zero

	requeue, err := h.Sweep(context.Background(), c, mlj)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if requeue != 0 {
		t.Fatalf("Sweep requeue: want 0 (TTL expired), got %d", requeue)
	}
	var pods corev1.PodList
	if err := c.List(context.Background(), &pods, client.InNamespace("tnt")); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pods.Items) != 0 {
		t.Fatalf("Sweep must delete terminal Pods after TTL, %d remain", len(pods.Items))
	}
}

// TestSweep_RequeuesBeforeTTL ensures Sweep schedules a follow-up
// invocation when the TTL deadline has not yet elapsed.
func TestSweep_RequeuesBeforeTTL(t *testing.T) {
	s := newScheme(t)
	finishedAt := metav1.NewTime(time.Now())
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gang-0", Namespace: "tnt",
			Labels: map[string]string{
				axislabels.JobIDLabel: "11111111-2222-3333-4444-555555555555",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
			ContainerStatuses: []corev1.ContainerStatus{{
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{FinishedAt: finishedAt}},
			}},
		},
	}
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(pod).Build()
	h := New()
	mlj := newMLJob(1, false)
	ttl := int32(60)
	mlj.Spec.RunPolicy.TTLSecondsAfterFinished = &ttl

	requeue, err := h.Sweep(context.Background(), c, mlj)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if requeue <= 0 {
		t.Fatalf("Sweep must request requeue before TTL expiry, got %d", requeue)
	}
	var got corev1.Pod
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "tnt", Name: "gang-0"}, &got); err != nil {
		t.Fatalf("Pod must NOT be deleted before TTL expiry: %v", err)
	}
}

// Compile-time assertion: the (native, podgroup) handler implements
// the optional Sweeper extension (used by the dispatcher's terminal
// short-circuit to drive post-terminal TTL GC).
var _ axishandler.Sweeper = (*Handler)(nil)
