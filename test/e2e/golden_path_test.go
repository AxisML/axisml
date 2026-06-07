//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	mljobv1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// the cross-service golden path. One ordered journey through all five
// system services + infra. If this passes, the system layer is wired correctly
// end-to-end. It provisions and tears down its own tenant.
func TestGoldenPath_TrainAndServeJourney(t *testing.T) {
	ctx := context.Background()
	tenant := uniqueName("e2e-golden")
	ns := tenant

	// 1) cluster-manager: the seeded default pool is available.
	pr := h.clusterManager.mustDo(t, ctx, http.MethodGet, "/api/v1/resource-pools/"+h.cfg.DefaultPool, nil)
	require.True(t, pr.is2xx(), "default pool must exist: %d", pr.status)

	// 2) compute-service: create tenant + quota -> tenant-operator provisions ns.
	cr, err := h.createTenant(ctx, csCreateTenantReq{
		Name:      tenant,
		Namespace: csNamespaceSpec{Name: ns},
		Quotas:    []csQuotaSpec{{Pool: h.cfg.DefaultPool, Name: "default", Max: map[string]string{"cpu": "4", "memory": "8Gi"}}},
	})
	require.NoError(t, err)
	require.True(t, cr.is2xx(), "create golden tenant: %d: %s", cr.status, string(cr.body))
	t.Cleanup(func() { _, _ = h.deleteTenant(context.Background(), tenant) })
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, ns) })

	var quota string
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		names, err := elasticQuotaNames(ctx, ns)
		if err != nil || len(names) == 0 {
			return assertErr("no quota yet (err=%v)", err)
		}
		quota = names[0]
		return nil
	})

	// 3) artifact-hub: upload a model to zot.
	modelName := uniqueName("golden-model")
	res := initiateModel(t, ctx, ns, modelName, "1.0.0")
	pf, err := startPortForward(h.cfg.InfraNamespace, "zot", 5000)
	require.NoError(t, err)
	defer pf.Stop()
	oc := &ociClient{base: pf.localURL(), creds: parseOCICreds(res.Upload.Credentials), http: &http.Client{}}
	repo, ref := parseRepoRef(res.Upload.URI)
	digest, err := oc.pushConfigOnlyManifest(ctx, repo, ref)
	require.NoError(t, err, "push model manifest")
	cc := h.artifactHub.mustDo(t, ctx, http.MethodPost, modelPath(ns, modelName)+"/1.0.0/complete", ahCompleteReq{Digest: digest})
	require.True(t, cc.is2xx(), "complete model: %d: %s", cc.status, string(cc.body))

	// 4) compute-service: run a training job under the tenant quota -> real pod.
	jobName := uniqueName("golden-job")
	jr, err := h.createJob(ctx, ns, busyboxJobReq(jobName, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota))
	require.NoError(t, err)
	require.True(t, jr.is2xx(), "create job: %d: %s", jr.status, string(jr.body))
	eventually(t, h.cfg.JobCompleteTimeout, func() error {
		var job mljobv1.MLJob
		if err := h.get(ctx, ns, jobName, &job); err != nil {
			return err
		}
		if job.Status.Phase != mljobv1.PhaseSucceeded {
			return assertErr("job phase=%q want Succeeded", job.Status.Phase)
		}
		return nil
	})

	// 5) compute-service: serve the model (nginx stand-in) with a route.
	svcName := uniqueName("golden-svc")
	route := &mlservicev1.Route{Enabled: true, Hostname: svcName + ".e2e.local"}
	sr, err := h.createService(ctx, ns, nginxServiceReq(svcName, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota, route))
	require.NoError(t, err)
	require.True(t, sr.is2xx(), "create service: %d: %s", sr.status, string(sr.body))
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		var svc mlservicev1.MLService
		if err := h.get(ctx, ns, svcName, &svc); err != nil {
			return err
		}
		if svc.Status.Phase != mlservicev1.PhaseReady {
			return assertErr("service phase=%q want Ready", svc.Status.Phase)
		}
		return nil
	})
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		hr := httpRouteObj()
		return h.get(ctx, ns, svcName, hr)
	})

	// 6) teardown: delete the service, then the tenant; namespace is GC'd.
	_, _ = h.deleteService(ctx, ns, svcName)
	_, _ = h.deleteTenant(ctx, tenant)
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		if err := h.namespaceExists(ctx, ns); isNotFound(err) {
			return nil
		}
		return assertErr("namespace %s not GC'd after tenant delete", ns)
	})
}
