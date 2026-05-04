//go:build envtest

package envtest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// doJSON drives the in-memory Gin engine without binding a real port. The
// request URL is server-relative (e.g. "/api/v1/tenants"); body, when
// non-nil, is JSON-marshalled. Returns the recorded response.
//
// Decoding into out is opt-in (pass nil to skip); when out is non-nil and
// the response body is empty (e.g. 204 No Content), out is left zero-valued.
func doJSON(t *testing.T, ctx context.Context, method, path string, body any, out any) *httptest.ResponseRecorder {
	t.Helper()
	require.NotNil(t, testEngine, "testEngine not bootstrapped (TestMain must call bootstrapHandlers)")

	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req, err := http.NewRequestWithContext(ctx, method, path, &buf)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	testEngine.ServeHTTP(rr, req)
	if out != nil && rr.Body.Len() > 0 {
		if err := json.Unmarshal(rr.Body.Bytes(), out); err != nil {
			t.Fatalf("decode response (status=%d body=%q): %v", rr.Code, rr.Body.String(), err)
		}
	}
	return rr
}

// requireStatus asserts the response status code with a body-quoting
// failure message — without the body, debugging "got 400, want 201" is
// useless.
func requireStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Fatalf("status=%d, want %d, body=%s", rr.Code, want, rr.Body.String())
	}
}

// idAndName extracts {id, name} fields from a JSON object response. Tests
// use this for fixture chaining (create something, capture its UUID, use
// it in the next request).
func idAndName(t *testing.T, rr *httptest.ResponseRecorder) (string, string) {
	t.Helper()
	var v struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &v))
	require.NotEmpty(t, v.ID)
	require.NotEmpty(t, v.Name)
	return v.ID, v.Name
}

// pathf is a tiny URL formatter to keep test bodies readable.
func pathf(format string, args ...any) string { return fmt.Sprintf(format, args...) }
