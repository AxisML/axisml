//go:build envtest

package envtest_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisv1alpha1 "github.com/axisml/axisml/components/operator/api/mljob/v1alpha1"
	axislabels "github.com/axisml/axisml/components/operator/internal/mljob/labels"

	"github.com/axisml/axisml/test/testutil"
)

// TestMLJob_NativeJob_SuspendCancels verifies the cancel propagation:
// flipping spec.runPolicy.suspend=true must (a) patch the underlying
// batch/v1.Job's spec.suspend=true and (b) set the MLJob's Suspended
// condition with reason CancelRequested. There is no PhaseSuspended;
// MLJob has only Pending/Running/Succeeded/Failed phases.
func TestMLJob_NativeJob_SuspendCancels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns        = "envt-mljob-susp"
		mljobName = "cancelme"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mljob := &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      mljobName,
			Labels: map[string]string{
				axislabels.JobIDLabel: "uuid-cancelme",
				axislabels.QuotaLabel: "axisml-acme-default-default",
			},
		},
		Spec: axisv1alpha1.MLJobSpec{
			Backend:    axisv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Scheduling: axisv1alpha1.SchedulingSpec{Quota: "axisml-acme-default-default"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          axisv1alpha1.DefaultRoleName,
				Replicas:      1,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "sleep 3600"},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mljob))

	// Wait for the Job to exist before flipping suspend; the reconcile-
	// before-create path is exercised separately at the unit-test level.
	var job batchv1.Job
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: mljobName}, &job, testWaitTimeout)

	// Toggle suspend.
	var fresh axisv1alpha1.MLJob
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &fresh))
	patch := client.MergeFrom(fresh.DeepCopy())
	fresh.Spec.RunPolicy.Suspend = true
	require.NoError(t, c.Patch(ctx, &fresh, patch))

	// Job's spec.suspend flips to true.
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got batchv1.Job
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &got); err != nil {
			return err
		}
		if got.Spec.Suspend == nil || !*got.Spec.Suspend {
			return fmt.Errorf("Job.spec.suspend not propagated")
		}
		return nil
	})

	// Suspended condition with CancelRequested reason appears on the MLJob.
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

// TestMLJob_ImmutableSpecRejected verifies the dispatcher's spec-fingerprint
// guard (design §3.3): once the applied-spec annotation is anchored, mutating
// any locked field (e.g. role template image) must surface an immutable-spec
// error in status.message and force phase=Failed.
func TestMLJob_ImmutableSpecRejected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns        = "envt-mljob-immut"
		mljobName = "immutable"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mljob := &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      mljobName,
			Labels: map[string]string{
				axislabels.JobIDLabel: "uuid-immutable",
				axislabels.QuotaLabel: "axisml-acme-default-default",
			},
		},
		Spec: axisv1alpha1.MLJobSpec{
			Backend:    axisv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Scheduling: axisv1alpha1.SchedulingSpec{Quota: "axisml-acme-default-default"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          axisv1alpha1.DefaultRoleName,
				Replicas:      1,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:1.36",
					Command: []string{"sh", "-c", "echo hello"},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mljob))

	// Wait for the dispatcher to anchor the applied-spec annotation
	// (otherwise the very first reconcile won't have anything to compare
	// our mutation against).
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisv1alpha1.MLJob
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &got); err != nil {
			return err
		}
		if got.Annotations[axislabels.AppliedSpecAnnotation] == "" {
			return fmt.Errorf("applied-spec annotation not yet anchored")
		}
		return nil
	})

	// Mutate an immutable field (image).
	var fresh axisv1alpha1.MLJob
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &fresh))
	patch := client.MergeFrom(fresh.DeepCopy())
	fresh.Spec.Roles[0].Template.Image = "busybox:1.37"
	require.NoError(t, c.Patch(ctx, &fresh, patch))

	// Dispatcher should drive phase=Failed with an immutable-spec message.
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisv1alpha1.MLJob
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisv1alpha1.PhaseFailed {
			return fmt.Errorf("phase=%q (msg=%q), want Failed", got.Status.Phase, got.Status.Message)
		}
		if !strings.Contains(got.Status.Message, "immutable") {
			return fmt.Errorf("message %q does not mention immutability", got.Status.Message)
		}
		return nil
	})
}

// TestMLJob_UnknownBackend verifies the dispatcher's behavior when the
// (backend, engine) tuple has no registered handler: phase must move to
// Failed with a "no handler" message, and no underlying resources should
// be created.
func TestMLJob_UnknownBackend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns        = "envt-mljob-unkbe"
		mljobName = "no-handler"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mljob := &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      mljobName,
			Labels: map[string]string{
				axislabels.JobIDLabel: "uuid-no-handler",
				axislabels.QuotaLabel: "axisml-acme-default-default",
			},
		},
		Spec: axisv1alpha1.MLJobSpec{
			Backend:    axisv1alpha1.BackendSpec{Name: "kubeflow-trainer", Engine: "pytorchjob"},
			Scheduling: axisv1alpha1.SchedulingSpec{Quota: "axisml-acme-default-default"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:     axisv1alpha1.DefaultRoleName,
				Replicas: 1,
				Template: axisv1alpha1.PodTemplateSubset{Image: "busybox:latest"},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mljob))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisv1alpha1.MLJob
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mljobName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisv1alpha1.PhaseFailed {
			return fmt.Errorf("phase=%q (msg=%q), want Failed", got.Status.Phase, got.Status.Message)
		}
		if !strings.Contains(got.Status.Message, "no handler") {
			return fmt.Errorf("message %q does not mention missing handler", got.Status.Message)
		}
		return nil
	})

	// Sanity: no batch/v1 Job was created for the unknown backend.
	var jobs batchv1.JobList
	require.NoError(t, c.List(ctx, &jobs, client.InNamespace(ns)))
	require.Empty(t, jobs.Items, "no Job should be created for unknown backend")
}
