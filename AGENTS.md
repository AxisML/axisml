# Repository Guidelines

## Project Structure & Module Organization

AxisML is organized around documentation, Helm deployment assets, and component implementations. System and development docs live in `docs/system_design/` and `docs/development/`; shared images are in `docs/assets/`. Helm charts are under `deploy/helm/axisml-infra/` for third-party infrastructure and `deploy/helm/axisml-system/` for CRDs, the merged operator, and control-plane services. Component code lives in `components/`: the merged Kubernetes operator is `components/operator/` (axisml-operator binary hosting Tenant + MLJob + MLService controllers), `components/compute/` is the active Go service, and `components/artifacts/` + `components/platform/*` are scaffolded service areas.

## Build, Test, and Development Commands

Use the top-level `Makefile` as the command hub.

- `make help`: list available targets and active component shortcuts.
- `make cluster-up` / `make cluster-status`: create, repair addons, and inspect the local `axisml` minikube cluster.
- `make helm-template`: render both Helm charts locally for review.
- `make helm-install`: install or upgrade infra first, then the system chart.
- `make build`, `make test`, `make image`: fan out to active components.
- `make operator-test` / `make compute-test`: per-component shortcuts (auto-generated from the COMPONENTS list).

Override cluster defaults with `make cluster-up MINIKUBE_CPUS=6 MINIKUBE_MEMORY=8192`.

## Coding Style & Naming Conventions

Go components use standard Go formatting. Run `make -C components/operator fmt vet` before submitting operator changes; component binaries build into `bin/`, do not commit build artifacts. Keep Helm names and values consistent with chart conventions in `deploy/helm/*/values.yaml`; image tags should track `deploy/helm/axisml-system/Chart.yaml` `appVersion` unless explicitly testing a local tag.

## Testing Guidelines

Tests are organized in three layers: unit (`make test`), L1 integration hermetic tests (`make integration-test` after `make setup-envtest`), and L2 e2e against minikube (`make e2e-test`, which itself brings up the cluster + helm-installs the stack). See `docs/development/testing.md` for layer choice, build tags (`integration` / `e2e`), conventions, and external-CRD vendoring under `test/crds/external/`. For Helm-only changes, run `make helm-template` and inspect rendered manifests.

## Commit & Pull Request Guidelines

Recent history uses Conventional Commit-style subjects such as `docs: ...`, `feat(operator): ...`, and `chore(build): ...`. Keep commits scoped and imperative. Pull requests should explain the behavioral or documentation change, list validation commands run, link related issues or design notes, and include screenshots only for UI-facing platform frontend changes.

## Security & Configuration Tips

Do not commit kubeconfigs, registry credentials, or generated local artifacts. Prefer documented `make` variables such as `MINIKUBE_DRIVER`, `MINIKUBE_PROFILE`, and `IMAGE_TAG` over editing scripts for local-only configuration.
