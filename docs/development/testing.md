# Testing

AxisML's tests are organized into two layers. Each layer answers a
different question, so write the kind that matches what you're trying to
verify.

## The pyramid

| Layer | Where | Cluster | When you need it |
|---|---|---|---|
| **Unit** | next to the package, `*_test.go` | none (uses `controller-runtime/pkg/client/fake`) | Pure logic, validation, mapping helpers — anything that doesn't need a real apiserver. |
| **Integration** | each component's `test/integration/` Go submodule | embedded apiserver+etcd via `setup-envtest` (controller-runtime), plus testcontainers for components with PostgreSQL (compute, artifacts) | Reconciler integration and HTTP API contracts: CRD apply + watch + status patch + ownerRef + cache; HTTP handlers driven in-process via `httptest`. Fast, hermetic, CI-friendly. |

A higher-level e2e layer is intentionally absent — running real Pods on
minikube was slow and flaky on hosted CI and rarely caught regressions
that integration didn't. If full-stack coverage is needed in the future,
add a self-hosted-runner job or a nightly cron rather than re-introducing
it on every PR.

## Choosing a layer

Pick the cheapest layer that proves what you need:

- "Does this validation reject a bad spec?" → unit.
- "Does the reconciler create the right Secret with the right owner ref?" → integration.
- "Does this REST endpoint return the right status code / body shape?" → integration (drive the gin engine in-process via `httptest`).

If you're adding a new handler, pair an integration happy-path with the
existing per-package unit tests.

## Running locally

```sh
# Once per machine: install the shared setup-envtest binary into test/setup-envtest/.
make setup-envtest

# Unit tests across every component. ~10 seconds.
make test

# Integration tests for every component (envtest-backed for the operators /
# cluster-manager / compute, testcontainers-backed for compute and artifacts
# Postgres). ~30-60 s. Needs Docker for the testcontainers suites; envtest
# assets are downloaded on demand.
make integration-test

# Per-component slices.
make tenant-operator-integration
make compute-operator-integration
make cluster-manager-integration
make compute-integration
make artifacts-integration
```

## Conventions

- **Build tag**: `//go:build integration` for integration tests. Default `go test ./...`
  skips it. Each gated test file has a sibling `doc.go` (no build tag) so
  the package compiles cleanly under `go test ./...`.
- **Framework**: plain `testing` + `github.com/stretchr/testify`
  (`require` for setup, `assert` for individual checks). No Ginkgo/Gomega.
- **Polling**: use `testutil.Eventually` / `EventuallyExists` /
  `EventuallyGone` from `test/testutil/`. They wrap the `Eventually(timeout,
  interval, func() error)` shape so async assertions stay readable.
- **Naming**: per-scenario file `<feature>_<scenario>_test.go`,
  `Test<Subject>_<Scenario>` function name (e.g.
  `TestTenant_HappyPath`).
- **Namespaces in tests**: use `testutil.RandomNamespace`.
- **HTTP API tests**: drive the gin engine in-process via
  `engine.ServeHTTP(rr, req)` — see `components/compute-service/test/integration/httptest_helpers_test.go`
  for the canonical `doJSON` / `requireStatus` helpers.

## Module layout

Each component's `test/integration/` is its own Go module so the
component's production `go.mod` stays free of test-only deps (`testify`,
`testcontainers-go`, `testutil`). This keeps `Dockerfile` build context
clean — sibling-test replace directives would otherwise fall outside the
build context.

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

## CI

`.github/workflows/ci.yml` runs three jobs on every PR:

- **lint**: `golangci-lint` per Go module (matrix). Build tag
  `integration` so lint covers tagged files too.
- **unit**: `make test`.
- **integration**: `make integration-test` with `test/setup-envtest/` and
  `~/.local/share/kubebuilder-envtest/` cached.

## Coverage

Unit and integration tests both produce coverage profiles.

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
`coverage.out` for unit, `integration-coverage.out` for integration. `make
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
| `unit` | `unit` job | All five components' unit profiles in one upload. |
| `integration`, `<component>` | `integration` job | One upload per component with both flags so PR comments break down by layer and by component. |

See `codecov.yml` at the repo root for thresholds and path filters.
`fail_ci_if_error: false` keeps a Codecov outage from failing the PR —
test status alone gates merge.
