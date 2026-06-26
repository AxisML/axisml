//go:build e2e || standard || lite

package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Black-box compute lifecycle tests. They drive only the typed compute-service
// client and assert over the HTTP contract (phase, pod projection, logs), so the
// same tests validate the Standard (Kubernetes operators) and Lite (in-process
// Standalone runtime) forms. White-box assertions on the materialized CR / Job /
// Deployment live at the integration layer, not here.

// TestMLRunLifecycle submits a native/job MLRun, waits for it to run to
// completion, and verifies the Pod projection and log streaming over HTTP.
func TestMLRunLifecycle(t *testing.T) {
	ctx := context.Background()
	ns, quota := h.Tenant(t)
	name := uniqueName("e2e-run")

	r, err := h.ComputeService().CreateMLRunWithResponse(ctx, ns, busyboxMLRunReq(name, h.config().DefaultPool, h.config().DefaultUnit, quota))
	require.NoError(t, err)
	require.Truef(t, is2xx(r.StatusCode()), "create mlrun: %d: %s", r.StatusCode(), string(r.Body))
	cleanupMLRun(t, ns, name)

	// Runs to Succeeded.
	eventually(t, h.config().MLRunCompleteTimeout, func() error {
		g, err := h.ComputeService().GetMLRunWithResponse(ctx, ns, name)
		if err != nil {
			return err
		}
		if g.JSON200 == nil {
			return assertErr("GET mlrun: %d", g.StatusCode())
		}
		if g.JSON200.Phase != "Succeeded" {
			return assertErr("phase=%q want Succeeded", g.JSON200.Phase)
		}
		return nil
	})

	// Pod projection is reachable and non-empty.
	pods, err := h.ComputeService().ListMLRunPodsWithResponse(ctx, ns, name)
	require.NoError(t, err)
	require.Truef(t, is2xx(pods.StatusCode()), "list pods: %d", pods.StatusCode())
	require.NotNil(t, pods.JSON200)
	require.NotEmpty(t, pods.JSON200.Items, "expected at least one pod")

	// Log streaming carries the container output.
	logs, err := h.ComputeService().GetMLRunPodLogsWithResponse(ctx, ns, name, pods.JSON200.Items[0].Name, nil)
	require.NoError(t, err)
	require.Truef(t, is2xx(logs.StatusCode()), "get logs: %d", logs.StatusCode())
	assert.Truef(t, strings.Contains(string(logs.Body), "hello"), "logs missing output: %q", string(logs.Body))
}

// TestMLRunCancel cancels a long-running job and verifies it converges to
// Cancelled over HTTP.
func TestMLRunCancel(t *testing.T) {
	ctx := context.Background()
	ns, quota := h.Tenant(t)
	name := uniqueName("e2e-cancel")

	req := busyboxMLRunReq(name, h.config().DefaultPool, h.config().DefaultUnit, quota)
	req.Roles[0].Template.Command = &[]string{"sh", "-c", "sleep 600"}
	r, err := h.ComputeService().CreateMLRunWithResponse(ctx, ns, req)
	require.NoError(t, err)
	require.Truef(t, is2xx(r.StatusCode()), "create mlrun: %d: %s", r.StatusCode(), string(r.Body))
	cleanupMLRun(t, ns, name)

	eventually(t, h.config().PodReadyTimeout, func() error {
		g, err := h.ComputeService().GetMLRunWithResponse(ctx, ns, name)
		if err != nil {
			return err
		}
		if g.JSON200 == nil || g.JSON200.Phase != "Running" {
			return assertErr("mlrun not Running yet")
		}
		return nil
	})

	c, err := h.ComputeService().CancelMLRunWithResponse(ctx, ns, name)
	require.NoError(t, err)
	require.Truef(t, is2xx(c.StatusCode()), "cancel: %d: %s", c.StatusCode(), string(c.Body))

	eventually(t, h.config().PodReadyTimeout, func() error {
		g, err := h.ComputeService().GetMLRunWithResponse(ctx, ns, name)
		if err != nil {
			return err
		}
		if g.JSON200 == nil || g.JSON200.Phase != "Cancelled" {
			return assertErr("phase not Cancelled yet")
		}
		return nil
	})
}

// TestMLServiceLifecycle brings up an nginx MLService, waits for Ready, verifies
// the pod projection, then deletes it and verifies it is gone.
func TestMLServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	ns, quota := h.Tenant(t)
	name := uniqueName("e2e-svc")

	r, err := h.ComputeService().CreateMLServiceWithResponse(ctx, ns, nginxMLServiceReq(name, h.config().DefaultPool, h.config().DefaultUnit, quota, nil))
	require.NoError(t, err)
	require.Truef(t, is2xx(r.StatusCode()), "create mlservice: %d: %s", r.StatusCode(), string(r.Body))
	cleanupMLService(t, ns, name)

	eventually(t, h.config().PodReadyTimeout, func() error {
		g, err := h.ComputeService().GetMLServiceWithResponse(ctx, ns, name)
		if err != nil {
			return err
		}
		if g.JSON200 == nil || g.JSON200.Phase != "Ready" {
			return assertErr("mlservice not Ready yet")
		}
		return nil
	})

	pods, err := h.ComputeService().ListMLServicePodsWithResponse(ctx, ns, name)
	require.NoError(t, err)
	require.Truef(t, is2xx(pods.StatusCode()), "list pods: %d", pods.StatusCode())
	require.NotNil(t, pods.JSON200)
	require.NotEmpty(t, pods.JSON200.Items)

	d, err := h.ComputeService().DeleteMLServiceWithResponse(ctx, ns, name)
	require.NoError(t, err)
	require.Truef(t, is2xx(d.StatusCode()), "delete: %d", d.StatusCode())

	eventually(t, h.config().CRProvisionTimeout, func() error {
		g, err := h.ComputeService().GetMLServiceWithResponse(ctx, ns, name)
		if err != nil {
			return err
		}
		if g.StatusCode() != http.StatusNotFound {
			return assertErr("mlservice still present: %d", g.StatusCode())
		}
		return nil
	})
}
