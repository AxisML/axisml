package standalone_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	standalone "github.com/axisml/axisml/axisml-standalone"
)

func TestDefaultSettings(t *testing.T) {
	s := standalone.DefaultSettings()
	assert.Equal(t, ":8080", s.APIBindAddress)
	assert.Equal(t, 2*time.Second, s.ReconcileInterval)
	assert.Equal(t, 5*time.Minute, s.GCInterval)
	assert.Equal(t, 24*time.Hour, s.UploadingTTL)
	assert.Equal(t, time.Hour, s.UploadTokenTTL)
	assert.Equal(t, "axisml-artifact-hub", s.DatasetBucket)
	assert.Equal(t, "/etc/axisml/pools", s.PoolConfigDir)
	assert.Equal(t, "/var/lib/axisml/gateway", s.GatewayConfigDir)
	assert.Equal(t, "/var/lib/axisml/configmaps", s.WorkloadConfigDir)
	assert.Equal(t, "axisml-configmaps", s.WorkloadConfigVolume)
	assert.Equal(t, "axisml-workloads", s.WorkloadsNetwork)
}
