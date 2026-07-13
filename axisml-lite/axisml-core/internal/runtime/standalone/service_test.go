package standalone

import (
	"os"
	"testing"

	"github.com/docker/docker/api/types/mount"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

// svcWithMount builds a minimal native/statefulset MLService whose single role
// declares one PVC-backed volume and mounts it, so subPath rendering can be
// exercised without repeating the whole spec in each test.
func svcWithMount(claimName string, vm corev1.VolumeMount) *mlservicev1alpha1.MLService {
	return &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ws"},
		Spec: mlservicev1alpha1.MLServiceSpec{
			Backend: mlservicev1alpha1.Backend{Name: "native", Engine: "statefulset"},
			Roles: []mlservicev1alpha1.RoleSpec{{
				Name:     "predictor",
				Replicas: 1,
				Template: mlservicev1alpha1.PodTemplate{
					Image: "codeserver:latest",
					Volumes: []corev1.Volume{{
						Name: vm.Name,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
						},
					}},
					VolumeMounts: []corev1.VolumeMount{vm},
				},
			}},
		},
	}
}

// TestRenderServicePlans_HostPathVolumeSubPath verifies subPath on a hostPath
// (bind) mount: the plan keeps the registered host root as the source plus the
// subPath, and toDocker resolves it into the bind source so several workloads
// share one host directory while each sees a different subtree.
func TestRenderServicePlans_HostPathVolumeSubPath(t *testing.T) {
	r := New(nil, Config{
		WorkloadsNetwork: "axisml-workloads",
		HostPathVolumes:  map[string]string{"host-ds": "/data/host-datasets"},
	}, logr.Discard())
	svc := svcWithMount("host-ds", corev1.VolumeMount{Name: "data", MountPath: "/data", SubPath: "images"})

	plans, err := r.renderServicePlans(svc)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Len(t, plans[0].Mounts, 1)
	m := plans[0].Mounts[0]
	assert.Equal(t, "bind", m.Type)
	assert.Equal(t, "/data/host-datasets", m.Source)
	assert.Equal(t, "images", m.SubPath)

	_, host, _ := plans[0].toDocker("axisml-workloads")
	require.Len(t, host.Mounts, 1)
	assert.Equal(t, mount.TypeBind, host.Mounts[0].Type)
	assert.Equal(t, "/data/host-datasets/images", host.Mounts[0].Source)
	assert.Nil(t, host.Mounts[0].VolumeOptions)
}

// TestRenderServicePlans_ManagedVolumeSubPath verifies subPath on a managed
// (named-volume) mount: the source volume is unchanged and the subPath rides on
// Docker's VolumeOptions.Subpath. The declared subPath is also path-cleaned.
func TestRenderServicePlans_ManagedVolumeSubPath(t *testing.T) {
	r := testRuntime()
	svc := svcWithMount("axisml-ws-ws-data", corev1.VolumeMount{Name: "workspace", MountPath: "/workspace", SubPath: "runs/./exp-1"})

	plans, err := r.renderServicePlans(svc)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Len(t, plans[0].Mounts, 1)
	m := plans[0].Mounts[0]
	assert.Equal(t, "volume", m.Type)
	assert.Equal(t, r.pvcVolumeName("default", "axisml-ws-ws-data"), m.Source)
	assert.Equal(t, "runs/exp-1", m.SubPath)

	_, host, _ := plans[0].toDocker("axisml-workloads")
	require.Len(t, host.Mounts, 1)
	assert.Equal(t, mount.TypeVolume, host.Mounts[0].Type)
	assert.Equal(t, r.pvcVolumeName("default", "axisml-ws-ws-data"), host.Mounts[0].Source)
	require.NotNil(t, host.Mounts[0].VolumeOptions)
	assert.Equal(t, "runs/exp-1", host.Mounts[0].VolumeOptions.Subpath)
}

// TestRenderServicePlans_SubPathRejected verifies a subPath that escapes the
// volume, or an unsupported subPathExpr, is surfaced as a CapabilityError
// rather than silently clamped or ignored.
func TestRenderServicePlans_SubPathRejected(t *testing.T) {
	cases := map[string]corev1.VolumeMount{
		"traversal": {Name: "workspace", MountPath: "/workspace", SubPath: "../etc"},
		"absolute":  {Name: "workspace", MountPath: "/workspace", SubPath: "/etc"},
		"expr":      {Name: "workspace", MountPath: "/workspace", SubPathExpr: "$(POD_NAME)"},
	}
	for name, vm := range cases {
		t.Run(name, func(t *testing.T) {
			r := testRuntime()
			_, err := r.renderServicePlans(svcWithMount("axisml-ws-ws-data", vm))
			var ce *CapabilityError
			require.ErrorAs(t, err, &ce)
		})
	}
}

// TestRenderServicePlans_WorkspacePVCVolume verifies a PVC-backed (workspace)
// volume mounts the volume keyed on its claim name — the same name the
// VolumeManager (Runtime.Ensure) provisions — rather than a generic per-mount
// volume.
func TestRenderServicePlans_WorkspacePVCVolume(t *testing.T) {
	r := testRuntime()
	svc := &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ws"},
		Spec: mlservicev1alpha1.MLServiceSpec{
			Backend: mlservicev1alpha1.Backend{Name: "native", Engine: "statefulset"},
			Roles: []mlservicev1alpha1.RoleSpec{{
				Name:     "predictor",
				Replicas: 1,
				Template: mlservicev1alpha1.PodTemplate{
					Image: "codeserver:latest",
					Volumes: []corev1.Volume{{
						Name: "workspace",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "axisml-ws-ws-data"},
						},
					}},
					VolumeMounts: []corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace"}},
				},
			}},
		},
	}
	plans, err := r.renderServicePlans(svc)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Len(t, plans[0].Mounts, 1)
	m := plans[0].Mounts[0]
	assert.Equal(t, r.pvcVolumeName("default", "axisml-ws-ws-data"), m.Source)
	assert.Equal(t, "/workspace", m.Target)
}

// TestRenderServicePlans_HostPathVolume verifies a predefined hostPath volume:
// the workload references it by claim name (its ordinary PVC form), and the
// runtime — knowing that claim name maps to a host directory (Config.
// HostPathVolumes) — renders a bind mount to the host path instead of a managed
// Docker volume.
func TestRenderServicePlans_HostPathVolume(t *testing.T) {
	r := New(nil, Config{
		WorkloadsNetwork: "axisml-workloads",
		HostPathVolumes:  map[string]string{"host-ds": "/data/host-datasets"},
	}, logr.Discard())
	svc := &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ws"},
		Spec: mlservicev1alpha1.MLServiceSpec{
			Backend: mlservicev1alpha1.Backend{Name: "native", Engine: "statefulset"},
			Roles: []mlservicev1alpha1.RoleSpec{{
				Name:     "predictor",
				Replicas: 1,
				Template: mlservicev1alpha1.PodTemplate{
					Image: "codeserver:latest",
					Volumes: []corev1.Volume{{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "host-ds"},
						},
					}},
					VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
				},
			}},
		},
	}
	plans, err := r.renderServicePlans(svc)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Len(t, plans[0].Mounts, 1)
	m := plans[0].Mounts[0]
	assert.Equal(t, "bind", m.Type)
	assert.Equal(t, "/data/host-datasets", m.Source)
	assert.Equal(t, "/data", m.Target)
}

// TestServiceRoute_ApplyObserveDelete verifies spec.route renders a Traefik
// config whose endpoint is readable back, and that delete removes it.
func TestServiceRoute_ApplyObserveDelete(t *testing.T) {
	dir := t.TempDir()
	r := New(nil, Config{WorkloadsNetwork: "axisml-workloads", TraefikDir: dir}, logr.Discard())

	svc := &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "infer"},
		Spec: mlservicev1alpha1.MLServiceSpec{
			Backend: mlservicev1alpha1.Backend{Name: "native", Engine: "deployment"},
			Roles: []mlservicev1alpha1.RoleSpec{{
				Name:     "predictor",
				Replicas: 1,
				Template: mlservicev1alpha1.PodTemplate{
					Image: "nginx:1.27",
					Ports: []mlservicev1alpha1.PodPort{{Name: "http", ContainerPort: 8080}},
				},
			}},
			Route: &mlservicev1alpha1.Route{Enabled: true, Path: "/app", PortName: "http"},
		},
	}
	plans, err := r.renderServicePlans(svc)
	require.NoError(t, err)

	require.NoError(t, r.applyServiceRoute(svc, plans))
	file := r.serviceRouteFileName("default", "infer")
	body, err := os.ReadFile(file)
	require.NoError(t, err)
	content := string(body)
	assert.Contains(t, content, "PathPrefix(`/app`)")
	assert.Contains(t, content, ":8080")
	assert.Equal(t, "/app", r.serviceRouteEndpoint("default", "infer"))

	require.NoError(t, r.deleteServiceRoute("default", "infer"))
	_, err = os.Stat(file)
	assert.True(t, os.IsNotExist(err))
	assert.Equal(t, "", r.serviceRouteEndpoint("default", "infer"))
}

// TestServiceRoute_AuthUnsupported verifies route auth is rejected as a
// capability error.
func TestServiceRoute_AuthUnsupported(t *testing.T) {
	dir := t.TempDir()
	r := New(nil, Config{TraefikDir: dir}, logr.Discard())
	svc := &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "infer"},
		Spec: mlservicev1alpha1.MLServiceSpec{
			Route: &mlservicev1alpha1.Route{Enabled: true, Path: "/", Auth: &mlservicev1alpha1.RouteAuth{Type: mlservicev1alpha1.RouteAuthJWT}},
		},
	}
	err := r.applyServiceRoute(svc, nil)
	var ce *CapabilityError
	require.ErrorAs(t, err, &ce)
}
