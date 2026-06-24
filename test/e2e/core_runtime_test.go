//go:build e2e || standard || lite

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// This file holds the form-neutral test runtime shared by every CORE test and
// both harnesses: the identity header, the raw HTTP probe client, status
// classification and the polling helpers. None of it touches Kubernetes, so it
// compiles under the lite build too.

// headerUser is the identity header every System call carries.
const headerUser = "X-Axisml-User"

// setUser is a client-wide request editor that stamps the identity header on
// every System-component call. The generated per-package RequestEditorFn types
// share this underlying signature, so one func value is assignable to all three.
func setUser(user string) func(context.Context, *http.Request) error {
	return func(_ context.Context, req *http.Request) error {
		if user != "" {
			req.Header.Set(headerUser, user)
		}
		return nil
	}
}

// is2xx classifies a status code from a generated response's StatusCode().
func is2xx(code int) bool { return code >= 200 && code < 300 }

// httpClient is a bare client for probing arbitrary in-cluster / in-container
// workloads over their base URL (e.g. GET / against a deployed nginx MLService),
// which have no OpenAPI contract to generate a typed client from. The AxisML
// System components are reached through their typed clients instead.
type httpClient struct {
	baseURL string
	c       *http.Client
}

func newHTTPClient(baseURL string) *httpClient {
	return &httpClient{baseURL: baseURL, c: &http.Client{Timeout: 30 * time.Second}}
}

// resp is the result of a raw HTTP call.
type resp struct {
	status int
	body   []byte
}

// do issues a request. body may be nil.
func (hc *httpClient) do(ctx context.Context, method, path string, body any) (resp, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return resp{}, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, hc.baseURL+path, rdr)
	if err != nil {
		return resp{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	httpResp, err := hc.c.Do(req)
	if err != nil {
		return resp{}, err
	}
	defer func() { _ = httpResp.Body.Close() }()
	b, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp{}, err
	}
	return resp{status: httpResp.StatusCode, body: b}, nil
}

// eventually polls fn until it returns nil or the timeout elapses, using the
// harness poll interval. Mirrors testutil.Eventually but keeps the e2e budgets.
func eventually(t *testing.T, timeout time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		if last = fn(); last == nil {
			return
		}
		time.Sleep(h.config().PollInterval)
	}
	if last == nil {
		last = fmt.Errorf("condition not met")
	}
	t.Fatalf("eventually: timed out after %s: %v", timeout, last)
}

// consistently asserts fn returns nil on every poll across the whole window (the
// inverse of eventually). Used for "must STAY in state X" checks where a single
// immediate read could pass before the system has had a chance to act.
func consistently(t *testing.T, window time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(window)
	for time.Now().Before(deadline) {
		if err := fn(); err != nil {
			t.Fatalf("consistently: condition violated within %s: %v", window, err)
		}
		time.Sleep(h.config().PollInterval)
	}
}

// assertErr is a tiny constructor for poll-condition errors.
func assertErr(format string, args ...any) error { return fmt.Errorf(format, args...) }
