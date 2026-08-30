package standalone

import "time"

// Default operational parameters. For the standalone binary these are fixed — it is
// a single-host, single-installation Compose stack, so they do not differ across
// deployments and are not exposed as configuration (see docs/configuration.md →
// "Not configurable by design"). They are the package-level defaults that
// DefaultSettings folds into a Settings value; an embedding host overrides them
// via WithSettings.
const (
	// HTTP API listener. Standalone serves probes on the same engine.
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
	DefaultPoolConfigDir        = "/etc/axisml/pools"          // Cluster Manager pool + tenant YAML (resourcepools/ + tenants/ subdirs)
	DefaultGatewayConfigDir     = "/var/lib/axisml/gateway"    // Envoy Gateway file-provider resources
	DefaultWorkloadConfigDir    = "/var/lib/axisml/configmaps" // workload-owned ConfigMap volume projections
	DefaultWorkloadConfigVolume = "axisml-configmaps"          // shared Docker volume mounted at DefaultWorkloadConfigDir

	// Docker network dynamic workloads join (Envoy Gateway also joins it). Shared
	// with the Compose networks: block and the binary's idempotent EnsureNetwork.
	DefaultWorkloadsNetwork = "axisml-workloads"
)

// Settings are the operational parameters axisml-standalone runs with. They are fixed
// constants for the standalone binary (DefaultSettings) but overridable when
// axisml-standalone is embedded as a library, so the host can choose its own bind
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

	// PoolConfigDir holds ResourcePool + Tenant bootstrap YAML read at startup.
	// Ignored when the static config is supplied in-memory via
	// WithStaticConfig.
	PoolConfigDir string

	// GatewayConfigDir is the Envoy Gateway file-provider resource directory on
	// the host filesystem, written per service / traffic policy.
	GatewayConfigDir string

	// WorkloadConfigDir is where the Standalone runtime materializes files for
	// workload-owned ConfigMap volumes. WorkloadConfigVolume names the Docker
	// volume mounted there by the standalone Compose stack. An embedded host may leave
	// WorkloadConfigVolume empty to bind WorkloadConfigDir directly instead, as
	// long as the Docker daemon sees that directory at the same absolute path.
	WorkloadConfigDir    string
	WorkloadConfigVolume string

	// WorkloadsNetwork is the Docker network dynamic workloads (and Envoy Gateway) join.
	WorkloadsNetwork string
}

// DefaultSettings returns the fixed standalone operational parameters. The binary uses
// these unchanged; an embedding host starts from these and overrides the fields
// it needs.
func DefaultSettings() Settings {
	return Settings{
		APIBindAddress:       DefaultAPIBindAddress,
		ReconcileInterval:    DefaultReconcileInterval,
		GCInterval:           DefaultGCInterval,
		UploadingTTL:         DefaultUploadingTTL,
		UploadTokenTTL:       DefaultUploadTokenTTL,
		DatasetBucket:        DefaultDatasetBucket,
		PoolConfigDir:        DefaultPoolConfigDir,
		GatewayConfigDir:     DefaultGatewayConfigDir,
		WorkloadConfigDir:    DefaultWorkloadConfigDir,
		WorkloadConfigVolume: DefaultWorkloadConfigVolume,
		WorkloadsNetwork:     DefaultWorkloadsNetwork,
	}
}
