// Package core holds the axisml-core composition root: configuration, the
// config-backed Cluster Manager providers (single default ResourcePool +
// Tenant), the GORM/DB coordination and the assembly that mounts the three
// System modules (Cluster Manager, Compute Service, Artifact Hub) plus the
// in-process Standalone Runtime on one router.
package core

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the axisml-core process configuration, sourced entirely from the
// environment (Compose injects it). A single binary fronts all three System
// modules, so the database, OCI and HTTP settings are shared.
type Config struct {
	// Database (shared by Compute Service + Artifact Hub migrations/business).
	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseSSLMode  string

	// HTTP listeners. Probes (/healthz, /readyz) are served on the same engine
	// as the API; Lite is a single process, so there is no separate probes port.
	APIBindAddress string

	// Reconciler / status-poller tick.
	ReconcileInterval time.Duration

	// Cluster Manager static config (resource-pool.yaml + tenant.yaml).
	ConfigDir string

	// Artifact Hub / OCI registry (zot) + dataset object store.
	OCIEndpoint      string
	OCIScheme        string
	OCIAdminUser     string
	OCIAdminPassword string
	DatasetBucket    string
	GCInterval       time.Duration
	UploadingTTL     time.Duration
	UploadTokenTTL   time.Duration

	// Standalone Runtime (Docker).
	DockerHost        string // empty = SDK default (DOCKER_HOST or unix socket)
	WorkloadsNetwork  string // docker network dynamic workloads join
	RuntimeDir        string // managed scratch dir (/var/lib/axisml/runtime)
	TraefikDir        string // Traefik file-provider dynamic config dir
	InstallationID    string // stable installation identity stamped on resources
	WorkloadImagePull bool   // pull images before creating containers

	// Logging.
	LogDevelopment bool
}

// Load builds the Config from environment variables with safe local defaults.
func Load() Config {
	return Config{
		DatabaseHost:     env("DATABASE_HOST", "localhost"),
		DatabasePort:     envInt("DATABASE_PORT", 5432),
		DatabaseName:     env("DATABASE_NAME", "axisml"),
		DatabaseUser:     env("DATABASE_USER", "axisml"),
		DatabasePassword: env("DATABASE_PASSWORD", "axisml"),
		DatabaseSSLMode:  env("DATABASE_SSLMODE", "disable"),

		APIBindAddress: env("API_BIND_ADDRESS", ":8080"),

		ReconcileInterval: envDuration("RECONCILE_INTERVAL", 2*time.Second),

		ConfigDir: env("CONFIG_DIR", "/etc/axisml/config"),

		OCIEndpoint:      env("OCI_ENDPOINT", "http://zot:5000"),
		OCIScheme:        env("OCI_SCHEME", "http"),
		OCIAdminUser:     env("OCI_ADMIN_USER", "admin"),
		OCIAdminPassword: env("OCI_ADMIN_PASSWORD", ""),
		DatasetBucket:    env("DATASET_BUCKET", "axisml-artifact-hub"),
		GCInterval:       envDuration("GC_INTERVAL", 5*time.Minute),
		UploadingTTL:     envDuration("UPLOADING_TTL", 24*time.Hour),
		UploadTokenTTL:   envDuration("UPLOAD_TOKEN_TTL", time.Hour),

		DockerHost:        env("DOCKER_HOST", ""),
		WorkloadsNetwork:  env("WORKLOADS_NETWORK", "axisml-workloads"),
		RuntimeDir:        env("RUNTIME_DIR", "/var/lib/axisml/runtime"),
		TraefikDir:        env("TRAEFIK_DIR", "/var/lib/axisml/traefik"),
		InstallationID:    env("INSTALLATION_ID", "axisml-lite"),
		WorkloadImagePull: envBool("WORKLOAD_IMAGE_PULL", true),

		LogDevelopment: envBool("LOG_DEVELOPMENT", false),
	}
}

// PostgresDSN returns a libpq-style DSN for GORM.
func (c Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DatabaseHost, c.DatabasePort, c.DatabaseUser, c.DatabasePassword,
		c.DatabaseName, c.DatabaseSSLMode,
	)
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
