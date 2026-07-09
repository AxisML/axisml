package core_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/axisml/axisml/axisml-lite/axisml-core/pkg/core"
)

func TestDefaultSettings(t *testing.T) {
	s := core.DefaultSettings()
	assert.Equal(t, ":8080", s.APIBindAddress)
	assert.Equal(t, 2*time.Second, s.ReconcileInterval)
	assert.Equal(t, 5*time.Minute, s.GCInterval)
	assert.Equal(t, 24*time.Hour, s.UploadingTTL)
	assert.Equal(t, time.Hour, s.UploadTokenTTL)
	assert.Equal(t, "axisml-artifact-hub", s.DatasetBucket)
	assert.Equal(t, "/etc/axisml/pools", s.PoolConfigDir)
	assert.Equal(t, "/var/lib/axisml/traefik", s.GatewayConfigDir)
	assert.Equal(t, "axisml-workloads", s.WorkloadsNetwork)
}
