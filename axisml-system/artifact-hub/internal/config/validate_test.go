package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/config"
)

func TestValidate(t *testing.T) {
	t.Run("missing host is rejected", func(t *testing.T) {
		err := config.Config{}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database.host")
	})

	t.Run("host set passes", func(t *testing.T) {
		cfg := config.Config{Common: config.Common{Database: config.Database{Host: "h"}}}
		assert.NoError(t, cfg.Validate())
	})
}

func TestLoad_MalformedFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bad.yaml"
	// Unterminated YAML flow sequence — ReadInConfig surfaces a parse error
	// that is neither "not found" nor a validation failure.
	require.NoError(t, os.WriteFile(path, []byte("foo: [unterminated\n"), 0o600))

	cfg, err := config.Load(config.Options{File: path})
	require.Error(t, err)
	assert.Equal(t, config.Config{}, cfg, "a load failure must return the zero Config")
}
