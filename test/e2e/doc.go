// Package e2e is the centralized, black-box end-to-end suite. It runs against
// either AxisML deployment form, selected at compile time by build tag:
//
//   - standard (alias: e2e): a REAL minikube cluster (profile "axisml") with the
//     infra + system Helm layers installed, reached over kubectl port-forwards.
//     Run with `make e2e-test`.
//   - lite: one axisml-core process (no Kubernetes) at $LITE_CORE_URL. Bring the
//     stack up with `make lite-up`, then run `make e2e-lite-test`.
//
// The shared CORE tests (files tagged `e2e || standard || lite`) drive only the
// System HTTP contract through the typed clients and the Harness seam, so they
// validate both forms unchanged. Form-specific code is confined to the harness
// implementations (harness_standard_test.go / harness_lite_test.go) and the
// form-tagged tests (Standard real-cluster white-box; lite_only_test.go).
//
// The suite is never part of the default `go test ./...` or the PR CI gate.
// This file carries no build tag so the package compiles cleanly under the
// default build constraints.
//
// See README.md for how to run each form and the environment it assumes.
package e2e
