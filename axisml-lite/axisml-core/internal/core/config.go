// Package core holds the axisml-core composition root: configuration, the
// config-backed Cluster Manager providers (single default ResourcePool +
// Tenant), the GORM/DB coordination and the assembly that mounts the three
// System modules (Cluster Manager, Compute Service, Artifact Hub) plus the
// in-process Standalone Runtime on one router.
package core

import (
	"fmt"

	"github.com/axisml/axisml/pkg/axismlconfig"
)

// Config is the axisml-core process configuration. Lite runs under Docker
// Compose, where the environment is the idiomatic config mechanism, so this
// binary reads NO config file — every key is sourced from AXISML_ env (secrets
// also accept AXISML_<KEY>_FILE). A single binary fronts all three System
// modules, so the database and OCI settings are shared. Everything operational
// (ports, reconcile cadence, GC, filesystem paths, Docker network) is a fixed
// constant — see consts.go.
type Config struct {
	axismlconfig.Common `mapstructure:",squash"`

	OCI OCI `mapstructure:"oci"`
}

// OCI is the artifact registry (zot) connection. The scheme is derived from the
// endpoint URL, so there is no separate scheme key.
type OCI struct {
	Endpoint      string `mapstructure:"endpoint" default:"http://zot:5000" doc:"OCI registry endpoint (full URL; scheme derived from it)"`
	AdminUser     string `mapstructure:"admin_user" default:"admin" doc:"OCI registry admin username"`
	AdminPassword string `mapstructure:"admin_password" secret:"true" doc:"OCI registry admin password"`
}

// Load resolves the configuration from defaults < AXISML_ env < AXISML_<KEY>_FILE
// secret files. axisml-core reads no config file (env-only).
func Load() (Config, error) {
	var c Config
	if err := axismlconfig.Load(&c, axismlconfig.Options{EnvOnly: true}); err != nil {
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
