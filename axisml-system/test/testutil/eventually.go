// Package testutil provides shared helpers for AxisML's L1 integration test
// suites. It is operator-agnostic: per-operator fixture builders live
// alongside each operator's test code.
package testutil

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultPollInterval is the default poll interval used by EventuallyExists /
// EventuallyGone when an explicit interval isn't supplied. Tests that need
// faster feedback can call Eventually directly with their own interval.
const DefaultPollInterval = 200 * time.Millisecond

// Eventually polls fn until it returns nil or timeout elapses. On timeout it
// fails t with the last error fn returned. Mirrors gomega.Eventually
// ergonomically without the dependency.
func Eventually(t *testing.T, timeout, interval time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		if err := fn(); err == nil {
			return
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(interval)
	}
	if lastErr == nil {
		lastErr = errors.New("(no error captured)")
	}
	t.Fatalf("Eventually timed out after %s: %v", timeout, lastErr)
}

// EventuallyExists waits until c.Get(ctx, key, obj) returns nil. obj is
// populated with the fetched value on success.
func EventuallyExists(t *testing.T, ctx context.Context, c client.Client, key client.ObjectKey, obj client.Object, timeout time.Duration) {
	t.Helper()
	Eventually(t, timeout, DefaultPollInterval, func() error {
		return c.Get(ctx, key, obj)
	})
}

// EventuallyGone waits until c.Get(ctx, key, obj) returns a NotFound error.
func EventuallyGone(t *testing.T, ctx context.Context, c client.Client, key client.ObjectKey, obj client.Object, timeout time.Duration) {
	t.Helper()
	Eventually(t, timeout, DefaultPollInterval, func() error {
		err := c.Get(ctx, key, obj)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("%s/%s still present", key.Namespace, key.Name)
	})
}
