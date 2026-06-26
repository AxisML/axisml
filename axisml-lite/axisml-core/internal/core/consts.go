package core

import "time"

// Fixed operational constants. Lite is a single-host, single-installation
// Compose stack, so these do not differ across deployments and are not
// configurable (see docs/configuration.md → "Not configurable by design").
const (
	// HTTP API listener. Lite serves probes on the same engine.
	APIBindAddress = ":8080"

	// Reconciler / status-poller tick.
	ReconcileInterval = 2 * time.Second

	// Artifact Hub GC + upload lifecycle.
	GCInterval     = 5 * time.Minute
	UploadingTTL   = 24 * time.Hour
	UploadTokenTTL = time.Hour

	// Object store bucket for datasets.
	DatasetBucket = "axisml-artifact-hub"

	// Filesystem layout (fixed by the image / Compose mounts).
	PoolConfigDir    = "/etc/axisml/pools"       // Cluster Manager pool + tenant YAML
	StateDir         = "/var/lib/axisml/runtime" // managed scratch dir
	GatewayConfigDir = "/var/lib/axisml/traefik" // Traefik file-provider dynamic config

	// Docker network dynamic workloads join (Traefik also joins it). Shared
	// with the Compose networks: block and the binary's idempotent EnsureNetwork.
	WorkloadsNetwork = "axisml-workloads"
)
