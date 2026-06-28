package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

const problemTypeBase = "https://axisml.io/errors/"

// statusFor maps a business error Class to an HTTP status.
func statusFor(class apperrors.Class) int {
	switch class {
	case apperrors.ClassValidation:
		return http.StatusBadRequest
	case apperrors.ClassUnauthorized:
		return http.StatusUnauthorized
	case apperrors.ClassForbidden:
		return http.StatusForbidden
	case apperrors.ClassNotFound:
		return http.StatusNotFound
	case apperrors.ClassConflict:
		return http.StatusConflict
	case apperrors.ClassGone:
		return http.StatusGone
	case apperrors.ClassUnprocessable:
		return http.StatusUnprocessableEntity
	case apperrors.ClassTooManyReq:
		return http.StatusTooManyRequests
	case apperrors.ClassUpstream:
		return http.StatusBadGateway
	case apperrors.ClassUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// WriteError renders err as an RFC 7807 application/problem+json response using
// the contract Problem shape. The stable machine code (Code / type URI) is the
// i18n contract; Title/Detail are English debug fallbacks only.
//
// The wrapped cause is never rendered: e.Error() expands the internal cause
// chain (DB/driver strings, downstream detail), which must not reach clients.
// Callers log the full chain server-side (see middleware.ErrorHandler / Recovery).
func WriteError(c *gin.Context, err error) {
	if e, ok := apperrors.As(err); ok {
		status := statusFor(e.Class)
		c.Header("Content-Type", "application/problem+json")
		c.AbortWithStatusJSON(status, Problem{
			Type:     URI(problemTypeBase + e.Code()),
			Title:    e.Message,
			Status:   status,
			Detail:   e.Message,
			Instance: c.Request.URL.Path,
			Code:     e.Code(),
		})
		return
	}
	if fieldErrs, ok := bindingFieldErrors(err); ok {
		c.Header("Content-Type", "application/problem+json")
		c.AbortWithStatusJSON(http.StatusBadRequest, Problem{
			Type:     URI(problemTypeBase + string(apperrors.ClassValidation)),
			Title:    "invalid request body",
			Status:   http.StatusBadRequest,
			Detail:   err.Error(),
			Instance: c.Request.URL.Path,
			Code:     string(apperrors.ClassValidation),
			Errors:   fieldErrs,
		})
		return
	}
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(http.StatusInternalServerError, Problem{
		Type:     URI(problemTypeBase + string(apperrors.ClassInternal)),
		Title:    "internal error",
		Status:   http.StatusInternalServerError,
		Detail:   "internal error",
		Instance: c.Request.URL.Path,
		Code:     string(apperrors.ClassInternal),
	})
}

// isServerError reports whether err renders as a 5xx — i.e. it carries an
// internal cause that WriteError suppresses and that callers should log.
func isServerError(err error) bool {
	if e, ok := apperrors.As(err); ok {
		return statusFor(e.Class) >= http.StatusInternalServerError
	}
	if _, ok := bindingFieldErrors(err); ok {
		return false // client-side 400 (bad body); no hidden cause to log
	}
	return true // unexpected raw error rendered as 500
}

// bindingFieldErrors reports whether err came from gin's body binding and, if a
// validator failure, projects per-field messages.
func bindingFieldErrors(err error) ([]ProblemFieldError, bool) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		out := make([]ProblemFieldError, 0, len(ve))
		for _, fe := range ve {
			out = append(out, ProblemFieldError{Field: fe.Field(), Message: fe.Tag()})
		}
		return out, true
	}
	var synErr *json.SyntaxError
	if errors.As(err, &synErr) {
		return nil, true
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return nil, true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, true
	}
	return nil, false
}
