//go:build integration

package integration_test

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

	axisv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	axislabels "github.com/axisml/axisml/components/compute-operator/internal/mlrun/labels"

	"github.com/axisml/axisml/test/testutil"
)

// TestMLRun_NativeJob_HappyPath drives the (native, job) handler end-to-end:
//
//  1. Create a Namespace and an MLRun{native, job} CR.
//  2. Assert the dispatcher creates a batch/v1.Job in the namespace.
//  3. Patch the Job's status to simulate kubelet completing it (envtest has
//     no kubelet, so the Job's Pods never run).
//  4. Assert MLRun.Status.Phase converges to Succeeded.
func TestMLRun_NativeJob_HappyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns        = "envt-mlrun"
		mlrunName = "smoke"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mlrun := &axisv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      mlrunName,
			Labels: map[string]string{
				axislabels.RunIDLabel: "uuid-smoke",
				axislabels.QuotaLabel: "training",
			},
		},
		Spec: axisv1alpha1.MLRunSpec{
			Backend:    axisv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Scheduling: axisv1alpha1.SchedulingSpec{Quota: "axisml-acme-default-training"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          axisv1alpha1.DefaultRoleName,
				Replicas:      1,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "echo hello"},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mlrun))

	// Dispatcher creates the underlying batch/v1.Job (named after the MLRun).
	var job batchv1.Job
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: mlrunName}, &job, testWaitTimeout)

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
		var got axisv1alpha1.MLRun
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mlrunName}, &got); err != nil {
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
