package docker

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryAuthFromFile(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.json")
	config := `{
		"auths": {
			"registry.example.com": {
				"auth": "dXNlcjpwYXNz"
			}
		}
	}`
	require.NoError(t, os.WriteFile(configFile, []byte(config), 0o600))

	encoded, err := registryAuthFromFile(
		configFile,
		"registry.example.com/team/model:v1",
	)
	require.NoError(t, err)

	auth := decodeRegistryAuth(t, encoded)
	assert.Equal(t, "user", auth.Username)
	assert.Equal(t, "pass", auth.Password)
	assert.Equal(t, "registry.example.com", auth.ServerAddress)
}

func TestRegistryAuthFromFileMatchesDockerHubLegacyKey(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.json")
	config := `{
		"auths": {
			"https://index.docker.io/v1/": {
				"auth": "aHViLXVzZXI6aHViLXRva2Vu"
			}
		}
	}`
	require.NoError(t, os.WriteFile(configFile, []byte(config), 0o600))

	encoded, err := registryAuthFromFile(configFile, "busybox:latest")
	require.NoError(t, err)

	auth := decodeRegistryAuth(t, encoded)
	assert.Equal(t, "hub-user", auth.Username)
	assert.Equal(t, "hub-token", auth.Password)
	assert.Equal(t, "https://index.docker.io/v1/", auth.ServerAddress)
}

func TestRegistryAuthFromFileReportsConfigErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := registryAuthFromFile(
			filepath.Join(t.TempDir(), "missing.json"),
			"busybox:latest",
		)
		require.ErrorContains(t, err, "open Docker config")
	})

	t.Run("malformed file", func(t *testing.T) {
		configFile := filepath.Join(t.TempDir(), "config.json")
		require.NoError(t, os.WriteFile(configFile, []byte("{"), 0o600))

		_, err := registryAuthFromFile(configFile, "busybox:latest")
		require.ErrorContains(t, err, "parse Docker config")
	})
}

func TestRegistryAuthResolverReloadsConfigForEachPull(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.json")
	writeConfig := func(auth string) {
		t.Helper()
		config := `{"auths":{"registry.example.com":{"auth":"` + auth + `"}}}`
		require.NoError(t, os.WriteFile(configFile, []byte(config), 0o600))
	}
	resolve := newRegistryAuthResolver(configFile)

	writeConfig("dXNlcjpmaXJzdA==")
	first, err := resolve("registry.example.com/model:v1")
	require.NoError(t, err)
	assert.Equal(t, "first", decodeRegistryAuth(t, first).Password)

	writeConfig("dXNlcjpzZWNvbmQ=")
	second, err := resolve("registry.example.com/model:v2")
	require.NoError(t, err)
	assert.Equal(t, "second", decodeRegistryAuth(t, second).Password)
}

func TestNewRegistryAuthResolverEmptyConfigPreservesAnonymousPull(t *testing.T) {
	assert.Nil(t, newRegistryAuthResolver(""))
}

func decodeRegistryAuth(t *testing.T, encoded string) registry.AuthConfig {
	t.Helper()
	raw, err := base64.URLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var auth registry.AuthConfig
	require.NoError(t, json.Unmarshal(raw, &auth))
	return auth
}
