// Package standalone holds the single-host System composition root:
// configuration, PostgreSQL-backed Cluster Manager providers, GORM/DB
// coordination, and the assembly that mounts the three System modules (Cluster
// Manager, Compute Service, Artifact Hub) plus the in-process Standalone Runtime
// on one router.
package standalone

import (
	"fmt"

	"github.com/axisml/axisml/axisml-standalone/internal/configutil"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Config is the axisml-standalone process configuration. The Compose
// distribution uses environment-only configuration, so this
// binary reads NO config file. Load uses the AXISML prefix by default; embedded
// callers can select another prefix with LoadWithOptions. Secret keys also
// accept a corresponding <PREFIX>_<KEY>_FILE variable. A single binary fronts
// all three System modules, so the database and OCI settings are shared.
// Everything operational (ports, reconcile cadence, GC, filesystem paths,
// Docker network) is a fixed constant — see consts.go.
type Config struct {
	Common `mapstructure:",squash"`

	OCI      OCI      `mapstructure:"oci"`
	Docker   Docker   `mapstructure:"docker"`
	GPU      GPU      `mapstructure:"gpu"`
	Workload Workload `mapstructure:"workload"`
}

// Docker configures access to the Docker Engine and registries used by the
// Standalone runtime.
type Docker struct {
	ConfigFile string `mapstructure:"config_file" doc:"Path to a Docker config.json used for workload image pull credentials; empty pulls anonymously"`
}

// Workload controls physical workload resource naming.
type Workload struct {
	TenantPrefix         bool   `mapstructure:"tenant_prefix" default:"false" doc:"Prefix physical workload names with a readable, collision-resistant tenant token"`
	SystemReservedCPU    string `mapstructure:"system_reserved_cpu" default:"0" doc:"Host CPU capacity reserved from workload queue admission"`
	SystemReservedMemory string `mapstructure:"system_reserved_memory" default:"0" doc:"Host memory capacity reserved from workload queue admission"`
}

// GPU configures single-host GPU scheduling. Devices names the physical GPU
// indices AxisML may schedule onto as a comma list ("0,1,2"), which turns on
// managed scheduling (pin to a free card, wait when none is free). Empty leaves
// managed scheduling off: services retain Docker's count-based request, while
// MLRun admission has no trustworthy GPU capacity and keeps GPU Runs queued.
type GPU struct {
	Devices string `mapstructure:"devices" doc:"Physical GPU indices available to AxisML (comma list, e.g. 0,1,2); empty keeps GPU Runs queued because capacity is unknown"`
}

// OCI is the artifact registry (zot) connection. The scheme is derived from the
// endpoint URL, so there is no separate scheme key.
type OCI struct {
	Endpoint      string `mapstructure:"endpoint" default:"http://zot:5000" doc:"OCI registry endpoint (full URL; scheme derived from it)"`
	AdminUser     string `mapstructure:"admin_user" default:"admin" doc:"OCI registry admin username"`
	AdminPassword string `mapstructure:"admin_password" secret:"true" doc:"OCI registry admin password"`
}

// DefaultEnvPrefix is the environment variable prefix used by Load and by
// LoadWithOptions when EnvPrefix is empty.
const DefaultEnvPrefix = "AXISML"

// LoadOptions controls how axisml-standalone loads its process configuration.
type LoadOptions struct {
	// EnvPrefix does not include a trailing underscore, for example AXISML or
	// AIOSML. An empty value uses DefaultEnvPrefix.
	EnvPrefix string
}

// Load resolves the configuration from defaults < AXISML_ env < AXISML_<KEY>_FILE
// secret files. axisml-standalone reads no config file (env-only).
func Load() (Config, error) {
	return LoadWithOptions(LoadOptions{EnvPrefix: DefaultEnvPrefix})
}

// LoadWithOptions resolves the configuration from defaults, then environment
// variables under the selected prefix, then the selected prefix's secret
// *_FILE variables. It does not read or merge variables under other prefixes.
func LoadWithOptions(opts LoadOptions) (Config, error) {
	prefix := opts.EnvPrefix
	if prefix == "" {
		prefix = DefaultEnvPrefix
	}
	var c Config
	if err := configutil.Load(&c, prefix); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Validate is invoked by the loader (fail-fast).
func (c Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	for key, value := range map[string]string{
		"workload.system_reserved_cpu":    c.Workload.SystemReservedCPU,
		"workload.system_reserved_memory": c.Workload.SystemReservedMemory,
	} {
		if value == "" {
			continue
		}
		quantity, err := resource.ParseQuantity(value)
		if err != nil || quantity.Sign() < 0 {
			return fmt.Errorf("%s must be a non-negative Kubernetes quantity", key)
		}
	}
	return nil
}
