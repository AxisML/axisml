package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
)

func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		assert.NotEmpty(t, c.GetString("requestID"))
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"),
		"middleware should echo a request ID header")
}

func TestRequestID_HonoursIncomingHeader(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "from-client")
	r.ServeHTTP(w, req)
	assert.Equal(t, "from-client", w.Header().Get("X-Request-ID"))
}

func TestRecovery_TranslatesPanicTo500(t *testing.T) {
	r := gin.New()
	r.Use(Recovery(logr.Discard()))
	r.GET("/", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestErrorHandler_RendersGinErrorAsProblem(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/", func(c *gin.Context) {
		_ = c.Error(apperrors.New(apperrors.CodeNotFound, "missing"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestErrorHandler_NoOpOnSuccess(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAccessLog_DoesNotChangeStatus(t *testing.T) {
	r := gin.New()
	r.Use(AccessLog(logr.Discard()))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusTeapot) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusTeapot, w.Code)
}
