package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/components/artifact-hub/internal/config"
	"github.com/axisml/axisml/pkg/axismlconfig"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load(axismlconfig.Options{EnvOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, "http://axisml-infra-zot.axisml-infra:5000", cfg.OCI.Endpoint)
	assert.Equal(t, "admin", cfg.OCI.AdminUser)
}

func TestLoad_EnvOverrideAndWeakTyping(t *testing.T) {
	t.Setenv("AXISML_DATABASE_HOST", "db.example")
	t.Setenv("AXISML_DATABASE_PORT", "6543")
	t.Setenv("AXISML_OCI_ENDPOINT", "https://oci.example:443")
	cfg, err := config.Load(axismlconfig.Options{EnvOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "db.example", cfg.Database.Host)
	assert.Equal(t, 6543, cfg.Database.Port)
	assert.Equal(t, "https://oci.example:443", cfg.OCI.Endpoint)
}

func TestLoad_SecretFromFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/pw"
	require.NoError(t, os.WriteFile(path, []byte("zot-admin\n"), 0o600))
	t.Setenv("AXISML_OCI_ADMIN_PASSWORD_FILE", path)
	cfg, err := config.Load(axismlconfig.Options{EnvOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "zot-admin", cfg.OCI.AdminPassword)
}

func TestPostgresDSN(t *testing.T) {
	cfg := config.Config{Common: axismlconfig.Common{Database: axismlconfig.Database{
		Host: "h", Port: 5432, User: "u", Password: "p", Name: "d", SSLMode: "disable",
	}}}
	dsn := cfg.PostgresDSN()
	for _, want := range []string{"host=h", "port=5432", "user=u", "password=p", "dbname=d", "sslmode=disable"} {
		assert.Contains(t, dsn, want)
	}
}

func TestPostgresURL(t *testing.T) {
	cfg := config.Config{Common: axismlconfig.Common{Database: axismlconfig.Database{
		Host: "h", Port: 5432, User: "u", Password: "p", Name: "d", SSLMode: "disable",
	}}}
	assert.Equal(t, "postgres://u:p@h:5432/d?sslmode=disable", cfg.PostgresURL())
}
