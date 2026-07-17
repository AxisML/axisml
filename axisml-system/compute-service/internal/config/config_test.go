package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load(config.Options{EnvOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "axisml", cfg.Database.Name)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
}

func TestLoad_EnvOverrideAndWeakTyping(t *testing.T) {
	t.Setenv("AXISML_DATABASE_HOST", "db.example")
	t.Setenv("AXISML_DATABASE_PORT", "6543")
	t.Setenv("AXISML_LOG_LEVEL", "debug")
	t.Setenv("AXISML_WORKLOAD_TENANT_PREFIX", "true")
	cfg, err := config.Load(config.Options{EnvOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "db.example", cfg.Database.Host)
	assert.Equal(t, 6543, cfg.Database.Port)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.True(t, cfg.Workload.TenantPrefix)
}

func TestLoad_SecretFromFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/pw"
	require.NoError(t, os.WriteFile(path, []byte("s3cret\n"), 0o600))
	t.Setenv("AXISML_DATABASE_PASSWORD_FILE", path)
	cfg, err := config.Load(config.Options{EnvOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "s3cret", cfg.Database.Password)
}

func TestPostgresDSN(t *testing.T) {
	cfg := config.Config{Common: config.Common{Database: config.Database{
		Host: "h", Port: 5432, User: "u", Password: "p", Name: "d", SSLMode: "disable",
	}}}
	dsn := cfg.PostgresDSN()
	for _, want := range []string{"host=h", "port=5432", "user=u", "password=p", "dbname=d", "sslmode=disable"} {
		assert.Contains(t, dsn, want)
	}
}

func TestPostgresURL(t *testing.T) {
	cfg := config.Config{Common: config.Common{Database: config.Database{
		Host: "h", Port: 5432, User: "u", Password: "p", Name: "d", SSLMode: "disable",
	}}}
	assert.Equal(t, "postgres://u:p@h:5432/d?sslmode=disable", cfg.PostgresURL())
}
