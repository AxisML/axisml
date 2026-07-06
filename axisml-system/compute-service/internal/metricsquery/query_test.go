package metricsquery_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/metricsquery"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

type fakeProvider struct {
	enabled   bool
	lastQuery string
	called    bool
	samples   []extensions.MetricSample
	err       error
}

func (f *fakeProvider) Enabled() bool { return f.enabled }

func (f *fakeProvider) QueryRange(_ context.Context, q string, _, _ time.Duration) ([]extensions.MetricSample, error) {
	f.called = true
	f.lastQuery = q
	return f.samples, f.err
}

func codeOf(t *testing.T, err error) apperrors.Code {
	t.Helper()
	var ae *apperrors.E
	require.ErrorAs(t, err, &ae)
	return ae.Code
}

func TestSeries_CPU_BuildsQueryAndMapsPoints(t *testing.T) {
	ts := time.Unix(1700000000, 0).UTC()
	fp := &fakeProvider{enabled: true, samples: []extensions.MetricSample{{Timestamp: ts, Value: 1.5}}}
	s, err := metricsquery.NewQuerier(fp).Series(context.Background(), "acme", []string{"pod-a", "pod-b"}, "cpu_util", "1h", "30s")
	require.NoError(t, err)
	assert.Equal(t, "cpu_util", s.Metric)
	assert.Equal(t, "cores", s.Unit)
	assert.Equal(t, "1h", s.Range)
	require.Len(t, s.Series, 1)
	assert.Equal(t, 1.5, s.Series[0].Value)
	assert.Equal(t, ts, s.Series[0].Timestamp)
	assert.Contains(t, fp.lastQuery, "container_cpu_usage_seconds_total")
	assert.Contains(t, fp.lastQuery, `namespace="acme"`)
	assert.Contains(t, fp.lastQuery, `pod=~"pod-a|pod-b"`)
}

func TestSeries_GPU_UsesDCGMExportedLabels(t *testing.T) {
	fp := &fakeProvider{enabled: true}
	s, err := metricsquery.NewQuerier(fp).Series(context.Background(), "acme", []string{"p1"}, "gpu_util", "24h", "")
	require.NoError(t, err)
	assert.Equal(t, "percent", s.Unit)
	assert.Contains(t, fp.lastQuery, "DCGM_FI_DEV_GPU_UTIL")
	assert.Contains(t, fp.lastQuery, `exported_namespace="acme"`)
	assert.Contains(t, fp.lastQuery, `exported_pod=~"p1"`)
}

func TestSeries_Memory_Unit(t *testing.T) {
	fp := &fakeProvider{enabled: true}
	s, err := metricsquery.NewQuerier(fp).Series(context.Background(), "acme", []string{"p1"}, "mem_util", "6h", "")
	require.NoError(t, err)
	assert.Equal(t, "bytes", s.Unit)
	assert.Contains(t, fp.lastQuery, "container_memory_working_set_bytes")
}

func TestSeries_EmptyPods_ReturnsEmptySeriesWithoutQuerying(t *testing.T) {
	fp := &fakeProvider{enabled: true}
	s, err := metricsquery.NewQuerier(fp).Series(context.Background(), "acme", nil, "cpu_util", "1h", "")
	require.NoError(t, err)
	assert.Empty(t, s.Series)
	assert.False(t, fp.called, "no live pods → no Prometheus query")
}

func TestSeries_UnsupportedMetric(t *testing.T) {
	_, err := metricsquery.NewQuerier(&fakeProvider{enabled: true}).
		Series(context.Background(), "acme", []string{"p"}, "request_rate", "1h", "")
	require.Error(t, err)
	assert.Equal(t, apperrors.CodeValidation, codeOf(t, err))
}

func TestSeries_ExplicitStepTooFine_Rejected(t *testing.T) {
	fp := &fakeProvider{enabled: true}
	_, err := metricsquery.NewQuerier(fp).
		Series(context.Background(), "acme", []string{"p"}, "cpu_util", "30d", "1s")
	require.Error(t, err)
	assert.Equal(t, apperrors.CodeValidation, codeOf(t, err), "an over-fine step is a client error, not a downstream failure")
	assert.False(t, fp.called, "must reject before querying Prometheus")
}

func TestSeries_DisabledProvider_Unavailable(t *testing.T) {
	_, err := metricsquery.NewQuerier(&fakeProvider{enabled: false}).
		Series(context.Background(), "acme", []string{"p"}, "cpu_util", "1h", "")
	require.Error(t, err)
	assert.Equal(t, apperrors.CodeUnavailable, codeOf(t, err))
}

func TestSeries_NilProvider_Unavailable(t *testing.T) {
	_, err := metricsquery.NewQuerier(nil).Series(context.Background(), "acme", []string{"p"}, "cpu_util", "1h", "")
	require.Error(t, err)
	assert.Equal(t, apperrors.CodeUnavailable, codeOf(t, err))
}

func TestSeries_RangeParsing(t *testing.T) {
	fp := &fakeProvider{enabled: true}
	q := metricsquery.NewQuerier(fp)
	for _, ok := range []string{"5m", "1h", "24h", "7d", "2w"} {
		_, err := q.Series(context.Background(), "acme", []string{"p"}, "cpu_util", ok, "")
		require.NoError(t, err, "range %s", ok)
	}
	for _, bad := range []string{"", "abc", "1x"} {
		_, err := q.Series(context.Background(), "acme", []string{"p"}, "cpu_util", bad, "")
		require.Error(t, err, "range %s", bad)
		assert.Equal(t, apperrors.CodeValidation, codeOf(t, err))
	}
}
