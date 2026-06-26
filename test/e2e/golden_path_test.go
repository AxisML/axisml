//go:build (e2e || standard) && !lite

package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlrunv1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/test/e2e/internal/clients/artifacthub"
	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
	"github.com/axisml/axisml/test/e2e/internal/clients/computeservice"
)

// the cross-service golden path: one ordered train-and-serve journey through
// the whole system layer in a single freshly-provisioned tenant.
//
// Unlike the per-service files (which own the happy/edge paths of each service
// in isolation), this test exists to assert the *cross-service contracts* that
// only emerge when the services are wired together — the seams no single-service
// test can see:
//
//   - a tenant+quota created through cluster-manager materializes a namespace +
//     ElasticQuota that compute-service can resolve and schedule a job into;
//   - a model uploaded to artifact-hub resolves back to the exact digest pushed
//     to the real registry;
//   - a service created through compute-service produces a gateway HTTPRoute
//     whose backend actually targets that service (not merely "a route exists");
//   - deleting the tenant removes the Tenant CR.
//
// Phases run as ordered subtests and short-circuit on the first failure (a
// broken tenant makes the later stages meaningless), giving per-stage failure
// localization while preserving the dependency order.
func TestGoldenPath_TrainAndServeJourney(t *testing.T) {
	ctx := context.Background()
	tenant := uniqueName("e2e-golden")
	ns := tenant

	var quota string
	modelName := uniqueName("golden-model")
	jobName := uniqueName("golden-job")
	svcName := uniqueName("golden-svc")

	// Tear the tenant (and its namespace) down only after the whole journey
	// completes. Registered on the parent t — a subtest-scoped t.Cleanup would
	// fire when the `tenant` subtest returns and delete the namespace out from
	// under the later train/serve subtests.
	t.Cleanup(func() { removeTenant(tenant, ns) })

	// --- tenant: cluster-manager -> tenant-operator -> namespace + ElasticQuota.
	ok := t.Run("tenant", func(t *testing.T) {
		pr, err := h.clusterManager.GetResourcePoolWithResponse(ctx, h.cfg.DefaultPool)
		require.NoError(t, err)
		require.True(t, is2xx(pr.StatusCode()), "default pool must exist: %d", pr.StatusCode())

		cr, err := h.createTenant(ctx, clustermanager.CreateTenantRequest{
			Name:      ptr(tenant),
			Namespace: &clustermanager.Apiv1alpha1NamespaceSpec{Name: ns},
			Quotas: &[]clustermanager.ServerQuota{{
				Pool:  h.cfg.DefaultPool,
				Units: []clustermanager.ServerQuotaUnit{{UnitName: h.cfg.DefaultUnit, Quantity: 2}},
			}},
		})
		require.NoError(t, err)
		require.True(t, is2xx(cr.StatusCode()), "create golden tenant: %d: %s", cr.StatusCode(), string(cr.Body))

		// Contract: the cluster-manager write fans out to a real namespace and a
		// real koord ElasticQuota the rest of the journey will schedule into.
		eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, ns) })
		eventually(t, h.cfg.CRProvisionTimeout, func() error {
			names, err := elasticQuotaNames(ctx, ns)
			if err != nil || len(names) == 0 {
				return assertErr("no quota yet (err=%v)", err)
			}
			quota = names[0]
			return nil
		})
	})
	if !ok {
		return
	}

	// --- model: artifact-hub two-phase upload resolves to the pushed digest.
	ok = t.Run("model", func(t *testing.T) {
		res := initiateModel(t, ctx, ns, modelName, "1.0.0")
		pf, err := startPortForward(h.cfg.InfraNamespace, "zot", 5000)
		require.NoError(t, err)
		defer pf.Stop()
		oc := &ociClient{base: pf.localURL(), creds: ociCredsFrom(res.Upload.Credentials), http: &http.Client{}}
		repo, ref := parseRepoRef(res.Upload.Uri)
		digest, err := oc.pushConfigOnlyManifest(ctx, repo, ref)
		require.NoError(t, err, "push model manifest")

		cc, err := h.artifactHub.CompleteModelWithResponse(ctx, ns, modelName, "1.0.0", artifacthub.ArtifactCompleteRequest{Digest: digest})
		require.NoError(t, err)
		require.True(t, is2xx(cc.StatusCode()), "complete model: %d: %s", cc.StatusCode(), string(cc.Body))

		// Contract: resolve round-trips the registry and echoes the exact digest
		// that was pushed — the metadata layer and the real OCI store agree.
		eventually(t, h.cfg.CRProvisionTimeout, func() error {
			r, err := h.artifactHub.ResolveModelWithResponse(ctx, ns, modelName, "1.0.0", nil)
			if err != nil {
				return err
			}
			if !is2xx(r.StatusCode()) {
				return assertErr("resolve: %d", r.StatusCode())
			}
			if r.JSON200 == nil || r.JSON200.Digest == nil {
				return assertErr("resolve: no digest in body")
			}
			if *r.JSON200.Digest != digest {
				return assertErr("resolve digest=%q want %q", *r.JSON200.Digest, digest)
			}
			return nil
		})
	})
	if !ok {
		return
	}

	// --- train: a job submitted to compute-service schedules under the
	// cluster-manager-originated quota and runs to completion.
	ok = t.Run("train", func(t *testing.T) {
		jr, err := h.createMLRun(ctx, ns, busyboxMLRunReq(jobName, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota))
		require.NoError(t, err)
		require.True(t, is2xx(jr.StatusCode()), "create job: %d: %s", jr.StatusCode(), string(jr.Body))
		cleanupMLRun(t, ns, jobName)

		// Contract: compute-service resolved (poolName, unitName) + the tenant
		// quota into a schedulable MLRun, and koord admitted it to completion.
		eventually(t, h.cfg.MLRunCompleteTimeout, func() error {
			var job mlrunv1.MLRun
			if err := h.get(ctx, ns, jobName, &job); err != nil {
				return err
			}
			if job.Status.Phase != mlrunv1.PhaseSucceeded {
				return assertErr("job phase=%q want Succeeded", job.Status.Phase)
			}
			return nil
		})
	})
	if !ok {
		return
	}

	// --- serve: a routed service yields an HTTPRoute that targets the service.
	ok = t.Run("serve", func(t *testing.T) {
		route := &computeservice.MLServiceRoute{Enabled: true, Hostname: ptr(svcName + ".e2e.local")}
		sr, err := h.createMLService(ctx, ns, nginxMLServiceReq(svcName, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota, route))
		require.NoError(t, err)
		require.True(t, is2xx(sr.StatusCode()), "create service: %d: %s", sr.StatusCode(), string(sr.Body))
		cleanupMLService(t, ns, svcName)

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

		// Contract: the operator derived a gateway HTTPRoute whose backendRef
		// actually points at the service (compute-service -> operator -> Envoy
		// Gateway). Asserting the backend target — not just the route's existence
		// — is what makes this an integration check.
		eventually(t, h.cfg.CRProvisionTimeout, func() error {
			w, err := httpRouteBackendWeights(ctx, ns, svcName)
			if err != nil {
				return err
			}
			for backend := range w {
				if strings.Contains(backend, svcName) {
					return nil
				}
			}
			return assertErr("HTTPRoute backends=%v do not target service %s", w, svcName)
		})
	})
	if !ok {
		return
	}

	// --- teardown: deleting the tenant removes the Tenant CR. The operator
	// intentionally retains the namespace (design §6.1), so the runner removes it.
	t.Run("teardown", func(t *testing.T) {
		t.Cleanup(func() {
			_ = h.k8s.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
		})
		_, _ = h.deleteTenant(ctx, tenant)
		eventually(t, h.cfg.CRProvisionTimeout, func() error {
			var ten tenantv1.Tenant
			if err := h.k8s.Get(ctx, client.ObjectKey{Name: tenant}, &ten); isNotFound(err) {
				return nil
			} else if err != nil {
				return err
			}
			return assertErr("tenant %s still present after delete", tenant)
		})
	})
}
