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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	"github.com/axisml/axisml/components/compute-service/internal/mlservice"
)

// TestWorkspace_PVCLifecycle drives the kind=workspace path end-to-end:
//   - POST creates a Service row + a deterministic PVC (axisml-ws-<name>-data)
//   - the rendered CR spec.roles[0].template.volumes references the PVC
//   - DELETE (default deletePvc=true) cascades to the PVC
func TestWorkspace_PVCLifecycle(t *testing.T) {
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

	const wsName = "demo-ws"
	body := map[string]any{
		"name":     wsName,
		"kind":     "workspace",
		"poolName": "ws-pool",
		"unitName": "small",
		"quota":    "axisml-default",
		"workspaceStorage": map[string]any{
			"size": "1Gi",
		},
		"roles": []map[string]any{{
			"name":     mlservicev1alpha1.DefaultRoleName,
			"replicas": 1,
			"template": map[string]any{
				"image": "busybox:1.36",
				"ports": []map[string]any{{
					"name": "http", "containerPort": 8080,
					"protocol": string(corev1.ProtocolTCP),
				}},
			},
		}},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	pvcName := mlservice.WorkspacePVCName(wsName)

	// PVC should exist immediately (created synchronously before the DB row).
	var pvc corev1.PersistentVolumeClaim
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: ns, Name: pvcName}, &pvc))
	assert.Equal(t, "workspace", pvc.Labels[mlservicev1alpha1.LabelServiceKind])

	// Reconciler patches the CR; its volume[] must reference the PVC.
	require.Eventually(t, func() bool {
		var cr mlservicev1alpha1.MLService
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: wsName}, &cr); err != nil {
			return false
		}
		if len(cr.Spec.Roles) == 0 || len(cr.Spec.Roles[0].Template.Volumes) == 0 {
			return false
		}
		for _, v := range cr.Spec.Roles[0].Template.Volumes {
			if v.PersistentVolumeClaim != nil && v.PersistentVolumeClaim.ClaimName == pvcName {
				return true
			}
		}
		return false
	}, 10*time.Second, 200*time.Millisecond, "MLService CR doesn't reference the workspace PVC")

	// DELETE cascades the PVC by default. envtest doesn't run the
	// pvc-protection controller, so PVCs marked for deletion may linger
	// with a deletionTimestamp until something strips the finalizer; in a
	// real cluster the controller-manager does that automatically. The
	// observable contract is "delete was dispatched": either the PVC is
	// gone, or it's been marked with a deletionTimestamp.
	rr = doJSON(t, ctx, http.MethodDelete,
		"/api/v1/namespaces/"+ns+"/mlservices/"+wsName, nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
	require.Eventually(t, func() bool {
		var fresh corev1.PersistentVolumeClaim
		err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: pvcName}, &fresh)
		if apierrors.IsNotFound(err) {
			return true
		}
		return err == nil && fresh.DeletionTimestamp != nil
	}, 10*time.Second, 200*time.Millisecond, "workspace PVC delete never dispatched")
}

// TestWorkspace_PVCRollbackOnDBFail forces the second create to fail at the
// PG unique-violation step (same name in the same namespace) and asserts
// the PVC created in that attempt is rolled back, so we don't leak storage.
func TestWorkspace_PVCRollbackOnDBFail(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	seedResourcePool(t, ctx, "ws-rollback-pool", "small")
	const ns = "ws-rollback-ns"
	mustCreateNamespace(t, ctx, ns)

	const wsName = "rollback-ws"
	body := map[string]any{
		"name":             wsName,
		"kind":             "workspace",
		"poolName":         "ws-rollback-pool",
		"unitName":         "small",
		"quota":            "axisml-default",
		"workspaceStorage": map[string]any{"size": "1Gi"},
		"roles": []map[string]any{{
			"name":     mlservicev1alpha1.DefaultRoleName,
			"replicas": 1,
			"template": map[string]any{
				"image": "busybox:1.36",
				"ports": []map[string]any{{
					"name": "http", "containerPort": 8080,
					"protocol": string(corev1.ProtocolTCP),
				}},
			},
		}},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", body, nil)
	requireStatus(t, rr, http.StatusCreated)
	t.Cleanup(func() {
		_ = doJSON(t, ctx, http.MethodDelete,
			"/api/v1/namespaces/"+ns+"/mlservices/"+wsName, nil, nil)
	})

	// First DELETE so the PVC is gone, then the second create is the real
	// rollback case: we'll force a hand-made duplicate row via direct SQL
	// so the second API call hits a unique-violation after creating the PVC.
	// Insert a duplicate row directly.
	row := map[string]any{}
	_ = row
	require.NoError(t, gormDB.Exec(
		`INSERT INTO mlservices (id, namespace, name, kind, spec, phase, status, generation, observed_generation)
		 VALUES (gen_random_uuid(), ?, ?, 'workspace', '{}', 'Creating', '{}', 1, 0)`,
		ns, wsName+"-dup").Error)

	// Now create a second workspace with the same name as our dup-row. The
	// PVC will be created, then PG insert fails (unique violation on
	// (namespace, name) — services_namespace_name_active_uniq), and the
	// rollback path should delete the PVC.
	body["name"] = wsName + "-dup"
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", body, nil)
	require.GreaterOrEqual(t, rr.Code, 400)
	require.Less(t, rr.Code, 600)

	// The PVC for the duplicate name should NOT exist (rollback removed it
	// or it was never reached on a fast-fail).
	require.Eventually(t, func() bool {
		var pvc corev1.PersistentVolumeClaim
		err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: mlservice.WorkspacePVCName(wsName + "-dup")}, &pvc)
		return apierrors.IsNotFound(err) || (err == nil && pvc.DeletionTimestamp != nil)
	}, 5*time.Second, 200*time.Millisecond, "duplicate workspace PVC was not rolled back")
}

// TestWorkspace_DeletePvcFalse retains the PVC when ?deletePvc=false.
func TestWorkspace_DeletePvcFalse(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	seedResourcePool(t, ctx, "ws-keep-pool", "small")
	const ns = "ws-keep-ns"
	mustCreateNamespace(t, ctx, ns)

	const wsName = "keep-ws"
	body := map[string]any{
		"name":     wsName,
		"kind":     "workspace",
		"poolName": "ws-keep-pool",
		"unitName": "small",
		"quota":    "axisml-default",
		"workspaceStorage": map[string]any{
			"size": "1Gi",
		},
		"roles": []map[string]any{{
			"name":     mlservicev1alpha1.DefaultRoleName,
			"replicas": 1,
			"template": map[string]any{
				"image": "busybox:1.36",
				"ports": []map[string]any{{
					"name": "http", "containerPort": 8080,
					"protocol": string(corev1.ProtocolTCP),
				}},
			},
		}},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	pvcName := mlservice.WorkspacePVCName(wsName)

	rr = doJSON(t, ctx, http.MethodDelete,
		"/api/v1/namespaces/"+ns+"/mlservices/"+wsName+"?deletePvc=false", nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
	t.Cleanup(func() {
		_ = c.Delete(context.Background(), &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: pvcName},
		})
	})

	// PVC should still be present a moment later.
	time.Sleep(2 * time.Second)
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: ns, Name: pvcName},
		&corev1.PersistentVolumeClaim{}), "PVC must survive deletePvc=false")
}
