//go:build envtest

package envtest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisv1alpha1 "github.com/axisml/axisml/components/operators/mljob-operator/api/v1alpha1"
	axislabels "github.com/axisml/axisml/components/operators/mljob-operator/internal/labels"

	"github.com/axisml/axisml/test/testutil"
)

// TestMLJob_NativePodGroup_HappyPath drives the (native, podgroup) handler
// end-to-end:
//
//  1. Create a Namespace and an MLJob{native, podgroup} with replicas=2.
//  2. Assert the dispatcher creates a PodGroup (minMember=2) plus 2 bare Pods.
//  3. Patch every Pod to Phase=Succeeded (envtest has no kubelet).
//  4. Assert MLJob.Status.Phase converges to Succeeded.
func TestMLJob_NativePodGroup_HappyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns        = "envt-mljob-pg"
		mljobName = "gang"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mljob := &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      mljobName,
			Labels: map[string]string{
				axislabels.JobIDLabel: "uuid-gang",
				axislabels.QuotaLabel: "axisml-acme-default-training",
			},
		},
		Spec: axisv1alpha1.MLJobSpec{
			Backend:    axisv1alpha1.BackendSpec{Name: "native", Engine: "podgroup"},
			Scheduling: axisv1alpha1.SchedulingSpec{Quota: "axisml-acme-default-training"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          axisv1alpha1.DefaultRoleName,
				Replicas:      2,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "echo hello"},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mljob))

	// Dispatcher creates the PodGroup, named after the MLJob.
	var pg schedulingv1alpha1.PodGroup
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: mljobName}, &pg, testWaitTimeout)
	require.Equal(t, int32(2), pg.Spec.MinMember, "PodGroup minMember mismatch")

	// And two bare Pods named gang-0, gang-1.
	pod0 := &corev1.Pod{}
	pod1 := &corev1.Pod{}
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: ns, Name: "gang-0"}, pod0, testWaitTimeout)
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: ns, Name: "gang-1"}, pod1, testWaitTimeout)
	for _, p := range []*corev1.Pod{pod0, pod1} {
		require.Equal(t, axislabels.KoordSchedulerName, p.Spec.SchedulerName,
			"Pod %s must use koord-scheduler", p.Name)
		require.Equal(t, mljobName, p.Labels[axislabels.PodGroupLabel],
			"Pod %s must carry pod-group label", p.Name)
	}

	// Simulate the kubelet completing every Pod.
	now := metav1.NewTime(time.Now())
	for _, name := range []string{"gang-0", "gang-1"} {
		var pod corev1.Pod
		require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, &pod))
		patch := client.MergeFrom(pod.DeepCopy())
		pod.Status.Phase = corev1.PodSucceeded
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
			Name: "default",
			State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
				FinishedAt: now,
				ExitCode:   0,
			}},
		}}
		require.NoError(t, c.Status().Patch(ctx, &pod, patch))
	}

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisv1alpha1.MLJob
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisv1alpha1.PhaseSucceeded {
			return fmt.Errorf("phase=%q (msg=%q)", got.Status.Phase, got.Status.Message)
		}
		return nil
	})
}

// TestMLJob_NativePodGroup_SuspendShutdown verifies the ordered shutdown
// path: spec.runPolicy.suspend=true must (a) patch PodGroup minMember=0
// before deleting Pods, and (b) cause the dispatcher to set the Suspended
// condition with reason CancelRequested. MLJob has no PhaseSuspended;
// cancellation is reflected through the condition.
func TestMLJob_NativePodGroup_SuspendShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns        = "envt-mljob-pg-susp"
		mljobName = "gang-susp"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mljob := &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      mljobName,
			Labels: map[string]string{
				axislabels.JobIDLabel: "uuid-gang-susp",
				axislabels.QuotaLabel: "axisml-acme-default-training",
			},
		},
		Spec: axisv1alpha1.MLJobSpec{
			Backend:    axisv1alpha1.BackendSpec{Name: "native", Engine: "podgroup"},
			Scheduling: axisv1alpha1.SchedulingSpec{Quota: "axisml-acme-default-training"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          axisv1alpha1.DefaultRoleName,
				Replicas:      2,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "sleep 3600"},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mljob))

	// Wait for both Pods to exist before suspend, otherwise the suspend
	// path runs before the dispatcher creates anything and the test never
	// observes the deletion semantics.
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: "gang-susp-0"}, &corev1.Pod{}, testWaitTimeout)
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: "gang-susp-1"}, &corev1.Pod{}, testWaitTimeout)

	// Toggle suspend.
	var fresh axisv1alpha1.MLJob
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &fresh))
	patch := client.MergeFrom(fresh.DeepCopy())
	fresh.Spec.RunPolicy.Suspend = true
	require.NoError(t, c.Patch(ctx, &fresh, patch))

	// PodGroup minMember must drop to 0 before Pods disappear.
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var pg schedulingv1alpha1.PodGroup
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &pg); err != nil {
			return err
		}
		if pg.Spec.MinMember != 0 {
			return fmt.Errorf("PodGroup.minMember=%d, want 0", pg.Spec.MinMember)
		}
		return nil
	})

	// Pods should be deleted (or have a deletionTimestamp).
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		for _, name := range []string{"gang-susp-0", "gang-susp-1"} {
			var p corev1.Pod
			err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, &p)
			if err == nil && p.DeletionTimestamp == nil {
				return fmt.Errorf("Pod %s still alive (no deletionTimestamp)", name)
			}
		}
		return nil
	})

	// Dispatcher records the Suspended condition with CancelRequested reason.
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisv1alpha1.MLJob
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &got); err != nil {
			return err
		}
		for _, cond := range got.Status.Conditions {
			if cond.Type == axisv1alpha1.ConditionSuspended &&
				cond.Status == metav1.ConditionTrue &&
				cond.Reason == axisv1alpha1.ReasonCancelRequested {
				return nil
			}
		}
		return fmt.Errorf("Suspended/CancelRequested condition missing; conditions=%v", got.Status.Conditions)
	})
}
