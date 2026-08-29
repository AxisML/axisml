package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
)

// Error is RFC7807 application/problem+json.
type Error struct {
	Type     string         `json:"type" desc:"URI identifying the problem type (https://axisml.io/errors/<code>)."`
	Title    string         `json:"title" desc:"Short, human-readable summary of the problem."`
	Status   int            `json:"status" desc:"HTTP status code for this occurrence of the problem."`
	Detail   string         `json:"detail,omitempty" desc:"Human-readable explanation specific to this occurrence."`
	Instance string         `json:"instance,omitempty" desc:"URI reference identifying the specific occurrence (the request path)."`
	Code     apperrors.Code `json:"code" desc:"Discrete business error class for programmatic handling."`
	Details  map[string]any `json:"details,omitempty" desc:"Optional structured field-level error details."`
}

// statusFor maps a business error code to an HTTP status.
func statusFor(code apperrors.Code) int {
	switch code {
	case apperrors.CodeValidation:
		return http.StatusBadRequest
	case apperrors.CodeNotFound:
		return http.StatusNotFound
	case apperrors.CodeConflict, apperrors.CodeImmutableField:
		return http.StatusConflict
	case apperrors.CodePrecondition:
		return http.StatusPreconditionFailed
	case apperrors.CodeUnauthorized:
		return http.StatusUnauthorized
	case apperrors.CodeForbidden:
		return http.StatusForbidden
	case apperrors.CodeUnavailable:
		return http.StatusServiceUnavailable
	case apperrors.CodeQuotaExceeded:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// WriteError emits a problem+json response.
func WriteError(c *gin.Context, err error) {
	if e, ok := apperrors.As(err); ok {
		status := statusFor(e.Code)
		c.AbortWithStatusJSON(status, Error{
			Type:     "https://axisml.io/errors/" + string(e.Code),
			Title:    e.Message,
			Status:   status,
			Detail:   e.Error(),
			Instance: c.Request.URL.Path,
			Code:     e.Code,
			Details:  e.Details,
		})
		return
	}
	if isBindingError(err) {
		c.AbortWithStatusJSON(http.StatusBadRequest, Error{
			Type:     "https://axisml.io/errors/" + string(apperrors.CodeValidation),
			Title:    "invalid request body",
			Status:   http.StatusBadRequest,
			Detail:   err.Error(),
			Instance: c.Request.URL.Path,
			Code:     apperrors.CodeValidation,
		})
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, Error{
		Type:     "https://axisml.io/errors/internal_error",
		Title:    "internal error",
		Status:   http.StatusInternalServerError,
		Detail:   err.Error(),
		Instance: c.Request.URL.Path,
		Code:     apperrors.CodeInternal,
	})
}

// isBindingError detects errors produced by gin's c.ShouldBindJSON path so
// they map to 400 instead of falling through to the catch-all 500. Covers:
//   - validator.ValidationErrors (custom tag failures: required, gte, etc.)
//   - *json.SyntaxError / *json.UnmarshalTypeError (malformed JSON)
//   - io.EOF / io.ErrUnexpectedEOF (empty or truncated body)
//
// Keep in sync with axisml-system/artifact-hub/internal/server/problem.go —
// duplication is preferred over a shared module to keep each component's
// production go.mod self-contained (see PR #25 discussion).
func isBindingError(err error) bool {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		return true
	}
	var synErr *json.SyntaxError
	if errors.As(err, &synErr) {
		return true
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	return false
}
