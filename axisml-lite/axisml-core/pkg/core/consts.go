package core

import "time"

// Default operational parameters. For the Lite binary these are fixed — Lite is
// a single-host, single-installation Compose stack, so they do not differ across
// deployments and are not exposed as configuration (see docs/configuration.md →
// "Not configurable by design"). They are the package-level defaults that
// DefaultSettings folds into a Settings value; an embedding host overrides them
// via WithSettings.
const (
	// HTTP API listener. Lite serves probes on the same engine.
	DefaultAPIBindAddress = ":8080"

	// Reconciler / status-poller tick.
	DefaultReconcileInterval = 2 * time.Second

	// Artifact Hub GC + upload lifecycle.
	DefaultGCInterval     = 5 * time.Minute
	DefaultUploadingTTL   = 24 * time.Hour
	DefaultUploadTokenTTL = time.Hour

	// Object store bucket for datasets.
	DefaultDatasetBucket = "axisml-artifact-hub"

	// Filesystem layout (fixed by the image / Compose mounts).
	DefaultPoolConfigDir    = "/etc/axisml/pools"       // Cluster Manager pool + tenant YAML
	DefaultStateDir         = "/var/lib/axisml/runtime" // managed scratch dir
	DefaultGatewayConfigDir = "/var/lib/axisml/traefik" // Traefik file-provider dynamic config

	// Docker network dynamic workloads join (Traefik also joins it). Shared
	// with the Compose networks: block and the binary's idempotent EnsureNetwork.
	DefaultWorkloadsNetwork = "axisml-workloads"
)

// Settings are the operational parameters axisml-core runs with. They are fixed
// constants for the Lite binary (DefaultSettings) but overridable when
// axisml-core is embedded as a library, so the host can choose its own bind
// address, filesystem layout, Docker network and background cadences. Unlike
// Config (database / OCI / logging, sourced from AXISML_ env), these are never
// read from the environment — pass them explicitly via WithSettings.
type Settings struct {
	// APIBindAddress is the listen address used by App.Serve. It is ignored when
	// the host mounts App.Handler on its own server.
	APIBindAddress string

	// ReconcileInterval is the Compute reconciler / status-poller tick.
	ReconcileInterval time.Duration

	// GCInterval, UploadingTTL and UploadTokenTTL drive the Artifact Hub GC and
	// upload lifecycle.
	GCInterval     time.Duration
	UploadingTTL   time.Duration
	UploadTokenTTL time.Duration

	// DatasetBucket is the object-store bucket for datasets.
	DatasetBucket string

	// PoolConfigDir holds the Cluster Manager ResourcePool + Tenant YAML read at
	// startup. Ignored when the static config is supplied in-memory via
	// WithStaticConfig.
	PoolConfigDir string

	// StateDir and GatewayConfigDir are the runtime scratch dir and the Traefik
	// file-provider config dir on the host filesystem.
	StateDir         string
	GatewayConfigDir string

	// WorkloadsNetwork is the Docker network dynamic workloads (and Traefik) join.
	WorkloadsNetwork string
}

// DefaultSettings returns the fixed Lite operational parameters. The binary uses
// these unchanged; an embedding host starts from these and overrides the fields
// it needs.
func DefaultSettings() Settings {
	return Settings{
		APIBindAddress:    DefaultAPIBindAddress,
		ReconcileInterval: DefaultReconcileInterval,
		GCInterval:        DefaultGCInterval,
		UploadingTTL:      DefaultUploadingTTL,
		UploadTokenTTL:    DefaultUploadTokenTTL,
		DatasetBucket:     DefaultDatasetBucket,
		PoolConfigDir:     DefaultPoolConfigDir,
		StateDir:          DefaultStateDir,
		GatewayConfigDir:  DefaultGatewayConfigDir,
		WorkloadsNetwork:  DefaultWorkloadsNetwork,
	}
}
