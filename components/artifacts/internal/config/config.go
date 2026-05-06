package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the runtime configuration for the artifacts service.
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

	// Leader election (only the GC worker is leader-gated; HTTP serves on every replica)
	LeaderElect         bool
	LeaderElectionID    string
	LeaderLeaseDuration time.Duration
	LeaderRenewDeadline time.Duration
	LeaderRetryPeriod   time.Duration
	LeaderResourceLock  string
	LeaderResourceNS    string

	// GC worker
	GCInterval     time.Duration
	UploadingTTL   time.Duration
	UploadTokenTTL time.Duration

	// OCI / zot — admin creds passthrough is the MVP simplification
	// per docs/system_design/artifacts.md §8.3 #4. Phase 2 replaces this
	// with a JWT-issuing bearer-token realm.
	OCIEndpoint      string
	OCIScheme        string
	OCIScopePrefix   string
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

		LeaderElect:         envBool("LEADER_ELECT", true),
		LeaderElectionID:    env("LEADER_ELECTION_ID", "axisml-artifacts.axisml.io"),
		LeaderLeaseDuration: envDuration("LEADER_LEASE_DURATION", 15*time.Second),
		LeaderRenewDeadline: envDuration("LEADER_RENEW_DEADLINE", 10*time.Second),
		LeaderRetryPeriod:   envDuration("LEADER_RETRY_PERIOD", 2*time.Second),
		LeaderResourceLock:  env("LEADER_RESOURCE_LOCK", "leases"),
		LeaderResourceNS:    env("LEADER_NAMESPACE", "axisml-system"),

		GCInterval:     envDuration("GC_INTERVAL", 5*time.Minute),
		UploadingTTL:   envDuration("UPLOADING_TTL", 24*time.Hour),
		UploadTokenTTL: envDuration("UPLOAD_TOKEN_TTL", time.Hour),

		OCIEndpoint:      env("OCI_ENDPOINT", "http://axisml-infra-zot.axisml-infra:5000"),
		OCIScheme:        env("OCI_SCHEME", "http"),
		OCIScopePrefix:   env("OCI_SCOPE_PREFIX", "tenants"),
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
