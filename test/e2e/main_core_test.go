//go:build e2e || standard || lite

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// TestMain wires the process-wide harness for whichever deployment form was
// selected by build tag (Standard: a real cluster over port-forwards; Lite: one
// axisml-core process), runs a fail-fast readiness gate, then the suite. The
// harness seam (newHarness / Ready / Close) is the only form-specific surface;
// every test below is black-box over the System HTTP contract.
func TestMain(m *testing.M) {
	hh, err := newHarness()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: cannot reach the target deployment: %v\n", err)
		fmt.Fprintf(os.Stderr, "e2e: Standard needs `make cluster-up && make helm-install`; Lite needs axisml-core up.\n")
		os.Exit(1)
	}
	h = hh

	// Gate before m.Run() so a half-installed target fails fast with a clear
	// message instead of every test timing out.
	if err := h.Ready(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: target not ready: %v\n", err)
		h.Close()
		os.Exit(1)
	}

	code := m.Run()
	h.Close()
	os.Exit(code)
}
