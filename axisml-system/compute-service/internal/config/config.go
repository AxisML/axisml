package config

import (
	"fmt"

	"github.com/axisml/axisml/pkg/axismlconfig"
)

// Config is compute-service's runtime configuration. It embeds the shared
// Common sections (database, log) and adds the optional Prometheus query
// endpoint that backs the per-workload metrics routes. Listen ports, reconcile
// cadence, and leader election are fixed constants (see consts.go).
type Config struct {
	axismlconfig.Common `mapstructure:",squash"`

	Prometheus Prometheus `mapstructure:"prometheus"`
	Workload   Workload   `mapstructure:"workload"`
}

// Workload controls physical workload resource naming.
type Workload struct {
	TenantPrefix bool `mapstructure:"tenant_prefix" default:"false" doc:"Prefix physical workload names with a readable, collision-resistant tenant token"`
}

// Prometheus configures the metrics-query backend. When URL is empty the
// per-workload metrics endpoints report metrics-unavailable instead of serving
// fabricated data.
type Prometheus struct {
	URL string `mapstructure:"url" default:"" doc:"Prometheus query API base URL (e.g. http://kube-prometheus-stack-prometheus.axisml-infra:9090). Empty disables the workload metrics endpoints."`
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
