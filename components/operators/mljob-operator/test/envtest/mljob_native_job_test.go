//go:build envtest

package envtest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisv1alpha1 "axisml.io/operators/mljob/api/v1alpha1"
	axislabels "axisml.io/operators/mljob/internal/labels"

	"github.com/axisml-io/axisml/test/testutil"
)

const testWaitTimeout = 30 * time.Second

// TestMLJob_NativeJob_HappyPath drives the (native, job) handler end-to-end:
//
//  1. Create a Namespace and an MLJob{native, job} CR.
//  2. Assert the dispatcher creates a batch/v1.Job in the namespace.
//  3. Patch the Job's status to simulate kubelet completing it (envtest has
//     no kubelet, so the Job's Pods never run).
//  4. Assert MLJob.Status.Phase converges to Succeeded.
func TestMLJob_NativeJob_HappyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns        = "envt-mljob"
		mljobName = "smoke"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mljob := &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      mljobName,
			Labels: map[string]string{
				axislabels.JobIDLabel: "uuid-smoke",
				axislabels.QuotaLabel: "training",
			},
		},
		Spec: axisv1alpha1.MLJobSpec{
			Backend:    axisv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Scheduling: axisv1alpha1.SchedulingSpec{Quota: "axisml-acme-default-training"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          "worker",
				Replicas:      1,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "echo hello"},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mljob))

	// Dispatcher creates the underlying batch/v1.Job (named after the MLJob).
	var job batchv1.Job
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: mljobName}, &job, testWaitTimeout)

	// Simulate the kubelet completing the Job. The dispatcher's MapStatus
	// reads JobComplete=True from Status.Conditions and maps it to Succeeded.
	now := metav1.NewTime(time.Now())
	patch := client.MergeFrom(job.DeepCopy())
	job.Status.Succeeded = 1
	job.Status.StartTime = &now
	job.Status.CompletionTime = &now
	job.Status.Conditions = []batchv1.JobCondition{{
		Type:               batchv1.JobComplete,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: now,
	}}
	require.NoError(t, c.Status().Patch(ctx, &job, patch))

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

// cleanupNamespace deletes a namespace and ignores NotFound. Used in
// t.Cleanup hooks; envtest deletes namespaces ~instantly because there's
// no controller manager doing finalizer GC.
func cleanupNamespace(t *testing.T, c client.Client, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := c.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
		t.Logf("cleanup namespace %q: %v", name, err)
	}
}
