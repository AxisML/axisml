package config

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"time"
)

// Config is the runtime configuration for the artifact-hub service.
// Every field has an env-var override; defaults target local development
// and the in-cluster Helm deployment.
type Config struct {
	// Database
	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseSSLMode  string

	// HTTP / probes / metrics
	APIBindAddress     string
	ProbesBindAddress  string
	MetricsBindAddress string

	// Leader election (only the GC worker is leader-gated; HTTP serves on every
	// replica). Election is backed by a Postgres session-level advisory lock —
	// no Kubernetes Lease, no client-go, no RBAC.
	LeaderElect       bool
	LeaderLockKey     int64         // pg_advisory_lock key shared by all replicas
	LeaderRetryPeriod time.Duration // acquisition retry + watchdog cadence

	// GC worker
	GCInterval     time.Duration
	UploadingTTL   time.Duration
	UploadTokenTTL time.Duration

	// OCI / zot — admin creds passthrough is the MVP simplification
	// per axisml-system/docs/artifact-hub.md §8.3 #4. Phase 2
	// replaces this with a JWT-issuing bearer-token realm.
	OCIEndpoint      string
	OCIScheme        string
	OCIAdminUser     string
	OCIAdminPassword string

	// Logging
	LogDevelopment bool
}

// Load reads env vars and returns a populated Config.
func Load() (Config, error) {
	port, err := envInt("DATABASE_PORT", 5432)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		DatabaseHost:     env("DATABASE_HOST", "localhost"),
		DatabasePort:     port,
		DatabaseName:     env("DATABASE_NAME", "axisml"),
		DatabaseUser:     env("DATABASE_USER", "axisml"),
		DatabasePassword: env("DATABASE_PASSWORD", "axisml"),
		DatabaseSSLMode:  env("DATABASE_SSLMODE", "disable"),

		APIBindAddress:     env("API_BIND_ADDRESS", ":8082"),
		ProbesBindAddress:  env("PROBES_BIND_ADDRESS", ":8083"),
		MetricsBindAddress: env("METRICS_BIND_ADDRESS", ":8080"),

		LeaderElect:       envBool("LEADER_ELECT", true),
		LeaderLockKey:     envInt64("LEADER_LOCK_KEY", defaultLockKey("axisml-artifact-hub-gc")),
		LeaderRetryPeriod: envDuration("LEADER_RETRY_PERIOD", 2*time.Second),

		GCInterval:     envDuration("GC_INTERVAL", 5*time.Minute),
		UploadingTTL:   envDuration("UPLOADING_TTL", 24*time.Hour),
		UploadTokenTTL: envDuration("UPLOAD_TOKEN_TTL", time.Hour),

		OCIEndpoint:      env("OCI_ENDPOINT", "http://axisml-infra-zot.axisml-infra:5000"),
		OCIScheme:        env("OCI_SCHEME", "http"),
		OCIAdminUser:     env("OCI_ADMIN_USER", "admin"),
		OCIAdminPassword: env("OCI_ADMIN_PASSWORD", ""),

		LogDevelopment: envBool("LOG_DEVELOPMENT", false),
	}

	if cfg.DatabaseHost == "" {
		return Config{}, errors.New("DATABASE_HOST is required")
	}
	return cfg, nil
}

// PostgresDSN returns a libpq-style DSN for GORM / pgx / golang-migrate.
func (c Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DatabaseHost, c.DatabasePort, c.DatabaseUser, c.DatabasePassword,
		c.DatabaseName, c.DatabaseSSLMode,
	)
}

// PostgresURL returns a postgres:// URL form (golang-migrate prefers this).
func (c Config) PostgresURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DatabaseUser, c.DatabasePassword, c.DatabaseHost, c.DatabasePort,
		c.DatabaseName, c.DatabaseSSLMode,
	)
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	switch v {
	case "1", "true", "TRUE", "True", "yes", "YES":
		return true
	case "0", "false", "FALSE", "False", "no", "NO":
		return false
	}
	return def
}

// defaultLockKey derives a stable advisory-lock key from a service name so
// distinct services on the same database don't collide.
func defaultLockKey(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64())
}

func envInt64(key string, def int64) int64 {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func envInt(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("env %s: %w", key, err)
	}
	return n, nil
}

func envDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
