package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func runWriteError(err error) (int, Error) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/compute/v1/jobs", nil)
	WriteError(c, err)
	var p Error
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	return w.Code, p
}

func TestWriteError_BusinessCodes(t *testing.T) {
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
		{apperrors.CodeQuotaExceeded, http.StatusUnprocessableEntity},
		{apperrors.CodeInternal, http.StatusInternalServerError},
		{"unknown_code", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(string(tc.code), func(t *testing.T) {
			status, p := runWriteError(apperrors.New(tc.code, "msg"))
			assert.Equal(t, tc.want, status)
			assert.Equal(t, tc.code, p.Code)
			assert.Equal(t, "/compute/v1/jobs", p.Instance)
			assert.True(t, strings.HasPrefix(p.Type, "https://axisml.io/errors/"),
				"problem type should be RFC7807 URL; got %q", p.Type)
		})
	}
}

func TestWriteError_PreservesDetails(t *testing.T) {
	err := apperrors.New(apperrors.CodeValidation, "bad").
		WithDetails(map[string]any{"field": "name"})
	_, p := runWriteError(err)
	require.NotNil(t, p.Details)
	assert.Equal(t, "name", p.Details["field"])
}

func TestWriteError_BindingErrorsMappedTo400(t *testing.T) {
	type body struct {
		Name string `validate:"required"`
	}
	v := validator.New()
	verr := v.Struct(body{}) // missing required field
	require.Error(t, verr, "validator should reject empty required field")

	cases := []struct {
		name string
		err  error
	}{
		{"validator", verr},
		{"json syntax", parseJSONErr(t, "{")},
		{"json type", parseJSONErr(t, `{"x": "string"}`)},
		{"io.EOF", io.EOF},
		{"io.ErrUnexpectedEOF", io.ErrUnexpectedEOF},
		{"wrapped EOF", fmt.Errorf("read: %w", io.EOF)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, p := runWriteError(tc.err)
			assert.Equal(t, http.StatusBadRequest, status)
			assert.Equal(t, apperrors.CodeValidation, p.Code)
		})
	}
}

// parseJSONErr returns the actual error value `encoding/json` produces for
// `body`, so isBindingError sees a real *json.SyntaxError or
// *json.UnmarshalTypeError instead of a zero-valued shell that panics on
// .Error().
func parseJSONErr(t *testing.T, body string) error {
	t.Helper()
	var dst struct{ X int }
	err := json.Unmarshal([]byte(body), &dst)
	require.Error(t, err)
	return err
}

func TestWriteError_PlainErrorBecomes500(t *testing.T) {
	status, p := runWriteError(errors.New("kaboom"))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, apperrors.CodeInternal, p.Code)
	assert.Equal(t, "internal error", p.Title)
}
