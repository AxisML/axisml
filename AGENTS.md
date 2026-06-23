# Repository Guidelines

## Project Structure & Module Organization

AxisML is a Kubernetes-native monorepo of independent Go modules, organized by deployment layer at the top level. `axisml-system/` holds the control-plane components — `tenant-operator`, `compute-operator`, `cluster-manager`, `compute-service`, `artifact-hub`. `axisml-platform/` holds the user-facing layer — `backend` (a contract-only shell that generates `axisml-platform/docs/apis/platform.yaml`) and `frontend` (still a scaffold). `axisml-infra/` and the Helm chart in each layer's `deploy/helm/` carry third-party infrastructure and packaging. `axisml-lite/` is the no-Kubernetes Docker Compose form (design doc only today). Each layer also keeps its design docs under `<layer>/docs/`. Shared Go test helpers are in `test/testutil`, external CRDs in `test/crds/external`, the manual real-cluster e2e suite in `test/e2e`, and generated OpenAPI specs in each layer's `docs/apis`. Helm install order is infra -> system -> platform. Cross-cutting design documents live in `docs/system_design`; update them when behavior or contracts change.

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

Per-component shortcuts follow `<basename>-<target>`, for example `make compute-service-test` or `make artifact-hub-doc-gen`; the basename is the directory name, so `axisml-platform/backend` → `platform-backend-test` / `platform-backend-doc-gen`.

## Coding Style & Naming Conventions

Go code must be formatted with `gofmt`; hooks also run `go vet` and push-time `golangci-lint` on touched modules. Keep package names short and lowercase. Component directory basenames must stay unique because Makefile shortcuts use them. Do not hand-edit generated files such as `zz_generated_deepcopy.go` or `<layer>/docs/apis/*.yaml`; regenerate them instead.

## Testing Guidelines

Unit tests sit next to packages as `*_test.go` and use Go `testing` with `testify`. Integration tests live in each component's `test/integration` submodule and are gated by the `integration` build tag. A manual, real-cluster e2e suite lives in `test/e2e` (gated by the `e2e` build tag, run via `make e2e-test` — not part of CI). Prefer `test/testutil` polling helpers for reconciler assertions. Add required external CRDs under `test/crds/external` when a new controller dependency needs them.

## Commit & Pull Request Guidelines

Follow Conventional Commit-style subjects seen in history, such as `fix(compute-service): ...`, `feat: ...`, and `chore: ...`. Keep commits scoped and imperative. Before opening a PR, run relevant `make ...-test`, `make doc-test`, and `make helm-template`/`make helm-lint` checks. PRs should describe behavior changes, link issues when available, and include screenshots only for UI-facing changes.

## Security & Configuration Tips

Do not commit real secrets, kubeconfigs, local binaries, `bin/`, coverage output, or generated cluster state. Development Helm defaults are for local use only; production installs must provide real secret values through values files or `HELM_EXTRA_ARGS`.
