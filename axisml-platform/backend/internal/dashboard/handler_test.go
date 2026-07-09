package dashboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
)

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name string
		in   string
		def  int
		want int
	}{
		{"empty uses default", "", 50, 50},
		{"non-numeric uses default", "abc", 50, 50},
		{"zero uses default", "0", 50, 50},
		{"negative uses default", "-5", 50, 50},
		{"valid value passes through", "10", 50, 10},
		{"exactly max passes through", "200", 50, 200},
		{"above max clamps to 200", "500", 50, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseLimit(tt.in, tt.def))
		})
	}
}

func TestToClusterPoolUsage_NilUnit(t *testing.T) {
	u := &clustermanager.PoolUsage{
		Pool:   "cpu-pool",
		Meters: []clustermanager.PoolMeter{{Resource: "cpu", Used: 1, Total: 4, Unit: nil}},
	}
	v := toClusterPoolUsage(u)
	assert.Equal(t, "cpu-pool", v.Pool)
	require.Len(t, v.Meters, 1)
	assert.Equal(t, "", v.Meters[0].Unit) // nil Unit stays empty
}

func TestToClusterPoolUsage_EmptyMeters(t *testing.T) {
	v := toClusterPoolUsage(&clustermanager.PoolUsage{Pool: "empty"})
	assert.Equal(t, "empty", v.Pool)
	assert.Empty(t, v.Meters)
	assert.NotNil(t, v.Meters) // constructed slice, not nil
}

func TestPoolMetricToServer_NilStepAndUnit(t *testing.T) {
	m := &clustermanager.PoolMetricSeries{Metric: "cpu_util", Range: "1h", Step: nil, Unit: nil}
	v := poolMetricToServer(m)
	assert.Equal(t, "cpu_util", v.Metric)
	assert.Equal(t, "1h", v.Range)
	assert.Equal(t, "", v.Step)
	assert.Equal(t, "", v.Unit)
	assert.Empty(t, v.Series)
	assert.NotNil(t, v.Series)
}
