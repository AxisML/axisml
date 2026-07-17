package axismlconfig

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// EnvPrefix is the mandatory prefix for every AxisML configuration variable.
const EnvPrefix = "AXISML"

// Options controls how a service resolves its config file.
type Options struct {
	// File is an explicit path (e.g. from a --config flag). Highest precedence.
	File string
	// EnvOnly skips the file layer entirely.
	EnvOnly bool
}

// DefaultFilePath is the in-container config file location.
const DefaultFilePath = "/etc/axisml/config.yaml"

func (o Options) resolveFile() string {
	if o.EnvOnly {
		return ""
	}
	if o.File != "" {
		return o.File
	}
	if p := os.Getenv(EnvPrefix + "_CONFIG"); p != "" {
		return p
	}
	for _, p := range []string{DefaultFilePath, "config.yaml"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// Load populates a pointer-to-Config from defaults < file < AXISML_ env <
// AXISML_<KEY>_FILE secret files. A missing config file is not an error. If the
// target implements Validate() error it is called last (fail-fast).
func Load(into any, opts Options) error {
	v := viper.New()
	fields := Walk(into)

	// Register every leaf so viper's AutomaticEnv reliably binds AXISML_ keys.
	for _, f := range fields {
		v.SetDefault(f.Path, f.Default)
	}

	v.SetConfigType("yaml")
	if path := opts.resolveFile(); path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			var notFound viper.ConfigFileNotFoundError
			if !errors.As(err, &notFound) && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("read config %s: %w", path, err)
			}
		}
	}

	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.Unmarshal(into,
		viper.DecodeHook(mapstructure.StringToTimeDurationHookFunc()),
		func(c *mapstructure.DecoderConfig) { c.WeaklyTypedInput = true },
	); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	// Secret files outrank file and plain env: applied after unmarshal.
	for _, f := range fields {
		if !f.Secret {
			continue
		}
		if p, ok := os.LookupEnv(f.EnvVar + "_FILE"); ok && p != "" {
			b, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("read secret file %s (%s): %w", f.EnvVar+"_FILE", p, err)
			}
			f.value.SetString(strings.TrimSpace(string(b)))
		}
	}

	if val, ok := into.(interface{ Validate() error }); ok {
		if err := val.Validate(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
	}
	return nil
}
