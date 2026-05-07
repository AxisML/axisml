package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

func TestUser_Default(t *testing.T) {
	assert.Equal(t, "anonymous", User(context.Background()))
}

func TestWithUser_RoundTrip(t *testing.T) {
	ctx := WithUser(context.Background(), "alice")
	assert.Equal(t, "alice", User(ctx))
}

func TestWithUser_EmptyTreatedAsAnonymous(t *testing.T) {
	ctx := WithUser(context.Background(), "")
	assert.Equal(t, "anonymous", User(ctx))
}

func TestMiddleware_ReadsHeader(t *testing.T) {
	r := gin.New()
	r.Use(Middleware())
	r.GET("/", func(c *gin.Context) {
		assert.Equal(t, "alice", User(c.Request.Context()))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderUser, "alice")
	r.ServeHTTP(w, req)
}

func TestMiddleware_DefaultsAnonymous(t *testing.T) {
	r := gin.New()
	r.Use(Middleware())
	r.GET("/", func(c *gin.Context) {
		assert.Equal(t, "anonymous", User(c.Request.Context()))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
}
