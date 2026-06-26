// Package config holds the platform-backend runtime configuration. The shape is
// the shared Common sections (database, log) plus the platform-specific system
// endpoints, cache, auth, and bootstrap-seed sections. Fixed operational values
// live in consts.go; the default tenant and the JWT kid are discovered/derived,
// not configured (see docs/configuration.md).
package config

import (
	"fmt"
	"time"

	"github.com/axisml/axisml/pkg/axismlconfig"
)

// Config is the runtime configuration for the platform backend.
type Config struct {
	axismlconfig.Common `mapstructure:",squash"`

	System    System    `mapstructure:"system"`
	Cache     Cache     `mapstructure:"cache"`
	Auth      Auth      `mapstructure:"auth"`
	Bootstrap Bootstrap `mapstructure:"bootstrap"`
}

// System holds the base endpoints of the System-layer services this backend
// calls (ClusterIP, internal-only).
type System struct {
	ClusterManager string `mapstructure:"cluster_manager" default:"http://axisml-cluster-manager.axisml-system:8080" doc:"cluster-manager endpoint"`
	ComputeService string `mapstructure:"compute_service" default:"http://axisml-compute-service.axisml-system:8080" doc:"compute-service endpoint"`
	ArtifactHub    string `mapstructure:"artifact_hub" default:"http://axisml-artifact-hub.axisml-system:8080" doc:"artifact-hub endpoint"`
}

// Cache is the Redis accelerator for the auth hot path. An empty Addr disables
// caching entirely (every lookup hits PostgreSQL).
type Cache struct {
	Addr     string `mapstructure:"addr" default:"" doc:"Redis address host:port (empty disables the cache)"`
	Password string `mapstructure:"password" secret:"true" doc:"Redis password"`
	DB       int    `mapstructure:"db" default:"0" doc:"Redis logical database"`
}

// Auth configures login-token issuance. The JWKS kid is derived from the key.
type Auth struct {
	LoginTokenTTL    time.Duration `mapstructure:"login_token_ttl" default:"12h" doc:"Login session token lifetime"`
	JWTPrivateKeyPEM string        `mapstructure:"jwt_private_key_pem" secret:"true" doc:"RS256 signing key PEM (ephemeral if unset; JWKS kid derived from the key)"`
}

// Bootstrap seeds the initial system-admin (consumed by `platform bootstrap`).
// The default tenant is NOT configured here — it is owned by the System chart's
// seed.tenant and discovered via cluster-manager.
type Bootstrap struct {
	Username string `mapstructure:"username" default:"admin" doc:"Initial system-admin username"`
	Password string `mapstructure:"password" secret:"true" doc:"Initial system-admin password"`
}

// Load resolves the configuration from defaults < file < AXISML_ env < secret
// files. opts.File carries the --config flag value (empty = auto-discover).
func Load(opts axismlconfig.Options) (Config, error) {
	var c Config
	if err := axismlconfig.Load(&c, opts); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Validate is invoked by the loader (fail-fast).
func (c Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	return nil
}
