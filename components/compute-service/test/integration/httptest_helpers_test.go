//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// doJSON drives the in-memory Gin engine without binding a real port.
// Body, when non-nil, is JSON-marshalled. Decoding into out is opt-in
// (pass nil to skip); empty bodies (e.g. 204) leave out zero-valued.
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

func requireStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Fatalf("status=%d, want %d, body=%s", rr.Code, want, rr.Body.String())
	}
}

// requireClientError asserts the response is a 4xx (validation, conflict,
// not-found, etc.) — used where a test cares that the request was
// rejected without prescribing the exact code.
func requireClientError(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if rr.Code < 400 || rr.Code >= 500 {
		t.Fatalf("expected 4xx, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func randSuffix(t *testing.T) string {
	t.Helper()
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return hex.EncodeToString(b[:])
}

func decodeJSONBody(rr *httptest.ResponseRecorder, out any) error {
	if rr.Body.Len() == 0 {
		return nil
	}
	return json.Unmarshal(rr.Body.Bytes(), out)
}
