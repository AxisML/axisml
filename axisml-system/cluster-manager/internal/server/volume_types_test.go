package server_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
	"github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
)

func TestStorageClassesToAPI(t *testing.T) {
	out := srv.StorageClassesToAPI([]extensions.StorageClass{
		{Name: "fast", Provisioner: "ebs", Default: true, AllowVolumeExpansion: true},
		{Name: "slow"},
	})
	require.Len(t, out, 2)
	assert.Equal(t, "fast", out[0].Name)
	assert.Equal(t, "ebs", out[0].Provisioner)
	assert.True(t, out[0].Default)
	assert.True(t, out[0].AllowVolumeExpansion)
	assert.False(t, out[1].Default)

	// Empty input yields a non-nil empty slice.
	assert.Empty(t, srv.StorageClassesToAPI(nil))
}

func TestVolumePatch(t *testing.T) {
	size := "10Gi"
	desc := "d"
	p := srv.PatchVolumeRequest{Size: &size, Description: &desc, Labels: map[string]string{"k": "v"}}
	ep := p.VolumePatch()
	require.NotNil(t, ep.Size)
	assert.Equal(t, "10Gi", *ep.Size)
	require.NotNil(t, ep.Description)
	assert.Equal(t, "d", *ep.Description)
	assert.Equal(t, map[string]string{"k": "v"}, ep.Labels)
}

func TestAPIToPVC(t *testing.T) {
	t.Run("full request", func(t *testing.T) {
		pvc, err := srv.APIToPVC(srv.CreateVolumeRequest{
			Namespace:    "acme",
			Name:         "data",
			Size:         "50Gi",
			StorageClass: "fast",
			AccessModes:  []string{"ReadWriteOnce", "ReadOnlyMany"},
			Description:  "my volume",
			Labels:       map[string]string{"app": "trainer"},
		})
		require.NoError(t, err)
		assert.Equal(t, "acme", pvc.Namespace)
		assert.Equal(t, "data", pvc.Name)
		q := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		assert.Equal(t, "50Gi", q.String())
		require.NotNil(t, pvc.Spec.StorageClassName)
		assert.Equal(t, "fast", *pvc.Spec.StorageClassName)
		require.Len(t, pvc.Spec.AccessModes, 2)
		assert.Equal(t, corev1.ReadWriteOnce, pvc.Spec.AccessModes[0])
		assert.Equal(t, map[string]string{"app": "trainer"}, pvc.Labels)
		assert.Equal(t, "my volume", pvc.Annotations["resource.axisml.io/description"])
	})

	t.Run("minimal request omits optionals", func(t *testing.T) {
		pvc, err := srv.APIToPVC(srv.CreateVolumeRequest{Namespace: "acme", Name: "d", Size: "1Gi"})
		require.NoError(t, err)
		assert.Nil(t, pvc.Spec.StorageClassName)
		assert.Empty(t, pvc.Spec.AccessModes)
		assert.Nil(t, pvc.Labels)
		assert.Nil(t, pvc.Annotations)
	})

	t.Run("invalid size errors", func(t *testing.T) {
		_, err := srv.APIToPVC(srv.CreateVolumeRequest{Namespace: "a", Name: "b", Size: "not-a-qty"})
		assert.Error(t, err)
	})
}

func TestVolumeToAPI(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "acme",
			Name:        "data",
			Annotations: map[string]string{"resource.axisml.io/description": "vol"},
			Labels: map[string]string{
				"app":                          "trainer",
				"app.kubernetes.io/managed-by": "cluster-manager",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("50Gi")},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}
	sc := "fast"
	pvc.Spec.StorageClassName = &sc
	pvc.Status.Phase = corev1.ClaimBound
	pvc.Status.Capacity = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("50Gi")}

	v := srv.VolumeToAPI(pvc)
	assert.Equal(t, "acme", v.Namespace)
	assert.Equal(t, "data", v.Name)
	assert.Equal(t, "vol", v.Description)
	assert.Equal(t, "50Gi", v.Size)
	assert.Equal(t, "fast", v.StorageClass)
	assert.Equal(t, []string{"ReadWriteOnce"}, v.AccessModes)
	require.NotNil(t, v.Status)
	assert.Equal(t, "Bound", v.Status.Phase)
	assert.Equal(t, "50Gi", v.Status.BoundCapacity)
	// The internal managed-by label is hidden from callers.
	assert.Equal(t, map[string]string{"app": "trainer"}, v.Labels)
}

func TestVolumeToAPI_OnlyManagedLabelHiddenToNil(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "acme", Name: "d",
			Labels: map[string]string{"app.kubernetes.io/managed-by": "cluster-manager"},
		},
	}
	v := srv.VolumeToAPI(pvc)
	assert.Nil(t, v.Labels)
	assert.Empty(t, v.Size)
	assert.Empty(t, v.StorageClass)
}

func TestMountsToAPI(t *testing.T) {
	assert.Nil(t, srv.MountsToAPI(nil))

	out := srv.MountsToAPI([]extensions.VolumeMount{
		{Workload: "trainer-0", Kind: "StatefulSet", MountPath: "/data", Running: true},
	})
	require.Len(t, out, 1)
	assert.Equal(t, "trainer-0", out[0].Workload)
	assert.Equal(t, "StatefulSet", out[0].Kind)
	assert.Equal(t, "/data", out[0].MountPath)
	assert.True(t, out[0].Running)
}
