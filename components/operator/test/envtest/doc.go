// Package envtest_test holds the tenant-operator L1 envtest suite. The actual
// tests are gated by `//go:build envtest`; this file lets the package compile
// (with no buildable Go files) under a default `go test ./...` invocation,
// so unit-test runs don't fail with "build constraints exclude all Go files".
package envtest_test
