# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`AGENTS.md` covers contributor-facing conventions (commit style, PR expectations, lint/format commands). This file focuses on the architectural shape and the gotchas that aren't obvious from reading any single file.

## Repository shape

AxisML is a Kubernetes-native ML platform. The repo is a monorepo split into:

- `components/tenant-operator/` — Go operator binary reconciling the `Tenant` CR (Namespace, Koordinator ElasticQuota, per-tenant Secret/CM/SA/RBAC). Single reconciler, no dispatcher.
- `components/compute-operator/` — Go operator binary reconciling `MLRun`, `MLService`, and `MLTrafficPolicy` CRs via the dispatcher + handler model (one dispatcher per CR, each gated by `--enable-mlrun` / `--enable-mlservice` / `--enable-mltrafficpolicy`).
- `components/cluster-manager/` — Stateless REST shell over the cluster-scoped `ResourcePool` CRD (CRUD of pools + inline `spec.units[]`) on the K8s API. Admin-tier entry point; no PG, no reconciler, no leader election — Kubernetes etcd is the source of truth.
- `components/compute-service/` — Go service and business authority for Tenant / Quota / Job / Service / Workspace, with PG as the sole source of truth. Emits `Tenant` / `MLRun` / `MLService` CRs derived from PG and reads back their status; partitioned by namespace (= tenant name). Resolves `(poolName, unitName)` against the `ResourcePool` CRD via Informer (it does not own the ResourcePool/ResourceUnit vocabulary — that's cluster-manager).
- `components/artifact-hub/` — Go service for the artifact registry. Partitioned by `(namespace, kind, name, version)` directly (no ArtifactRepo wrapper).
- `components/platform/backend/` — the user-facing API authority and only external entry point, currently a **contract-only Go shell**: `internal/server` declares the request/response DTOs that generate `docs/openapi/platform.yaml` via `cmd/openapi-gen`, while `cmd/platform-backend` serves only health probes + a `501` fallback (real handlers are TODO). `components/platform/frontend/` is still a README-only scaffold.
- Deployment splits into three Helm charts along the Platform / System / Infra responsibility layers (install order infra → system → platform, uninstall reverse):
  - `deploy/helm/axisml-infra/` — Infra layer: third-party infrastructure (Envoy Gateway, RustFS, zot, Koordinator, GPU Operator, kube-prometheus-stack) **plus PostgreSQL**.
  - `deploy/helm/axisml-system/` — System layer: CRDs, both operators, Cluster Manager, Compute Service, Artifact Hub. No PostgreSQL — it consumes the infra DB cross-namespace.
  - `deploy/helm/axisml-platform/` — Platform layer: the user-facing entry point (Platform frontend + backend). The only externally-exposed layer.
- `docs/system_design/` — authoritative design docs: `high_level_design.md` (system-level overview) plus per-layer subdirs `infra/` · `system/` · `platform/` (each with its own `overview.md` + per-component docs), and cross-cutting `database.md` / `deployment.md`. Other doc trees: `docs/openapi/` (generated API specs), `docs/product_design/` (product/UX, incl. an interactive `prototype/`), `docs/development/` (dev guides).
- `test/` — shared test infrastructure: `setup-envtest/` binary, `testutil/` helpers, `crds/external/` vendored upstream CRDs, and `e2e/` (the centralized real-cluster e2e suite — see testing section).

The system design lives ahead of the code. When code and `docs/system_design/` disagree, the design doc is usually the intended target — confirm before "fixing" code to match incomplete scaffolding.

## Design doc style

Design docs capture the **final intended state only**, not the reasoning journey:

- No "before / after" comparisons, deprecated approaches, alternatives considered, or change rationale inside the doc body.
- No phrases like "previously X, now Y", "we used to ...", "this replaces ...", or migration narratives — when something changes, rewrite the affected section in-place to describe the new state directly.
- Discussion artifacts (why this approach beat the others, what was rejected, ADR-style trade-off analysis) belong in PR descriptions, commit messages, or a dedicated ADR file — never in `docs/system_design/`.
- `## 9. 后续工作` / "future work" sections may list **forward-looking** TODOs, but should not narrate the doc's editing history.

A reader opening the doc cold should not be able to tell which parts were rewritten in which round.

## Multi-module Go workspace

Each component is its own Go module, and each has a sibling `test/integration/` Go submodule that holds its integration tests:

```
components/tenant-operator/                       (production — Tenant CR reconciler)
components/tenant-operator/test/integration/      (integration tests, separate module)
components/compute-operator/                      (production — MLRun + MLService + MLTrafficPolicy controllers)
components/compute-operator/test/integration/     (integration tests, separate module)
components/cluster-manager/                       (production — REST shell over ResourcePool CR)
components/cluster-manager/test/integration/      (integration tests, separate module)
components/compute-service/                       (production)
components/compute-service/test/integration/      (integration tests, separate module — envtest + testcontainers Postgres)
components/artifact-hub/                          (production)
components/artifact-hub/test/integration/         (integration tests, separate module — testcontainers Postgres + httptest OCI stub)
components/platform/backend/                       (production — contract-only API shell; generates docs/openapi/platform.yaml)
components/platform/backend/test/integration/     (integration tests — drives in-process gin via httptest; no envtest/Docker)
test/testutil/                                    (shared helpers, no operator deps)
```

Why split: keeps test-only deps (`testify`, `testcontainers-go`, `testutil`) out of each component's production `go.mod` and Dockerfile build context. `testutil` is imported via `replace` from each test module — keep it operator-agnostic to avoid circular deps.

Practical implications:
- `go test ./...` from the repo root won't traverse all of these. Use `make test` / `make integration-test` instead, or `cd` into the right module.
- `go mod tidy` runs per-module — `make tidy` fans out across all of them.
- CI walks every module that contains a `go.mod` automatically (driven by `find`); newly added modules are picked up without editing the workflow.

## Build / test / install commands

The top-level `Makefile` is the command hub. The most common targets:

```sh
make help                # list targets + auto-generated per-component shortcuts
make build               # fan out `make build` to every active component
make test                # unit tests across every component (no cluster)
make integration-test    # integration tests for every component (envtest + testcontainers, needs Docker; ~30-60s)

# Per-component shortcuts (auto-generated from the COMPONENTS list:
# tenant-operator, compute-operator, cluster-manager, compute-service, artifact-hub,
# plus platform/backend whose basename is `backend` — so `make backend-test`):
make tenant-operator-test
make tenant-operator-integration
make compute-operator-test
make compute-operator-integration
make cluster-manager-test
make cluster-manager-integration
make compute-service-test
make compute-service-integration
make artifact-hub-test
make artifact-hub-integration

# Cluster + Helm:
make cluster-up                      # minikube profile "axisml"
make helm-install                    # infra → system → platform (idempotent upgrade --install)
make helm-template                   # render all three charts for review
```

Per-component dev loop (run from inside the component dir):

```sh
make fmt vet               # before every commit
make build / make image    # binary into bin/, container image
```

Single test invocation: `go test -run TestTenant_HappyPath ./internal/...` (use `-tags=integration` for integration tests).

Per-component shortcuts are auto-generated from the `COMPONENTS` list in the top-level Makefile. Pattern: `<basename>-{build,image,image-load,test,integration,coverage,fmt,tidy,clean}` (e.g., `make compute-operator-image-load`); API services also get `<basename>-{doc-gen,doc-test}`. **Basename = `notdir` of the path**, so `components/platform/backend` → `backend-test` / `backend-doc-gen` (not `platform-*`). Top-level `make fmt` walks every module via `GO_MODULES` (`gofmt -w` doesn't cross module boundaries on its own).

Pre-commit hooks (`pre-commit` framework, see `.pre-commit-config.yaml`) are staged:
- **pre-commit** (fast, <5s): gofmt, basic hygiene, `go vet` on touched modules, `make doc-test` when Go in `cluster-manager` / `compute-service` / `artifact-hub` changes, `make helm-lint` when `deploy/helm/**` changes.
- **pre-push** (30-60s): `golangci-lint` and `go test -short` on every Go module containing a pushed file.

Install once per clone: `make install-hooks`. Bypass for a single commit: `git commit --no-verify`. Vendored CRDs (`test/crds/external/`) and Helm sub-charts are excluded from hooks. If `doc-test` fails after editing DTOs, run `make <component>-doc-gen` (or top-level `make doc-gen`) to regenerate `docs/openapi/<component>.yaml` and re-stage — see next section.

## Testing layers

Documented in detail in `docs/development/testing.md`. The short version:

| Layer | Build tag | Where | Backing |
|---|---|---|---|
| Unit | none | `*_test.go` next to package | none — uses `controller-runtime/pkg/client/fake` |
| Integration | `//go:build integration` | each component's `test/integration/` Go submodule | embedded apiserver+etcd via `setup-envtest` (controller-runtime), plus testcontainers Postgres for compute-service and artifact-hub |
| E2E | `//go:build e2e` | `test/e2e/` (centralized, **not** per-component) | a **real** `axisml` minikube cluster (infra+system installed); reaches in-cluster HTTP via `kubectl port-forward` |

The e2e suite is **manual and not in CI**: run `make e2e-test` after `make cluster-up && make helm-install` (details in `test/e2e/README.md`). HTTP API contracts for the service components are *also* covered at the integration layer by driving the in-process gin engine via `httptest` — see `components/compute-service/test/integration/httptest_helpers_test.go` for the canonical helpers.

Conventions that bite if you don't know them:
- **Framework is plain `testing` + `testify`** (`require` for setup, `assert` for checks). **No Ginkgo/Gomega** — don't add them.
- Each gated test file needs a sibling `doc.go` (no build tag) so the package compiles cleanly under `go test ./...`.
- Polling: use `testutil.Eventually` / `EventuallyExists` / `EventuallyGone` from `test/testutil/`.
- **External CRDs**: any CRD the operator imports from outside this repo (Koordinator's ElasticQuota, scheduler-plugins' PodGroup, gateway-api's HTTPRoute, etc.) must be vendored under `test/crds/external/` and added to the merged TestMain's `CRDPaths`. Tests hang on "no matches for kind X" otherwise.

## Operator architecture: backend handler routing

The compute-operator runs three dispatchers (MLRun / MLService / MLTrafficPolicy); none reconciles a single backend type. Each CR's `spec.backend.{name, engine}` tuple routes reconciliation to a different handler:

| Backend | CRD | engine examples | Notes |
|---|---|---|---|
| `native` | MLRun, MLService, MLTrafficPolicy | `job`, `podgroup`, `deployment`, `statefulset`, `httproute` | Direct K8s primitives + sigs.k8s.io scheduler-plugins `PodGroup` for gang scheduling; MLTrafficPolicy → Envoy Gateway `HTTPRoute` with weighted `backendRefs` (canary / blue-green over member MLServices) |
| `kubeflow-trainer` | MLRun | `pytorchjob`, `tfjob`, `mpijob`, ... | Delegates to Kubeflow Training Operator |
| `kserve` | MLService | `inference`, `llminference` | KServe `InferenceService` |
| `custom` | MLRun, MLService | any | User-defined target GVK via `backend.config` |

Defaults: MLRun `(native, job)`, MLService `(native, deployment)`, MLTrafficPolicy `(native, httproute)`.

When adding a new handler:
1. Implement under `internal/<backend>/<engine>/`.
2. Wire it into the dispatch table in the operator's reconciler.
3. **All backend-derived Pods MUST set `schedulerName: koord-scheduler` and carry the `quota.scheduling.koordinator.sh/name` label** — this is non-negotiable; bypassing koord-scheduler bypasses ElasticQuota.
4. Vendor any new external CRDs into `test/crds/external/` in the same PR.
5. Pair an integration happy-path with the unit tests.

## OpenAPI specs are generated, not hand-written

Four components own a generated spec under `docs/openapi/<component>.yaml`, produced from their Go request/response DTOs: `cluster-manager`, `compute-service`, `artifact-hub`, and `platform/backend` (a server-less contract shell that nonetheless owns `docs/openapi/platform.yaml`). The two operators have no HTTP surface and are excluded. The generated specs under `docs/openapi/` are the single source of truth for HTTP API contracts.

- `make doc-gen` (or `make <basename>-doc-gen`) regenerates the spec(s).
- `make doc-test` (or `make <basename>-doc-test`) verifies that the spec matches the current Go types — this is the CI guard and the pre-commit hook described above.

When you change a handler signature or DTO, regenerate before committing and never hand-edit `docs/openapi/*.yaml`. **Gotcha:** the pre-commit `doc-test` hook only watches `cluster-manager` / `compute-service` / `artifact-hub` Go files, so editing `platform/backend` DTOs won't trip it — run `make backend-doc-gen` yourself (its basename is `backend`, not `platform`).

## Image tag synchronization

Operator images are pulled by Helm using `Chart.appVersion` as the default tag. `deploy/helm/axisml-system/Chart.yaml`'s `appVersion` is the single version authority across all three charts: the top-level Makefile exports `IMAGE_TAG` from it and injects it into both the system and platform charts' `--set <component>.image.tag`, overriding each component's local default. (Platform still ships an nginx placeholder image, so `HELM_PLATFORM_IMAGE_SET` is intentionally empty until Platform publishes a real appVersion-tracked image.) This means:

- `make image` from the repo root tags images to match what the chart will pull.
- `make image` from inside a component dir uses the component's local default (`0.1.0`) — fine for ad-hoc testing, but `minikube image load` won't satisfy the rendered Deployment unless the tags match.
- For dev loops: `make image-load IMAGE_TAG=dev` and override the chart's `image.tag` in a values override.

## Helm: install order matters

Three charts, installed infra → system → platform (uninstall reverse). `axisml-infra` provides CRDs, components, and PostgreSQL that `axisml-system` depends on (Koordinator, Envoy Gateway, the DB, etc.); `axisml-platform` depends on the system-layer services being up (its bootstrap calls compute-service). `make helm-install` / `make helm-uninstall` enforce this ordering.

PostgreSQL lives in the infra namespace, so the system services reach it cross-namespace at `axisml-database.axisml-infra:5432`. Because Secrets are namespace-scoped, each system service renders its own DB-credentials Secret from `database.auth.password` — that password must match `database.auth.password` in the infra chart (a shared input present in both values files).

`make helm-install-system` runs `helm-crds-system` first, which `kubectl apply`s `deploy/helm/axisml-system/crds/` directly — Helm only installs files under `crds/` on initial install, so this picks up schema upgrades. If you add or change a CRD, the chart upgrade alone won't apply it; the make target handles this.

## Conventions worth knowing

- Conventional Commit subjects: `docs:`, `feat(operator):`, `chore(build):`, `fix:`, etc. Keep scoped and imperative.
- Component basenames must be unique across the `COMPONENTS` list in the top-level Makefile (the per-component shortcut targets are derived from `notdir`).
- `bin/` directories are build artifacts — never commit.
- Lint config (`.golangci.yml`) is shared across all Go modules; CI runs `golangci-lint` once per module (matrix in `.github/workflows/ci.yml`). Active linters: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `misspell`, plus `gofmt` + `goimports` formatters. The `integration` build tag is enabled so tagged files are linted too.
