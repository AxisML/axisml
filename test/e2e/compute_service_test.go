//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	mlrunv1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/test/e2e/internal/clients/computeservice"
)

// compute-service. Drive the HTTP API; assert on the HTTP response, the
// materialized CR, and (for jobs/services) the real workload. Scenarios share
// one tenant provisioned for this file and run as subtests.
func TestComputeService(t *testing.T) {
	ns, quota := provisionTenant(t)

	t.Run("MLRunLifecycleTopToBottom", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-apijob")

		r, err := h.createMLRun(ctx, ns, busyboxMLRunReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota))
		require.NoError(t, err)
		require.True(t, is2xx(r.StatusCode()), "create job: %d: %s", r.StatusCode(), string(r.Body))
		cleanupMLRun(t, ns, name)

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
			if !is2xx(g.StatusCode()) {
				return assertErr("GET job: %d", g.StatusCode())
			}
			if g.JSON200 == nil {
				return assertErr("GET job: empty body")
			}
			if g.JSON200.Phase != string(mlrunv1.PhaseSucceeded) {
				return assertErr("phase=%q want Succeeded", g.JSON200.Phase)
			}
			return nil
		})
	})

	t.Run("MLRunCancel", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-apicancel")

		req := busyboxMLRunReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota)
		req.Roles[0].Template.Command = &[]string{"sh", "-c", "sleep 600"}
		r, err := h.createMLRun(ctx, ns, req)
		require.NoError(t, err)
		require.True(t, is2xx(r.StatusCode()), "create job: %d: %s", r.StatusCode(), string(r.Body))
		cleanupMLRun(t, ns, name)

		eventually(t, h.cfg.CRProvisionTimeout, func() error {
			var job mlrunv1.MLRun
			return h.get(ctx, ns, name, &job)
		})

		c, err := h.cancelMLRun(ctx, ns, name)
		require.NoError(t, err)
		require.True(t, is2xx(c.StatusCode()), "cancel job: %d: %s", c.StatusCode(), string(c.Body))

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
	})

	t.Run("KubeproxyPodsAndLogs", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-proxy")

		r, err := h.createMLRun(ctx, ns, busyboxMLRunReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota))
		require.NoError(t, err)
		require.True(t, is2xx(r.StatusCode()))
		cleanupMLRun(t, ns, name)

		// Pods sub-resource is reachable.
		eventually(t, h.cfg.PodReadyTimeout, func() error {
			p, err := h.computeService.ListMLRunPodsWithResponse(ctx, ns, name)
			if err != nil {
				return err
			}
			if !is2xx(p.StatusCode()) {
				return assertErr("GET pods: %d", p.StatusCode())
			}
			if !strings.Contains(string(p.Body), name) {
				return assertErr("pods list does not yet reference the job")
			}
			return nil
		})
		// Events sub-resource is reachable.
		e, err := h.computeService.ListMLRunEventsWithResponse(ctx, ns, name)
		require.NoError(t, err)
		assert.True(t, is2xx(e.StatusCode()), "GET events: %d", e.StatusCode())
	})

	t.Run("ServiceLifecycleScaleDelete", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-apisvc")

		r, err := h.createMLService(ctx, ns, nginxMLServiceReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota, nil))
		require.NoError(t, err)
		require.True(t, is2xx(r.StatusCode()), "create service: %d: %s", r.StatusCode(), string(r.Body))
		// The test deletes the service explicitly below, but register fallback
		// teardown so a mid-test failure can't leak it into the shared namespace.
		cleanupMLService(t, ns, name)

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
		s, err := h.scaleMLService(ctx, ns, name, 2)
		require.NoError(t, err)
		require.True(t, is2xx(s.StatusCode()), "scale: %d: %s", s.StatusCode(), string(s.Body))
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
		d, err := h.deleteMLService(ctx, ns, name)
		require.NoError(t, err)
		require.True(t, is2xx(d.StatusCode()), "delete: %d", d.StatusCode())
		eventually(t, h.cfg.CRProvisionTimeout, func() error {
			var svc mlservicev1.MLService
			if err := h.get(ctx, ns, name, &svc); isNotFound(err) {
				return nil
			} else if err != nil {
				return err
			}
			return assertErr("MLService %s still present", name)
		})
	})

	t.Run("WorkspacePVC", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-workspace")

		req := nginxMLServiceReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota, nil)
		req.Kind = ptr(mlservicev1.ServiceKindWorkspace)
		req.WorkspaceStorage = &computeservice.MLServiceWorkspaceStorageSpec{Size: "1Gi"}
		r, err := h.createMLService(ctx, ns, req)
		require.NoError(t, err)
		require.True(t, is2xx(r.StatusCode()), "create workspace: %d: %s", r.StatusCode(), string(r.Body))
		cleanupMLService(t, ns, name)

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
		d, err := h.deleteMLService(ctx, ns, name)
		require.NoError(t, err)
		require.True(t, is2xx(d.StatusCode()))
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
	})

	// Note: input-validation negatives (unknown pool, unknown namespace -> 4xx, no
	// CR) are NOT covered here — they schedule no workload and gain nothing from a
	// real cluster. The hermetic integration suite owns them (TestMLRunValidation).
}
