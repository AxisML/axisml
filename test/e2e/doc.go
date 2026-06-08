// Package e2e contains the system-layer end-to-end test suite.
//
// These tests run against a REAL minikube cluster (profile "axisml") with the
// infra and system Helm layers already installed. They are gated behind the
// "e2e" build tag and are never part of the default `go test ./...` or the PR
// CI gate — invoke them explicitly via `make e2e-test`.
//
// This file carries no build tag so the package compiles cleanly under the
// default build constraints; all actual test code lives in `*_test.go` files
// tagged `//go:build e2e`.
//
// See README.md in this directory for how to run the suite and the environment
// it assumes.
package e2e
