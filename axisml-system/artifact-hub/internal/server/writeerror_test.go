package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/axisml/axisml/components/artifact-hub/pkg/errors"
)

func runWriteError(err error) (int, Error) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/artifacts", nil)
	WriteError(c, err)
	var p Error
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	return w.Code, p
}

func TestStatusFor_AllCodes(t *testing.T) {
	cases := []struct {
		code apperrors.Code
		want int
	}{
		{apperrors.CodeValidation, http.StatusBadRequest},
		{apperrors.CodeNotFound, http.StatusNotFound},
		{apperrors.CodeConflict, http.StatusConflict},
		{apperrors.CodePrecondition, http.StatusPreconditionFailed},
		{apperrors.CodeUnauthorized, http.StatusUnauthorized},
		{apperrors.CodeForbidden, http.StatusForbidden},
		{apperrors.CodeUnavailable, http.StatusServiceUnavailable},
		{apperrors.CodeGone, http.StatusGone},
		{apperrors.CodeInternal, http.StatusInternalServerError},
		{"unknown_code", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(string(tc.code), func(t *testing.T) {
			assert.Equal(t, tc.want, statusFor(tc.code))
		})
	}
}

func TestWriteError_BusinessErrorRendersProblem(t *testing.T) {
	status, p := runWriteError(apperrors.New(apperrors.CodeNotFound, "missing"))
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, apperrors.CodeNotFound, p.Code)
	assert.Equal(t, "missing", p.Title)
	assert.True(t, strings.HasPrefix(p.Type, "https://axisml.io/errors/"))
	assert.Equal(t, "/v1/artifacts", p.Instance)
}

func TestWriteError_PreservesDetails(t *testing.T) {
	err := apperrors.New(apperrors.CodeValidation, "bad").
		WithDetails(map[string]any{"field": "name"})
	_, p := runWriteError(err)
	require.NotNil(t, p.Details)
	assert.Equal(t, "name", p.Details["field"])
}

func TestWriteError_BindingErrorBecomes400(t *testing.T) {
	status, p := runWriteError(io.EOF)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, apperrors.CodeValidation, p.Code)
}

func TestWriteError_PlainErrorBecomes500(t *testing.T) {
	status, p := runWriteError(errors.New("kaboom"))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, apperrors.CodeInternal, p.Code)
	assert.Equal(t, "internal error", p.Title)
}
