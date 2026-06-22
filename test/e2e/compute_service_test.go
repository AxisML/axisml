//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	mlrunv1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// compute-service. Drive the HTTP API; assert on the HTTP response, the
// materialized CR, and (for jobs/services) the real workload.

func TestComputeService_MLRunLifecycleTopToBottom(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-apijob")

	r, err := h.createMLRun(ctx, ns, busyboxMLRunReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota))
	require.NoError(t, err)
	require.True(t, r.is2xx(), "create job: %d: %s", r.status, string(r.body))
	t.Cleanup(func() { _, _ = h.deleteMLRun(context.Background(), ns, name) })

	// MLRun CR carries the resolved resource snapshot from the unit.
	var job mlrunv1.MLRun
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.get(ctx, ns, name, &job) })
	require.NotEmpty(t, job.Spec.Roles, "MLRun should have a role")
	assert.NotEmpty(t, job.Spec.Roles[0].Template.Resources.Requests,
		"unit resources should be snapshotted onto the MLRun role")

	// GET reports the job and it runs to Succeeded.
	eventually(t, h.cfg.MLRunCompleteTimeout, func() error {
		g, err := h.getMLRun(ctx, ns, name)
		if err != nil {
			return err
		}
		if !g.is2xx() {
			return assertErr("GET job: %d", g.status)
		}
		var v csMLRunView
		if err := g.decode(&v); err != nil {
			return err
		}
		if v.Phase != string(mlrunv1.PhaseSucceeded) {
			return assertErr("phase=%q want Succeeded", v.Phase)
		}
		return nil
	})
}

func TestComputeService_MLRunCancel(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-apicancel")

	req := busyboxMLRunReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota)
	req.Roles[0].Template.Command = []string{"sh", "-c", "sleep 600"}
	r, err := h.createMLRun(ctx, ns, req)
	require.NoError(t, err)
	require.True(t, r.is2xx(), "create job: %d: %s", r.status, string(r.body))
	t.Cleanup(func() { _, _ = h.deleteMLRun(context.Background(), ns, name) })

	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var job mlrunv1.MLRun
		return h.get(ctx, ns, name, &job)
	})

	c := h.computeService.mustDo(t, ctx, http.MethodPost, mlrunPath(ns, name)+"/cancel", nil)
	require.True(t, c.is2xx(), "cancel job: %d: %s", c.status, string(c.body))

	eventually(t, h.cfg.PodReadyTimeout, func() error {
		suspended, err := batchJobSuspended(ctx, ns, name)
		if err != nil {
			return err
		}
		if !suspended {
			return assertErr("batch Job not suspended after cancel")
		}
		return nil
	})
}

func TestComputeService_KubeproxyPodsAndLogs(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-proxy")

	r, err := h.createMLRun(ctx, ns, busyboxMLRunReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota))
	require.NoError(t, err)
	require.True(t, r.is2xx())
	t.Cleanup(func() { _, _ = h.deleteMLRun(context.Background(), ns, name) })

	// Pods sub-resource is reachable.
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		p := h.computeService.mustDo(t, ctx, http.MethodGet, mlrunPath(ns, name)+"/pods", nil)
		if !p.is2xx() {
			return assertErr("GET pods: %d", p.status)
		}
		if !strings.Contains(string(p.body), name) {
			return assertErr("pods list does not yet reference the job")
		}
		return nil
	})
	// Events sub-resource is reachable.
	e := h.computeService.mustDo(t, ctx, http.MethodGet, mlrunPath(ns, name)+"/events", nil)
	assert.True(t, e.is2xx(), "GET events: %d", e.status)
}

func TestComputeService_ServiceLifecycleScaleDelete(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-apisvc")

	r, err := h.createMLService(ctx, ns, nginxMLServiceReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota, nil))
	require.NoError(t, err)
	require.True(t, r.is2xx(), "create service: %d: %s", r.status, string(r.body))

	// MLService CR + Deployment become ready.
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		var dep appsv1.Deployment
		if err := h.get(ctx, ns, name, &dep); err != nil {
			return err
		}
		var svc mlservicev1.MLService
		if err := h.get(ctx, ns, name, &svc); err != nil {
			return err
		}
		if svc.Status.Phase != mlservicev1.PhaseReady {
			return assertErr("phase=%q want Ready", svc.Status.Phase)
		}
		return nil
	})

	// Scale 1 -> 2 via the API.
	s := h.computeService.mustDo(t, ctx, http.MethodPost, mlservicePath(ns, name)+"/scale", csScaleReq{Replicas: 2})
	require.True(t, s.is2xx(), "scale: %d: %s", s.status, string(s.body))
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		var dep appsv1.Deployment
		if err := h.get(ctx, ns, name, &dep); err != nil {
			return err
		}
		if dep.Spec.Replicas == nil || *dep.Spec.Replicas != 2 {
			return assertErr("deployment not scaled to 2")
		}
		return nil
	})

	// Delete cascades the workload.
	d := h.computeService.mustDo(t, ctx, http.MethodDelete, mlservicePath(ns, name), nil)
	require.True(t, d.is2xx(), "delete: %d", d.status)
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var svc mlservicev1.MLService
		if err := h.get(ctx, ns, name, &svc); isNotFound(err) {
			return nil
		} else if err != nil {
			return err
		}
		return assertErr("MLService %s still present", name)
	})
}

func TestComputeService_WorkspacePVC(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-workspace")

	req := nginxMLServiceReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota, nil)
	req.Kind = mlservicev1.ServiceKindWorkspace
	req.WorkspaceStorage = &csWorkspaceStorage{Size: "1Gi"}
	r, err := h.createMLService(ctx, ns, req)
	require.NoError(t, err)
	require.True(t, r.is2xx(), "create workspace: %d: %s", r.status, string(r.body))

	// A PVC is created for the workspace.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var pvcs corev1.PersistentVolumeClaimList
		if err := h.k8s.List(ctx, &pvcs); err != nil {
			return err
		}
		for i := range pvcs.Items {
			if pvcs.Items[i].Namespace == ns && strings.Contains(pvcs.Items[i].Name, name) {
				return nil
			}
		}
		return assertErr("no PVC for workspace %s yet", name)
	})

	// Delete cascades the PVC.
	d := h.computeService.mustDo(t, ctx, http.MethodDelete, mlservicePath(ns, name), nil)
	require.True(t, d.is2xx())
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var pvcs corev1.PersistentVolumeClaimList
		if err := h.k8s.List(ctx, &pvcs); err != nil {
			return err
		}
		for i := range pvcs.Items {
			if pvcs.Items[i].Namespace == ns && strings.Contains(pvcs.Items[i].Name, name) {
				return assertErr("PVC for %s still present", name)
			}
		}
		return nil
	})
}

func TestComputeService_UnknownPoolRejected(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-badpool")

	r, err := h.createMLRun(ctx, ns, busyboxMLRunReq(name, "does-not-exist", "nope", quota))
	require.NoError(t, err)
	assert.True(t, r.is4xx(), "unknown pool should be 4xx, got %d", r.status)

	// No MLRun CR was created.
	var job mlrunv1.MLRun
	err = h.get(ctx, ns, name, &job)
	assert.True(t, isNotFound(err), "no MLRun should exist for a rejected request")
}

func TestComputeService_MLRunIntoUnknownNamespace(t *testing.T) {
	ctx := context.Background()
	quota := "whatever"
	ns := uniqueName("nope-ns")
	r, err := h.createMLRun(ctx, ns, busyboxMLRunReq("j", h.cfg.DefaultPool, h.cfg.DefaultUnit, quota))
	require.NoError(t, err)
	assert.True(t, r.is4xx(), "job into unknown namespace should be 4xx, got %d", r.status)
}
