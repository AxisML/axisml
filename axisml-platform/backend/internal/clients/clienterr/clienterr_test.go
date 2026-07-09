package clienterr_test

import (
	stderrors "errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clienterr"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

func TestTransport(t *testing.T) {
	cause := stderrors.New("dial tcp: connection refused")
	err := clienterr.Transport("compute", cause)

	e, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ClassUpstream, e.Class)
	assert.Equal(t, "upstream-failure", e.Reason)
	assert.Equal(t, "compute: request failed", e.Message)
	assert.Equal(t, "compute", e.Details["service"])
	// The transport cause is preserved for unwrapping.
	assert.Equal(t, cause, e.Unwrap())
}

func TestFromResponse_NilResponseIsUpstream(t *testing.T) {
	err := clienterr.FromResponse("compute", nil, nil)
	e, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ClassUpstream, e.Class)
	assert.Equal(t, "upstream-failure", e.Reason)
	// Empty body → fallback message.
	assert.Equal(t, "compute: downstream error", e.Message)
	assert.Equal(t, "compute", e.Details["service"])
	assert.Equal(t, http.StatusBadGateway, e.Details["status"])
}

func TestFromResponse_UpstreamStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"500", http.StatusInternalServerError},
		{"503", http.StatusServiceUnavailable},
		{"401", http.StatusUnauthorized},
		{"403", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A parsed Code must NOT leak through as the Reason for upstream failures.
			body := []byte(`{"detail":"kaboom","code":"downstream-code"}`)
			err := clienterr.FromResponse("compute", &http.Response{StatusCode: tt.status}, body)
			e, ok := apperrors.As(err)
			require.True(t, ok)
			assert.Equal(t, apperrors.ClassUpstream, e.Class)
			assert.Equal(t, "upstream-failure", e.Reason)
			assert.Equal(t, "compute: kaboom", e.Message)
			assert.Equal(t, tt.status, e.Details["status"])
		})
	}
}

func TestFromResponse_MappedClasses(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantClass apperrors.Class
	}{
		{"400", http.StatusBadRequest, apperrors.ClassValidation},
		{"404", http.StatusNotFound, apperrors.ClassNotFound},
		{"409", http.StatusConflict, apperrors.ClassConflict},
		{"410", http.StatusGone, apperrors.ClassGone},
		{"422", http.StatusUnprocessableEntity, apperrors.ClassUnprocessable},
		{"default (429) falls back to validation", http.StatusTooManyRequests, apperrors.ClassValidation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"detail":"invalid field","code":"bad-field"}`)
			err := clienterr.FromResponse("compute", &http.Response{StatusCode: tt.status}, body)
			e, ok := apperrors.As(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantClass, e.Class)
			// Parsed problem Code becomes the Reason for 4xx pass-through.
			assert.Equal(t, "bad-field", e.Reason)
			assert.Equal(t, "invalid field", e.Message)
		})
	}
}

func TestFromResponse_TitleFallbackWhenNoDetail(t *testing.T) {
	body := []byte(`{"title":"Not Found"}`)
	err := clienterr.FromResponse("compute", &http.Response{StatusCode: http.StatusNotFound}, body)
	e, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, "Not Found", e.Message)
	// No code in the body → no Reason override → falls back to Class code.
	assert.Empty(t, e.Reason)
	assert.Equal(t, "not_found", e.Code())
}

func TestFromResponse_GarbageBodyFallsBackToMessage(t *testing.T) {
	body := []byte(`<html><body>502 Bad Gateway</body></html>`)
	err := clienterr.FromResponse("compute", &http.Response{StatusCode: http.StatusBadRequest}, body)
	e, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ClassValidation, e.Class)
	// Never surface the raw non-problem body; use the fallback string instead.
	assert.Equal(t, "downstream error", e.Message)
	assert.Empty(t, e.Reason)
}
