package promql_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/promql"
)

// mockProm serves a canned Prometheus matrix and captures the PromQL query.
func mockProm(captured *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if captured != nil {
			*captured = r.FormValue("query")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1700000000,"1.5"],[1700000030,"2.5"]]}]}}`))
	}))
}

func TestSeries_CPU_NodeScoped(t *testing.T) {
	var q string
	srv := mockProm(&q)
	defer srv.Close()

	qr, err := promql.New(srv.URL)
	require.NoError(t, err)
	require.True(t, qr.Enabled())

	s, err := qr.Series(context.Background(), "acme", map[string]string{"node.axisml.io/pool": "gpu"}, "cpu_util", "1h", "30s")
	require.NoError(t, err)
	assert.Equal(t, "cpu_util", s.Metric)
	assert.Equal(t, "cores", s.Unit)
	require.Len(t, s.Series, 2)
	assert.Equal(t, 1.5, s.Series[0].Value)

	assert.Contains(t, q, "container_cpu_usage_seconds_total")
	assert.Contains(t, q, `namespace="acme"`)
	assert.Contains(t, q, "kube_pod_info")
	assert.Contains(t, q, `label_node_axisml_io_pool="gpu"`)
}

func TestSeries_Memory_NamespaceOnly(t *testing.T) {
	var q string
	srv := mockProm(&q)
	defer srv.Close()

	qr, _ := promql.New(srv.URL)
	s, err := qr.Series(context.Background(), "acme", nil, "mem_util", "1h", "")
	require.NoError(t, err)
	assert.Equal(t, "GiB", s.Unit)
	assert.Contains(t, q, "container_memory_working_set_bytes")
	assert.Contains(t, q, "/ 1073741824")
	assert.NotContains(t, q, "kube_node_labels")
}

func TestSeries_Disabled(t *testing.T) {
	qr, err := promql.New("")
	require.NoError(t, err)
	assert.False(t, qr.Enabled())
	_, err = qr.Series(context.Background(), "acme", nil, "cpu_util", "1h", "")
	assert.ErrorIs(t, err, promql.ErrUnavailable)
}

func TestSeries_BadRequests(t *testing.T) {
	srv := mockProm(nil)
	defer srv.Close()
	qr, _ := promql.New(srv.URL)

	_, err := qr.Series(context.Background(), "acme", nil, "bogus", "1h", "")
	assert.ErrorIs(t, err, promql.ErrBadRequest)

	_, err = qr.Series(context.Background(), "acme", nil, "cpu_util", "nope", "")
	assert.ErrorIs(t, err, promql.ErrBadRequest)
}
