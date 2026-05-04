# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`AGENTS.md` covers contributor-facing conventions (commit style, PR expectations, lint/format commands). This file focuses on the architectural shape and the gotchas that aren't obvious from reading any single file.

## Repository shape

AxisML is a Kubernetes-native ML platform. The repo is a monorepo split into:

- `components/operator/` — single Go operator binary (`axisml-operator`) hosting Tenant, MLJob, and MLService controllers in one Manager.
- `components/compute/` — Go service.
- `components/{artifacts,platform/{backend,frontend}}/` — scaffolded service areas with READMEs only; no code yet.
- `deploy/helm/axisml-infra/` — third-party infrastructure chart (Envoy Gateway, RustFS, zot, Koordinator, GPU Operator, kube-prometheus-stack).
- `deploy/helm/axisml-system/` — control-plane chart: CRDs, the merged operator, and (eventually) Platform/Compute/Artifacts. Includes PostgreSQL.
- `docs/system_design/` — authoritative design docs (overview, compute, operator, artifacts, infra, platform). Read these before making non-trivial design changes; they encode decisions that the code doesn't yet reflect.
- `test/` — shared test infrastructure: `setup-envtest/` binary, `testutil/` helpers, `e2e/` suites, and `crds/external/` vendored upstream CRDs for envtest.

The system design lives ahead of the code. When code and `docs/system_design/` disagree, the design doc is usually the intended target — confirm before "fixing" code to match incomplete scaffolding.

## Multi-module Go workspace

There are **six separate Go modules**, each with its own `go.mod`:

```
components/operator/                  (production — Tenant + MLJob + MLService controllers)
components/operator/test/envtest/     (L1 tests, separate module)
components/compute/                   (production)
components/compute/test/envtest/      (L1 tests, separate module)
test/testutil/                        (shared helpers, no operator deps)
test/e2e/                             (L2 tests)
```

Why split: keeps test-only deps (`testify`, `testutil`) out of each component's production `go.mod` and Dockerfile build context. `testutil` is imported via `replace` from each test module — keep it operator-agnostic to avoid circular deps.

Practical implications:
- `go test ./...` from the repo root won't traverse all of these. Use `make test` / `make envtest-test` / `make e2e-test` instead, or `cd` into the right module.
- `go mod tidy` runs per-module — `make tidy` fans out across all of them.
- The CI lint matrix in `.github/workflows/ci.yml` enumerates every module; add new modules there too.

## Build / test / install commands

The top-level `Makefile` is the command hub. The most common targets:

```sh
make help                # list targets + auto-generated per-component shortcuts
make build               # fan out `make build` to every active component
make test                # unit tests across every component (no cluster)
make envtest-test        # L1 envtest for the merged operator + compute (hermetic, ~25s)
make e2e-test            # L2 e2e on real minikube (~10 min, brings up cluster + helm-installs)

# Per-component shortcuts (auto-generated from the COMPONENTS list):
make operator-test
make operator-envtest
make operator-image-load   # builds image, loads into minikube
make compute-test

# Cluster + Helm:
make cluster-up                      # minikube profile "axisml"
make helm-install                    # infra first, then system (idempotent upgrade --install)
make helm-template                   # render both charts for review
```

Per-component dev loop (run from inside the component dir):

```sh
make fmt vet               # before every commit
make build / make image    # binary into bin/, container image
```

Single test invocation: `go test -run TestTenant_HappyPath ./internal/...` (use `-tags=envtest` or `-tags=e2e` for those layers).

Per-component shortcuts are auto-generated from the `COMPONENTS` list in the top-level Makefile. Pattern: `<basename>-{build,image,image-load,test,envtest,fmt,tidy,clean}` (e.g., `make operator-image-load`). Top-level `make fmt` walks every module via `GO_MODULES` (`gofmt -w` doesn't cross module boundaries on its own).

Pre-commit hooks (`pre-commit` framework, see `.pre-commit-config.yaml`) run gofmt + basic hygiene checks. Install once per clone: `make install-hooks`. Bypass for a single commit: `git commit --no-verify`. Vendored CRDs (`test/crds/external/`) and Helm sub-charts are excluded from hooks.

## Three-layer testing pyramid

Documented in detail in `docs/development/testing.md`. The short version:

| Layer | Build tag | Where | Cluster |
|---|---|---|---|
| Unit | none | `*_test.go` next to package | none — uses `controller-runtime/pkg/client/fake` |
| L1 envtest | `//go:build envtest` | `components/operator/test/envtest/` (merged) and `components/compute/test/envtest/` | embedded apiserver+etcd via `setup-envtest` |
| L2 e2e | `//go:build e2e` | `test/e2e/` | real minikube with helm-installed stack |

Conventions that bite if you don't know them:
- **Framework is plain `testing` + `testify`** (`require` for setup, `assert` for checks). **No Ginkgo/Gomega** — don't add them.
- Each gated test file needs a sibling `doc.go` (no build tag) so the package compiles cleanly under `go test ./...`.
- Polling: use `testutil.Eventually` / `EventuallyExists` / `EventuallyGone` from `test/testutil/`.
- **External CRDs**: any CRD the operator imports from outside this repo (Koordinator's ElasticQuota, scheduler-plugins' PodGroup, gateway-api's HTTPRoute, etc.) must be vendored under `test/crds/external/` and added to the merged TestMain's `CRDPaths`. Tests hang on "no matches for kind X" otherwise.
- L2 doesn't run in CI (minikube is too slow/flaky on hosted runners). Don't expect green CI to imply L2 passed.

## Operator architecture: backend handler routing

The MLJob and MLService operators don't reconcile a single backend type. Each CR's `spec.backend.{name, engine}` tuple routes reconciliation to a different handler:

| Backend | CRD | engine examples | Notes |
|---|---|---|---|
| `native` | MLJob, MLService | `job`, `podgroup`, `deployment`, `statefulset` | Direct K8s primitives + sigs.k8s.io scheduler-plugins `PodGroup` for gang scheduling |
| `kubeflow-trainer` | MLJob | `pytorchjob`, `tfjob`, `mpijob`, ... | Delegates to Kubeflow Training Operator |
| `kserve` | MLService | `inference`, `llminference` | KServe `InferenceService` |
| `custom` | MLJob, MLService | any | User-defined target GVK via `backend.config` |

Defaults: MLJob `(native, job)`, MLService `(native, deployment)`.

When adding a new handler:
1. Implement under `internal/<backend>/<engine>/`.
2. Wire it into the dispatch table in the operator's reconciler.
3. **All backend-derived Pods MUST set `schedulerName: koord-scheduler` and carry the `quota.scheduling.koordinator.sh/name` label** — this is non-negotiable; bypassing koord-scheduler bypasses tenant quota.
4. Vendor any new external CRDs into `test/crds/external/` in the same PR.
5. Pair an L1 envtest happy-path with the unit tests.

## Image tag synchronization

Operator images are pulled by Helm using `Chart.appVersion` as the default tag. The top-level Makefile exports `IMAGE_TAG` from `deploy/helm/axisml-system/Chart.yaml`'s `appVersion`, overriding each component's local default. This means:

- `make image` from the repo root tags images to match what the chart will pull.
- `make image` from inside a component dir uses the component's local default (`0.1.0`) — fine for ad-hoc testing, but `minikube image load` won't satisfy the rendered Deployment unless the tags match.
- For dev loops: `make image-load IMAGE_TAG=dev` and override the chart's `image.tag` in a values override.

## Helm: install order matters

`axisml-infra` provides CRDs and components that `axisml-system` depends on (Koordinator, Envoy Gateway, etc.). Always install infra first, uninstall system first. `make helm-install` / `make helm-uninstall` enforce this ordering.

`make helm-install-system` runs `helm-crds-system` first, which `kubectl apply`s `deploy/helm/axisml-system/crds/` directly — Helm only installs files under `crds/` on initial install, so this picks up schema upgrades. If you add or change a CRD, the chart upgrade alone won't apply it; the make target handles this.

## Conventions worth knowing

- Conventional Commit subjects: `docs:`, `feat(operator):`, `chore(build):`, `fix:`, etc. Keep scoped and imperative.
- Component basenames must be unique across the `COMPONENTS` list in the top-level Makefile (the per-component shortcut targets are derived from `notdir`).
- `bin/` directories are build artifacts — never commit.
- Lint config (`.golangci.yml`) is shared across all Go modules; CI runs `golangci-lint` once per module (matrix in `.github/workflows/ci.yml`). Active linters: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `misspell`, plus `gofmt` + `goimports` formatters. The `envtest,e2e` build tags are enabled so tagged files are linted too.
