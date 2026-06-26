//go:build e2e || standard || lite

package e2e

import (
	"context"
	"fmt"
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

// assertErr is a tiny constructor for poll-condition errors.
func assertErr(format string, args ...any) error { return fmt.Errorf(format, args...) }
