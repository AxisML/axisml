// Package configutil owns the standalone distribution's environment-only configuration loader.
// It is intentionally internal so standalone does not expose or version a shared
// configuration helper module as part of its embedding API.
package configutil

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

var envPrefixPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

type Field struct {
	Path    string
	EnvVar  string
	Default string
	Secret  bool
	Doc     string
	value   reflect.Value
}

// Load resolves defaults, environment overrides under envPrefix, and
// file-mounted secrets into the supplied configuration value.
func Load(into any, envPrefix string) error {
	if err := validateEnvPrefix(envPrefix); err != nil {
		return err
	}
	v := viper.New()
	fields := Walk(into, envPrefix)
	for _, field := range fields {
		v.SetDefault(field.Path, field.Default)
	}
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.Unmarshal(into,
		viper.DecodeHook(mapstructure.StringToTimeDurationHookFunc()),
		func(c *mapstructure.DecoderConfig) { c.WeaklyTypedInput = true },
	); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	for _, field := range fields {
		if !field.Secret {
			continue
		}
		if path, ok := os.LookupEnv(field.EnvVar + "_FILE"); ok && path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read secret file %s (%s): %w", field.EnvVar+"_FILE", path, err)
			}
			field.value.SetString(strings.TrimSpace(string(data)))
		}
	}
	if validator, ok := into.(interface{ Validate() error }); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
	}
	return nil
}

// Walk returns configuration leaf metadata in declaration order, using
// envPrefix to derive every field's environment variable name.
func Walk(into any, envPrefix string) []Field {
	var out []Field
	walk(reflect.ValueOf(into).Elem(), nil, envPrefix, &out)
	return out
}

func walk(v reflect.Value, prefix []string, envPrefix string, out *[]Field) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		name, squash := parseTag(sf.Tag.Get("mapstructure"))
		fv := v.Field(i)
		var path []string
		switch {
		case squash:
			path = prefix
		case name == "" || name == "-":
			continue
		default:
			path = append(append([]string{}, prefix...), name)
		}
		if fv.Kind() == reflect.Struct && fv.Type().PkgPath() != "time" {
			walk(fv, path, envPrefix, out)
			continue
		}
		dotted := strings.Join(path, ".")
		*out = append(*out, Field{
			Path: dotted, EnvVar: envPrefix + "_" + strings.ToUpper(strings.ReplaceAll(dotted, ".", "_")),
			Default: sf.Tag.Get("default"), Secret: sf.Tag.Get("secret") == "true", Doc: sf.Tag.Get("doc"), value: fv,
		})
	}
}

func validateEnvPrefix(prefix string) error {
	if !envPrefixPattern.MatchString(prefix) || strings.HasSuffix(prefix, "_") {
		return fmt.Errorf("invalid environment variable prefix %q: must match [A-Z][A-Z0-9_]* and must not end with an underscore", prefix)
	}
	return nil
}

func parseTag(tag string) (string, bool) {
	parts := strings.Split(tag, ",")
	for _, option := range parts[1:] {
		if option == "squash" {
			return parts[0], true
		}
	}
	return parts[0], false
}
