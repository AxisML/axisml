# Development Workflow

Everything you need to build, test, and run AxisML locally. For *how a change gets
merged* (issues, commit style, PR rules) see [`CONTRIBUTING.md`](../CONTRIBUTING.md);
for architecture and gotchas see [`CLAUDE.md`](../CLAUDE.md) / [`AGENTS.md`](../AGENTS.md).

## Prerequisites

macOS (Intel or Apple Silicon), ≥16 GB RAM, ~40 GB free disk. Install via [Homebrew](https://brew.sh/):

| Tool | Install | Verify |
|---|---|---|
| Docker Desktop **or** Podman | `brew install --cask docker` / `brew install podman` | `docker info` / `podman info` |
| minikube | `brew install minikube` | `minikube version` |
| kubectl | `brew install kubectl` | `kubectl version --client` |
| Helm | `brew install helm` | `helm version` |
| Go 1.26+ | `brew install go` | `go version` |

One container runtime is enough; the cluster script auto-detects it (prefers Docker).
Force one with `make cluster-up MINIKUBE_DRIVER=podman`. With Podman, start the
machine first: `podman machine init --cpus 4 --memory 8192 --disk-size 40 && podman machine start`.

Then, once per clone:

```sh
make install-hooks     # pre-commit + pre-push hooks (needs `pre-commit` on PATH)
make setup-envtest     # shared envtest binary for integration tests
```

## Local cluster

```sh
make cluster-up        # minikube profile "axisml" (addons: ingress, metrics-server, storage-provisioner, dashboard)
make cluster-status
make helm-install      # infra → system → platform (idempotent upgrade --install)

make cluster-down      # stop, keep state
make cluster-delete    # destroy
```

Defaults (override via env vars): `MINIKUBE_PROFILE=axisml`, `MINIKUBE_CPUS=4`,
`MINIKUBE_MEMORY=8192`, `MINIKUBE_DISK=20g`, driver auto-detected. Example:
`make cluster-up MINIKUBE_CPUS=6 MINIKUBE_MEMORY=16384`. Cluster logic lives in
`axisml-infra/` (`axisml-infra/scripts/minikube.sh`).

## Build, test, lint

This is a multi-module Go monorepo — `go test ./...` from the root does **not**
traverse the components. The repo-root `Makefile` is a thin orchestrator that fans
out to the three layer Makefiles (`axisml-infra/`, `axisml-system/`,
`axisml-platform/`) and the distribution Makefile (`axisml-standalone/`), which
hold the real logic.

```sh
# Repo-root aggregates (run across every layer):
make build               # build all components
make test                # unit tests (no cluster)        ~10s
make integration-test    # envtest + testcontainers (needs Docker)   ~30–60s
make fmt vet tidy        # format / vet / mod tidy every module
make coverage            # unit + integration, merged into coverage/coverage.out

# Per-component work goes through the owning layer Makefile:
make -C axisml-system build                    # all system components
make -C axisml-system compute-service-test     # one component
make -C axisml-system compute-service-integration
make -C axisml-platform integration            # backend httptest suite
make -C axisml-platform frontend-dev           # frontend (pnpm)
```

Per-component shortcut pattern (generated from each layer's `COMPONENTS` list):
`<component>-{build,image,image-load,test,integration,coverage,fmt,vet,tidy,clean}`;
the API services also get `<component>-{doc-gen,doc-test}`.

Single test: `cd axisml-system/compute-service && go test -run TestX ./internal/...`
(add `-tags=integration` inside a `test/integration` submodule).

## Generated OpenAPI specs

`cluster-manager`, `compute-service`, `artifact-hub`, `platform/backend`, and the
Standalone aggregate generate specs under their owner’s `docs/apis/` directory
from Go DTOs — **never hand-edit them**.

```sh
make docs-gen   # regenerate every spec and the configuration manual
make docs-test  # verify generated docs match Go sources — CI guard
```

If `doc-test` fails after changing a DTO, run `doc-gen` and re-stage.

## Testing layers

Pick the cheapest layer that proves what you need.

| Layer | Build tag | Where | Backing | Use for |
|---|---|---|---|---|
| **Unit** | none | `*_test.go` next to package | `controller-runtime` fake client | pure logic, validation, mapping |
| **Integration** | `//go:build integration` | each component's `test/integration/` submodule | envtest apiserver+etcd; testcontainers Postgres for compute-service / artifact-hub; gin via `httptest` for HTTP contracts | reconciler behavior, REST endpoints |
| **E2E (black-box)** | none (Python) | `tests/` (centralized, Python + pytest) | Kubernetes/minikube or standalone/Compose selected with `--mode`; Playwright for the UI | full-stack smoke over the HTTP contract + UI |

The E2E suite is Python + pytest (uv-managed), not Go. Bring an environment up
with `uv run test-setup --mode kubernetes|standalone`, then pass the same mode to
`uv run pytest --mode <mode> api` (API tests) or
`uv run pytest --mode <mode> e2e` (UI). It drives clients generated from the
OpenAPI specs and treats the system as a black box. CI runs the standalone path;
see [`tests/README.md`](../tests/README.md).

Conventions that bite:

- **Framework**: plain `testing` + `testify` (`require` for setup, `assert` for checks). **No Ginkgo/Gomega.**
- Each gated (`integration`) file needs a sibling `doc.go` (no build tag) so the package compiles under bare `go test ./...`.
- **Polling**: `testutil.Eventually` / `EventuallyExists` / `EventuallyGone` from `axisml-system/test/testutil/` (keep it operator-agnostic — no circular deps).
- **Naming**: `<feature>_<scenario>_test.go`, `Test<Subject>_<Scenario>`; namespaces via `testutil.RandomNamespace`.
- **External CRDs**: any CRD an operator imports from outside the repo (ElasticQuota, scheduler-plugins PodGroup, gateway-api HTTPRoute, …) must be vendored under `axisml-system/test/crds/external/` and added to the operator's `TestMain` `CRDPaths`, or integration tests hang on "no matches for kind X". Vendor it in the same PR as the handler.

## Pre-commit hooks

Staged via the `pre-commit` framework (`.pre-commit-config.yaml`):

- **pre-commit** (<5s): gofmt, hygiene, `go vet` on touched modules, `doc-test` when API Go files change, `helm-lint` when `deploy/helm/**` changes.
- **pre-push** (30–60s): `golangci-lint` and `go test -short` on every Go module with a pushed file.

Bypass once with `git commit --no-verify`. CI (`.github/workflows/ci.yml`) re-runs
lint + unit + integration and the standalone API/UI path on every PR.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `Docker daemon is not running` | Start Docker Desktop, wait for `docker info` to succeed. |
| `Podman machine is not running` | `podman machine start` (or `init` then `start` if none exists). |
| Pods stuck `Pending` / minikube won't start | Give the runtime ≥10 GB RAM + 4 CPUs, then retry `make cluster-up`. |
| Cluster broken after a macOS update | `make cluster-delete && make cluster-up`. |
| Port 80/443 conflict (ingress addon) | Stop local web servers/proxies before `make cluster-up`. |
| Integration test hangs on "no matches for kind X" | Vendor the missing CRD under `axisml-system/test/crds/external/` and add it to `CRDPaths`. |
