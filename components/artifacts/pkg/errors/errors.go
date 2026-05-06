package errors

import (
	"errors"
	"fmt"
)

// Code is the discrete business error class. The HTTP layer maps each Code
// to a status / problem type.
type Code string

const (
	CodeValidation   Code = "validation_failed"
	CodeNotFound     Code = "not_found"
	CodeConflict     Code = "conflict"
	CodePrecondition Code = "precondition_failed"
	CodeUnauthorized Code = "unauthorized"
	CodeForbidden    Code = "forbidden"
	CodeUnavailable  Code = "service_unavailable"
	CodeInternal     Code = "internal_error"
	CodeGone         Code = "gone"
)

// AllCodes lists every Code constant in declaration order.
func AllCodes() []string {
	return []string{
		string(CodeValidation),
		string(CodeNotFound),
		string(CodeConflict),
		string(CodePrecondition),
		string(CodeUnauthorized),
		string(CodeForbidden),
		string(CodeUnavailable),
		string(CodeInternal),
		string(CodeGone),
	}
}

// E is the canonical business error.
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

func New(code Code, msg string) *E {
	return &E{Code: code, Message: msg}
}

func Newf(code Code, format string, args ...any) *E {
	return &E{Code: code, Message: fmt.Sprintf(format, args...)}
}

func Wrap(code Code, msg string, cause error) *E {
	return &E{Code: code, Message: msg, cause: cause}
}

func (e *E) WithDetails(d map[string]any) *E {
	e.Details = d
	return e
}

func As(err error) (*E, bool) {
	var target *E
	ok := errors.As(err, &target)
	return target, ok
}
