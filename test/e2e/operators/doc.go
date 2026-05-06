// Package operators_test once held direct-CR e2e tests for tenant /
// mljob / mlservice. Those flows were folded into the compute API e2e
// suite (test/e2e/compute) so a single test exercises the full vertical
// slice (Compute HTTP → DB → reconciler → CR → operator → Pod). The
// per-operator integration coverage now lives in
// components/operator/test/integration at L1.
//
// This file keeps the directory in the package list so future operator-
// only smoke tests have an obvious home.
package operators_test
