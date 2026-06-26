package config

import (
	"hash/fnv"
	"time"
)

// Fixed operational constants. These do not differ across deployments, so they
// are not configurable (see docs/configuration.md → "Not configurable by design").
const (
	// Listen addresses, uniform across all AxisML services.
	APIBindAddress     = ":8080"
	ProbesBindAddress  = ":8081"
	MetricsBindAddress = ":9090"

	// GC worker cadence and TTLs.
	GCInterval     = 5 * time.Minute
	UploadingTTL   = 24 * time.Hour
	UploadTokenTTL = time.Hour

	// Leader election (Postgres advisory lock; gates only the GC worker — HTTP
	// serves on every replica). Always on: a no-op at one replica, required
	// automatically when scaled.
	LeaderElect       = true
	LeaderRetryPeriod = 2 * time.Second
)

// LeaderLockKey is the stable pg_advisory_lock key shared by all replicas,
// derived from the service name so distinct services on one database never
// collide.
var LeaderLockKey = lockKey("axisml-artifact-hub-gc")

func lockKey(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64())
}
