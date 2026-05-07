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

func TestUser_DefaultsToAnonymous(t *testing.T) {
	assert.Equal(t, "anonymous", User(context.Background()))
}

func TestUser_RetrievesStoredValue(t *testing.T) {
	ctx := WithUser(context.Background(), "alice")
	assert.Equal(t, "alice", User(ctx))
}

func TestUser_EmptyStringTreatedAsAnonymous(t *testing.T) {
	ctx := WithUser(context.Background(), "")
	assert.Equal(t, "anonymous", User(ctx))
}

func TestMiddleware_StampsHeaderOnContext(t *testing.T) {
	r := gin.New()
	r.Use(Middleware())
	r.GET("/", func(c *gin.Context) {
		assert.Equal(t, "alice", User(c.Request.Context()))
		assert.Equal(t, "alice", c.GetString(string(HeaderUser)))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderUser, "alice")
	r.ServeHTTP(w, req)
}

func TestMiddleware_DefaultsAnonymousWhenHeaderAbsent(t *testing.T) {
	r := gin.New()
	r.Use(Middleware())
	r.GET("/", func(c *gin.Context) {
		assert.Equal(t, "anonymous", User(c.Request.Context()))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
}
