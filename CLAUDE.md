# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`AGENTS.md` covers contributor-facing conventions (commit style, PR expectations, lint/format commands). This file focuses on the architectural shape and the gotchas that aren't obvious from reading any single file.

## Repository shape

AxisML is a Kubernetes-native ML platform. The repo is a monorepo organized by deployment layer at the top level — `axisml-platform/`, `axisml-system/`, `axisml-infra/`, and `axisml-lite/` — where each layer dir holds its components, its Helm chart under `deploy/helm/`, and its design docs under `docs/`:

- `axisml-system/tenant-operator/` — Go operator binary reconciling the `Tenant` CR (Namespace, Koordinator ElasticQuota, per-tenant Secret/CM/SA/RBAC). Single reconciler, no dispatcher.
- `axisml-system/compute-operator/` — Go operator binary reconciling `MLRun`, `MLService`, and `MLTrafficPolicy` CRs via the dispatcher + handler model (one dispatcher per CR, each gated by `--enable-mlrun` / `--enable-mlservice` / `--enable-mltrafficpolicy`).
- `axisml-system/cluster-manager/` — Stateless REST shell over the cluster-scoped `ResourcePool` CRD (CRUD of pools + inline `spec.units[]`) on the K8s API. Admin-tier entry point; no PG, no reconciler, no leader election — Kubernetes etcd is the source of truth.
- `axisml-system/compute-service/` — Go service and business authority for Tenant / Quota / Job / Service / Workspace, with PG as the sole source of truth. Emits `Tenant` / `MLRun` / `MLService` CRs derived from PG and reads back their status; partitioned by namespace (= tenant name). Resolves `(poolName, unitName)` against the `ResourcePool` CRD via Informer (it does not own the ResourcePool/ResourceUnit vocabulary — that's cluster-manager).
- `axisml-system/artifact-hub/` — Go service for the artifact registry. Partitioned by `(namespace, kind, name, version)` directly (no ArtifactRepo wrapper).
- `axisml-platform/backend/` — the user-facing API authority and only external entry point, currently a **contract-only Go shell**: `internal/server` declares the request/response DTOs that generate `axisml-platform/docs/apis/platform.yaml` via `cmd/openapi-gen`, while `cmd/platform-backend` serves only health probes + a `501` fallback (real handlers are TODO). `axisml-platform/frontend/` is still a README-only scaffold.
- Deployment splits into three Helm charts along the Platform / System / Infra responsibility layers (install order infra → system → platform, uninstall reverse):
  - `axisml-infra/deploy/helm/` — Infra layer: third-party infrastructure (Envoy Gateway, RustFS, zot, Koordinator, GPU Operator, kube-prometheus-stack) **plus PostgreSQL**.
  - `axisml-system/deploy/helm/` — System layer: CRDs, both operators, Cluster Manager, Compute Service, Artifact Hub. No PostgreSQL — it consumes the infra DB cross-namespace.
  - `axisml-platform/deploy/helm/` — Platform layer: the user-facing entry point (Platform frontend + backend). The only externally-exposed layer.
- `axisml-lite/` — the no-Kubernetes single-host Docker Compose form (design doc only today, at `axisml-lite/docs/overview.md`; `cmd/` · `internal/` · `deploy/compose/` to be built per that doc).
- Design docs: each layer owns its per-component docs under `<layer>/docs/` (`axisml-system/docs/`, `axisml-platform/docs/`, `axisml-infra/docs/`, `axisml-lite/docs/`). Cross-cutting design docs stay in `docs/system_design/` — `high_level_design.md` (system-level overview), `database.md`, `deployment.md`. Generated API specs live in each layer's `docs/apis/`. Other doc trees: `docs/development/` (dev guides); product/UX docs live in `axisml-platform/docs/product_design/` (incl. an interactive `prototype/`).
- `axisml-system/test/` — System-layer test infrastructure used by the System integration suites: `setup-envtest/` binary, `testutil/` helpers, `crds/external/` vendored upstream CRDs.
- `test/` — repo-level shared tests: `e2e/`, the centralized real-cluster e2e suite (see testing section). It's the only cross-layer test tree left at the root.

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
axisml-system/tenant-operator/                       (production — Tenant CR reconciler)
axisml-system/tenant-operator/test/integration/      (integration tests, separate module)
axisml-system/compute-operator/                      (production — MLRun + MLService + MLTrafficPolicy controllers)
axisml-system/compute-operator/test/integration/     (integration tests, separate module)
axisml-system/cluster-manager/                       (production — REST shell over ResourcePool CR)
axisml-system/cluster-manager/test/integration/      (integration tests, separate module)
axisml-system/compute-service/                       (production)
axisml-system/compute-service/test/integration/      (integration tests, separate module — envtest + testcontainers Postgres)
axisml-system/artifact-hub/                          (production)
axisml-system/artifact-hub/test/integration/         (integration tests, separate module — testcontainers Postgres + httptest OCI stub)
axisml-platform/backend/                       (production — contract-only API shell; generates axisml-platform/docs/apis/platform.yaml)
axisml-platform/backend/test/integration/     (integration tests — drives in-process gin via httptest; no envtest/Docker)
axisml-system/test/testutil/                                    (shared helpers, no operator deps)
```

Why split: keeps test-only deps (`testify`, `testcontainers-go`, `testutil`) out of each component's production `go.mod` and Dockerfile build context. `testutil` is imported via `replace` from each test module — keep it operator-agnostic to avoid circular deps.

Practical implications:
- `go test ./...` from the repo root won't traverse all of these. Use `make test` / `make integration-test` instead, or `cd` into the right module.
- `go mod tidy` runs per-module — `make tidy` fans out across all of them.
- CI walks every module that contains a `go.mod` automatically (driven by `find`); newly added modules are picked up without editing the workflow.

## Build / test / install commands

There are exactly **four Makefiles**: the repo-root orchestrator plus one per layer (`axisml-infra/`, `axisml-system/`, `axisml-platform/`). The root Makefile is a thin delegator — its aggregate targets fan out to the layer Makefiles, which hold the real build/test/helm logic. The most common root targets:

```sh
make help                # list root targets (delegated to layers)
make build               # → make -C each Go layer build (system 5 components + platform backend)
make test                # unit tests across every layer
make integration-test    # integration tests across layers (envtest + testcontainers, needs Docker)
make doc-gen / doc-test  # regenerate / verify every OpenAPI spec
make coverage            # unit + integration coverage, merged into coverage/coverage.out

# Cluster + Helm (root orchestrates ordering; per-layer logic lives in each layer Makefile):
make cluster-up                      # delegates to axisml-infra (minikube profile "axisml")
make helm-install                    # infra → system → platform (idempotent upgrade --install)
make helm-template                   # render all three charts for review
```

Per-component work runs through the **layer** Makefile, which owns both layer aggregates and per-component shortcuts:

```sh
make -C axisml-system build                    # all 5 system components
make -C axisml-system compute-service-test     # one component
make -C axisml-system cluster-manager-doc-gen  # API services also get doc-gen/doc-test
make -C axisml-platform integration            # backend httptest suite
make -C axisml-platform frontend-dev           # frontend (pnpm; not in the default aggregate)
```

Single test invocation: `cd axisml-system/compute-service && go test -run TestX ./internal/...` (use `-tags=integration` inside the `test/integration` submodule).

The system layer Makefile generates each component's targets from its `COMPONENTS` list (`<component>-{build,image,image-load,test,integration,coverage,integration-coverage,coverage-html,fmt,vet,tidy,clean}`; the three API services also get `<component>-{doc-gen,doc-test}`). Generated as explicit targets (not pattern rules) so overlapping suffixes like `-test` / `-doc-test` never collide. Root `make fmt` delegates to each layer's `fmt`.

Pre-commit hooks (`pre-commit` framework, see `.pre-commit-config.yaml`) are staged:
- **pre-commit** (fast, <5s): gofmt, basic hygiene, `go vet` on touched modules, `make doc-test` when Go in `cluster-manager` / `compute-service` / `artifact-hub` changes, `make helm-lint` when `deploy/helm/**` changes.
- **pre-push** (30-60s): `golangci-lint` and `go test -short` on every Go module containing a pushed file.

Install once per clone: `make install-hooks`. Bypass for a single commit: `git commit --no-verify`. Vendored CRDs (`axisml-system/test/crds/external/`) and Helm sub-charts are excluded from hooks. If `doc-test` fails after editing DTOs, run `make <component>-doc-gen` (or top-level `make doc-gen`) to regenerate `the layer's docs/apis/<component>.yaml` and re-stage — see next section.

## Testing layers

Documented in detail in `docs/development/testing.md`. The short version:

| Layer | Build tag | Where | Backing |
|---|---|---|---|
| Unit | none | `*_test.go` next to package | none — uses `controller-runtime/pkg/client/fake` |
| Integration | `//go:build integration` | each component's `test/integration/` Go submodule | embedded apiserver+etcd via `setup-envtest` (controller-runtime), plus testcontainers Postgres for compute-service and artifact-hub |
| E2E | `//go:build e2e` | `test/e2e/` (centralized, **not** per-component) | a **real** `axisml` minikube cluster (infra+system installed); reaches in-cluster HTTP via `kubectl port-forward` |

The e2e suite is **manual and not in CI**: run `make e2e-test` after `make cluster-up && make helm-install` (details in `test/e2e/README.md`). HTTP API contracts for the service components are *also* covered at the integration layer by driving the in-process gin engine via `httptest` — see `axisml-system/compute-service/test/integration/httptest_helpers_test.go` for the canonical helpers.

Conventions that bite if you don't know them:
- **Framework is plain `testing` + `testify`** (`require` for setup, `assert` for checks). **No Ginkgo/Gomega** — don't add them.
- Each gated test file needs a sibling `doc.go` (no build tag) so the package compiles cleanly under `go test ./...`.
- Polling: use `testutil.Eventually` / `EventuallyExists` / `EventuallyGone` from `axisml-system/test/testutil/`.
- **External CRDs**: any CRD the operator imports from outside this repo (Koordinator's ElasticQuota, scheduler-plugins' PodGroup, gateway-api's HTTPRoute, etc.) must be vendored under `axisml-system/test/crds/external/` and added to the merged TestMain's `CRDPaths`. Tests hang on "no matches for kind X" otherwise.

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
4. Vendor any new external CRDs into `axisml-system/test/crds/external/` in the same PR.
5. Pair an integration happy-path with the unit tests.

## OpenAPI specs are generated, not hand-written

Four components own a generated spec under their layer's `docs/apis/<component>.yaml`, produced from their Go request/response DTOs: `cluster-manager`, `compute-service`, `artifact-hub` (under `axisml-system/docs/apis/`), and `platform/backend` (a server-less contract shell that owns `axisml-platform/docs/apis/platform.yaml`). The two operators have no HTTP surface and are excluded. The generated specs under each layer's `docs/apis/` are the single source of truth for HTTP API contracts.

- `make doc-gen` (root, all specs) or per-component, e.g. `make -C axisml-system compute-service-doc-gen`, regenerates the spec(s). Platform's is `make -C axisml-platform doc-gen` (backend is its only API component).
- `make doc-test` (root) or per-component verifies the spec matches the current Go types — this is the CI guard and the pre-commit hook described above.

When you change a handler signature or DTO, regenerate before committing and never hand-edit `<layer>/docs/apis/*.yaml`. **Gotcha:** the pre-commit `doc-test` hook only watches `axisml-system/{cluster-manager,compute-service,artifact-hub}` Go files, so editing Platform backend DTOs won't trip it — run `make -C axisml-platform doc-gen` yourself.

## Image tag synchronization

Images are pulled by Helm using `Chart.appVersion` as the default tag. `axisml-system/deploy/helm/Chart.yaml`'s `appVersion` is the single version authority across all three charts: the root Makefile exports `IMAGE_TAG` from it; each layer Makefile defaults `IMAGE_TAG` to the same value and injects it into its chart's `--set <component>.image.tag`. This means:

- `make image` from the repo root (or `make -C <layer> image`) tags images to match what the chart will pull.
- Each layer Makefile recomputes `IMAGE_TAG` from the system chart's appVersion when invoked standalone, so the tags stay aligned without the root.
- For dev loops: `make image-load IMAGE_TAG=dev` and override the chart's `image.tag` in a values override.

## Helm: install order matters

Three charts, installed infra → system → platform (uninstall reverse). `axisml-infra` provides CRDs, components, and PostgreSQL that `axisml-system` depends on (Koordinator, Envoy Gateway, the DB, etc.); `axisml-platform` depends on the system-layer services being up (its bootstrap calls compute-service). `make helm-install` / `make helm-uninstall` enforce this ordering.

PostgreSQL lives in the infra namespace, so the system services reach it cross-namespace at `axisml-database.axisml-infra:5432`. Because Secrets are namespace-scoped, each system service renders its own DB-credentials Secret from `database.auth.password` — that password must match `database.auth.password` in the infra chart (a shared input present in both values files).

The system layer's `helm-install` / `helm-upgrade` depend on `helm-crds`, which `kubectl apply`s `axisml-system/deploy/helm/crds/` directly — Helm only installs files under `crds/` on initial install, so this picks up schema upgrades. If you add or change a CRD, the chart upgrade alone won't apply it; the make target handles this. (`make helm-install` at root runs the whole infra → system → platform chain.)

## Conventions worth knowing

- Conventional Commit subjects: `docs:`, `feat(operator):`, `chore(build):`, `fix:`, etc. Keep scoped and imperative.
- The build system is four Makefiles: root (orchestrator) + one per layer. The system layer Makefile owns the `COMPONENTS` list; component dir names must stay unique within it (per-component shortcut targets are `<component>-<target>`).
- `bin/` directories are build artifacts — never commit.
- Lint config (`.golangci.yml`) is shared across all Go modules; CI runs `golangci-lint` once per module (matrix in `.github/workflows/ci.yml`). Active linters: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `misspell`, plus `gofmt` + `goimports` formatters. The `integration` build tag is enabled so tagged files are linted too.
