package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/axisml/axisml/components/artifact-hub/pkg/errors"
)

func init() { gin.SetMode(gin.TestMode) }

func newCtxWithQuery(qs string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+qs, nil)
	return c
}

func TestParsePagination_Defaults(t *testing.T) {
	p, err := ParsePagination(newCtxWithQuery(""))
	require.NoError(t, err)
	assert.Equal(t, defaultLimit, p.Limit)
	assert.Equal(t, 0, p.Offset)
}

func TestParsePagination_AppliesValues(t *testing.T) {
	p, err := ParsePagination(newCtxWithQuery("limit=10&offset=5"))
	require.NoError(t, err)
	assert.Equal(t, 10, p.Limit)
	assert.Equal(t, 5, p.Offset)
}

func TestParsePagination_ClampsLimit(t *testing.T) {
	p, err := ParsePagination(newCtxWithQuery("limit=99999"))
	require.NoError(t, err)
	assert.Equal(t, maxLimit, p.Limit)
}

func TestParsePagination_RejectsBadInputs(t *testing.T) {
	for _, qs := range []string{"limit=abc", "limit=0", "limit=-1", "offset=abc", "offset=-1"} {
		t.Run(qs, func(t *testing.T) {
			_, err := ParsePagination(newCtxWithQuery(qs))
			require.Error(t, err)
			e, ok := apperrors.As(err)
			require.True(t, ok)
			assert.Equal(t, apperrors.CodeValidation, e.Code)
		})
	}
}
