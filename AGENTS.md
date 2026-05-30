# Repository Guidelines

## Project Structure & Module Organization

AxisML is a Kubernetes-native monorepo made of independent Go modules. Active components live under `components/`: `tenant-operator`, `compute-operator`, `cluster-manager`, `compute-service`, and `artifact-hub`. Scaffolded Platform areas are in `components/platform/backend` and `components/platform/frontend`. Shared Go helpers are in `test/testutil`, external CRDs for tests are in `test/crds/external`, and generated OpenAPI specs are in `docs/openapi`. Helm charts split along the Platform / System / Infra layers: `deploy/helm/axisml-infra` (third-party infrastructure + PostgreSQL), `deploy/helm/axisml-system` (control-plane services + CRDs), and `deploy/helm/axisml-platform` (user-facing entry point); install order infra → system → platform. Design documents live in `docs/system_design`; update them when behavior or contracts change.

## Build, Test, and Development Commands

Use the top-level `Makefile` as the command entry point:

- `make help` lists aggregate and per-component targets.
- `make build` builds all active components.
- `make test` runs unit tests across active components.
- `make integration-test` runs envtest/testcontainers integration suites and requires Docker.
- `make fmt` runs `gofmt` across every Go module.
- `make helm-template` renders both Helm charts locally.
- `make helm-lint` lints chart changes.
- `make doc-gen` regenerates OpenAPI specs for HTTP services.
- `make doc-test` verifies generated specs match Go DTOs.

Per-component shortcuts follow `<component>-<target>`, for example `make compute-service-test` or `make artifact-hub-doc-gen`.

## Coding Style & Naming Conventions

Go code must be formatted with `gofmt`; hooks also run `go vet` and push-time `golangci-lint` on touched modules. Keep package names short and lowercase. Component directory basenames must stay unique because Makefile shortcuts are derived from them. Do not hand-edit generated files such as `zz_generated_deepcopy.go` or `docs/openapi/*.yaml`; regenerate them instead.

## Testing Guidelines

Unit tests sit next to packages as `*_test.go` and use Go `testing` with `testify`. Integration tests live in each component's `test/integration` submodule and are gated by the `integration` build tag. Prefer `test/testutil` polling helpers for reconciler assertions. Add required external CRDs under `test/crds/external` when a new controller dependency needs them.

## Commit & Pull Request Guidelines

Follow Conventional Commit-style subjects seen in history, such as `fix(compute-service): ...`, `feat: ...`, and `chore: ...`. Keep commits scoped and imperative. Before opening a PR, run the relevant `make ...-test`, `make doc-test`, and `make helm-template`/`make helm-lint` checks for touched areas. PRs should describe behavior changes, link issues when available, and include screenshots only for UI-facing changes.

## Security & Configuration Tips

Do not commit real secrets, kubeconfigs, local binaries, `bin/`, coverage output, or generated cluster state. Development Helm defaults are for local use only; production installs must provide real secret values through values files or `HELM_EXTRA_ARGS`.
