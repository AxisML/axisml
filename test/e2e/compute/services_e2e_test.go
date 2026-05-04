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

	mlservicev1alpha1 "github.com/axisml/axisml/components/operators/mlservice-operator/api/v1alpha1"

	"github.com/axisml/axisml/test/e2e"
)

// TestComputeAPI_ServiceLifecycle drives the service vertical slice via
// the compute HTTP API: POST /services → MLService CR → mlservice-operator
// (native/deployment) → Deployment → real Pod ready → operator → CR
// status=Ready → informer → DB → GET /services/:service returns
// status=Ready. Then POST /:service/scale {replicas: 2} flows back through
// the same path and verifies the Deployment scales.
//
// Replaces the direct-CR `TestMLService_NativeDeployment` with the same
// integration breadth plus the /:scale custom action.
func TestComputeAPI_ServiceLifecycle(t *testing.T) {
	_, c := e2e.SetupOrSkip(t)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	api := e2e.PortForwardCompute(t)
	fx := setupComputeFixture(t, ctx, c, api, "e2e-svc")

	const svcName = "predictor"
	t.Cleanup(func() {
		cleanCtx, cancelClean := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancelClean()
		_ = api.DoJSON(t, cleanCtx, http.MethodDelete,
			fmt.Sprintf("/api/v1/tenants/%s/services/%s", fx.TenantName, svcName), nil, nil).Body.Close()
	})

	// Submit.
	resp := api.DoJSON(t, ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/tenants/%s/services", fx.TenantName),
		map[string]any{
			"name":           svcName,
			"resourceUnitId": fx.UnitID,
			"quotaId":        fx.QuotaID,
			"modelRef":       map[string]any{"name": "demo", "version": "v1"},
			"roles": []map[string]any{{
				"name":     mlservicev1alpha1.DefaultRoleName,
				"replicas": 1,
				"template": map[string]any{
					"image": "nginx:1.27",
					"ports": []map[string]any{{
						"name":          "http",
						"containerPort": 8080,
						"protocol":      string(corev1.ProtocolTCP),
					}},
				},
			}},
		}, nil)
	body := e2e.ReadBody(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create service: %s", e2e.PrettyResp(resp, body))

	// Wait for status=Ready.
	require.Eventually(t, func() bool {
		var view struct {
			Status string `json:"status"`
		}
		grr := api.DoJSON(t, ctx, http.MethodGet,
			fmt.Sprintf("/api/v1/tenants/%s/services/%s", fx.TenantName, svcName), nil, &view)
		_ = grr.Body.Close()
		t.Logf("service status: %s (http %d)", view.Status, grr.StatusCode)
		return view.Status == "Ready"
	}, 5*time.Minute, 5*time.Second, "service did not reach Ready via compute API")

	// Scale.
	resp = api.DoJSON(t, ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/tenants/%s/services/%s/scale", fx.TenantName, svcName),
		map[string]any{"replicas": 2}, nil)
	body = e2e.ReadBody(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode, "scale: %s", e2e.PrettyResp(resp, body))

	// Wait for replicas=2 reflected back via the API view.
	require.Eventually(t, func() bool {
		var view struct {
			Replicas int32 `json:"replicas"`
		}
		grr := api.DoJSON(t, ctx, http.MethodGet,
			fmt.Sprintf("/api/v1/tenants/%s/services/%s", fx.TenantName, svcName), nil, &view)
		_ = grr.Body.Close()
		return view.Replicas == 2
	}, 2*time.Minute, 5*time.Second, "service did not scale to 2 replicas")
}
