package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	e := New(CodeValidation, "boom")
	if e.Code != CodeValidation || e.Message != "boom" {
		t.Errorf("unexpected: %+v", e)
	}
}

func TestNewf(t *testing.T) {
	e := Newf(CodeNotFound, "missing %d", 7)
	if e.Message != "missing 7" {
		t.Errorf("Message = %q", e.Message)
	}
}

func TestErrorString(t *testing.T) {
	t.Run("no cause", func(t *testing.T) {
		got := New(CodeValidation, "boom").Error()
		if !strings.Contains(got, "validation_failed") {
			t.Errorf("missing code in %q", got)
		}
	})
	t.Run("with cause", func(t *testing.T) {
		cause := fmt.Errorf("root")
		got := Wrap(CodeInternal, "wrap", cause).Error()
		if !strings.Contains(got, "root") {
			t.Errorf("missing cause in %q", got)
		}
	})
}

func TestUnwrap_AndIs(t *testing.T) {
	cause := fmt.Errorf("root")
	e := Wrap(CodeInternal, "x", cause)
	if !errors.Is(e, cause) {
		t.Error("errors.Is should match cause")
	}
}

func TestAs_DirectAndWrapped(t *testing.T) {
	e := New(CodeNotFound, "x")
	if got, ok := As(e); !ok || got != e {
		t.Errorf("As() failed for direct error")
	}
	if got, ok := As(fmt.Errorf("wrap: %w", e)); !ok || got != e {
		t.Errorf("As() failed through fmt.Errorf chain")
	}
	if _, ok := As(fmt.Errorf("plain")); ok {
		t.Error("As() returned true for plain error")
	}
}

func TestWithDetails(t *testing.T) {
	e := New(CodeValidation, "x").WithDetails(map[string]any{"k": 1})
	if e.Details["k"] != 1 {
		t.Errorf("Details not set: %v", e.Details)
	}
}

func TestAllCodes_ContainsEveryCode(t *testing.T) {
	codes := AllCodes()
	want := []Code{
		CodeValidation, CodeNotFound, CodeConflict, CodePrecondition,
		CodeUnauthorized, CodeForbidden, CodeUnavailable, CodeInternal,
		CodeGone,
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
