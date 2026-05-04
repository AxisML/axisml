// Package envtest_test holds the merged axisml-operator L1 envtest suite
// (Tenant + MLJob + MLService reconcilers running in one Manager). The
// actual tests are gated by `//go:build envtest`; this file lets the
// package compile (with no buildable Go files) under a default
// `go test ./...` invocation, so unit-test runs don't fail with
// "build constraints exclude all Go files".
package envtest_test
