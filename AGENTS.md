# Repository Guidelines

## Project Structure & Module Organization

AxisML is organized around documentation, Helm deployment assets, and component implementations. System and development docs live in `docs/system_design/` and `docs/development/`; shared images are in `docs/assets/`. Helm charts are under `deploy/helm/axisml-infra/` for third-party infrastructure and `deploy/helm/axisml-system/` for CRDs, operators, and control-plane services. Component code lives in `components/`: active Go operators are in `components/operators/{tenant-operator,mljob-operator,mlservice-operator}`, while `components/compute`, `components/artifacts`, and `components/platform/*` are scaffolded service areas.

## Build, Test, and Development Commands

Use the top-level `Makefile` as the command hub.

- `make help`: list available targets and active component shortcuts.
- `make cluster-up` / `make cluster-status`: create and inspect the local `axisml` minikube cluster.
- `make helm-template`: render both Helm charts locally for review.
- `make helm-install`: install or upgrade infra first, then the system chart.
- `make build`, `make test`, `make image`: fan out to active operator components.
- `make mljob-operator-test`: run a single component shortcut; the same pattern applies to `tenant-operator` and `mlservice-operator`.

Local cluster defaults can be overridden, for example `make cluster-up MINIKUBE_CPUS=6 MINIKUBE_MEMORY=8192`.

## Coding Style & Naming Conventions

Go components use standard Go formatting. Run `make -C components/operators/mljob-operator fmt vet` before submitting operator changes, and keep generated code current with that component’s `make generate` target when API types change. Component binaries build into `bin/`; do not commit build artifacts. Keep Helm names and values consistent with chart conventions in `deploy/helm/*/values.yaml`; image tags should track `deploy/helm/axisml-system/Chart.yaml` `appVersion` unless explicitly testing a local tag.

## Testing Guidelines

Unit tests are Go `*_test.go` files and run with `make test` from the repo root or `make -C components/operators/<name> test` for a single operator. Integration coverage currently exists for the tenant operator: run `make integration-test` after `make cluster-up && make helm-install-infra`, with in-cluster operators scaled to zero as noted in the target help. For Helm-only changes, run `make helm-template` and inspect rendered manifests.

## Commit & Pull Request Guidelines

Recent history uses Conventional Commit-style subjects such as `docs: ...`, `feat(operator): ...`, and `chore(build): ...`. Keep commits scoped and imperative. Pull requests should explain the behavioral or documentation change, list validation commands run, link related issues or design notes, and include screenshots only for UI-facing platform frontend changes.

## Security & Configuration Tips

Do not commit kubeconfigs, registry credentials, or generated local artifacts. Prefer documented `make` variables such as `MINIKUBE_DRIVER`, `MINIKUBE_PROFILE`, and `IMAGE_TAG` over editing scripts for local-only configuration.
