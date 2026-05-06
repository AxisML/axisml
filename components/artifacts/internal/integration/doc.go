// Package integration is the home of integration tests behind the
// `integration` build tag. They boot a real PostgreSQL via testcontainers +
// an httptest server stubbing zot's OCI v2 endpoints and exercise the full
// HTTP API state machine end-to-end.
//
// Run with `make artifacts-integration` from the repo root, or
// `go test -tags=integration ./internal/integration/...` from the
// component directory.
//
// This file has no build tag so the package compiles cleanly under
// `go test ./...`; the actual test code uses `package integration_test`
// (external) so it can import the production internal packages.
package integration
