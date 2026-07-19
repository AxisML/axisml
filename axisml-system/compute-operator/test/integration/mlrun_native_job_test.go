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

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	configapi "github.com/axisml/axisml/axisml-system/apis/pkg/workloadconfig"
	"github.com/axisml/axisml/axisml-system/apis/pkg/workloadname"
	axislabels "github.com/axisml/axisml/axisml-system/compute-operator/internal/mlrun/labels"

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
			Scheduling: axisv1alpha1.SchedulingSpec{Quota: "axisml-acme-default"},
			ConfigMaps: []configapi.ConfigMap{{
				Name: "trainer-config",
				Data: map[string]string{"trainer.yaml": "epochs: 3"},
			}},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          axisv1alpha1.DefaultRoleName,
				Replicas:      1,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "echo hello"},
					EnvFrom: []corev1.EnvFromSource{{
						ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "trainer-config"}},
					}},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mlrun))
	jobName := workloadname.Role(mlrun, axisv1alpha1.DefaultRoleName)

	// Dispatcher creates the underlying batch/v1.Job using the shared
	// workload-plus-role naming contract.
	var job batchv1.Job
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: jobName}, &job, testWaitTimeout)
	var configMap corev1.ConfigMap
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: "trainer-config"}, &configMap, testWaitTimeout)
	require.Equal(t, "epochs: 3", configMap.Data["trainer.yaml"])
	require.True(t, metav1.IsControlledBy(&configMap, mlrun))
	require.Equal(t, "trainer-config", job.Spec.Template.Spec.Containers[0].EnvFrom[0].ConfigMapRef.Name)

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

// TestMLRun_NativeJob_MountsVolume drives the (native, job) handler with a
// PVC-backed dataset volume declared on the role template and asserts the
// dispatcher injects it into the underlying Job: the PodSpec carries the
// volume (keyed on its claim) and the container carries the matching mount.
func TestMLRun_NativeJob_MountsVolume(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns        = "envt-mlrun-vol"
		mlrunName = "with-data"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mlrun := &axisv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      mlrunName,
			Labels: map[string]string{
				axislabels.RunIDLabel: "uuid-vol",
				axislabels.QuotaLabel: "training",
			},
		},
		Spec: axisv1alpha1.MLRunSpec{
			Backend:    axisv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Scheduling: axisv1alpha1.SchedulingSpec{Quota: "axisml-acme-default"},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          axisv1alpha1.DefaultRoleName,
				Replicas:      1,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "ls /data"},
					Volumes: []corev1.Volume{{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "dataset-1"},
						},
					}},
					VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mlrun))
	jobName := workloadname.Role(mlrun, axisv1alpha1.DefaultRoleName)

	var job batchv1.Job
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: jobName}, &job, testWaitTimeout)

	podSpec := job.Spec.Template.Spec
	require.Len(t, podSpec.Volumes, 1, "PodSpec must carry the declared volume")
	require.Equal(t, "data", podSpec.Volumes[0].Name)
	require.NotNil(t, podSpec.Volumes[0].PersistentVolumeClaim, "volume source must survive")
	require.Equal(t, "dataset-1", podSpec.Volumes[0].PersistentVolumeClaim.ClaimName)

	require.Len(t, podSpec.Containers, 1)
	mounts := podSpec.Containers[0].VolumeMounts
	require.Len(t, mounts, 1, "container must carry the declared volumeMount")
	require.Equal(t, "data", mounts[0].Name)
	require.Equal(t, "/data", mounts[0].MountPath)
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
