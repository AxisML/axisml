//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	srv "github.com/axisml/axisml/components/cluster-manager/internal/server"
)

// TestVolume_Lifecycle drives the Volume REST surface against the live K8s API
// (envtest): POST materialises a PVC, the response echoes the request, the PVC
// carries cluster-manager's managed-by label and requested size, POST is
// idempotent, and DELETE reclaims it (idempotent).
func TestVolume_Lifecycle(t *testing.T) {
	const ns = "default" // always present in envtest
	const name = "axisml-ws-demo-data"

	body := `{"namespace":"` + ns + `","name":"` + name + `","size":"1Gi"}`

	rr := doRequest(t, "POST", "/api/v1/volumes", body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var created srv.Volume
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &created))
	require.Equal(t, ns, created.Namespace)
	require.Equal(t, name, created.Name)
	require.Equal(t, "1Gi", created.Size)

	// The backing PVC exists with the managed-by label and the requested size.
	var pvc corev1.PersistentVolumeClaim
	require.NoError(t, testCli.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, &pvc))
	assert.Equal(t, "axisml-cluster-manager", pvc.Labels["app.kubernetes.io/managed-by"])
	q := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	assert.Equal(t, "1Gi", q.String())

	// POST is idempotent: an existing PVC is treated as success.
	rr = doRequest(t, "POST", "/api/v1/volumes", body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// DELETE reclaims it. envtest doesn't run the pvc-protection controller, so
	// a deleted PVC may linger with a deletionTimestamp; the observable contract
	// is "delete dispatched": gone, or marked for deletion.
	rr = doRequest(t, "DELETE", "/api/v1/volumes/"+ns+"/"+name, "")
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())
	require.Eventually(t, func() bool {
		var fresh corev1.PersistentVolumeClaim
		err := testCli.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, &fresh)
		return apierrors.IsNotFound(err) || (err == nil && fresh.DeletionTimestamp != nil)
	}, 10*time.Second, 200*time.Millisecond, "volume PVC delete never dispatched")

	// DELETE is idempotent: a missing volume is still 204.
	rr = doRequest(t, "DELETE", "/api/v1/volumes/"+ns+"/"+name, "")
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())
}

// TestVolume_Validation rejects a missing/invalid size with 400.
func TestVolume_Validation(t *testing.T) {
	// Missing size.
	rr := doRequest(t, "POST", "/api/v1/volumes", `{"namespace":"default","name":"axisml-ws-novalsize-data"}`)
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())

	// Invalid Quantity.
	rr = doRequest(t, "POST", "/api/v1/volumes", `{"namespace":"default","name":"axisml-ws-badsize-data","size":"notaquantity"}`)
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())

	// Missing name.
	rr = doRequest(t, "POST", "/api/v1/volumes", `{"namespace":"default","size":"1Gi"}`)
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
}

// TestVolume_ListGetPatch covers the data-volume catalog surface: create with
// accessModes/description, read back (with live status), list, and patch the
// description.
func TestVolume_ListGetPatch(t *testing.T) {
	const ns = "default"
	const name = "shared-data"

	rr := doRequest(t, "POST", "/api/v1/volumes",
		`{"namespace":"default","name":"shared-data","size":"1Gi","accessModes":["ReadWriteMany"],"description":"team share"}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	t.Cleanup(func() { doRequest(t, "DELETE", "/api/v1/volumes/"+ns+"/"+name+"?force=true", "") })

	// GET echoes accessModes + description and carries a status block.
	rr = doRequest(t, "GET", "/api/v1/volumes/"+ns+"/"+name, "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got srv.Volume
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, "team share", got.Description)
	assert.Contains(t, got.AccessModes, "ReadWriteMany")
	require.NotNil(t, got.Status)

	// LIST returns the managed volume.
	rr = doRequest(t, "GET", "/api/v1/volumes?namespace="+ns, "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var list srv.VolumeList
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	found := false
	for _, v := range list.Items {
		if v.Name == name {
			found = true
		}
	}
	assert.True(t, found, "created volume must appear in the list")

	// PATCH updates the description.
	rr = doRequest(t, "PATCH", "/api/v1/volumes/"+ns+"/"+name, `{"description":"expanded share"}`)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var patched srv.Volume
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &patched))
	assert.Equal(t, "expanded share", patched.Description)
}

// TestVolume_StorageClasses lists storage classes, flagging the default and
// expansion capability.
func TestVolume_StorageClasses(t *testing.T) {
	expand := true
	sc := &storagev1.StorageClass{
		ObjectMeta:           metav1.ObjectMeta{Name: "it-fast-ssd", Annotations: map[string]string{"storageclass.kubernetes.io/is-default-class": "true"}},
		Provisioner:          "test.csi.k8s.io",
		AllowVolumeExpansion: &expand,
	}
	require.NoError(t, testCli.Create(context.Background(), sc))
	t.Cleanup(func() { _ = testCli.Delete(context.Background(), sc) })

	rr := doRequest(t, "GET", "/api/v1/storageclasses", "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var list srv.StorageClassList
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	var found *srv.StorageClass
	for i := range list.Items {
		if list.Items[i].Name == "it-fast-ssd" {
			found = &list.Items[i]
		}
	}
	require.NotNil(t, found, "created storage class must be listed")
	assert.True(t, found.Default)
	assert.True(t, found.AllowVolumeExpansion)
	assert.Equal(t, "test.csi.k8s.io", found.Provisioner)
}

// TestVolume_DeleteOccupancyGuard verifies a volume mounted by a running pod is
// refused (409) and the mount surfaces in the detail status; force=true deletes.
func TestVolume_DeleteOccupancyGuard(t *testing.T) {
	const ns = "default"
	const name = "occupied-data"

	rr := doRequest(t, "POST", "/api/v1/volumes", `{"namespace":"default","name":"occupied-data","size":"1Gi"}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:         "c",
				Image:        "busybox",
				VolumeMounts: []corev1.VolumeMount{{Name: "d", MountPath: "/data"}},
			}},
			Volumes: []corev1.Volume{{
				Name: "d",
				VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: name,
				}},
			}},
		},
	}
	require.NoError(t, testCli.Create(context.Background(), pod))
	pod.Status.Phase = corev1.PodRunning
	require.NoError(t, testCli.Status().Update(context.Background(), pod))
	t.Cleanup(func() { _ = testCli.Delete(context.Background(), pod) })

	// DELETE without force is blocked while a running pod mounts the volume.
	rr = doRequest(t, "DELETE", "/api/v1/volumes/"+ns+"/"+name, "")
	require.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())

	// The mount surfaces in the detail status.
	rr = doRequest(t, "GET", "/api/v1/volumes/"+ns+"/"+name, "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got srv.Volume
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.NotNil(t, got.Status)
	require.NotEmpty(t, got.Status.Mounts)
	assert.Equal(t, "consumer", got.Status.Mounts[0].Workload)
	assert.True(t, got.Status.Mounts[0].Running)

	// force=true deletes despite occupancy.
	rr = doRequest(t, "DELETE", "/api/v1/volumes/"+ns+"/"+name+"?force=true", "")
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())
}
