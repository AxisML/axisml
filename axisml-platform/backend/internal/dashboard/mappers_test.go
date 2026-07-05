package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

func TestToClusterPoolUsage(t *testing.T) {
	unit := "cores"
	u := &clustermanager.PoolUsage{
		Pool:   "gpu-a100",
		Tenant: "acme",
		Meters: []clustermanager.PoolMeter{{Resource: "cpu", Used: 8, Total: 32, Unit: &unit}},
	}
	v := toClusterPoolUsage(u)
	assert.Equal(t, "gpu-a100", v.Pool)
	require.Len(t, v.Meters, 1)
	assert.Equal(t, server.ClusterMeter{Resource: "cpu", Used: 8, Total: 32, Unit: "cores"}, v.Meters[0])
}

func TestPoolMetricToServer(t *testing.T) {
	ts := time.Unix(1700000000, 0).UTC()
	step, unit := "30s", "GiB"
	m := &clustermanager.PoolMetricSeries{
		Metric: "mem_util", Range: "1h", Step: &step, Unit: &unit,
		Series: []clustermanager.PoolMetricPoint{{Timestamp: ts, Value: 12.5}},
	}
	v := poolMetricToServer(m)
	assert.Equal(t, "mem_util", v.Metric)
	assert.Equal(t, "GiB", v.Unit)
	assert.Equal(t, "30s", v.Step)
	require.Len(t, v.Series, 1)
	assert.Equal(t, 12.5, v.Series[0].Value)
}
