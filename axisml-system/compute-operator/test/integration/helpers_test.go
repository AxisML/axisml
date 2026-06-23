//go:build integration

package integration_test

import "time"

// Shared timeout for `testutil.Eventually` polls across MLRun / MLService
// integration tests. Tuned for envtest reconciler latency on a busy CI
// runner; bump if tests start flaking.
const testWaitTimeout = 30 * time.Second
