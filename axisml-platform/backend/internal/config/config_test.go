package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/config"
	"github.com/axisml/axisml/pkg/axismlconfig"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load(axismlconfig.Options{EnvOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "http://axisml-cluster-manager.axisml-system:8080", cfg.System.ClusterManager)
	assert.Equal(t, "http://axisml-compute-service.axisml-system:8080", cfg.System.ComputeService)
	assert.Equal(t, "http://axisml-artifact-hub.axisml-system:8080", cfg.System.ArtifactHub)
	assert.Equal(t, "", cfg.Cache.Addr)
	assert.Equal(t, 0, cfg.Cache.DB)
	assert.Equal(t, "admin", cfg.Bootstrap.Username)
	// Duration default decoded via the loader's StringToTimeDurationHookFunc.
	assert.Equal(t, 12*time.Hour, cfg.Auth.LoginTokenTTL)
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("AXISML_DATABASE_HOST", "db.example")
	t.Setenv("AXISML_SYSTEM_COMPUTE_SERVICE", "http://compute:9000")
	t.Setenv("AXISML_CACHE_DB", "3")
	t.Setenv("AXISML_AUTH_LOGIN_TOKEN_TTL", "1h")
	cfg, err := config.Load(axismlconfig.Options{EnvOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "db.example", cfg.Database.Host)
	assert.Equal(t, "http://compute:9000", cfg.System.ComputeService)
	assert.Equal(t, 3, cfg.Cache.DB)
	assert.Equal(t, time.Hour, cfg.Auth.LoginTokenTTL)
}

func TestLoad_SecretsFromFileAndEnv(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/dbpw"
	require.NoError(t, os.WriteFile(path, []byte("filepw\n"), 0o600))
	t.Setenv("AXISML_DATABASE_PASSWORD_FILE", path)
	t.Setenv("AXISML_BOOTSTRAP_PASSWORD", "envpw")
	cfg, err := config.Load(axismlconfig.Options{EnvOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "filepw", cfg.Database.Password) // _FILE trimmed
	assert.Equal(t, "envpw", cfg.Bootstrap.Password) // plain env
}

func TestPostgresDSN(t *testing.T) {
	cfg := config.Config{Common: axismlconfig.Common{Database: axismlconfig.Database{
		Host: "h", Port: 5432, User: "u", Password: "p", Name: "d", SSLMode: "disable",
	}}}
	assert.Contains(t, cfg.PostgresDSN(), "host=h")
	assert.Contains(t, cfg.PostgresDSN(), "dbname=d")
}
