# Repository Guidelines

## Project Structure & Module Organization

AxisML is a Kubernetes-native monorepo of independent Go modules. Active components live under `components/`: `tenant-operator`, `compute-operator`, `cluster-manager`, `compute-service`, `artifact-hub`, and `platform/backend` (a contract-only shell that generates `docs/openapi/platform.yaml`; `components/platform/frontend` is still a scaffold). Shared Go test helpers are in `test/testutil`, external CRDs in `test/crds/external`, the manual real-cluster e2e suite in `test/e2e`, and generated OpenAPI specs in `docs/openapi`. Helm charts are split by layer: `deploy/helm/axisml-infra`, `deploy/helm/axisml-system`, and `deploy/helm/axisml-platform`; install order is infra -> system -> platform. Design documents live in `docs/system_design`; update them when behavior or contracts change.

## Build, Test, and Development Commands

Use the top-level `Makefile` as the command entry point:

- `make help` lists aggregate and per-component targets.
- `make build` builds all active components.
- `make test` runs unit tests across active components.
- `make integration-test` runs envtest/testcontainers integration suites and requires Docker.
- `make fmt` runs `gofmt` across every Go module.
- `make helm-template` renders all three Helm charts locally.
- `make helm-lint` lints chart changes.
- `make doc-gen` regenerates OpenAPI specs for HTTP services.
- `make doc-test` verifies generated specs match Go DTOs.

Per-component shortcuts follow `<basename>-<target>`, for example `make compute-service-test` or `make artifact-hub-doc-gen`; the basename is the directory name, so `components/platform/backend` → `backend-test` / `backend-doc-gen`.

## Coding Style & Naming Conventions

Go code must be formatted with `gofmt`; hooks also run `go vet` and push-time `golangci-lint` on touched modules. Keep package names short and lowercase. Component directory basenames must stay unique because Makefile shortcuts use them. Do not hand-edit generated files such as `zz_generated_deepcopy.go` or `docs/openapi/*.yaml`; regenerate them instead.

## Testing Guidelines

Unit tests sit next to packages as `*_test.go` and use Go `testing` with `testify`. Integration tests live in each component's `test/integration` submodule and are gated by the `integration` build tag. A manual, real-cluster e2e suite lives in `test/e2e` (gated by the `e2e` build tag, run via `make e2e-test` — not part of CI). Prefer `test/testutil` polling helpers for reconciler assertions. Add required external CRDs under `test/crds/external` when a new controller dependency needs them.

## Commit & Pull Request Guidelines

Follow Conventional Commit-style subjects seen in history, such as `fix(compute-service): ...`, `feat: ...`, and `chore: ...`. Keep commits scoped and imperative. Before opening a PR, run relevant `make ...-test`, `make doc-test`, and `make helm-template`/`make helm-lint` checks. PRs should describe behavior changes, link issues when available, and include screenshots only for UI-facing changes.

## Security & Configuration Tips

Do not commit real secrets, kubeconfigs, local binaries, `bin/`, coverage output, or generated cluster state. Development Helm defaults are for local use only; production installs must provide real secret values through values files or `HELM_EXTRA_ARGS`.
