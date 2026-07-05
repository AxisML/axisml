package compview_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	"github.com/axisml/axisml/axisml-platform/backend/internal/compview"
)

func TestMetricSeries(t *testing.T) {
	ts := time.Unix(1700000000, 0).UTC()
	step, unit := "30s", "cores"
	m := &computeservice.MetricSeries{
		Metric: "cpu_util", Range: "1h", Step: &step, Unit: &unit,
		Series: []computeservice.MetricPoint{{Timestamp: ts, Value: 1.5}},
	}
	v := compview.MetricSeries(m)
	assert.Equal(t, "cpu_util", v.Metric)
	assert.Equal(t, "1h", v.Range)
	assert.Equal(t, "30s", v.Step)
	assert.Equal(t, "cores", v.Unit)
	require.Len(t, v.Series, 1)
	assert.Equal(t, ts, v.Series[0].Timestamp)
	assert.Equal(t, 1.5, v.Series[0].Value)
}

func TestMetricSeries_NilOptionals(t *testing.T) {
	v := compview.MetricSeries(&computeservice.MetricSeries{Metric: "mem_util", Range: "1h"})
	assert.Empty(t, v.Step)
	assert.Empty(t, v.Unit)
	assert.Empty(t, v.Series)
}
