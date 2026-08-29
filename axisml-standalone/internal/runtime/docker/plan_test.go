package docker

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

func testRuntime() *Runtime {
	return New(nil, Config{WorkloadsNetwork: "axisml-workloads"}, logr.Discard())
}

func TestRenderRunPlans(t *testing.T) {
	r := testRuntime()
	run := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "trainer", Labels: map[string]string{mlrunv1alpha1.LabelRunID: "run-uuid"}},
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
	assert.Empty(t, p.Name)
	assert.Equal(t, "trainer-worker", p.NamePrefix)
	assert.Equal(t, 0, p.Replica)
	assert.False(t, p.StableOrdinal)
	assert.Equal(t, "busybox:latest", p.Image)
	assert.Equal(t, corev1.PullAlways, p.ImagePullPolicy)
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
	assert.Equal(t, "run-uuid", p.Labels[mlrunv1alpha1.LabelRunID])
	assert.NotEmpty(t, p.Labels[LabelSpecHash])
	assert.Equal(t, "trainer-worker", plans[1].NamePrefix)
	assert.Equal(t, 1, plans[1].Replica)
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

func TestRenderRunPlans_VolumeMounts(t *testing.T) {
	r := testRuntime()
	run := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "trainer"},
		Spec: mlrunv1alpha1.MLRunSpec{
			Backend: mlrunv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Roles: []mlrunv1alpha1.RoleSpec{{
				Name:     "worker",
				Replicas: 1,
				Template: mlrunv1alpha1.PodTemplateSubset{
					Image: "busybox",
					Volumes: []corev1.Volume{{
						Name: "dataset",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "yolo26-dataset"},
						},
					}},
					VolumeMounts: []corev1.VolumeMount{{Name: "dataset", MountPath: "/data", ReadOnly: true}},
				},
			}},
		},
	}
	plans, err := r.renderRunPlans(run)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Len(t, plans[0].Mounts, 1)
	m := plans[0].Mounts[0]
	assert.Equal(t, "volume", m.Type)
	// PVC-backed volume keys the Docker volume on the claim name, matching what
	// the VolumeManager (Runtime.Ensure) materialises for the same claim.
	assert.Equal(t, "axisml-default-yolo26-dataset", m.Source)
	assert.Equal(t, "/data", m.Target)
	assert.True(t, m.ReadOnly)
}

func TestRenderRunPlans_WritableHostPathSubPathUsesDaemonCreatingBind(t *testing.T) {
	r := New(nil, Config{
		WorkloadsNetwork: "axisml-workloads",
		HostPathVolumes:  map[string]string{"default-workspaces": "/host/data/tenants/default/workspaces"},
	}, logr.Discard())
	run := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "hello-world-1"},
		Spec: mlrunv1alpha1.MLRunSpec{
			Backend: mlrunv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Roles: []mlrunv1alpha1.RoleSpec{{
				Name:     "worker",
				Replicas: 1,
				Template: mlrunv1alpha1.PodTemplateSubset{
					Image: "busybox",
					Volumes: []corev1.Volume{{
						Name: "workspaces",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "default-workspaces"},
						},
					}},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "workspaces",
						MountPath: "/workspaces",
						SubPath:   "hello-world-1",
					}},
				},
			}},
		},
	}

	plans, err := r.renderRunPlans(run)
	require.NoError(t, err)
	require.Len(t, plans, 1)

	_, host, _ := plans[0].toDocker("axisml-workloads")
	assert.Empty(t, host.Mounts)
	assert.Equal(t, []string{
		"/host/data/tenants/default/workspaces/hello-world-1:/workspaces",
	}, host.Binds)
}

func TestRenderRunPlans_VolumeMountUndeclared(t *testing.T) {
	r := testRuntime()
	run := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "x"},
		Spec: mlrunv1alpha1.MLRunSpec{
			Backend: mlrunv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Roles: []mlrunv1alpha1.RoleSpec{{Name: "worker", Replicas: 1, Template: mlrunv1alpha1.PodTemplateSubset{
				Image:        "busybox",
				VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
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
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "infer", Labels: map[string]string{mlservicev1alpha1.LabelServiceID: "service-uuid"}},
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
	assert.Empty(t, p.Name)
	assert.Equal(t, "infer-predictor", p.NamePrefix)
	assert.False(t, p.StableOrdinal)
	assert.Equal(t, corev1.PullIfNotPresent, p.ImagePullPolicy)
	assert.Equal(t, string("unless-stopped"), p.RestartPolicy)
	require.Len(t, p.Ports, 1)
	assert.Equal(t, int32(80), p.Ports[0].ContainerPort)
	assert.Equal(t, 1, p.Resources.GPUCount)
	assert.Equal(t, KindService, p.Labels[LabelResourceKind])
	assert.Equal(t, "service-uuid", p.Labels[mlservicev1alpha1.LabelServiceID])
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
	name := instanceName("weird-name-here-worker", 0, false)
	assert.Regexp(t, `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`, name)
	assert.Regexp(t, `^weird-name-here-worker-[a-z0-9]{5}$`, name)
}

func TestStatefulSetInstanceUsesOrdinal(t *testing.T) {
	assert.Equal(t, "hello-world-predictor-0", instanceName("hello-world-predictor", 0, true))
}

// The assigned GPU device indices must NOT be part of the spec hash: a re-apply
// re-runs the allocator and would otherwise see a "changed" spec and rebuild the
// running container every tick.
func TestPlanIdentityExcludesGPUDeviceIDs(t *testing.T) {
	base := ContainerPlan{
		Image:     "busybox",
		Resources: ResourcePlan{GPUCount: 1},
	}
	withDev := base
	withDev.Resources.GPUDeviceIDs = []string{"0"}
	otherDev := base
	otherDev.Resources.GPUDeviceIDs = []string{"3"}

	assert.Equal(t, specHash(planIdentity(&base)), specHash(planIdentity(&withDev)),
		"assigning a device must not change the spec hash")
	assert.Equal(t, specHash(planIdentity(&withDev)), specHash(planIdentity(&otherDev)),
		"different assigned devices must yield the same spec hash")

	// The GPU request count is still part of the identity.
	twoCards := base
	twoCards.Resources.GPUCount = 2
	assert.NotEqual(t, specHash(planIdentity(&base)), specHash(planIdentity(&twoCards)),
		"a different GPU request count must change the spec hash")
}

func TestResolveImagePullPolicy(t *testing.T) {
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tests := []struct {
		name     string
		image    string
		explicit corev1.PullPolicy
		want     corev1.PullPolicy
	}{
		{name: "untagged", image: "busybox", want: corev1.PullAlways},
		{name: "registry port is not a tag", image: "localhost:5000/team/image", want: corev1.PullAlways},
		{name: "latest", image: "busybox:latest", want: corev1.PullAlways},
		{name: "non-latest tag", image: "busybox:1.37", want: corev1.PullIfNotPresent},
		{name: "digest", image: "busybox@" + digest, want: corev1.PullIfNotPresent},
		{name: "tag and digest", image: "busybox:latest@" + digest, want: corev1.PullIfNotPresent},
		{name: "explicit always", image: "busybox:1.37", explicit: corev1.PullAlways, want: corev1.PullAlways},
		{name: "explicit if not present", image: "busybox:latest", explicit: corev1.PullIfNotPresent, want: corev1.PullIfNotPresent},
		{name: "explicit never", image: "busybox:latest", explicit: corev1.PullNever, want: corev1.PullNever},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveImagePullPolicy(tt.image, tt.explicit)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveImagePullPolicyRejectsInvalidValues(t *testing.T) {
	_, err := resolveImagePullPolicy("busybox", corev1.PullPolicy("Sometimes"))
	var ce *CapabilityError
	require.ErrorAs(t, err, &ce)

	_, err = resolveImagePullPolicy("INVALID/IMAGE", "")
	require.ErrorAs(t, err, &ce)
}

func TestPlanIdentityIncludesImagePullPolicy(t *testing.T) {
	ifNotPresent := ContainerPlan{
		Image:           "busybox:1.37",
		ImagePullPolicy: corev1.PullIfNotPresent,
	}
	always := ifNotPresent
	always.ImagePullPolicy = corev1.PullAlways

	assert.NotEqual(t, specHash(planIdentity(&ifNotPresent)), specHash(planIdentity(&always)),
		"changing imagePullPolicy must rebuild the container")
}

func TestToDockerBindsAssignedDevices(t *testing.T) {
	p := ContainerPlan{
		Image:     "busybox",
		Resources: ResourcePlan{GPUCount: 2, GPUDeviceIDs: []string{"1", "3"}},
	}
	_, host, _ := p.toDocker("axisml-workloads")
	require.Len(t, host.DeviceRequests, 1)
	dr := host.DeviceRequests[0]
	assert.Equal(t, "nvidia", dr.Driver)
	assert.Equal(t, []string{"1", "3"}, dr.DeviceIDs)
	assert.Equal(t, 0, dr.Count) // pinned by ID, never by count
	assert.Equal(t, [][]string{{"gpu"}}, dr.Capabilities)

	// Unmanaged mode (AXISML_GPU_DEVICES unset): a GPU request with no assigned
	// devices falls back to Docker's default count-based request.
	unassigned := ContainerPlan{Image: "busybox", Resources: ResourcePlan{GPUCount: 2}}
	_, host2, _ := unassigned.toDocker("axisml-workloads")
	require.Len(t, host2.DeviceRequests, 1)
	assert.Equal(t, 2, host2.DeviceRequests[0].Count)
	assert.Empty(t, host2.DeviceRequests[0].DeviceIDs)
}
