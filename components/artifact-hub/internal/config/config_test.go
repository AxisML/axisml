package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var envKeys = []string{
	"DATABASE_HOST", "DATABASE_PORT", "DATABASE_NAME", "DATABASE_USER",
	"DATABASE_PASSWORD", "DATABASE_SSLMODE",
	"API_BIND_ADDRESS", "PROBES_BIND_ADDRESS", "METRICS_BIND_ADDRESS",
	"LEADER_ELECT", "LEADER_ELECTION_ID", "LEADER_LEASE_DURATION",
	"LEADER_RENEW_DEADLINE", "LEADER_RETRY_PERIOD",
	"LEADER_RESOURCE_LOCK", "LEADER_NAMESPACE",
	"GC_INTERVAL", "UPLOADING_TTL", "UPLOAD_TOKEN_TTL",
	"OCI_ENDPOINT", "OCI_SCHEME", "OCI_ADMIN_USER", "OCI_ADMIN_PASSWORD",
	"LOG_DEVELOPMENT",
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
	assert.Equal(t, ":8082", cfg.APIBindAddress)
	assert.True(t, cfg.LeaderElect)
	assert.Equal(t, 5*time.Minute, cfg.GCInterval)
	assert.Equal(t, 24*time.Hour, cfg.UploadingTTL)
	assert.Equal(t, "http", cfg.OCIScheme)
}

func TestLoad_Overrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_HOST", "db.example")
	t.Setenv("DATABASE_PORT", "6543")
	t.Setenv("LEADER_ELECT", "false")
	t.Setenv("GC_INTERVAL", "30s")
	t.Setenv("OCI_ENDPOINT", "https://oci.example:443")
	t.Setenv("LOG_DEVELOPMENT", "yes")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "db.example", cfg.DatabaseHost)
	assert.Equal(t, 6543, cfg.DatabasePort)
	assert.False(t, cfg.LeaderElect)
	assert.Equal(t, 30*time.Second, cfg.GCInterval)
	assert.Equal(t, "https://oci.example:443", cfg.OCIEndpoint)
	assert.True(t, cfg.LogDevelopment)
}

func TestLoad_BadDatabasePort(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_PORT", "not-a-port")
	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_PORT")
}

func TestLoad_BadDurationFallsBack(t *testing.T) {
	clearEnv(t)
	t.Setenv("UPLOADING_TTL", "garbage")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 24*time.Hour, cfg.UploadingTTL)
}

func TestLoad_UnknownBoolFallsBack(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEADER_ELECT", "perhaps")
	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.LeaderElect)
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
	assert.Equal(t, "postgres://u:p@h:5432/d?sslmode=disable", cfg.PostgresURL())
}
