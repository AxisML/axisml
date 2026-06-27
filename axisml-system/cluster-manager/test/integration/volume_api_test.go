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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
