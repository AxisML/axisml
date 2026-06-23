package server

import (
	"reflect"
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

// fakeFL satisfies the validator.FieldLevel surface that the predicate uses.
// We only need Field().String(), so most methods can panic — they're never
// reached.
type fakeFL struct{ v string }

func (f fakeFL) Top() reflect.Value      { panic("not impl") }
func (f fakeFL) Parent() reflect.Value   { panic("not impl") }
func (f fakeFL) Field() reflect.Value    { return reflect.ValueOf(f.v) }
func (f fakeFL) FieldName() string       { return "" }
func (f fakeFL) StructFieldName() string { return "" }
func (f fakeFL) Param() string           { return "" }
func (f fakeFL) GetTag() string          { return "" }
func (f fakeFL) ExtractType(reflect.Value) (reflect.Value, reflect.Kind, bool) {
	panic("not impl")
}
func (f fakeFL) GetStructFieldOK() (reflect.Value, reflect.Kind, bool) {
	panic("not impl")
}
func (f fakeFL) GetStructFieldOKAdvanced(reflect.Value, string) (reflect.Value, reflect.Kind, bool) {
	panic("not impl")
}
func (f fakeFL) GetStructFieldOK2() (reflect.Value, reflect.Kind, bool, bool) {
	panic("not impl")
}
func (f fakeFL) GetStructFieldOKAdvanced2(reflect.Value, string) (reflect.Value, reflect.Kind, bool, bool) {
	panic("not impl")
}

func TestIsAxisMLName_Predicate(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"alpha-1", true},
		{"BAD", false},
		{"--bad", false},
		{"a/b", false},
		{"a", false},
	}
	for _, tc := range cases {
		if got := isAxisMLName(fakeFL{v: tc.in}); got != tc.want {
			t.Errorf("isAxisMLName(%q) = %v; want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsResourceUnitName_Predicate(t *testing.T) {
	if !isResourceUnitName(fakeFL{v: "ru-1"}) {
		t.Error("expected true for valid resource-unit name")
	}
	if isResourceUnitName(fakeFL{v: "BAD"}) {
		t.Error("expected false for invalid resource-unit name")
	}
}

func TestRegisterValidators_NoErrorOnSecondCall(t *testing.T) {
	require.NoError(t, RegisterValidators())
	require.NoError(t, RegisterValidators(),
		"RegisterValidators must be idempotent — gin uses a process-wide singleton")

	v, ok := binding.Validator.Engine().(*validator.Validate)
	require.True(t, ok)
	if v == nil {
		t.Fatal("nil validator engine")
	}
}
