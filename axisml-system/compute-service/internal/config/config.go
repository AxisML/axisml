package config

import (
	"fmt"

	"github.com/axisml/axisml/pkg/axismlconfig"
)

// Config is compute-service's runtime configuration. compute-service is a
// controller whose only deployment-varying inputs are the database connection
// and log settings, so it embeds only the shared Common sections — there is no
// service-specific config. Listen ports, reconcile cadence, and leader election
// are fixed constants (see consts.go).
type Config struct {
	axismlconfig.Common `mapstructure:",squash"`
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
