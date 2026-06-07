//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	poolv1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
)

// cluster-manager: REST over the cluster-scoped ResourcePool CRD.

func TestClusterManager_CreatePoolRoundTripsToCR(t *testing.T) {
	ctx := context.Background()
	pool := uniqueName("e2e-pool")
	req := cmCreatePoolReq{
		Name: pool,
		Units: []cmCreateUnitReq{{
			Name:     "small",
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("2Gi")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("2Gi")},
		}},
	}
	r := h.clusterManager.mustDo(t, ctx, http.MethodPost, "/api/v1/resource-pools", req)
	require.True(t, r.is2xx(), "create pool: %d: %s", r.status, string(r.body))
	t.Cleanup(func() {
		_, _ = h.clusterManager.do(context.Background(), http.MethodDelete, "/api/v1/resource-pools/"+pool, nil)
	})

	// The ResourcePool CR materializes in-cluster.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var rp poolv1.ResourcePool
		return h.k8s.Get(ctx, client.ObjectKey{Name: pool}, &rp)
	})
}

func TestClusterManager_AddUnitAndImmutability(t *testing.T) {
	ctx := context.Background()
	pool := uniqueName("e2e-pool")
	require.True(t, h.clusterManager.mustDo(t, ctx, http.MethodPost, "/api/v1/resource-pools",
		cmCreatePoolReq{Name: pool}).is2xx())
	t.Cleanup(func() {
		_, _ = h.clusterManager.do(context.Background(), http.MethodDelete, "/api/v1/resource-pools/"+pool, nil)
	})

	unitPath := "/api/v1/resource-pools/" + pool + "/resource-units"
	add := cmCreateUnitReq{
		Name:     "u1",
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
	}
	r := h.clusterManager.mustDo(t, ctx, http.MethodPost, unitPath, add)
	require.True(t, r.is2xx(), "add unit: %d: %s", r.status, string(r.body))

	// Units are immutable: PATCH must be rejected.
	patch := map[string]any{"requests": map[string]string{"cpu": "4"}}
	r = h.clusterManager.mustDo(t, ctx, http.MethodPatch, unitPath+"/u1", patch)
	assert.True(t, r.is4xx(), "patching an immutable unit should be 4xx, got %d", r.status)
}

func TestClusterManager_DeletePoolGC(t *testing.T) {
	ctx := context.Background()
	pool := uniqueName("e2e-pool")
	require.True(t, h.clusterManager.mustDo(t, ctx, http.MethodPost, "/api/v1/resource-pools",
		cmCreatePoolReq{Name: pool}).is2xx())

	r := h.clusterManager.mustDo(t, ctx, http.MethodDelete, "/api/v1/resource-pools/"+pool, nil)
	require.True(t, r.is2xx(), "delete pool: %d", r.status)

	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var rp poolv1.ResourcePool
		err := h.k8s.Get(ctx, client.ObjectKey{Name: pool}, &rp)
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return assertErr("ResourcePool %s still present", pool)
	})
}

func TestClusterManager_MissingIdentity401(t *testing.T) {
	ctx := context.Background()
	r, err := h.clusterManager.doNoAuth(ctx, http.MethodGet, "/api/v1/resource-pools", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, r.status, "missing %s must be 401", headerUser)
}
