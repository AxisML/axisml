package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
)

func init() { gin.SetMode(gin.TestMode) }

// runRequireUser exercises the RequireUser middleware with the given header
// value (set==false means the header is omitted entirely) and reports the
// resulting recorder plus whether the chain aborted.
func runRequireUser(t *testing.T, value string, set bool) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if set {
		req.Header.Set(srv.HeaderUser, value)
	}
	c.Request = req
	srv.RequireUser(c)
	return w, c.IsAborted()
}

func TestRequireUser(t *testing.T) {
	t.Run("valid user passes", func(t *testing.T) {
		_, aborted := runRequireUser(t, "alice@example.com", true)
		assert.False(t, aborted)
	})

	t.Run("missing header is 401", func(t *testing.T) {
		w, aborted := runRequireUser(t, "", false)
		assert.True(t, aborted)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty header is 401", func(t *testing.T) {
		w, aborted := runRequireUser(t, "", true)
		assert.True(t, aborted)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("whitespace is 400", func(t *testing.T) {
		w, aborted := runRequireUser(t, "bad user", true)
		assert.True(t, aborted)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("control char is 400", func(t *testing.T) {
		w, aborted := runRequireUser(t, "bad\tuser", true)
		assert.True(t, aborted)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("too long is 400", func(t *testing.T) {
		w, aborted := runRequireUser(t, strings.Repeat("a", srv.MaxUserHeaderLen+1), true)
		assert.True(t, aborted)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAbortWithProblem(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	srv.AbortWithProblem(c, http.StatusConflict, "AlreadyExists", "resource already exists", "pool p")
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusConflict, w.Code)

	var got srv.Error
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "about:blank", got.Type)
	assert.Equal(t, http.StatusConflict, got.Status)
	assert.Equal(t, "AlreadyExists", got.Code)
	assert.Equal(t, "resource already exists", got.Title)
	assert.Equal(t, "pool p", got.Detail)
}
