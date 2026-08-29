package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/mount"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	configapi "github.com/axisml/axisml/axisml-system/apis/pkg/workloadconfig"
)

func configMapRun() *mlrunv1alpha1.MLRun {
	return &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "trainer"},
		Spec: mlrunv1alpha1.MLRunSpec{
			Backend: mlrunv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			ConfigMaps: []configapi.ConfigMap{{
				Name: "trainer-config",
				Data: map[string]string{
					"MODE":         "train",
					"trainer.yaml": "epochs: 1\n",
					"obsolete":     "remove-me",
				},
			}},
			Roles: []mlrunv1alpha1.RoleSpec{{
				Name:     "worker",
				Replicas: 1,
				Template: mlrunv1alpha1.PodTemplateSubset{
					Image: "busybox",
					EnvFrom: []corev1.EnvFromSource{{
						Prefix: "CFG_",
						ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{
							Name: "trainer-config",
						}},
					}},
					Env: []corev1.EnvVar{{
						Name: "TRAIN_MODE",
						ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "trainer-config"},
							Key:                  "MODE",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config-files",
						VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "trainer-config"},
						}},
					}},
					VolumeMounts: []corev1.VolumeMount{{Name: "config-files", MountPath: "/etc/trainer"}},
				},
			}},
		},
	}
}

func TestRenderRunPlans_ConfigMapConsumption(t *testing.T) {
	r := New(nil, Config{
		WorkloadsNetwork: "axisml-workloads",
		ConfigMapsDir:    t.TempDir(),
		ConfigMapsVolume: "axisml-configmaps",
	}, logr.Discard())

	plans, err := r.renderRunPlans(configMapRun())
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, []string{
		"CFG_MODE=train",
		"CFG_obsolete=remove-me",
		"CFG_trainer.yaml=epochs: 1\n",
		"TRAIN_MODE=train",
	}, plans[0].Env)

	require.Len(t, plans[0].Mounts, 1)
	projection := plans[0].Mounts[0]
	assert.Equal(t, "volume", projection.Type)
	assert.Equal(t, "axisml-configmaps", projection.Source)
	assert.Equal(t, "run/default/trainer/worker/config-files", projection.SubPath)
	assert.Equal(t, "/etc/trainer", projection.Target)
	assert.True(t, projection.ReadOnly)
	assert.Equal(t, "epochs: 1\n", projection.ConfigMapFiles["trainer.yaml"].Data)

	_, host, _ := plans[0].toDocker("axisml-workloads")
	require.Len(t, host.Mounts, 1)
	assert.Equal(t, mount.TypeVolume, host.Mounts[0].Type)
	require.NotNil(t, host.Mounts[0].VolumeOptions)
	assert.Equal(t, projection.SubPath, host.Mounts[0].VolumeOptions.Subpath)
}

func TestRenderServicePlans_ConfigMapConsumption(t *testing.T) {
	r := New(nil, Config{ConfigMapsDir: t.TempDir(), ConfigMapsVolume: "axisml-configmaps"}, logr.Discard())
	svc := &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "inference"},
		Spec: mlservicev1alpha1.MLServiceSpec{
			Backend:    mlservicev1alpha1.Backend{Name: "native", Engine: "deployment"},
			ConfigMaps: []configapi.ConfigMap{{Name: "server-config", Data: map[string]string{"PORT": "8080"}}},
			Roles: []mlservicev1alpha1.RoleSpec{{
				Name:     "predictor",
				Replicas: 1,
				Template: mlservicev1alpha1.PodTemplate{
					Image: "server:latest",
					EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "server-config"},
					}}},
					Volumes: []corev1.Volume{{Name: "config", VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "server-config"}},
					}}},
					VolumeMounts: []corev1.VolumeMount{{Name: "config", MountPath: "/etc/server"}},
				},
			}},
		},
	}

	plans, err := r.renderServicePlans(svc)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, []string{"PORT=8080"}, plans[0].Env)
	require.Len(t, plans[0].Mounts, 1)
	assert.Equal(t, "service/default/inference/predictor/config", plans[0].Mounts[0].ConfigMapPath)
}

func TestConfigMapFiles_ItemsAndModes(t *testing.T) {
	defaultMode := int32(0o600)
	itemMode := int32(0o400)
	files, err := configMapFiles(configMapSet{"config": {
		"app.yaml": "debug: false\n",
		"unused":   "value",
	}}, &corev1.ConfigMapVolumeSource{
		LocalObjectReference: corev1.LocalObjectReference{Name: "config"},
		DefaultMode:          &defaultMode,
		Items: []corev1.KeyToPath{{
			Key: "app.yaml", Path: "nested/app.yaml", Mode: &itemMode,
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]ConfigMapFile{
		"nested/app.yaml": {Data: "debug: false\n", Mode: 0o400},
	}, files)
}

func TestReconcileConfigMapProjections_CreatesAndCorrectsDrift(t *testing.T) {
	dir := t.TempDir()
	r := New(nil, Config{ConfigMapsDir: dir, ConfigMapsVolume: "axisml-configmaps"}, logr.Discard())
	run := configMapRun()

	plans, err := r.renderRunPlans(run)
	require.NoError(t, err)
	require.NoError(t, r.reconcileConfigMapProjections(KindRun, "default", "trainer", plans))
	projectionDir := filepath.Join(dir, "run", "default", "trainer", "worker", "config-files")
	body, err := os.ReadFile(filepath.Join(projectionDir, "trainer.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "epochs: 1\n", string(body))

	// A later reconcile restores desired data and removes keys no longer present.
	require.NoError(t, os.WriteFile(filepath.Join(projectionDir, "trainer.yaml"), []byte("manual drift"), 0o644))
	run.Spec.ConfigMaps[0].Data["trainer.yaml"] = "epochs: 2\n"
	delete(run.Spec.ConfigMaps[0].Data, "obsolete")
	plans, err = r.renderRunPlans(run)
	require.NoError(t, err)
	require.NoError(t, r.reconcileConfigMapProjections(KindRun, "default", "trainer", plans))
	body, err = os.ReadFile(filepath.Join(projectionDir, "trainer.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "epochs: 2\n", string(body))
	_, err = os.Stat(filepath.Join(projectionDir, "obsolete"))
	assert.True(t, os.IsNotExist(err))

	require.NoError(t, r.removeConfigMapProjections(KindRun, "default", "trainer"))
	_, err = os.Stat(filepath.Join(dir, "run", "default", "trainer"))
	assert.True(t, os.IsNotExist(err))
}

func TestRenderRunPlans_ConfigMapReferenceMustBeWorkloadOwned(t *testing.T) {
	r := New(nil, Config{ConfigMapsDir: t.TempDir()}, logr.Discard())
	run := configMapRun()
	run.Spec.ConfigMaps = nil

	_, err := r.renderRunPlans(run)
	var capability *CapabilityError
	require.ErrorAs(t, err, &capability)
	assert.Contains(t, err.Error(), "not declared in spec.configMaps")
}

func TestRenderRunPlans_ConfigMapVolumeSubPathUnsupported(t *testing.T) {
	r := New(nil, Config{ConfigMapsDir: t.TempDir()}, logr.Discard())
	run := configMapRun()
	run.Spec.Roles[0].Template.VolumeMounts[0].SubPath = "trainer.yaml"

	_, err := r.renderRunPlans(run)
	var capability *CapabilityError
	require.ErrorAs(t, err, &capability)
	assert.Contains(t, err.Error(), "subPath is unsupported")
}

func TestRemoveConfigMapProjections_RejectsEscapingWorkloadName(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "sentinel")
	require.NoError(t, os.WriteFile(sentinel, []byte("keep"), 0o600))
	r := New(nil, Config{ConfigMapsDir: dir}, logr.Discard())

	err := r.removeConfigMapProjections(KindRun, "default", "../escape")
	require.Error(t, err)
	_, statErr := os.Stat(sentinel)
	require.NoError(t, statErr)
}
