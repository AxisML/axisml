package docker

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

func testRuntime() *Runtime {
	return New(nil, Config{WorkloadsNetwork: "axisml-workloads", Tenant: "default"}, logr.Discard())
}

func TestRenderRunPlans(t *testing.T) {
	r := testRuntime()
	run := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "trainer"},
		Spec: mlrunv1alpha1.MLRunSpec{
			Backend: mlrunv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Roles: []mlrunv1alpha1.RoleSpec{{
				Name:     "worker",
				Replicas: 2,
				Template: mlrunv1alpha1.PodTemplateSubset{
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []string{"echo hi"},
					Env:     []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
			}},
		},
	}

	plans, err := r.renderRunPlans(run)
	require.NoError(t, err)
	require.Len(t, plans, 2)

	p := plans[0]
	assert.Equal(t, "axisml-run-default-trainer-worker-0", p.Name)
	assert.Equal(t, "busybox:latest", p.Image)
	assert.Equal(t, []string{"sh", "-c"}, p.Command)
	assert.Equal(t, []string{"echo hi"}, p.Args)
	assert.Equal(t, []string{"FOO=bar"}, p.Env)
	assert.Equal(t, string("no"), p.RestartPolicy)
	assert.Equal(t, int64(1_000_000_000), p.Resources.NanoCPUs)
	assert.Equal(t, int64(512*1024*1024), p.Resources.MemoryBytes)
	assert.Equal(t, "true", p.Labels[LabelManaged])
	assert.Equal(t, KindRun, p.Labels[LabelResourceKind])
	assert.Equal(t, "default", p.Labels[LabelNamespace])
	assert.Equal(t, "trainer", p.Labels[LabelName])
	assert.Equal(t, "worker", p.Labels[LabelRole])
	assert.Equal(t, "0", p.Labels[LabelReplicaIndex])
	assert.NotEmpty(t, p.Labels[LabelSpecHash])
	assert.Equal(t, "axisml-run-default-trainer-worker-1", plans[1].Name)
}

func TestRenderRunPlans_UnsupportedBackend(t *testing.T) {
	r := testRuntime()
	run := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "x"},
		Spec: mlrunv1alpha1.MLRunSpec{
			Backend: mlrunv1alpha1.BackendSpec{Name: "kubeflow-trainer", Engine: "pytorchjob"},
			Roles:   []mlrunv1alpha1.RoleSpec{{Name: "worker", Replicas: 1, Template: mlrunv1alpha1.PodTemplateSubset{Image: "x"}}},
		},
	}
	_, err := r.renderRunPlans(run)
	require.Error(t, err)
	var ce *CapabilityError
	assert.ErrorAs(t, err, &ce)
}

func TestRenderRunPlans_EnvFromUnsupported(t *testing.T) {
	r := testRuntime()
	run := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "x"},
		Spec: mlrunv1alpha1.MLRunSpec{
			Backend: mlrunv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Roles: []mlrunv1alpha1.RoleSpec{{Name: "worker", Replicas: 1, Template: mlrunv1alpha1.PodTemplateSubset{
				Image:   "busybox",
				EnvFrom: []corev1.EnvFromSource{{}},
			}}},
		},
	}
	_, err := r.renderRunPlans(run)
	var ce *CapabilityError
	require.ErrorAs(t, err, &ce)
}

func TestRenderServicePlans(t *testing.T) {
	r := testRuntime()
	svc := &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "infer"},
		Spec: mlservicev1alpha1.MLServiceSpec{
			Backend: mlservicev1alpha1.Backend{Name: "native", Engine: "deployment"},
			Roles: []mlservicev1alpha1.RoleSpec{{
				Name:     "predictor",
				Replicas: 2,
				Template: mlservicev1alpha1.PodTemplate{
					Image: "nginx:1.27",
					Ports: []mlservicev1alpha1.PodPort{{Name: "http", ContainerPort: 80}},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("1")},
					},
				},
			}},
		},
	}
	plans, err := r.renderServicePlans(svc)
	require.NoError(t, err)
	require.Len(t, plans, 2)
	p := plans[0]
	assert.Equal(t, "axisml-service-default-infer-predictor-0", p.Name)
	assert.Equal(t, string("unless-stopped"), p.RestartPolicy)
	require.Len(t, p.Ports, 1)
	assert.Equal(t, int32(80), p.Ports[0].ContainerPort)
	assert.Equal(t, 1, p.Resources.GPUCount)
	assert.Equal(t, KindService, p.Labels[LabelResourceKind])
}

func TestRenderServicePlans_VolumeMountUndeclared(t *testing.T) {
	r := testRuntime()
	svc := &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ws"},
		Spec: mlservicev1alpha1.MLServiceSpec{
			Backend: mlservicev1alpha1.Backend{Name: "native", Engine: "statefulset"},
			Roles: []mlservicev1alpha1.RoleSpec{{
				Name:     "predictor",
				Replicas: 1,
				Template: mlservicev1alpha1.PodTemplate{
					Image:        "nginx",
					VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
				},
			}},
		},
	}
	_, err := r.renderServicePlans(svc)
	var ce *CapabilityError
	require.ErrorAs(t, err, &ce)
}

func TestContainerNameSanitized(t *testing.T) {
	r := testRuntime()
	name := r.containerName(KindRun, "default", "weird/name*here", "worker", 0)
	assert.Regexp(t, `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`, name)
}
