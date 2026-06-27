package core

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRunnable struct{}

func (stubRunnable) Start(context.Context) error { return nil }

// TestLoopGuard asserts the background loops can be claimed only once, whether
// the host uses Runnables or Serve — so a host that touches both never starts
// the reconcilers twice against the same DB.
func TestLoopGuard(t *testing.T) {
	app := &App{log: logr.Discard(), runnables: []Runnable{stubRunnable{}, stubRunnable{}}}

	// First claim hands the loops to the host.
	got := app.Runnables()
	require.Len(t, got, 2)

	// Second claim via Runnables is refused (returns nil, not the slice again).
	assert.Nil(t, app.Runnables())

	// Serve, the other claim path, is refused too — and fails fast before it
	// binds a listener, so this is safe to call in a unit test.
	err := app.Serve(context.Background())
	assert.True(t, errors.Is(err, ErrLoopsAlreadyStarted), "want ErrLoopsAlreadyStarted, got %v", err)
}

// TestServeClaimsLoops asserts the reverse order: once Serve has claimed the
// loops, Runnables returns nil rather than handing them out a second time.
func TestServeClaimsLoops(t *testing.T) {
	app := &App{log: logr.Discard(), runnables: []Runnable{stubRunnable{}}}

	require.True(t, app.loopsClaimed.CompareAndSwap(false, true), "precondition: loops unclaimed")

	assert.Nil(t, app.Runnables(), "Runnables must not hand out loops already claimed by Serve")
}
