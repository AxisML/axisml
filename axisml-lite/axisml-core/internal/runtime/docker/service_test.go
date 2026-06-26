package docker

import (
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// TestRenderServicePlans_WorkspacePVCVolume verifies a PVC-backed (workspace)
// volume mounts the workspace volume name — the same name EnsureWorkspaceVolume
// provisions — rather than a generic per-mount volume.
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
	assert.Equal(t, r.workspaceVolumeName("default", "ws"), m.Source)
	assert.Equal(t, "/workspace", m.Target)
}

// TestServiceRoute_ApplyObserveDelete verifies spec.route renders a Traefik
// config whose endpoint is readable back, and that delete removes it.
func TestServiceRoute_ApplyObserveDelete(t *testing.T) {
	dir := t.TempDir()
	r := New(nil, Config{WorkloadsNetwork: "axisml-workloads", Tenant: "default", TraefikDir: dir}, logr.Discard())

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
