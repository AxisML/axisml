// Package axisconfig is the one canonical configuration loader shared by every
// AxisML service binary. It layers built-in defaults, an optional YAML file,
// and AXISML_-prefixed environment overrides (with file-mounted secrets) into a
// typed Config struct, and exposes the same struct walk that the configuration
// reference manual is generated from.
//
// A field's behaviour is driven entirely by struct tags so the loader and the
// docs generator stay in sync:
//
//	mapstructure:"<key>"   the YAML/env key segment (use ",squash" to inline an embedded struct)
//	default:"<literal>"    the built-in default (string form; weakly converted)
//	secret:"true"          a secret: resolved from AXISML_<KEY>_FILE and redacted in dumps
//	doc:"<description>"     human description, surfaced in the reference manual
package axismlconfig

import (
	"reflect"
	"strings"
)

// Field describes one leaf configuration key discovered by Walk.
type Field struct {
	Path    string // dotted config path, e.g. "database.host"
	EnvVar  string // AXISML_ override variable, e.g. "AXISML_DATABASE_HOST"
	Default string // raw default from the `default` tag (empty for secrets)
	Secret  bool   // true when the field carries `secret:"true"`
	Doc     string // description from the `doc` tag

	value reflect.Value // settable leaf, used to inject resolved secret files
}

// Walk traverses a pointer-to-struct and returns its leaf configuration fields
// in declaration order. Embedded structs tagged `mapstructure:",squash"` are
// inlined without contributing a path segment; named struct fields nest under
// their key.
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

		if fv.Kind() == reflect.Struct && !isLeaf(fv.Type()) {
			walk(fv, path, out)
			continue
		}

		dotted := strings.Join(path, ".")
		*out = append(*out, Field{
			Path:    dotted,
			EnvVar:  "AXISML_" + strings.ToUpper(strings.ReplaceAll(dotted, ".", "_")),
			Default: sf.Tag.Get("default"),
			Secret:  sf.Tag.Get("secret") == "true",
			Doc:     sf.Tag.Get("doc"),
			value:   fv,
		})
	}
}

// parseTag splits a mapstructure tag into its name and whether ",squash" is set.
func parseTag(tag string) (name string, squash bool) {
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, o := range parts[1:] {
		if o == "squash" {
			squash = true
		}
	}
	return name, squash
}

// isLeaf reports whether a struct type should be treated as a scalar leaf rather
// than recursed into. time.Duration is an int64 (not a struct) so it is already
// a leaf; this guards the rare struct that implements its own decoding.
func isLeaf(t reflect.Type) bool {
	return t.PkgPath() == "time" // time.Time and friends — none nested today
}
