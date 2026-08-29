package docker

import (
	"fmt"
	"os"

	"github.com/distribution/reference"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types/registry"
)

type registryAuthResolver func(imageRef string) (string, error)

const dockerHubAuthConfigKey = "https://index.docker.io/v1/"

func newRegistryAuthResolver(configFile string) registryAuthResolver {
	if configFile == "" {
		return nil
	}
	return func(imageRef string) (string, error) {
		return registryAuthFromFile(configFile, imageRef)
	}
}

// registryAuthFromFile applies the same registry lookup used by the Docker CLI:
// normalize the image reference, select credentials for its registry, and
// serialize them for the Engine API's X-Registry-Auth header.
func registryAuthFromFile(configFile, imageRef string) (string, error) {
	named, err := reference.ParseNormalizedNamed(imageRef)
	if err != nil {
		return "", fmt.Errorf("parse image reference: %w", err)
	}

	f, err := os.Open(configFile)
	if err != nil {
		return "", fmt.Errorf("open Docker config %q: %w", configFile, err)
	}
	defer func() { _ = f.Close() }()

	cfg, err := cliconfig.LoadFromReader(f)
	if err != nil {
		return "", fmt.Errorf("parse Docker config %q: %w", configFile, err)
	}
	registryDomain := reference.Domain(named)
	configKey := registryDomain
	if registryDomain == "docker.io" || registryDomain == "index.docker.io" {
		configKey = dockerHubAuthConfigKey
	}
	auth, err := cfg.GetAuthConfig(configKey)
	if err != nil {
		return "", fmt.Errorf("get credentials for registry %q: %w", reference.Domain(named), err)
	}

	return registry.EncodeAuthConfig(registry.AuthConfig{
		Username:      auth.Username,
		Password:      auth.Password,
		Auth:          auth.Auth,
		ServerAddress: auth.ServerAddress,
		IdentityToken: auth.IdentityToken,
		RegistryToken: auth.RegistryToken,
	})
}
