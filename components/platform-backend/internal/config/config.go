// Package config holds the platform-backend runtime configuration. Every field
// has an env-var override; defaults target local development and the in-cluster
// Helm deployment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the runtime configuration for the platform backend.
type Config struct {
	// Database (shared axisml DB, Platform tables are prefix-isolated).
	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseSSLMode  string

	// HTTP listeners.
	APIBindAddress    string
	ProbesBindAddress string

	// Downstream System-layer base URLs (ClusterIP, internal-only).
	ClusterManagerURL string
	ComputeURL        string
	ArtifactsURL      string
	UpstreamTimeout   time.Duration

	// Auth / JWT.
	JWTPrivateKeyPEM  string        // RS256 private key (PEM); generated ephemerally when empty
	JWTKeyID          string        // kid published in JWKS
	LoginTokenTTL     time.Duration // control-plane login JWT lifetime
	WorkspaceTokenTTL time.Duration // data-plane workspace access JWT lifetime (planned)
	PublicTenantScope string        // tenant scope that hosts visibility=public artifacts

	// Bootstrap defaults (consumed by `platform bootstrap`).
	BootstrapUsername string
	BootstrapPassword string
	BootstrapTenant   string
	BootstrapTenantNS string

	// Logging.
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

		APIBindAddress:    env("API_BIND_ADDRESS", ":8080"),
		ProbesBindAddress: env("PROBES_BIND_ADDRESS", ":8082"),

		ClusterManagerURL: env("CLUSTER_MANAGER_URL", "http://axisml-cluster-manager.axisml-system:8080"),
		ComputeURL:        env("COMPUTE_URL", "http://axisml-compute-service.axisml-system:8081"),
		ArtifactsURL:      env("ARTIFACTS_URL", "http://axisml-artifact-hub.axisml-system:8080"),
		UpstreamTimeout:   envDuration("UPSTREAM_TIMEOUT", 30*time.Second),

		JWTPrivateKeyPEM:  env("JWT_PRIVATE_KEY_PEM", ""),
		JWTKeyID:          env("JWT_KEY_ID", "axisml-platform-key-1"),
		LoginTokenTTL:     envDuration("LOGIN_TOKEN_TTL", 12*time.Hour),
		WorkspaceTokenTTL: envDuration("WORKSPACE_ACCESS_JWT_TTL", time.Hour),
		PublicTenantScope: env("PUBLIC_TENANT_SCOPE", "default"),

		BootstrapUsername: env("AXISML_BOOTSTRAP_USERNAME", "admin"),
		BootstrapPassword: env("AXISML_BOOTSTRAP_PASSWORD", "admin"),
		BootstrapTenant:   env("AXISML_BOOTSTRAP_TENANT", "default"),
		BootstrapTenantNS: env("AXISML_BOOTSTRAP_TENANT_NAMESPACE", "axisml-tenant"),

		LogDevelopment: envBool("LOG_DEVELOPMENT", false),
	}

	if cfg.DatabaseHost == "" {
		return Config{}, errors.New("DATABASE_HOST is required")
	}
	return cfg, nil
}

// PostgresDSN returns a libpq-style DSN for GORM / golang-migrate.
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
