package httpx_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/artifact-hub/pkg/httpx"
)

// freeAddr grabs an ephemeral localhost port and releases it, returning the
// addr string. There is a small race between release and re-bind, but the
// readiness poll below tolerates it.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}

func TestServe_GracefulShutdownOnCtxCancel(t *testing.T) {
	addr := freeAddr(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- httpx.Serve(ctx, addr, mux, logr.Discard(), "test")
	}()

	// Wait until the server is actually accepting connections so we are
	// certain to exercise the ctx.Done() shutdown path (not the error path).
	client := &http.Client{Timeout: 200 * time.Millisecond}
	ready := false
	for i := 0; i < 100; i++ {
		resp, err := client.Get("http://" + addr + "/")
		if err == nil {
			_ = resp.Body.Close()
			ready = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.True(t, ready, "server never became ready")

	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err, "graceful shutdown after ctx cancel must return nil")
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after ctx cancel")
	}
}

func TestServe_InvalidAddrReturnsError(t *testing.T) {
	// Port out of the valid 0-65535 range fails at ListenAndServe with a
	// non-ErrServerClosed error, which Serve must surface. ctx is never
	// cancelled, so a nil return would be a bug.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- httpx.Serve(ctx, "127.0.0.1:99999", http.NewServeMux(), logr.Discard(), "bad")
	}()

	select {
	case err := <-done:
		assert.Error(t, err, "invalid listen addr must produce a non-nil error")
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return for an invalid addr")
	}
}
