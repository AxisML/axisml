package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

const envPrefix = "AXISML"
const defaultFilePath = "/etc/axisml/config.yaml"

type Options struct {
	File    string
	EnvOnly bool
}

type Field struct {
	Path    string
	EnvVar  string
	Default string
	Secret  bool
	Doc     string
	value   reflect.Value
}

func Load(opts Options) (Config, error) {
	var c Config
	if err := load(&c, opts); err != nil {
		return Config{}, err
	}
	return c, nil
}

func load(into any, opts Options) error {
	v := viper.New()
	fields := Walk(into)
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

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.Unmarshal(into,
		viper.DecodeHook(mapstructure.StringToTimeDurationHookFunc()),
		func(c *mapstructure.DecoderConfig) { c.WeaklyTypedInput = true },
	); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	for _, f := range fields {
		if !f.Secret {
			continue
		}
		if path, ok := os.LookupEnv(f.EnvVar + "_FILE"); ok && path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read secret file %s (%s): %w", f.EnvVar+"_FILE", path, err)
			}
			f.value.SetString(strings.TrimSpace(string(data)))
		}
	}
	if validator, ok := into.(interface{ Validate() error }); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
	}
	return nil
}

func (o Options) resolveFile() string {
	if o.EnvOnly {
		return ""
	}
	if o.File != "" {
		return o.File
	}
	if path := os.Getenv(envPrefix + "_CONFIG"); path != "" {
		return path
	}
	for _, path := range []string{defaultFilePath, "config.yaml"} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func Walk(into any) []Field {
	var out []Field
	walk(reflect.ValueOf(into).Elem(), nil, &out)
	return out
}

func walk(v reflect.Value, prefix []string, out *[]Field) {
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
			walk(fv, path, out)
			continue
		}
		dotted := strings.Join(path, ".")
		*out = append(*out, Field{
			Path: dotted, EnvVar: envPrefix + "_" + strings.ToUpper(strings.ReplaceAll(dotted, ".", "_")),
			Default: sf.Tag.Get("default"), Secret: sf.Tag.Get("secret") == "true", Doc: sf.Tag.Get("doc"), value: fv,
		})
	}
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
