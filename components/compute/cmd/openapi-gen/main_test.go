package main

import (
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mljobv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
)

// TestComponentName covers the naming rules that decide where types land in
// the components/schemas map. The interesting cases are (a) operator API
// packages, which share a v1alpha1 suffix and would collide without the
// per-operator prefix, and (b) Kubernetes-style versioned packages, which
// would all collapse to "V1*" without the parent qualifier.
func TestComponentName(t *testing.T) {
	cases := []struct {
		name string
		in   reflect.Type
		want string
	}{
		{"operator: MLJob double prefix collapses",
			reflect.TypeOf(mljobv1alpha1.MLJobSpec{}), "MLJobSpec"},
		{"operator: MLJob nested type prefixed",
			reflect.TypeOf(mljobv1alpha1.RoleSpec{}), "MLJobRoleSpec"},
		{"corev1 → Corev1*, not V1*",
			reflect.TypeOf(corev1.Toleration{}), "Corev1Toleration"},
		{"metav1 → Metav1* (no V1 collision)",
			reflect.TypeOf(metav1.LabelSelector{}), "Metav1LabelSelector"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := componentName(tc.in); got != tc.want {
				t.Errorf("componentName(%s) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsVersionSegment(t *testing.T) {
	cases := map[string]bool{
		"v1":       true,
		"v1alpha1": true,
		"v2beta1":  true,
		"v":        false, // too short
		"version":  false, // 'e' is not a digit
		"core":     false,
		"":         false,
	}
	for in, want := range cases {
		if got := isVersionSegment(in); got != want {
			t.Errorf("isVersionSegment(%q) = %v, want %v", in, got, want)
		}
	}
}

// TestIsFieldRequired pins the input/response semantics so a future refactor
// can't silently flip them. The bug to guard against: response Views start
// reporting required=false on always-present fields, which breaks typed
// client codegen.
func TestIsFieldRequired(t *testing.T) {
	strType := reflect.TypeOf("")
	ptrStrType := reflect.TypeOf((*string)(nil))

	cases := []struct {
		name    string
		mode    apiMode
		binding string
		opts    []string
		t       reflect.Type
		want    bool
	}{
		{"input: binding=required, no omitempty → required",
			inputMode, "required", nil, strType, true},
		{"input: no binding → not required",
			inputMode, "", nil, strType, false},
		{"input: binding=required but omitempty → not required",
			inputMode, "required", []string{"omitempty"}, strType, false},
		{"input: pointer field is never required",
			inputMode, "required", nil, ptrStrType, false},
		{"response: no omitempty, value type → required",
			responseMode, "", nil, strType, true},
		{"response: omitempty → not required",
			responseMode, "", []string{"omitempty"}, strType, false},
		{"response: pointer → not required",
			responseMode, "", nil, ptrStrType, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFieldRequired(tc.mode, tc.binding, tc.opts, tc.t); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestApplyValidators covers the binding tags that exist in the codebase
// today (axisml_name/_resource_unit, min, gte) plus their min-on-string and
// min-on-number cousins to anchor the dispatch behavior. Missing constraints
// here means client SDKs lose input validation.
func TestApplyValidators(t *testing.T) {
	t.Run("axisml_name on string sets pattern + lengths", func(t *testing.T) {
		s := &schema{Type: "string"}
		applyValidators(s, "required,axisml_name", reflect.TypeOf(""))
		if s.Pattern != axisMLNamePattern {
			t.Errorf("pattern = %q, want %q", s.Pattern, axisMLNamePattern)
		}
		if s.MinLength == nil || *s.MinLength != 3 {
			t.Errorf("minLength = %v, want 3", s.MinLength)
		}
		if s.MaxLength == nil || *s.MaxLength != 40 {
			t.Errorf("maxLength = %v, want 40", s.MaxLength)
		}
	})

	t.Run("axisml_resource_unit on string sets pattern", func(t *testing.T) {
		s := &schema{Type: "string"}
		applyValidators(s, "required,axisml_resource_unit", reflect.TypeOf(""))
		if s.Pattern == "" {
			t.Error("expected pattern to be set")
		}
	})

	t.Run("axisml_name on non-string is a no-op", func(t *testing.T) {
		s := &schema{Type: "integer"}
		applyValidators(s, "axisml_name", reflect.TypeOf(int32(0)))
		if s.Pattern != "" || s.MinLength != nil {
			t.Error("expected no-op on non-string schema")
		}
	})

	t.Run("min=N on slice → minItems", func(t *testing.T) {
		s := &schema{Type: "array"}
		applyValidators(s, "required,min=1", reflect.TypeOf([]int{}))
		if s.MinItems == nil || *s.MinItems != 1 {
			t.Errorf("minItems = %v, want 1", s.MinItems)
		}
		if s.MinLength != nil || s.Minimum != nil {
			t.Error("min on slice should not set minLength/minimum")
		}
	})

	t.Run("min=N on string → minLength", func(t *testing.T) {
		s := &schema{Type: "string"}
		applyValidators(s, "min=2", reflect.TypeOf(""))
		if s.MinLength == nil || *s.MinLength != 2 {
			t.Errorf("minLength = %v, want 2", s.MinLength)
		}
	})

	t.Run("gte=0 on int → minimum", func(t *testing.T) {
		s := &schema{Type: "integer"}
		applyValidators(s, "required,gte=0", reflect.TypeOf(int32(0)))
		if s.Minimum == nil || *s.Minimum != 0 {
			t.Errorf("minimum = %v, want 0", s.Minimum)
		}
	})

	t.Run("len=N on string → exact bounds", func(t *testing.T) {
		s := &schema{Type: "string"}
		applyValidators(s, "len=5", reflect.TypeOf(""))
		if s.MinLength == nil || *s.MinLength != 5 || s.MaxLength == nil || *s.MaxLength != 5 {
			t.Errorf("expected min=max=5, got min=%v max=%v", s.MinLength, s.MaxLength)
		}
	})

	t.Run("unknown rule is ignored, not erroring", func(t *testing.T) {
		s := &schema{Type: "string"}
		applyValidators(s, "required,oneof=a b c", reflect.TypeOf(""))
		// We don't translate oneof yet; just assert no panic and no incorrect
		// constraints leaked through.
		if s.Pattern != "" || s.MinLength != nil {
			t.Error("oneof should be a no-op for now")
		}
	})
}

// TestStructSchemaNullableRef is the regression for the OpenAPI 3.0
// nullable-next-to-$ref bug: pointer-to-named-struct fields must wrap the
// $ref in allOf so the nullability isn't silently dropped by validators.
func TestStructSchemaNullableRef(t *testing.T) {
	type Inner struct {
		X string `json:"x"`
	}
	type Outer struct {
		// Pointer to a named struct: must be wrapped.
		Ptr *Inner `json:"ptr"`
		// Pointer to a primitive: nullable on the primitive is fine in 3.0.
		Str *string `json:"str"`
	}
	g := newGenerator()
	g.register("Outer", Outer{}, inputMode)
	out, ok := g.defs["Outer"]
	if !ok {
		t.Fatal("Outer not registered")
	}
	ptr := out.Properties["ptr"]
	if ptr == nil {
		t.Fatal("missing ptr property")
	}
	if ptr.Ref != "" {
		t.Errorf("ptr should NOT have a top-level $ref (sibling-of-$ref bug), got %q", ptr.Ref)
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

// TestStructSchemaResponseRequired pins the responseMode required-list rule:
// every field that lacks omitempty and isn't a pointer must appear, so that
// generated client types don't make always-present fields optional.
func TestStructSchemaResponseRequired(t *testing.T) {
	type View struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Optional   string  `json:"optional,omitempty"`
		MaybeNil   *string `json:"maybeNil"`
		HiddenFunc string  `json:"-"`
	}
	g := newGenerator()
	g.register("View", View{}, responseMode)
	got := g.defs["View"]
	wantReq := []string{"id", "name"}
	if !reflect.DeepEqual(got.Required, wantReq) {
		t.Errorf("required = %v, want %v", got.Required, wantReq)
	}
	if _, ok := got.Properties["HiddenFunc"]; ok {
		t.Error("json:\"-\" field should be omitted from properties")
	}
}

// TestWellKnownTypes anchors the type table that hides Kubernetes/google
// types' messy reflective shape behind clean OpenAPI primitives.
func TestWellKnownTypes(t *testing.T) {
	cases := []struct {
		name string
		t    reflect.Type
		// validate inspects the returned schema; must call t.Fatal/Error.
		validate func(*testing.T, *schema)
	}{
		{
			"apperrors.Code → enum string",
			reflect.TypeOf(apperrors.CodeValidation),
			func(t *testing.T, s *schema) {
				if s == nil {
					t.Fatal("nil schema")
				}
				if s.Type != "string" {
					t.Errorf("type = %q, want string", s.Type)
				}
				if len(s.Enum) != len(apperrors.AllCodes()) {
					t.Errorf("enum len = %d, want %d", len(s.Enum), len(apperrors.AllCodes()))
				}
				for _, want := range []string{"validation_failed", "not_found", "internal_error"} {
					found := false
					for _, got := range s.Enum {
						if got == want {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("enum missing %q", want)
					}
				}
			},
		},
		{
			"runtime.RawExtension → free-form object",
			reflect.TypeOf(runtime.RawExtension{}),
			func(t *testing.T, s *schema) {
				if s == nil {
					t.Fatal("nil schema")
				}
				if s.Type != "object" {
					t.Errorf("type = %q, want object", s.Type)
				}
				if !strings.Contains(s.Description, "free-form") {
					t.Errorf("description missing 'free-form': %q", s.Description)
				}
			},
		},
		{
			"metav1.Duration → string",
			reflect.TypeOf(metav1.Duration{}),
			func(t *testing.T, s *schema) {
				if s == nil {
					t.Fatal("nil schema")
				}
				if s.Type != "string" {
					t.Errorf("type = %q, want string", s.Type)
				}
			},
		},
		{
			"corev1.Container → not well-known (falls through)",
			reflect.TypeOf(corev1.Container{}),
			func(t *testing.T, s *schema) {
				if s != nil {
					t.Errorf("expected nil (fall-through), got %+v", s)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.validate(t, wellKnown(tc.t))
		})
	}
}

// TestBuildDocumentSchemaIntegrity catches drift in the explicit schema
// inventory: every registered route's $ref must resolve, and operationIds
// must be unique across the whole spec.
func TestBuildDocumentSchemaIntegrity(t *testing.T) {
	doc := buildDocument("test")
	schemas := doc.Components.Schemas
	const prefix = "#/components/schemas/"

	checkRef := func(where, target string) {
		if !strings.HasPrefix(target, prefix) {
			t.Errorf("non-component $ref %q at %s", target, where)
			return
		}
		if _, ok := schemas[strings.TrimPrefix(target, prefix)]; !ok {
			t.Errorf("dangling $ref %q (used at %s)", target, where)
		}
	}

	for name, s := range schemas {
		walkSchema(s, "components."+name, checkRef)
	}
	for path, item := range doc.Paths {
		for method, op := range opsOf(item) {
			if op == nil {
				continue
			}
			where := method + " " + path
			if op.RequestBody != nil {
				for _, mt := range op.RequestBody.Content {
					walkSchema(mt.Schema, where+".requestBody", checkRef)
				}
			}
			for code, r := range op.Responses {
				for _, mt := range r.Content {
					walkSchema(mt.Schema, where+".responses."+code, checkRef)
				}
			}
		}
	}

	seenOpIDs := map[string]string{}
	for path, item := range doc.Paths {
		for method, op := range opsOf(item) {
			if op == nil {
				continue
			}
			if existing, ok := seenOpIDs[op.OperationID]; ok {
				t.Errorf("duplicate operationId %q at %s %s (also %s)",
					op.OperationID, method, path, existing)
				continue
			}
			seenOpIDs[op.OperationID] = method + " " + path
		}
	}
}

// walkSchema recurses through every $ref-bearing branch of s and invokes
// visit for each. Used by the integrity test to verify every reference
// resolves; kept narrow rather than reflective so a future schema field
// (e.g. oneOf) is a deliberate update here, not a silent gap in coverage.
func walkSchema(s *schema, where string, visit func(where, ref string)) {
	if s == nil {
		return
	}
	if s.Ref != "" {
		visit(where, s.Ref)
	}
	walkSchema(s.Items, where+".items", visit)
	walkSchema(s.AdditionalProperties, where+".additionalProperties", visit)
	for i, sub := range s.AllOf {
		walkSchema(sub, where+".allOf["+itoa(i)+"]", visit)
	}
	for k, v := range s.Properties {
		walkSchema(v, where+".properties."+k, visit)
	}
}

func opsOf(p pathItem) map[string]*operation {
	return map[string]*operation{
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

// TestWithErrorsCoversPrecondition is the regression test for the spec/code
// drift that hid 412 responses from clients. Every business code that maps to
// an HTTP status in problem.go should appear in the standard error overlay.
func TestWithErrorsCoversPrecondition(t *testing.T) {
	resps := withErrors(map[string]response{"200": {Description: "ok"}})
	for _, code := range []string{"400", "401", "403", "404", "409", "412", "422", "503", "default"} {
		if _, ok := resps[code]; !ok {
			t.Errorf("withErrors missing %q response", code)
		}
	}
}
