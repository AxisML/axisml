# Repository Guidelines

## Project Structure & Module Organization

AxisML is a Kubernetes-native, multi-module Go monorepo organized by deployment layer. `axisml-system/` contains control-plane components: `tenant-operator`, `compute-operator`, `cluster-manager`, `compute-service`, and `artifact-hub`, plus the shared `apis` module that holds the five CRD Go types (MLRun/MLService/MLTrafficPolicy/Tenant/ResourcePool) every producer and consumer imports. `axisml-platform/` contains the user-facing layer: `backend/` plus the Vite/React `frontend/`. `axisml-infra/` owns local cluster and infrastructure Helm logic. Shared packages live in `pkg/`, cross-cutting docs in `docs/`, the black-box test suite (Python + pytest) in `tests/`, and generated API specs under each layer's `docs/apis/`.

## Build, Test, and Development Commands

Use the root `Makefile` for repo-wide work; it delegates to layer Makefiles.

- `make help`: list supported root targets and examples.
- `make build`: build all Go components.
- `make test`: run unit tests across Go layers.
- `make integration-test`: run envtest/testcontainers integration suites; requires Docker.
- `make fmt vet tidy`: format, vet, and tidy every Go module.
- `make doc-gen` / `make doc-test`: regenerate or verify generated OpenAPI specs.
- `make helm-template` / `make helm-lint`: render or lint all Helm charts.
- `make -C axisml-system compute-service-test`: run one component target.
- `make -C axisml-platform frontend-dev`: start the frontend dev server with pnpm.

## Coding Style & Naming Conventions

Go code must be `gofmt`/`goimports` clean and pass `go vet`; `golangci-lint` uses `.golangci.yml` with `errcheck`, `govet`, `staticcheck`, `unused`, `misspell`, and related checks. Keep Go package names short and lowercase. Component directory basenames must stay unique because Makefile shortcuts use them. Do not hand-edit generated files such as `zz_generated_deepcopy.go` or `<layer>/docs/apis/*.yaml`; regenerate them.

## Testing Guidelines

Unit tests sit next to packages as `*_test.go` and use Go `testing` plus `testify`; avoid Ginkgo/Gomega. Integration tests live under each component's `test/integration/` module and use the `integration` build tag. The black-box E2E suite lives in `tests/` (Python + pytest + Playwright, uv-managed): API tests per component over the HTTP contract, plus UI e2e; it requires a running Kubernetes cluster brought up with `uv run test-setup`. Prefer `axisml-system/test/testutil` polling helpers for reconciler assertions.

## Commit & Pull Request Guidelines

Use Conventional Commit subjects, e.g. `feat(system): add cache probe` or `docs(platform): clarify runtime contract`. The scope must be one of the three deployment layers — `infra`, `system`, or `platform` — or, for cross-cutting changes, `build` (Makefiles/CI/tooling), `repo` (repo-wide reorg), or `deps` (dependency bumps); omit it when a change spans everything (`docs:`, `chore:`). This is enforced by commitlint (`.commitlintrc.yml`) via the `commit-msg` hook and on PR titles in CI. PR titles must also be valid Conventional Commits because PRs are squash-merged. Before opening a PR, run the relevant component tests plus `make fmt`, `make doc-test` for DTO/API changes, and `make helm-template`/`make helm-lint` for chart changes. Link issues, describe behavior changes, and include screenshots for UI-facing work.

## Security & Configuration Tips

Never commit real secrets, kubeconfigs, generated cluster state, local binaries, or coverage output. Use values files or `HELM_EXTRA_ARGS` for environment-specific Helm settings.
