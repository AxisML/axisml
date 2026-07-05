//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmapp "github.com/axisml/axisml/axisml-system/cluster-manager/internal/app"
	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/promql"
	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
)

// TestResourcePoolMetrics_Series drives the N3 endpoint end-to-end against a
// fake Prometheus: a metrics-enabled router returns a projected PoolMetricSeries.
func TestResourcePoolMetrics_Series(t *testing.T) {
	const pool = "n3-pool"
	seedPool(t, pool)

	prom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1700000000,"3.5"]]}]}}`))
	}))
	defer prom.Close()

	querier, err := promql.New(prom.URL)
	require.NoError(t, err)
	rtr := cmapp.NewRouter(testCli, querier)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/resourcepools/"+pool+"/metrics?tenant=team&metric=cpu_util&range=1h&step=30s", nil)
	req.Header.Set("X-Axisml-User", "test-user")
	rr := httptest.NewRecorder()
	rtr.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var series srv.PoolMetricSeries
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &series))
	assert.Equal(t, "cpu_util", series.Metric)
	assert.Equal(t, "cores", series.Unit)
	require.Len(t, series.Series, 1)
	assert.Equal(t, 3.5, series.Series[0].Value)

	// An unsupported metric is a client error even with a live backend.
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/resourcepools/"+pool+"/metrics?tenant=team&metric=bogus&range=1h", nil)
	req.Header.Set("X-Axisml-User", "test-user")
	rr = httptest.NewRecorder()
	rtr.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
}
