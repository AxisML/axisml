package server

// Capabilities describes what the Cluster Manager supports in the current
// deployment form. Standard backs the stores with the cluster-scoped ResourcePool
// and Tenant CRs (full CRUD); Lite backs them with read-only static config, so
// the write-bearing capabilities report false (design §5.1). The document is
// derived from the injected stores, never hardcoded per binary.
type Capabilities struct {
	// MultiTenant reports whether Tenant CRUD is available (false = single
	// static default tenant; tenant writes return 409 CapabilityUnavailable).
	MultiTenant bool `json:"multiTenant" desc:"Whether Tenant CRUD is available (false = single static default tenant)."`
	// ResourcePoolsWritable reports whether ResourcePool CRUD is available
	// (false = single read-only default pool).
	ResourcePoolsWritable bool `json:"resourcePoolsWritable" desc:"Whether ResourcePool CRUD is available (false = single read-only default pool)."`
}
