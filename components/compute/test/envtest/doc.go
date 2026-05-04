// Package envtest_test runs the compute service against a hermetic
// controller-runtime envtest (embedded apiserver+etcd) and a testcontainers-
// managed PostgreSQL instance. The L1 layer covers Outbox + reconciler +
// Informer paths end-to-end without depending on minikube or a remote DB.
//
// Triggered by `make compute-envtest`.
package envtest_test
