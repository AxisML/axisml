package server

import (
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

// RequestID injects an X-Request-ID per request.
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
func AccessLog(log *slog.Logger) gin.HandlerFunc {
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

// Recovery catches panics and renders an internal_error problem.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic recovered",
					"requestID", c.GetString("requestID"),
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				// The panic value is logged above; never render it to the client.
				WriteError(c, apperrors.New(apperrors.ClassInternal, "internal error"))
			}
		}()
		c.Next()
	}
}

// ErrorHandler converts c.Error() entries into problem responses. Runs last so
// handlers/middleware Next() returns before the error is rendered. For 5xx it
// logs the full error chain (which WriteError deliberately omits from the
// response body) keyed by requestID, so the suppressed cause isn't lost.
func ErrorHandler(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		err := c.Errors.Last().Err
		if isServerError(err) {
			log.Error("request failed",
				"requestID", c.GetString("requestID"),
				"path", c.Request.URL.Path,
				"error", err.Error(),
			)
		}
		WriteError(c, err)
	}
}
