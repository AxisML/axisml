package errors

import (
	"errors"
	"fmt"
)

// Code is the discrete business error class. The HTTP layer maps each Code
// to a status / problem type.
type Code string

const (
	CodeValidation     Code = "validation_failed"
	CodeNotFound       Code = "not_found"
	CodeConflict       Code = "conflict"
	CodeImmutableField Code = "immutable-field"
	CodePrecondition   Code = "precondition_failed"
	CodeUnauthorized   Code = "unauthorized"
	CodeForbidden      Code = "forbidden"
	CodeUnavailable    Code = "service_unavailable"
	CodeInternal       Code = "internal_error"
	CodeQuotaExceeded  Code = "quota_exceeded"
)

// AllCodes lists every Code constant in declaration order. It exists so
// schema generators (cmd/openapi-gen) can surface the enum without repeating
// the list and silently going stale when a new code is added here.
func AllCodes() []string {
	return []string{
		string(CodeValidation),
		string(CodeNotFound),
		string(CodeConflict),
		string(CodeImmutableField),
		string(CodePrecondition),
		string(CodeUnauthorized),
		string(CodeForbidden),
		string(CodeUnavailable),
		string(CodeInternal),
		string(CodeQuotaExceeded),
	}
}

// E is the canonical business error. Use New / Wrap to build one.
type E struct {
	Code    Code
	Message string
	Details map[string]any
	cause   error
}

func (e *E) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *E) Unwrap() error { return e.cause }

// New returns a fresh business error.
func New(code Code, msg string) *E {
	return &E{Code: code, Message: msg}
}

// Newf is sprintf for New.
func Newf(code Code, format string, args ...any) *E {
	return &E{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap attaches a cause to a business error.
func Wrap(code Code, msg string, cause error) *E {
	return &E{Code: code, Message: msg, cause: cause}
}

// WithDetails attaches a key/value detail map (rendered into RFC7807).
func (e *E) WithDetails(d map[string]any) *E {
	e.Details = d
	return e
}

// As is a convenience for errors.As(err, &target).
func As(err error) (*E, bool) {
	var target *E
	ok := errors.As(err, &target)
	return target, ok
}
