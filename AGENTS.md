# Repository Guidelines

## Project Structure & Module Organization

AxisML combines Go services, Kubernetes operators, Helm charts, and design docs. Active component code lives in `components/tenant-operator/`, `components/compute-operator/`, `components/cluster-manager/`, `components/compute/`, and `components/artifacts/`; each has its own `go.mod`, `Dockerfile`, and `Makefile`. Platform backend/frontend folders under `components/platform/` are scaffolded. Helm charts live in `deploy/helm/axisml-infra/` for third-party infrastructure and `deploy/helm/axisml-system/` for CRDs and AxisML services. Development and design docs are in `docs/development/` and `docs/system_design/`. Shared test infrastructure (`testutil/`, `setup-envtest/`, vendored `crds/external/`) lives under `test/`.

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

Go code uses standard `gofmt`; run `make fmt` for all modules or `<component>-fmt` for a slice. `.golangci.yml` enables `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`, and `misspell` with the `integration` build tag. Keep generated files committed when sources require them, such as CRDs or `components/compute/docs/openapi.yaml` after `make -C components/compute openapi`. Do not commit `bin/`, coverage output, local kubeconfigs, or secrets.

## Testing Guidelines

Unit tests use Go `testing` and normally sit beside packages as `*_test.go`. Integration tests use `//go:build integration` and live in each component's `test/integration/` Go submodule (envtest-backed for operators / cluster-manager / compute, with testcontainers for compute and artifacts Postgres). All run via `make integration-test` (after `make setup-envtest`). HTTP API tests drive the gin engine in-process via `httptest`; see `components/compute/test/integration/httptest_helpers_test.go`. The repo deliberately has no e2e layer — see `docs/development/testing.md` for layer choice and external CRD rules.

## Commit & Pull Request Guidelines

Recent history follows Conventional Commit-style subjects: `feat(operator): ...`, `docs(infra): ...`, `ci: ...`. Keep commits scoped and imperative. PRs should summarize behavior or doc changes, link related issues or design notes, list validation commands run, and include screenshots only for UI-facing changes.

## Hooks & Configuration

Install hooks with `make install-hooks`; run all hooks with `make pre-commit-run`. Prefer Make variables such as `IMAGE_TAG`, `MINIKUBE_PROFILE`, `MINIKUBE_CPUS`, and `HELM_EXTRA_ARGS` for local overrides instead of editing scripts or checked-in values.
