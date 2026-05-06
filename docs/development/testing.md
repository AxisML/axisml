# Testing

AxisML's tests are organized into three layers. Each layer answers a
different question, so write the kind that matches what you're trying to
verify.

## The pyramid

| Layer | Where | Cluster | When you need it |
|---|---|---|---|
| **Unit** | next to the package, `*_test.go` | none (uses `controller-runtime/pkg/client/fake`) | Pure logic, validation, mapping helpers — anything that doesn't need a real apiserver. |
| **L1 integration** | `components/operator/test/integration/` (merged) and `components/compute/test/integration/` | embedded apiserver+etcd via `setup-envtest` (controller-runtime), plus testcontainers for compute's PostgreSQL | Reconciler integration: CRD apply + watch + status patch + ownerRef + cache. Fast, hermetic, CI-friendly. |
| **L2 e2e** | `test/e2e/` | real minikube + helm-installed AxisML stack | Full-stack happy paths: real Pods running, real Koordinator scheduler, real cross-operator interactions. |

There's a 4th implicit layer for future services (compute / artifacts /
platform) — those tests live under `test/e2e/{compute,artifacts,platform}/`
and run on the same minikube cluster as L2; we don't use testcontainers
because the services strongly depend on real K8s, real zot, real RustFS
which are easier to provision from the helm chart than to reproduce locally.

## Choosing a layer

Pick the cheapest layer that proves what you need:

- "Does this validation reject a bad spec?" → unit.
- "Does the reconciler create the right Secret with the right owner ref?" → L1 integration.
- "Does an MLJob actually finish on a real cluster, with real koord-scheduler accounting?" → L2 e2e.

If you're adding a new handler, pair an L1 integration happy-path with the
existing per-package unit tests. L2 coverage for new handlers is optional
but encouraged for handlers that touch external systems (Kubeflow, KServe,
etc.) — that's where reality bites.

## Running locally

```sh
# Once per machine: install the shared setup-envtest binary into test/setup-envtest/.
make setup-envtest

# Unit tests across every component. ~10 seconds.
make test

# L1 integration for the merged operator + compute. ~25 seconds. Hermetic.
make integration-test

# L2 e2e (operator + future-service suites). Brings up minikube + helm-installs
# infra and system; runs real Pods. ~10 minutes from a clean clone.
make e2e-test

# Per-component slices.
make operator-integration
make compute-integration
```

`make e2e-test`'s prerequisites (`cluster-up image-load helm-install
e2e-wait`) are wired into the target — you don't need to run them
separately. If a step fails (e.g. helm-install times out), inspect with
`make cluster-status` and re-run.

## Conventions

- **Build tags**: `//go:build integration` for L1, `//go:build e2e` for L2.
  Default `go test ./...` skips both. Each gated test file has a sibling
  `doc.go` (no build tag) so the package compiles cleanly under `go test
  ./...`.
- **Framework**: plain `testing` + `github.com/stretchr/testify`
  (`require` for setup, `assert` for individual checks). No Ginkgo/Gomega.
- **Polling**: use `testutil.Eventually` / `EventuallyExists` /
  `EventuallyGone` from `test/testutil/`. They wrap the `Eventually(timeout,
  interval, func() error)` shape so async assertions stay readable.
- **Naming**: per-scenario file `<feature>_<scenario>_test.go`,
  `Test<Subject>_<Scenario>` function name (e.g.
  `TestTenant_HappyPath`).
- **Namespaces in tests**: use `testutil.RandomNamespace` for L1; in L2,
  hardcode a deterministic `e2e-` prefix and explicit `t.Cleanup`. The
  Tenant controller design intentionally does NOT delete Namespaces, so
  e2e tenant cleanup must `kubectl delete ns` explicitly.

## Module layout

Each component's `test/integration/` is its own Go module so the
component's production `go.mod` stays free of test-only deps (`testify`,
`testutil`). This keeps `Dockerfile` build context clean — sibling-test
replace directives would otherwise fall outside the build context. Same
applies to `test/e2e/`.

The shared `test/testutil/` is a tiny module with no operator deps; it's
imported via `replace` from each test module. If you need an operator-
specific helper (e.g., a Tenant fixture builder), put it inside the
operator's `test/integration/` package, NOT in testutil — testutil must
remain operator-agnostic to avoid circular deps.

## External CRDs (integration)

The controller-runtime envtest apiserver only knows about CRDs you feed
it. Any CRD an operator imports from outside the repo (Koordinator's
ElasticQuota, scheduler-plugins' PodGroup, gateway-api's HTTPRoute, etc.)
must be vendored under `test/crds/external/` and added to the per-operator
TestMain's `CRDPaths`. See `test/crds/external/README.md` for the upstream
sources, version pins, and refresh procedure.

When you add a new handler that imports an external API package, copy its
CRD into `test/crds/external/` in the same PR — tests will hang on
"no matches for kind X" otherwise. Don't worry about CRDs that the
operator only references via the scheme but never creates resources for;
those still need to be loaded if the controller-runtime SetupWithManager
sets up a watch on them.

## Future services

When a service under `components/{compute,artifacts,platform/backend}/`
ships, its e2e tests go under `test/e2e/<svc>/`. The skeleton is already
there with a README explaining the conventions:

- Build tag `//go:build e2e`, runs as part of `make e2e-test`.
- Use `e2e.SetupOrSkip(t)` for the cluster client, then port-forward to
  the deployed Service to call its API.
- Don't introduce testcontainers — the service relies on real K8s, real
  zot/RustFS, real Postgres (helm-installed); duplicating those locally
  is more work than running on minikube.

See `test/e2e/compute/README.md`, `test/e2e/artifacts/README.md`,
`test/e2e/platform/README.md` for service-specific scenario suggestions.

## CI

`.github/workflows/ci.yml` runs three jobs on every PR:

- **lint**: `golangci-lint` per Go module (matrix). Build tags
  `integration,e2e` so lint covers tagged files too.
- **unit**: `make test`.
- **integration**: `make integration-test` with `test/setup-envtest/` and
  `~/.local/share/kubebuilder-envtest/` cached.

L2 e2e is intentionally NOT in CI — minikube on GitHub-hosted runners is
slow and flaky, and most regressions show up at L1 first. If you need L2
coverage in CI later, add a self-hosted runner job or a nightly cron
workflow rather than putting it on every PR.

## Coverage

Unit and L1 integration both produce coverage profiles. L2 e2e is excluded
(not in CI, minikube-dependent — collection isn't reliable).

```sh
# Unit + integration with merged profile at coverage/coverage.out.
make coverage

# Per-component HTML reports (one per operator, written to
# <component>/coverage/coverage.html and integration-coverage.html).
make coverage-html

# One layer at a time.
make coverage-unit
make coverage-integration

# Per-component slices (auto-generated shortcuts).
make operator-coverage
make operator-integration-coverage
make operator-coverage-html

# Wipe everything (root + per-component coverage/ dirs).
make coverage-clean
```

Each component writes its own profiles to `<component>/coverage/`:
`coverage.out` for unit, `integration-coverage.out` for L1. `make
coverage-merge` (called by `make coverage`) concatenates them into
`coverage/coverage.out` at the repo root for Codecov / external tools.

`go tool cover -html` resolves package paths against the current Go
module, so HTML rendering happens **per component** rather than from the
merged file (the repo root has no `go.mod`, and the project intentionally
avoids `go.work`). Open the per-component HTML files for navigation; use
the merged profile only for aggregate tooling.

**Why integration needs `-coverpkg`**: each component's `test/integration/`
is its own Go module that imports the component via `replace`. Without an
explicit `-coverpkg=<component-import-path>/...`, `go test -coverprofile`
only counts the test/integration module itself and the merged report would
be empty. Each component's Makefile sets `MODULE_PATH` and threads it into
`-coverpkg` for you.

**Atomic mode**: every coverage invocation uses `-covermode=atomic` so
`scripts/merge-coverage.sh` can stitch profiles together without a
gocovmerge dependency. If you add new coverage targets, keep the mode
consistent.

CI uploads to [Codecov](https://codecov.io/) (public repo, no token
needed) with these flags:

| Flag | Source | What it covers |
|---|---|---|
| `unit` | `unit` job | All three operators' unit profiles in one upload. |
| `integration`, `<operator>` | `integration` matrix job | One upload per operator with both flags so PR comments break down by layer and by operator. |

See `codecov.yml` at the repo root for thresholds and path filters.
`fail_ci_if_error: false` keeps a Codecov outage from failing the PR —
test status alone gates merge.
