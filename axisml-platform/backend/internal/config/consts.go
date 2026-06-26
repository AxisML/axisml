package config

import "time"

// Fixed operational constants. These do not differ across deployments, so they
// are not configurable (see docs/configuration.md → "Not configurable by design").
const (
	// Listen addresses (platform-backend has no metrics endpoint).
	APIBindAddress    = ":8080"
	ProbesBindAddress = ":8081"

	// HTTP client timeout for System-layer upstream calls.
	UpstreamTimeout = 30 * time.Second

	// Auth-hot-path cache TTLs and the expired-session sweep cadence.
	SessionCacheTTL      = 5 * time.Minute
	IdentityCacheTTL     = time.Minute
	SessionSweepInterval = time.Hour

	// Data-plane workspace access JWT lifetime.
	WorkspaceAccessJWTTTL = time.Hour

	// DefaultTenant is the System-seeded built-in tenant name. It is the home for
	// visibility=public artifacts and the tenant platform-backend imports at
	// bootstrap (it is created by the System chart's seed.tenant, not here).
	DefaultTenant = "default"
)
