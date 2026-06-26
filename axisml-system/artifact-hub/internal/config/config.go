package config

import (
	"fmt"

	"github.com/axisml/axisml/pkg/axismlconfig"
)

// Config is artifact-hub's runtime configuration: the shared database and log
// sections plus the OCI registry connection. Listen ports, GC cadence/TTLs, and
// leader election are fixed constants (see consts.go).
type Config struct {
	axismlconfig.Common `mapstructure:",squash"`
	OCI                 OCI `mapstructure:"oci"`
}

// OCI is the zot/OCI registry connection. The scheme is derived from Endpoint
// (a full URL), so there is no separate scheme key.
type OCI struct {
	Endpoint      string `mapstructure:"endpoint" default:"http://axisml-infra-zot.axisml-infra:5000" doc:"OCI registry endpoint (full URL; scheme derived from it)"`
	AdminUser     string `mapstructure:"admin_user" default:"admin" doc:"OCI registry admin username"`
	AdminPassword string `mapstructure:"admin_password" secret:"true" doc:"OCI registry admin password"`
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
