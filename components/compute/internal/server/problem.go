package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
)

// Problem is RFC7807 application/problem+json.
type Problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail,omitempty"`
	Instance string         `json:"instance,omitempty"`
	Code     apperrors.Code `json:"code"`
	Details  map[string]any `json:"details,omitempty"`
}

// statusFor maps a business error code to an HTTP status.
func statusFor(code apperrors.Code) int {
	switch code {
	case apperrors.CodeValidation:
		return http.StatusBadRequest
	case apperrors.CodeNotFound:
		return http.StatusNotFound
	case apperrors.CodeConflict:
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
		c.AbortWithStatusJSON(status, Problem{
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
	c.AbortWithStatusJSON(http.StatusInternalServerError, Problem{
		Type:     "https://axisml.io/errors/internal_error",
		Title:    "internal error",
		Status:   http.StatusInternalServerError,
		Detail:   err.Error(),
		Instance: c.Request.URL.Path,
		Code:     apperrors.CodeInternal,
	})
}
