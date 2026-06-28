package config

import (
	"os"
	"time"
)

// Fixed operational constants. These do not differ across deployments, so they
// are not configurable (see docs/configuration.md → "Not configurable by design").
const (
	// Listen addresses, uniform across all AxisML services.
	APIBindAddress     = ":8080"
	MetricsBindAddress = ":8081"
	ProbesBindAddress  = ":8082"

	// Reconcile cadence.
	ReconcileInterval = 2 * time.Second

	// Leader election (controller-runtime lease). Always on: a no-op at one
	// replica, required automatically when scaled. The lease backend is always
	// present (in-cluster API + RBAC).
	LeaderElect         = true
	LeaderElectionID    = "axisml-compute-service.axisml.io"
	LeaderResourceLock  = "leases"
	LeaderLeaseDuration = 15 * time.Second
	LeaderRenewDeadline = 10 * time.Second
	LeaderRetryPeriod   = 2 * time.Second
)

// LeaderNamespace is the namespace holding the leader-election lease, derived
// from the downward-API POD_NAMESPACE (falls back to the system namespace).
func LeaderNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return "axisml-system"
}
