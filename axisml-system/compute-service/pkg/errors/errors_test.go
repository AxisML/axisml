package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	e := New(CodeValidation, "boom")
	if e.Code != CodeValidation {
		t.Errorf("Code = %s; want %s", e.Code, CodeValidation)
	}
	if e.Message != "boom" {
		t.Errorf("Message = %s", e.Message)
	}
}

func TestNewf(t *testing.T) {
	e := Newf(CodeNotFound, "missing id=%d", 42)
	if e.Message != "missing id=42" {
		t.Errorf("Message = %s", e.Message)
	}
}

func TestErrorString_NoCause(t *testing.T) {
	e := New(CodeValidation, "boom")
	got := e.Error()
	if !strings.Contains(got, "validation_failed") || !strings.Contains(got, "boom") {
		t.Errorf("Error() = %q", got)
	}
}

func TestErrorString_WithCause(t *testing.T) {
	cause := fmt.Errorf("underlying")
	e := Wrap(CodeInternal, "wrapper", cause)
	got := e.Error()
	if !strings.Contains(got, "internal_error") {
		t.Errorf("missing code in %q", got)
	}
	if !strings.Contains(got, "underlying") {
		t.Errorf("missing cause in %q", got)
	}
}

func TestUnwrap_AndErrorsIs(t *testing.T) {
	cause := fmt.Errorf("root")
	e := Wrap(CodeInternal, "wrapper", cause)
	if !errors.Is(e, cause) {
		t.Error("errors.Is should match wrapped cause")
	}
}

func TestAs(t *testing.T) {
	e := New(CodeNotFound, "x")
	got, ok := As(e)
	if !ok || got != e {
		t.Errorf("As() failed to retrieve typed error")
	}

	wrapped := fmt.Errorf("outer: %w", e)
	got, ok = As(wrapped)
	if !ok || got != e {
		t.Errorf("As() did not unwrap through fmt.Errorf chain")
	}

	if _, ok := As(fmt.Errorf("plain")); ok {
		t.Error("As() returned true for non-business error")
	}
}

func TestWithDetails(t *testing.T) {
	e := New(CodeValidation, "boom").WithDetails(map[string]any{"field": "x"})
	if e.Details["field"] != "x" {
		t.Errorf("Details not attached: %v", e.Details)
	}
}

func TestAllCodes_ContainsEveryCode(t *testing.T) {
	codes := AllCodes()
	want := []Code{
		CodeValidation, CodeNotFound, CodeConflict, CodeImmutableField, CodePrecondition,
		CodeUnauthorized, CodeForbidden, CodeUnavailable, CodeInternal,
		CodeQuotaExceeded,
	}
	if len(codes) != len(want) {
		t.Fatalf("len(AllCodes) = %d; want %d", len(codes), len(want))
	}
	for i, c := range want {
		if codes[i] != string(c) {
			t.Errorf("AllCodes()[%d] = %s; want %s", i, codes[i], c)
		}
	}
}
