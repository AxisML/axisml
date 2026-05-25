package server

import (
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"

	apperrors "github.com/axisml/axisml/components/artifact-hub/pkg/errors"
)

// RequestID injects a unique X-Request-ID per request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("requestID", id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}

// AccessLog emits a structured log line per request.
func AccessLog(log logr.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"requestID", c.GetString("requestID"),
		)
	}
}

// Recovery catches panics and renders an RFC7807 internal_error.
func Recovery(log logr.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error(nil, "panic recovered",
					"requestID", c.GetString("requestID"),
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				WriteError(c, apperrors.Newf(apperrors.CodeInternal, "internal panic: %v", rec))
			}
		}()
		c.Next()
	}
}

// ErrorHandler converts c.Error() entries into RFC7807 responses. Must run
// last so handlers' Next() returns before the error is rendered.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		WriteError(c, c.Errors.Last().Err)
	}
}
