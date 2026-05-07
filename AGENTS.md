# Repository Guidelines

## Project Structure & Module Organization

AxisML combines Go services, Kubernetes operators, Helm charts, and design docs. Active component code lives in `components/tenant-operator/`, `components/compute-operator/`, `components/cluster-manager/`, `components/compute/`, and `components/artifacts/`; each has its own `go.mod`, `Dockerfile`, and `Makefile`. Platform backend/frontend folders under `components/platform/` are scaffolded. Helm charts live in `deploy/helm/axisml-infra/` for third-party infrastructure and `deploy/helm/axisml-system/` for CRDs and AxisML services. Development and design docs are in `docs/development/` and `docs/system_design/`. Shared test modules and e2e suites live under `test/`.

## Build, Test, and Development Commands

Use the top-level `Makefile` as the command hub.

- `make help`: list available targets and generated component shortcuts.
- `make build`: build all active components into their local `bin/` directories.
- `make test`: run unit tests for active components.
- `make image` / `make image-load`: build images and load them into minikube.
- `make cluster-up` / `make cluster-status`: manage the local `axisml` minikube profile.
- `make helm-template`: render both Helm charts locally for review.
- `make helm-install`: install infra first, then the AxisML system chart.
- `make operator-integration`, `make compute-test`, `make artifacts-test`: run focused component targets.

## Coding Style & Naming Conventions

Go code uses standard `gofmt`; run `make fmt` for all modules or `<component>-fmt` for a slice. `.golangci.yml` enables `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`, and `misspell` with `integration,e2e` build tags. Keep generated files committed when sources require them, such as CRDs or `components/compute/docs/openapi.yaml` after `make -C components/compute openapi`. Do not commit `bin/`, coverage output, local kubeconfigs, or secrets.

## Testing Guidelines

Unit tests use Go `testing` and normally sit beside packages as `*_test.go`. L1 tests use `//go:build integration`; the operators / cluster-manager / compute keep them under separate `test/integration/` modules (envtest-backed), while artifacts keeps them in-module under `internal/integration/` (testcontainers-backed). All flavours run via `make integration-test` (after `make setup-envtest`). L2 tests live under `test/e2e/`, use `//go:build e2e`, and run with `make e2e-test` against minikube. See `docs/development/testing.md` for layer choice and external CRD rules.

## Commit & Pull Request Guidelines

Recent history follows Conventional Commit-style subjects: `feat(operator): ...`, `docs(infra): ...`, `ci: ...`. Keep commits scoped and imperative. PRs should summarize behavior or doc changes, link related issues or design notes, list validation commands run, and include screenshots only for UI-facing changes.

## Hooks & Configuration

Install hooks with `make install-hooks`; run all hooks with `make pre-commit-run`. Prefer Make variables such as `IMAGE_TAG`, `MINIKUBE_PROFILE`, `MINIKUBE_CPUS`, and `HELM_EXTRA_ARGS` for local overrides instead of editing scripts or checked-in values.
