package openapigen

import (
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestIsVersionSegment(t *testing.T) {
	cases := map[string]bool{
		"v1":       true,
		"v1alpha1": true,
		"v2beta1":  true,
		"v":        false,
		"version":  false,
		"core":     false,
		"":         false,
	}
	for in, want := range cases {
		if got := isVersionSegment(in); got != want {
			t.Errorf("isVersionSegment(%q) = %v, want %v", in, got, want)
		}
	}
}

// TestComponentName covers the package-prefix rules. The PackageNamer hook
// lets per-service generators collapse colliding v1alpha1 packages; the
// default heuristic qualifies Kubernetes-style version segments with their
// parent so corev1 / metav1 don't collide.
func TestComponentName(t *testing.T) {
	g := New(Options{
		PackageNamer: func(pkg string) (string, bool) {
			// Mock the per-operator prefix scheme used by the compute generator.
			switch {
			case strings.Contains(pkg, "/mlrun-operator/api"):
				return "MLRun", true
			case strings.Contains(pkg, "/mlservice-operator/api"):
				return "MLService", true
			}
			return "", false
		},
	})
	cases := []struct {
		name string
		in   reflect.Type
		want string
	}{
		{"corev1 → Corev1*", reflect.TypeOf(corev1.Toleration{}), "Corev1Toleration"},
		{"metav1 → Metav1*", reflect.TypeOf(metav1.LabelSelector{}), "Metav1LabelSelector"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := g.componentName(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsFieldRequired(t *testing.T) {
	strType := reflect.TypeOf("")
	ptrStrType := reflect.TypeOf((*string)(nil))

	cases := []struct {
		name    string
		mode    Mode
		binding string
		opts    []string
		t       reflect.Type
		want    bool
	}{
		{"input: binding=required, no omitempty → required",
			InputMode, "required", nil, strType, true},
		{"input: no binding → not required",
			InputMode, "", nil, strType, false},
		{"input: binding=required but omitempty → not required",
			InputMode, "required", []string{"omitempty"}, strType, false},
		{"input: pointer field is never required",
			InputMode, "required", nil, ptrStrType, false},
		{"response: no omitempty, value type → required",
			ResponseMode, "", nil, strType, true},
		{"response: omitempty → not required",
			ResponseMode, "", []string{"omitempty"}, strType, false},
		{"response: pointer → not required",
			ResponseMode, "", nil, ptrStrType, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFieldRequired(tc.mode, tc.binding, tc.opts, tc.t); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestApplyValidators(t *testing.T) {
	rules := []PatternRule{
		{Tag: "axisml_name", Pattern: "^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])?$", MinLength: 3, MaxLength: 40},
		{Tag: "axisml_version", Pattern: "^[a-zA-Z0-9._-]+$"},
	}

	t.Run("axisml_name on string sets pattern + lengths", func(t *testing.T) {
		s := &Schema{Type: "string"}
		applyValidators(s, "required,axisml_name", reflect.TypeOf(""), rules)
		if s.Pattern == "" {
			t.Error("expected pattern")
		}
		if s.MinLength == nil || *s.MinLength != 3 {
			t.Errorf("minLength = %v, want 3", s.MinLength)
		}
		if s.MaxLength == nil || *s.MaxLength != 40 {
			t.Errorf("maxLength = %v, want 40", s.MaxLength)
		}
	})

	t.Run("axisml_version on string sets pattern only", func(t *testing.T) {
		s := &Schema{Type: "string"}
		applyValidators(s, "required,axisml_version", reflect.TypeOf(""), rules)
		if s.Pattern == "" {
			t.Error("expected pattern")
		}
		if s.MinLength != nil || s.MaxLength != nil {
			t.Errorf("expected no length bounds, got min=%v max=%v", s.MinLength, s.MaxLength)
		}
	})

	t.Run("axisml_name on non-string is a no-op", func(t *testing.T) {
		s := &Schema{Type: "integer"}
		applyValidators(s, "axisml_name", reflect.TypeOf(int32(0)), rules)
		if s.Pattern != "" || s.MinLength != nil {
			t.Error("expected no-op on non-string schema")
		}
	})

	t.Run("min=N on slice → minItems", func(t *testing.T) {
		s := &Schema{Type: "array"}
		applyValidators(s, "required,min=1", reflect.TypeOf([]int{}), nil)
		if s.MinItems == nil || *s.MinItems != 1 {
			t.Errorf("minItems = %v, want 1", s.MinItems)
		}
		if s.MinLength != nil || s.Minimum != nil {
			t.Error("min on slice should not set minLength/minimum")
		}
	})

	t.Run("min=N on string → minLength", func(t *testing.T) {
		s := &Schema{Type: "string"}
		applyValidators(s, "min=2", reflect.TypeOf(""), nil)
		if s.MinLength == nil || *s.MinLength != 2 {
			t.Errorf("minLength = %v, want 2", s.MinLength)
		}
	})

	t.Run("gte=0 on int → minimum", func(t *testing.T) {
		s := &Schema{Type: "integer"}
		applyValidators(s, "required,gte=0", reflect.TypeOf(int32(0)), nil)
		if s.Minimum == nil || *s.Minimum != 0 {
			t.Errorf("minimum = %v, want 0", s.Minimum)
		}
	})

	t.Run("len=N on string → exact bounds", func(t *testing.T) {
		s := &Schema{Type: "string"}
		applyValidators(s, "len=5", reflect.TypeOf(""), nil)
		if s.MinLength == nil || *s.MinLength != 5 || s.MaxLength == nil || *s.MaxLength != 5 {
			t.Errorf("expected min=max=5, got min=%v max=%v", s.MinLength, s.MaxLength)
		}
	})

	t.Run("unknown rule is ignored", func(t *testing.T) {
		s := &Schema{Type: "string"}
		applyValidators(s, "required,oneof=a b c", reflect.TypeOf(""), nil)
		if s.Pattern != "" || s.MinLength != nil {
			t.Error("oneof should be a no-op")
		}
	})
}

// TestStructSchemaNullableRef is the regression for the OpenAPI 3.0
// nullable-next-to-$ref bug: pointer-to-named-struct fields must wrap the
// $ref in allOf so the nullability isn't silently dropped.
func TestStructSchemaNullableRef(t *testing.T) {
	type Inner struct {
		X string `json:"x"`
	}
	type Outer struct {
		Ptr *Inner  `json:"ptr"`
		Str *string `json:"str"`
	}
	g := New(Options{})
	g.Register("Outer", Outer{}, InputMode)
	out, ok := g.defs["Outer"]
	if !ok {
		t.Fatal("Outer not registered")
	}
	ptr := out.Properties["ptr"]
	if ptr == nil {
		t.Fatal("missing ptr property")
	}
	if ptr.Ref != "" {
		t.Errorf("ptr should NOT have a top-level $ref, got %q", ptr.Ref)
	}
	if !ptr.Nullable {
		t.Error("ptr should be nullable")
	}
	if len(ptr.AllOf) != 1 || ptr.AllOf[0].Ref == "" {
		t.Errorf("ptr should be wrapped in allOf with one $ref, got %+v", ptr.AllOf)
	}

	str := out.Properties["str"]
	if str == nil {
		t.Fatal("missing str property")
	}
	if !str.Nullable || str.Type != "string" {
		t.Errorf("str should be nullable string, got %+v", str)
	}
	if len(str.AllOf) != 0 {
		t.Errorf("str should NOT use allOf wrapping for primitive, got %+v", str.AllOf)
	}
}

// TestStructSchemaInlineEmbed pins that anonymous struct fields with an
// empty json name are flattened — both an absent tag and an explicit
// `json:",inline"` — matching encoding/json's promotion of embedded fields.
// A non-empty json name keeps the embed as a nested named property.
func TestStructSchemaInlineEmbed(t *testing.T) {
	type Source struct {
		Claim string `json:"claim"`
	}
	type NamedEmbed struct {
		Extra string `json:"extra"`
	}
	type Volume struct {
		Name       string `json:"name"`
		Source     `json:",inline"`
		NamedEmbed `json:"named"`
	}
	g := New(Options{})
	g.Register("Volume", Volume{}, InputMode)
	out, ok := g.defs["Volume"]
	if !ok {
		t.Fatal("Volume not registered")
	}
	// The `,inline` embed is flattened: claim surfaces at the top level.
	if _, ok := out.Properties["claim"]; !ok {
		t.Errorf("`json:\",inline\"` embed should flatten `claim` to the top level; got %v", keys(out.Properties))
	}
	if _, ok := out.Properties["Source"]; ok {
		t.Error("`,inline` embed must NOT appear as a nested `Source` property")
	}
	// A named embed stays nested under its json name.
	if _, ok := out.Properties["named"]; !ok {
		t.Errorf("named embed should stay nested under `named`; got %v", keys(out.Properties))
	}
	if _, ok := out.Properties["extra"]; ok {
		t.Error("named embed must NOT flatten `extra` to the top level")
	}
}

func keys(m map[string]*Schema) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestStructSchemaCorev1VolumeFlattened is the concrete regression: the real
// corev1.Volume embeds VolumeSource with `json:",inline"`, so its members
// (persistentVolumeClaim, emptyDir, …) must surface at the volume's top level —
// matching how corev1.Volume actually (de)serializes. A nested `VolumeSource`
// property here would mismatch the runtime shape and silently drop the source.
func TestStructSchemaCorev1VolumeFlattened(t *testing.T) {
	g := New(Options{})
	g.Register("Corev1Volume", corev1.Volume{}, InputMode)
	out, ok := g.defs["Corev1Volume"]
	if !ok {
		t.Fatal("Corev1Volume not registered")
	}
	if _, ok := out.Properties["VolumeSource"]; ok {
		t.Error("corev1.Volume must NOT expose a nested `VolumeSource` property")
	}
	for _, want := range []string{"name", "persistentVolumeClaim", "emptyDir", "configMap"} {
		if _, ok := out.Properties[want]; !ok {
			t.Errorf("corev1.Volume must flatten %q to the top level; got %v", want, keys(out.Properties))
		}
	}
}

// TestStructSchemaResponseRequired pins the responseMode required-list rule.
func TestStructSchemaResponseRequired(t *testing.T) {
	type View struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Optional   string  `json:"optional,omitempty"`
		MaybeNil   *string `json:"maybeNil"`
		HiddenFunc string  `json:"-"`
	}
	g := New(Options{})
	g.Register("View", View{}, ResponseMode)
	got := g.defs["View"]
	wantReq := []string{"id", "name"}
	if !reflect.DeepEqual(got.Required, wantReq) {
		t.Errorf("required = %v, want %v", got.Required, wantReq)
	}
	if _, ok := got.Properties["HiddenFunc"]; ok {
		t.Error(`json:"-" field should be omitted from properties`)
	}
}

func TestWellKnownTypes(t *testing.T) {
	cases := []struct {
		name     string
		t        reflect.Type
		validate func(*testing.T, *Schema)
	}{
		{
			"runtime.RawExtension → free-form object",
			reflect.TypeOf(runtime.RawExtension{}),
			func(t *testing.T, s *Schema) {
				if s == nil {
					t.Fatal("nil schema")
				}
				if s.Type != "object" || !strings.Contains(s.Description, "free-form") {
					t.Errorf("got %+v", s)
				}
			},
		},
		{
			"metav1.Duration → string",
			reflect.TypeOf(metav1.Duration{}),
			func(t *testing.T, s *Schema) {
				if s == nil || s.Type != "string" {
					t.Errorf("got %+v", s)
				}
			},
		},
		{
			"corev1.Container → not well-known",
			reflect.TypeOf(corev1.Container{}),
			func(t *testing.T, s *Schema) {
				if s != nil {
					t.Errorf("expected nil fall-through, got %+v", s)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.validate(t, builtinWellKnown(tc.t))
		})
	}
}

// TestUserWellKnownOverridesBuiltin verifies that an Options.WellKnown hook
// fires before the built-in table and can also fall through (return nil).
func TestUserWellKnownOverridesBuiltin(t *testing.T) {
	g := New(Options{
		WellKnown: func(t reflect.Type) *Schema {
			if t == reflect.TypeOf("") {
				return &Schema{Type: "string", Description: "intercepted"}
			}
			return nil
		},
	})
	s := g.SchemaForType(reflect.TypeOf(""))
	if s.Description != "intercepted" {
		t.Errorf("user well-known not applied: %+v", s)
	}
	// Fall-through still hits builtin.
	s = g.SchemaForType(reflect.TypeOf(metav1.Duration{}))
	if s == nil || s.Type != "string" {
		t.Errorf("builtin not consulted on fall-through: %+v", s)
	}
}

// TestListEnvelope sanity-checks the helper used for paginated responses.
func TestListEnvelope(t *testing.T) {
	s := ListEnvelope("FooView")
	if s.Type != "object" {
		t.Errorf("type = %q", s.Type)
	}
	if !reflect.DeepEqual(s.Required, []string{"items", "total"}) {
		t.Errorf("required = %v", s.Required)
	}
	if s.Properties["items"].Items.Ref != "#/components/schemas/FooView" {
		t.Errorf("items ref wrong: %+v", s.Properties["items"].Items)
	}
}

// TestWalkSchema covers every $ref-bearing branch we currently emit.
func TestWalkSchema(t *testing.T) {
	s := &Schema{
		Properties: map[string]*Schema{
			"a": {Ref: "#/components/schemas/A"},
		},
		Items:                &Schema{Ref: "#/components/schemas/I"},
		AdditionalProperties: &Schema{Ref: "#/components/schemas/M"},
		AllOf:                []*Schema{{Ref: "#/components/schemas/L"}},
	}
	seen := map[string]bool{}
	WalkSchema(s, "root", func(_, ref string) { seen[ref] = true })
	for _, want := range []string{
		"#/components/schemas/A",
		"#/components/schemas/I",
		"#/components/schemas/M",
		"#/components/schemas/L",
	} {
		if !seen[want] {
			t.Errorf("walk missed %s", want)
		}
	}
}
