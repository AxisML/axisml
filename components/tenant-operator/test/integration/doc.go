// Package integration_test holds the axisml-tenant-operator L1 integration
// suite. The actual tests are gated by `//go:build integration`; this file
// lets the package compile (with no buildable Go files) under a default
// `go test ./...` invocation, so unit-test runs don't fail with
// "build constraints exclude all Go files".
package integration_test
