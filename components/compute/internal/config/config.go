package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the runtime configuration for the compute service.
// Every field has an env-var override; defaults are tuned for local
// development and the in-cluster Helm deployment.
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

	// Leader election
	LeaderElect         bool
	LeaderElectionID    string
	LeaderLeaseDuration time.Duration
	LeaderRenewDeadline time.Duration
	LeaderRetryPeriod   time.Duration
	LeaderResourceLock  string
	LeaderResourceNS    string

	// Reconciler tuning
	ReconcileInterval time.Duration

	// Bootstrap defaults (only consumed by `compute bootstrap`)
	BootstrapTenant          string
	BootstrapTenantNamespace string
	BootstrapPool            string
	BootstrapResourceMax     map[string]string

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

		APIBindAddress:     env("API_BIND_ADDRESS", ":8081"),
		ProbesBindAddress:  env("PROBES_BIND_ADDRESS", ":8082"),
		MetricsBindAddress: env("METRICS_BIND_ADDRESS", ":8080"),

		LeaderElect:         envBool("LEADER_ELECT", true),
		LeaderElectionID:    env("LEADER_ELECTION_ID", "axisml-compute.axisml.io"),
		LeaderLeaseDuration: envDuration("LEADER_LEASE_DURATION", 15*time.Second),
		LeaderRenewDeadline: envDuration("LEADER_RENEW_DEADLINE", 10*time.Second),
		LeaderRetryPeriod:   envDuration("LEADER_RETRY_PERIOD", 2*time.Second),
		LeaderResourceLock:  env("LEADER_RESOURCE_LOCK", "leases"),
		LeaderResourceNS:    env("LEADER_NAMESPACE", "axisml-system"),

		ReconcileInterval: envDuration("RECONCILE_INTERVAL", 2*time.Second),

		BootstrapTenant:          env("BOOTSTRAP_TENANT", "default"),
		BootstrapTenantNamespace: env("BOOTSTRAP_TENANT_NAMESPACE", "axisml-default"),
		BootstrapPool:            env("BOOTSTRAP_POOL", "default"),
		BootstrapResourceMax: map[string]string{
			"cpu":    env("BOOTSTRAP_QUOTA_CPU", "8"),
			"memory": env("BOOTSTRAP_QUOTA_MEMORY", "16Gi"),
		},

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
