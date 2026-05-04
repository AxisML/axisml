//go:build e2e

package compute_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/operators/mljob-operator/api/v1alpha1"

	"github.com/axisml/axisml/test/e2e"
)

// TestComputeAPI_JobLifecycle: POST /jobs through to a real Pod completing
// on minikube, asserting GET /jobs/:job returns status=Succeeded.
func TestComputeAPI_JobLifecycle(t *testing.T) {
	_, c := e2e.SetupOrSkip(t)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	api := e2e.PortForwardCompute(t)
	fx := setupComputeFixture(t, ctx, c, api, "e2e-job")

	const jobName = "hello"
	t.Cleanup(func() {
		cleanCtx, cancelClean := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancelClean()
		_ = api.DoJSON(t, cleanCtx, http.MethodDelete,
			fmt.Sprintf("/api/v1/tenants/%s/jobs/%s", fx.TenantName, jobName), nil, nil).Body.Close()
	})

	// Submit.
	resp := api.DoJSON(t, ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/tenants/%s/jobs", fx.TenantName),
		map[string]any{
			"name":           jobName,
			"resourceUnitId": fx.UnitID,
			"quotaId":        fx.QuotaID,
			"roles": []map[string]any{{
				"name":          mljobv1alpha1.DefaultRoleName,
				"replicas":      1,
				"restartPolicy": string(corev1.RestartPolicyNever),
				"template": map[string]any{
					"image":   "busybox:latest",
					"command": []string{"sh", "-c", "echo hello"},
				},
			}},
		}, nil)
	body := e2e.ReadBody(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create job: %s", e2e.PrettyResp(resp, body))

	// Poll the API until the informer surfaces status=Succeeded — that
	// covers compute → DB → reconciler → CR → operator → Pod → CR.status →
	// informer → DB. ~2-3 min on minikube depending on image-pull cache.
	require.Eventually(t, func() bool {
		var view struct {
			Status string `json:"status"`
		}
		grr := api.DoJSON(t, ctx, http.MethodGet,
			fmt.Sprintf("/api/v1/tenants/%s/jobs/%s", fx.TenantName, jobName), nil, &view)
		_ = grr.Body.Close()
		t.Logf("job status: %s (http %d)", view.Status, grr.StatusCode)
		return view.Status == "Succeeded"
	}, 5*time.Minute, 5*time.Second, "job did not reach Succeeded via compute API")
}

// computeFixture is the e2e analog of the envtest apiFixture: a tenant
// namespace + pool unit + quota wired through the compute HTTP API. The
// pool is the bootstrap-shared `default` pool (created by the helm
// post-install Job) so the e2e cluster doesn't accumulate pools across
// runs; the unit and quota are test-scoped and explicitly cleaned up.
type computeFixture struct {
	TenantName string
	TenantNS   string
	PoolName   string
	UnitName   string
	UnitID     string
	QuotaName  string
	QuotaID    string
}

func setupComputeFixture(t *testing.T, ctx context.Context, _ any, api *e2e.ComputeHTTP, stem string) computeFixture {
	t.Helper()
	fx := computeFixture{
		TenantName: stem + "-tnt",
		TenantNS:   stem + "-ns",
		PoolName:   "default",
		UnitName:   stem + "-unit",
		QuotaName:  stem + "-quota",
	}
	t.Cleanup(func() {
		cleanCtx, cancelClean := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancelClean()
		_ = api.DoJSON(t, cleanCtx, http.MethodDelete,
			fmt.Sprintf("/api/v1/tenants/%s/quotas/%s", fx.TenantName, fx.QuotaName), nil, nil).Body.Close()
		_ = api.DoJSON(t, cleanCtx, http.MethodDelete,
			fmt.Sprintf("/api/v1/resource-pools/%s/resource-units/%s", fx.PoolName, fx.UnitName), nil, nil).Body.Close()
		_ = api.DoJSON(t, cleanCtx, http.MethodDelete,
			fmt.Sprintf("/api/v1/tenants/%s", fx.TenantName), nil, nil).Body.Close()
		// Tenant operator doesn't delete the namespace; do it from the
		// generic clusterClient. Imported via setup() in setup_test.go.
		if clusterClient != nil {
			_ = clusterClient.Delete(cleanCtx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: fx.TenantNS},
			})
		}
	})

	// Tenant.
	resp := api.DoJSON(t, ctx, http.MethodPost, "/api/v1/tenants", map[string]any{
		"name":      fx.TenantName,
		"namespace": map[string]any{"name": fx.TenantNS},
	}, nil)
	body := e2e.ReadBody(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create tenant: %s", e2e.PrettyResp(resp, body))

	// Resource Unit (small enough to fit any minikube node).
	var unit struct {
		ID string `json:"id"`
	}
	resp = api.DoJSON(t, ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/resource-pools/%s/resource-units", fx.PoolName),
		map[string]any{
			"name":     fx.UnitName,
			"requests": map[string]any{"cpu": "50m", "memory": "32Mi"},
			"limits":   map[string]any{"cpu": "100m", "memory": "64Mi"},
		}, &unit)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create unit: %s", e2e.PrettyResp(resp, e2e.ReadBody(resp)))
	fx.UnitID = unit.ID

	// Quota.
	var qview struct {
		ID string `json:"id"`
	}
	resp = api.DoJSON(t, ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/tenants/%s/quotas", fx.TenantName),
		map[string]any{
			"pool": fx.PoolName,
			"name": fx.QuotaName,
			"max":  map[string]any{"cpu": "1", "memory": "256Mi"},
		}, &qview)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create quota: %s", e2e.PrettyResp(resp, e2e.ReadBody(resp)))
	fx.QuotaID = qview.ID

	return fx
}
