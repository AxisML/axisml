package config

import (
	"fmt"
)

// Config is artifact-hub's runtime configuration: the shared database and log
// sections plus the OCI registry and S3 object-store connections. Listen ports,
// GC cadence/TTLs, and leader election are fixed constants (see consts.go).
type Config struct {
	Common `mapstructure:",squash"`
	OCI    OCI `mapstructure:"oci"`
	S3     S3  `mapstructure:"s3"`
}

// OCI is the zot/OCI registry connection. The scheme is derived from Endpoint
// (a full URL), so there is no separate scheme key.
type OCI struct {
	Endpoint      string `mapstructure:"endpoint" default:"http://axisml-infra-zot.axisml-infra:5000" doc:"OCI registry endpoint (full URL; scheme derived from it)"`
	AdminUser     string `mapstructure:"admin_user" default:"admin" doc:"OCI registry admin username"`
	AdminPassword string `mapstructure:"admin_password" secret:"true" doc:"OCI registry admin password"`
}

// S3 is the RustFS/S3 object-store connection backing the dataset kind. When
// Endpoint is empty the dataset handler runs without a backend: it records the
// client-supplied digest unverified (single-host dev form). When Endpoint is
// set, complete verifies the digest against the stored artifact-manifest.json.
type S3 struct {
	Endpoint  string `mapstructure:"endpoint" default:"" doc:"S3/RustFS endpoint (host:port or full URL; scheme derived from it). Empty disables dataset digest verification."`
	AccessKey string `mapstructure:"access_key" default:"" doc:"S3/RustFS access key"`
	SecretKey string `mapstructure:"secret_key" secret:"true" doc:"S3/RustFS secret key"`
	Bucket    string `mapstructure:"bucket" default:"axisml-artifact-hub" doc:"S3 bucket datasets are stored in"`
}

// Validate is invoked by the loader (fail-fast).
func (c Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	return nil
}
