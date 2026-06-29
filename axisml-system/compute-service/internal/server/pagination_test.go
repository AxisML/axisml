package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
)

func newCtxWithQuery(qs string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+qs, nil)
	return c
}

func TestParsePagination_Defaults(t *testing.T) {
	c := newCtxWithQuery("")
	p, err := ParsePagination(c)
	require.NoError(t, err)
	assert.Equal(t, defaultLimit, p.Limit)
	assert.Equal(t, 0, p.Offset)
}

func TestParsePagination_AppliesValues(t *testing.T) {
	// "NQ" is base64.RawURLEncoding of "5" — the continue token format.
	c := newCtxWithQuery("limit=10&continue=NQ")
	p, err := ParsePagination(c)
	require.NoError(t, err)
	assert.Equal(t, 10, p.Limit)
	assert.Equal(t, 5, p.Offset)
}

func TestParsePagination_RejectsBadContinue(t *testing.T) {
	for _, qs := range []string{"continue=not-base64!!", "continue=" + base64Of("-1")} {
		t.Run(qs, func(t *testing.T) {
			_, err := ParsePagination(newCtxWithQuery(qs))
			require.Error(t, err)
			e, ok := apperrors.As(err)
			require.True(t, ok)
			assert.Equal(t, apperrors.CodeValidation, e.Code)
		})
	}
}

func TestParsePagination_ClampsLimitToMax(t *testing.T) {
	c := newCtxWithQuery("limit=999999")
	p, err := ParsePagination(c)
	require.NoError(t, err)
	assert.Equal(t, maxLimit, p.Limit, "limit should be clamped to maxLimit")
}

func TestParsePagination_RejectsBadLimit(t *testing.T) {
	for _, qs := range []string{"limit=abc", "limit=0", "limit=-1"} {
		t.Run(qs, func(t *testing.T) {
			_, err := ParsePagination(newCtxWithQuery(qs))
			require.Error(t, err)
			e, ok := apperrors.As(err)
			require.True(t, ok)
			assert.Equal(t, apperrors.CodeValidation, e.Code)
		})
	}
}

func TestParsePagination_ContinueZeroAccepted(t *testing.T) {
	c := newCtxWithQuery("continue=" + base64Of("0"))
	p, err := ParsePagination(c)
	require.NoError(t, err)
	assert.Equal(t, 0, p.Offset)
}

func base64Of(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
