//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

// TestWorkspace_ReferencesPreProvisionedVolume drives the kind=workspace path:
// the durable volume is pre-provisioned by Platform (via cluster-manager) and
// arrives as a PVC entry in the role template. compute must relay it into the
// MLService CR verbatim and must NOT itself create or delete any PVC — volume
// lifecycle is entirely out of compute's scope.
func TestWorkspace_ReferencesPreProvisionedVolume(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	seedResourcePool(t, ctx, "ws-pool", "small")
	const ns = "ws-ns"
	mustCreateNamespace(t, ctx, ns)
	mustSetTenantQuota(t, ctx, ns, "ws-pool", resourceList("100", "1Ti"))

	const wsName = "demo-ws"
	const claimName = "axisml-ws-demo-ws-data" // pre-provisioned by Platform
	body := map[string]any{
		"name":     wsName,
		"kind":     "workspace",
		"poolName": "ws-pool",
		"unitName": "small",
		"roles": []map[string]any{{
			"name":     mlservicev1alpha1.DefaultRoleName,
			"replicas": 1,
			"template": map[string]any{
				"image": "busybox:1.36",
				"ports": []map[string]any{{
					"name": "http", "containerPort": 8080,
					"protocol": string(corev1.ProtocolTCP),
				}},
				// Platform injects the PVC reference + mount for the
				// pre-provisioned workspace volume.
				"volumes": []map[string]any{{
					"name": "workspace",
					"persistentVolumeClaim": map[string]any{
						"claimName": claimName,
					},
				}},
				"volumeMounts": []map[string]any{{
					"name": "workspace", "mountPath": "/workspace",
				}},
			},
		}},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	// compute must NOT create a PVC — that's Platform's job via cluster-manager.
	var pvc corev1.PersistentVolumeClaim
	err = c.Get(ctx, types.NamespacedName{Namespace: ns, Name: claimName}, &pvc)
	assert.True(t, apierrors.IsNotFound(err), "compute must not create a PVC; got %v", err)

	// The reconciler patches the CR; its volume[] must relay the PVC reference
	// the caller supplied.
	require.Eventually(t, func() bool {
		var cr mlservicev1alpha1.MLService
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: wsName}, &cr); err != nil {
			return false
		}
		if len(cr.Spec.Roles) == 0 {
			return false
		}
		for _, v := range cr.Spec.Roles[0].Template.Volumes {
			if v.PersistentVolumeClaim != nil && v.PersistentVolumeClaim.ClaimName == claimName {
				return true
			}
		}
		return false
	}, 10*time.Second, 200*time.Millisecond, "MLService CR doesn't relay the workspace PVC reference")

	// DELETE soft-deletes the row + tears down the CR; it must not touch the PVC.
	rr = doJSON(t, ctx, http.MethodDelete,
		"/api/v1/namespaces/"+ns+"/mlservices/"+wsName, nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
}
