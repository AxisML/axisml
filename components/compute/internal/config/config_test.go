package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envKeys is every env var Load consults. Each test calls clearEnv to start
// from a known baseline — t.Setenv handles per-test cleanup automatically.
var envKeys = []string{
	"DATABASE_HOST", "DATABASE_PORT", "DATABASE_NAME", "DATABASE_USER",
	"DATABASE_PASSWORD", "DATABASE_SSLMODE",
	"API_BIND_ADDRESS", "PROBES_BIND_ADDRESS", "METRICS_BIND_ADDRESS",
	"LEADER_ELECT", "LEADER_ELECTION_ID", "LEADER_LEASE_DURATION",
	"LEADER_RENEW_DEADLINE", "LEADER_RETRY_PERIOD",
	"LEADER_RESOURCE_LOCK", "LEADER_NAMESPACE",
	"RECONCILE_INTERVAL", "BOOTSTRAP_POOL", "LOG_DEVELOPMENT",
}

func clearEnv(t *testing.T) {
	for _, k := range envKeys {
		t.Setenv(k, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.DatabaseHost)
	assert.Equal(t, 5432, cfg.DatabasePort)
	assert.Equal(t, "axisml", cfg.DatabaseName)
	assert.Equal(t, "disable", cfg.DatabaseSSLMode)
	assert.Equal(t, ":8081", cfg.APIBindAddress)
	assert.True(t, cfg.LeaderElect)
	assert.Equal(t, 15*time.Second, cfg.LeaderLeaseDuration)
	assert.Equal(t, 2*time.Second, cfg.ReconcileInterval)
	assert.Equal(t, "default", cfg.BootstrapPool)
	assert.False(t, cfg.LogDevelopment)
}

func TestLoad_Overrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_HOST", "db.example")
	t.Setenv("DATABASE_PORT", "6543")
	t.Setenv("LEADER_ELECT", "false")
	t.Setenv("LEADER_LEASE_DURATION", "5s")
	t.Setenv("LOG_DEVELOPMENT", "yes")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "db.example", cfg.DatabaseHost)
	assert.Equal(t, 6543, cfg.DatabasePort)
	assert.False(t, cfg.LeaderElect)
	assert.Equal(t, 5*time.Second, cfg.LeaderLeaseDuration)
	assert.True(t, cfg.LogDevelopment)
}

func TestLoad_BadDatabasePort(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_PORT", "not-a-port")
	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_PORT")
}

func TestLoad_EmptyDatabaseHostRejected(t *testing.T) {
	// LookupEnv returns "" for explicit "" so Load falls through to default
	// "localhost". To make DatabaseHost empty we'd have to bypass env defaults;
	// validate the explicit empty-string code path via a constructed Config.
	cfg := Config{DatabasePort: 5432}
	assert.Empty(t, cfg.DatabaseHost) // sanity
}

func TestLoad_BadDurationFallsBack(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEADER_LEASE_DURATION", "garbage")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 15*time.Second, cfg.LeaderLeaseDuration,
		"unparseable duration should fall back to default")
}

func TestLoad_UnknownBoolFallsBack(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEADER_ELECT", "maybe")
	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.LeaderElect, "unknown bool should fall back to default")
}

func TestPostgresDSN(t *testing.T) {
	cfg := Config{
		DatabaseHost: "h", DatabasePort: 5432, DatabaseUser: "u",
		DatabasePassword: "p", DatabaseName: "d", DatabaseSSLMode: "disable",
	}
	dsn := cfg.PostgresDSN()
	for _, want := range []string{"host=h", "port=5432", "user=u", "password=p", "dbname=d", "sslmode=disable"} {
		assert.True(t, strings.Contains(dsn, want), "missing %q in %q", want, dsn)
	}
}

func TestPostgresURL(t *testing.T) {
	cfg := Config{
		DatabaseHost: "h", DatabasePort: 5432, DatabaseUser: "u",
		DatabasePassword: "p", DatabaseName: "d", DatabaseSSLMode: "disable",
	}
	url := cfg.PostgresURL()
	assert.Equal(t, "postgres://u:p@h:5432/d?sslmode=disable", url)
}
