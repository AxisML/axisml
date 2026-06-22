// Package errors is Platform's canonical business-error type. The HTTP layer
// (internal/server) maps each Class to an RFC 7807 status and renders the
// stable machine-readable Reason as the problem `code` / `type` URI — the i18n
// contract from backend.md §5.6 (the frontend localises off Reason, never the
// English Message).
package errors

import (
	"errors"
	"fmt"
)

// Class is the coarse error category. It selects the HTTP status; Reason
// carries the fine-grained, stable machine code (e.g. "tenant-suspended").
type Class string

const (
	ClassValidation    Class = "validation_failed"    // 400
	ClassUnauthorized  Class = "unauthenticated"      // 401
	ClassForbidden     Class = "forbidden"            // 403
	ClassNotFound      Class = "not_found"            // 404
	ClassConflict      Class = "conflict"             // 409
	ClassGone          Class = "gone"                 // 410
	ClassUnprocessable Class = "unprocessable_entity" // 422
	ClassUpstream      Class = "upstream_failure"     // 502
	ClassUnavailable   Class = "service_unavailable"  // 503
	ClassInternal      Class = "internal_error"       // 500
)

// E is the canonical business error. Build it with New / Newf / Wrap and refine
// it with WithReason / WithDetails.
type E struct {
	Class   Class
	Reason  string // stable machine code surfaced as problem.code; defaults to Class when empty
	Message string
	Details map[string]any
	cause   error
}

func (e *E) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.code(), e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.code(), e.Message)
}

func (e *E) Unwrap() error { return e.cause }

func (e *E) code() string {
	if e.Reason != "" {
		return e.Reason
	}
	return string(e.Class)
}

// Code returns the stable machine code (Reason, or Class when no Reason set).
func (e *E) Code() string { return e.code() }

// New returns a fresh business error.
func New(class Class, msg string) *E { return &E{Class: class, Message: msg} }

// Newf is sprintf for New.
func Newf(class Class, format string, args ...any) *E {
	return &E{Class: class, Message: fmt.Sprintf(format, args...)}
}

// Wrap attaches a cause.
func Wrap(class Class, msg string, cause error) *E {
	return &E{Class: class, Message: msg, cause: cause}
}

// WithReason sets the stable machine code (e.g. "last-tenant-admin").
func (e *E) WithReason(reason string) *E { e.Reason = reason; return e }

// WithDetails attaches a key/value detail map (rendered into the problem body).
func (e *E) WithDetails(d map[string]any) *E { e.Details = d; return e }

// As is a convenience for errors.As(err, &target).
func As(err error) (*E, bool) {
	var target *E
	ok := errors.As(err, &target)
	return target, ok
}
