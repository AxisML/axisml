package openapigen

import (
	"reflect"
	"sort"
	"strings"
)

// Mode controls how a top-level Schema's required-list is computed. Inputs
// take their cue from gin `binding` tags; responses treat any field that
// doesn't carry `omitempty` (and isn't a pointer) as always present, which is
// what client codegens want for typed accessors.
//
// Mode propagates only to the root struct's fields. Nested schemas (Kubernetes
// API types, etc.) always use the response/omitempty rule because the same
// component schema may be reached from both an input and an output context,
// and we want one canonical shape per type.
type Mode int

const (
	// ResponseMode marks every non-omitempty, non-pointer field as required.
	ResponseMode Mode = iota
	// InputMode only marks fields with binding=required (and no omitempty).
	InputMode
)

// Options configures a Generator. Both fields are optional.
type Options struct {
	// WellKnown is consulted before the built-in well-known table. Use it to
	// inject service-specific enum types (e.g. an apperrors.Code) without
	// editing this package.
	WellKnown WellKnownFunc

	// PatternRules describes custom validator/v10 tags that translate to
	// regex + length bounds on string fields. Tag matching is exact.
	PatternRules []PatternRule

	// PackageNamer customizes how a Go type's package path becomes the prefix
	// of its component-schema name. Override when the default heuristic
	// (capitalize last segment, join with type name) collides — e.g. multiple
	// versioned packages producing "V1*". Returning ("", false) falls through
	// to the default.
	PackageNamer func(pkgPath string) (prefix string, ok bool)
}

// Generator accumulates component schemas during a single document build.
// Concurrent use is not supported — instantiate one per cmd run.
type Generator struct {
	defs       map[string]*Schema
	inProgress map[reflect.Type]string // cycle-break: type → name being built
	opts       Options
}

// New returns a fresh Generator.
func New(opts Options) *Generator {
	return &Generator{
		defs:       map[string]*Schema{},
		inProgress: map[reflect.Type]string{},
		opts:       opts,
	}
}

// Schemas returns the accumulated component schemas. Callers typically pass
// this to Document.Components.Schemas.
func (g *Generator) Schemas() map[string]*Schema { return g.defs }

// Set inserts a fully-formed Schema under the given name. Useful for list
// envelopes and other shapes that aren't backed by a single Go type.
func (g *Generator) Set(name string, s *Schema) { g.defs[name] = s }

// SetExample attaches a whole-object example to an already-registered schema.
// The value is rendered as the schema's `example` and is what the frontend mock
// codegen reads to build fixtures. Call after Register/Set. Panics on an unknown
// schema name so a typo breaks `make doc-gen` instead of silently dropping the
// example.
func (g *Generator) SetExample(name string, ex any) {
	s := g.defs[name]
	if s == nil {
		panic("openapigen: SetExample for unregistered schema " + name)
	}
	s.Example = ex
}

// ExampleNames returns the schema names that currently carry an example. Used
// by the generator's own tests to guard example coverage.
func (g *Generator) ExampleNames() []string {
	var names []string
	for name, s := range g.defs {
		if s != nil && s.Example != nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// Register derives a Schema for v and stores it under the provided component
// name (so handlers can reference it as #/components/schemas/<name>). For
// named struct types we call StructSchema directly so the slot holds the
// expanded schema rather than a $ref to itself.
func (g *Generator) Register(name string, v any, mode Mode) {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	g.inProgress[t] = name
	var s *Schema
	if ws := g.lookupWellKnown(t); ws != nil {
		s = ws
	} else if t.Kind() == reflect.Struct {
		s = g.StructSchema(t, mode)
	} else {
		s = g.SchemaForType(t)
	}
	g.defs[name] = s
	delete(g.inProgress, t)
}

// SchemaForType is the workhorse. For named struct types it emits a $ref
// (defining the target lazily) so we get a tidy components/schemas section
// instead of inline duplication. Nested types always use response-mode
// required semantics (see Mode docs).
func (g *Generator) SchemaForType(t reflect.Type) *Schema {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if s := g.lookupWellKnown(t); s != nil {
		return s
	}
	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Note: uint64 is rendered as int64 since OpenAPI 3.0 has no native
		// unsigned integer type.
		return &Schema{Type: "integer", Format: intFormat(t)}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number", Format: floatFormat(t)}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 { // []byte → base64 string
			return &Schema{Type: "string", Format: "byte"}
		}
		return &Schema{Type: "array", Items: g.SchemaForType(t.Elem())}
	case reflect.Map:
		return &Schema{Type: "object", AdditionalProperties: g.SchemaForType(t.Elem())}
	case reflect.Struct:
		// Anonymous structs are inlined; named ones become $ref.
		if t.Name() != "" {
			name := g.componentName(t)
			if existing, ok := g.inProgress[t]; ok {
				return Ref(existing)
			}
			if _, defined := g.defs[name]; defined {
				return Ref(name)
			}
			g.inProgress[t] = name
			s := g.StructSchema(t, ResponseMode)
			g.defs[name] = s
			delete(g.inProgress, t)
			return Ref(name)
		}
		return g.StructSchema(t, ResponseMode)
	case reflect.Interface:
		return &Schema{} // free-form
	}
	return &Schema{}
}

// StructSchema builds the object schema for a struct type. Anonymous fields
// without a json tag are flattened (their properties merge into the parent),
// matching encoding/json's behavior.
func (g *Generator) StructSchema(t reflect.Type, mode Mode) *Schema {
	out := &Schema{Type: "object", Properties: map[string]*Schema{}}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get("json") == "" {
			sub := g.StructSchema(f.Type, mode)
			for k, v := range sub.Properties {
				out.Properties[k] = v
			}
			required = append(required, sub.Required...)
			continue
		}
		jsonTag := f.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		name, opts := splitTag(jsonTag)
		if name == "" {
			name = f.Name
		}
		fs := g.SchemaForType(f.Type)
		applyValidators(fs, f.Tag.Get("binding"), f.Type, g.opts.PatternRules)
		desc := f.Tag.Get("desc")
		ptr := isPtr(f.Type)
		if fs.Ref != "" && (ptr || desc != "") {
			// OpenAPI 3.0 ignores sibling keys next to $ref (fixed in 3.1), so a
			// bare `{$ref, nullable/description}` would silently drop them. Wrap
			// in `allOf` to surface the description and/or nullability hint.
			fs = &Schema{Description: desc, Nullable: ptr, AllOf: []*Schema{{Ref: fs.Ref}}}
		} else {
			if desc != "" {
				fs.Description = desc
			}
			if ptr {
				fs.Nullable = true
			}
		}
		out.Properties[name] = fs
		if isFieldRequired(mode, f.Tag.Get("binding"), opts, f.Type) {
			required = append(required, name)
		}
	}
	if len(required) > 0 {
		sort.Strings(required)
		out.Required = required
	}
	return out
}

func splitTag(tag string) (name string, opts []string) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

// isFieldRequired decides whether a struct field appears in the schema's
// required list. See Mode docs for the rationale.
func isFieldRequired(mode Mode, binding string, jsonOpts []string, t reflect.Type) bool {
	for _, o := range jsonOpts {
		if o == "omitempty" {
			return false
		}
	}
	if isPtr(t) {
		return false
	}
	if mode == ResponseMode {
		return true
	}
	for _, b := range strings.Split(binding, ",") {
		if strings.TrimSpace(b) == "required" {
			return true
		}
	}
	return false
}

func isPtr(t reflect.Type) bool { return t.Kind() == reflect.Pointer }

func intFormat(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int32, reflect.Uint32, reflect.Int16, reflect.Uint16, reflect.Int8, reflect.Uint8:
		return "int32"
	case reflect.Int64, reflect.Uint64:
		return "int64"
	}
	return ""
}

func floatFormat(t reflect.Type) string {
	if t.Kind() == reflect.Float32 {
		return "float"
	}
	return "double"
}

func (g *Generator) lookupWellKnown(t reflect.Type) *Schema {
	if g.opts.WellKnown != nil {
		if s := g.opts.WellKnown(t); s != nil {
			return s
		}
	}
	return builtinWellKnown(t)
}

// componentName produces a stable, human-readable name for a Go type. The
// PackageNamer hook lets services collapse colliding versioned packages
// (e.g. mlrun-operator/api/v1alpha1 vs mlservice-operator/api/v1alpha1) to
// per-operator prefixes; otherwise we fall back to capitalising the last
// segment, qualifying Kubernetes-style version segments with their parent.
func (g *Generator) componentName(t reflect.Type) string {
	pkg := t.PkgPath()
	if g.opts.PackageNamer != nil {
		if prefix, ok := g.opts.PackageNamer(pkg); ok {
			if strings.HasPrefix(t.Name(), prefix) {
				return t.Name()
			}
			return prefix + t.Name()
		}
	}
	parts := strings.Split(pkg, "/")
	last := parts[len(parts)-1]
	if last == "" {
		return t.Name()
	}
	if isVersionSegment(last) && len(parts) >= 2 {
		last = parts[len(parts)-2] + last
	}
	return capitalize(last) + t.Name()
}

// isVersionSegment reports whether s looks like a Kubernetes-style API
// version (v1, v1alpha1, v2beta1, …).
func isVersionSegment(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	return s[1] >= '0' && s[1] <= '9'
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ListEnvelope returns a {items: []ref(itemRef), total: int64} schema. Both
// fields are required. Used by the generic list response shape.
func ListEnvelope(itemRef string) *Schema {
	return &Schema{
		Type:     "object",
		Required: []string{"items", "total"},
		Properties: map[string]*Schema{
			"items": {Type: "array", Items: Ref(itemRef)},
			"total": {Type: "integer", Format: "int64"},
		},
	}
}

// PagedListEnvelope returns a paginated list schema
// {items: []ref(itemRef), count: int, total: int64, continueToken: string}.
// items, count and total are always present; continueToken is the Kubernetes
// continuation token for fetching the next page and is omitted on the final
// page. Used by handlers that page via ParsePagination + EncodeContinue.
func PagedListEnvelope(itemRef string) *Schema {
	return &Schema{
		Type:     "object",
		Required: []string{"items", "count", "total"},
		Properties: map[string]*Schema{
			"items":         {Type: "array", Items: Ref(itemRef), Description: "The page of items for the current offset."},
			"count":         {Type: "integer", Description: "Number of items returned in this page (len(items))."},
			"total":         {Type: "integer", Format: "int64", Description: "Total number of matching items across all pages."},
			"continueToken": {Type: "string", Description: "Kubernetes-style continuation token for the next page; empty/absent on the final page."},
		},
	}
}

// WalkSchema recurses through every $ref-bearing branch of s and invokes
// visit for each. Used by per-service integrity tests to verify every
// reference resolves; kept narrow rather than reflective so a future Schema
// field is a deliberate update here, not a silent gap in coverage.
func WalkSchema(s *Schema, where string, visit func(where, ref string)) {
	if s == nil {
		return
	}
	if s.Ref != "" {
		visit(where, s.Ref)
	}
	WalkSchema(s.Items, where+".items", visit)
	WalkSchema(s.AdditionalProperties, where+".additionalProperties", visit)
	for i, sub := range s.AllOf {
		WalkSchema(sub, where+".allOf["+itoa(i)+"]", visit)
	}
	for k, v := range s.Properties {
		WalkSchema(v, where+".properties."+k, visit)
	}
}

// OpsOf returns each non-nil method on a PathItem keyed by lowercase verb.
func OpsOf(p PathItem) map[string]*Operation {
	return map[string]*Operation{
		"get": p.Get, "post": p.Post, "patch": p.Patch, "delete": p.Delete, "put": p.Put,
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
