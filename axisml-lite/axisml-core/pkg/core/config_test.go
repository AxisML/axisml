package core_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-lite/axisml-core/pkg/core"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := core.Load()
	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "axisml", cfg.Database.Name)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, "http://zot:5000", cfg.OCI.Endpoint)
	assert.Equal(t, "admin", cfg.OCI.AdminUser)
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("AXISML_DATABASE_HOST", "axisml-database")
	t.Setenv("AXISML_OCI_ENDPOINT", "http://zot.internal:5000")
	t.Setenv("AXISML_LOG_FORMAT", "console")
	cfg, err := core.Load()
	require.NoError(t, err)
	assert.Equal(t, "axisml-database", cfg.Database.Host)
	assert.Equal(t, "http://zot.internal:5000", cfg.OCI.Endpoint)
	assert.Equal(t, "console", cfg.Log.Format)
}

func TestLoad_SecretFromFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/oci-pw"
	require.NoError(t, os.WriteFile(path, []byte("zot-admin\n"), 0o600))
	t.Setenv("AXISML_OCI_ADMIN_PASSWORD_FILE", path)
	cfg, err := core.Load()
	require.NoError(t, err)
	assert.Equal(t, "zot-admin", cfg.OCI.AdminPassword)
}

func TestPostgresDSN(t *testing.T) {
	cfg, err := core.Load()
	require.NoError(t, err)
	dsn := cfg.PostgresDSN()
	for _, want := range []string{"host=localhost", "port=5432", "dbname=axisml", "sslmode=disable"} {
		assert.Contains(t, dsn, want)
	}
}
