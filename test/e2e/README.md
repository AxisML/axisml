# System-layer E2E suite

Real-cluster end-to-end tests for the five system-layer services, built bottom-up
by dependency.

This layer is **manual / local only** — it is gated behind the `e2e` build tag and is
not part of `go test ./...` or PR CI.

## Prerequisites

1. A running cluster with infra + system installed:
   ```sh
   make cluster-up
   make helm-install        # infra -> system -> platform
   ```
2. Workload images preloaded (offline-deterministic; pods use `imagePullPolicy: IfNotPresent`):
   ```sh
   minikube image load busybox:latest -p axisml
   minikube image load nginx:1.27   -p axisml
   ```
3. `kubectl` on PATH with its context pointing at the `axisml` cluster (the suite reaches
   the ClusterIP-only HTTP services through `kubectl port-forward`).

## Run

```sh
make e2e-test            # from repo root
# or
cd test/e2e && go test -tags=e2e -v -timeout=30m ./...
```

A single case:
```sh
cd test/e2e && go test -tags=e2e -run TestComputeService_JobLifecycleTopToBottom -v ./...
```

## Layout

| File | Covers |
|---|---|
| `main_test.go` | `TestMain` + `gateReady` readiness gate + `provisionTenant` (per-file tenant) |
| `harness_test.go` | K8s client, scheme, port-forward, generated component clients + auth editors, polling |
| `config_test.go` | env-driven knobs (namespaces, ports, images, timeouts) |
| `internal/clients/{clustermanager,computeservice,artifacthub,platform}/` | `oapi-codegen`-generated typed clients + models for each AxisML HTTP component, produced from the OpenAPI specs (`make e2e-client-gen`) — the suite's single source of request/response types |
| `actions_test.go` | reusable client actions + request builders |
| `util_test.go` / `k8shelpers_test.go` | tenant/pod builders, CRD/quota/pod helpers |
| `oci_test.go` | minimal OCI-distribution push for artifact-hub |
| `preflight_test.go` | environment-readiness diagnostics (the gate itself runs in `TestMain`) |
| `tenant_operator_test.go` | tenant-operator (incl. real ElasticQuota admission) |
| `cluster_manager_test.go` | cluster-manager ResourcePool CRUD |
| `compute_operator_test.go` | MLRun/MLService -> real workloads, scheduler labels, route |
| `compute_service_test.go` | HTTP -> CR -> running pod chain, scale, workspace PVC |
| `traffic_policy_test.go` | HTTP -> MLTrafficPolicy CR -> weighted HTTPRoute; canary split / promote / delete |
| `artifact_hub_test.go` | artifact metadata lifecycle + real zot two-phase upload |
| `golden_path_test.go` | cross-service train-and-serve journey |

## Test organization

Each service test file owns **one tenant for its lifetime**: a top-level `Test`
calls `provisionTenant(t)` once and runs its scenarios as `t.Run` subtests that
share `(namespace, quota)` and tear their workloads down before the next subtest.
The tenant is removed by `t.Cleanup` when the file finishes. Cases that need a
differently-sized tenant (e.g. the over-quota check) or none at all keep their own
top-level `Test`. There is no process-wide shared tenant.

Readiness is gated in `TestMain` (`gateReady`) before any test: the required CRDs
must be Established and the default pool must exist, else the run fails fast with
guidance. `provisionTenant` uses run-unique names, so interrupted runs may leave
`e2e-*` tenants behind — `make e2e-clean` reaps them.

## Configuration (env overrides)

All have cluster-default values; override only when your install differs:
`E2E_INFRA_NAMESPACE`, `E2E_SYSTEM_NAMESPACE`, `E2E_CLUSTER_MANAGER_SVC`,
`E2E_COMPUTE_SERVICE_SVC`, `E2E_ARTIFACT_HUB_SVC`, `E2E_USER`, `E2E_DEFAULT_POOL`,
`E2E_DEFAULT_UNIT`, `E2E_JOB_IMAGE`, `E2E_SERVICE_IMAGE`.

## Known validation point

`oci_test.go` parses artifact-hub's upload credentials/URI defensively
(`username`/`password` or bearer `token`, scheme-stripped repo path). This is the one
contract the suite cannot verify offline; if a live run of `TestArtifactHub_ModelTwoPhaseUploadResolve`
shows a different credential or URI shape, adjust `parseOCICreds` / `parseRepoRef` — the
rest of the artifact flow is contract-stable.
