# axisml-system

The **System layer** of AxisML is the Kubernetes control plane and a collection
of public Go modules that supply domain services shared by both deployment forms. The
top-level [`axisml-standalone`](../axisml-standalone/) module composes those
services into one process and supplies the Docker runtime.

System is **not externally exposed**. Platform is the product entry point and
propagates the trusted `X-Axisml-User` identity to System APIs.

## Components

The shared API contract and each deployable component are released as independent
Go modules under `github.com/axisml/axisml/axisml-system/...`:

| Component | Kind | One-line responsibility | Key model |
| --- | --- | --- | --- |
| **[cluster-manager](cluster-manager/)** | REST (admin) | Stateless REST shell over K8s for the admin domain — `ResourcePool` + `Tenant` CRUD, quota folding, cluster capacity/metrics — so upstream never calls the K8s API directly | `ResourcePool` CR (inline `units[]`) + `Tenant` CR |
| **[compute-service](compute-service/)** | REST (business) | Business authority for Run / Service / Workspace / TrafficPolicy with **PG as the sole source of truth**; derives three CRs and reads their status back | `MLRun` + `MLService` + `MLTrafficPolicy`, PG, namespace-partitioned |
| **[artifact-hub](artifact-hub/)** | REST (business) | Artifact registry with a metadata (PG) / storage (zot · RustFS) split | Artifact tuple `(namespace, kind, name, version)` |
| **[tenant-operator](tenant-operator/)** | Operator | Reconciles the `Tenant` CR into Namespace / `ElasticQuota` / per-tenant Secret·CM·SA·RBAC; reflects `status.used` | `Tenant` CR (cluster-scoped) |
| **[compute-operator](compute-operator/)** | Operator | dispatcher + handler routing the three CRs by `spec.backend.{name,engine}` into K8s & gateway resources | three CRs + backend handler registry |

The Kubernetes form deploys all five components. The standalone distribution
imports the three REST modules in-process and does not run either operator.

## Layer invariants

- **PG owns mutable business state.** compute-service and artifact-hub share the
  same schema and migration chain in both forms.
- **Runtime adapters are private.** System writes workload CRs for the
  Kubernetes operator; the separate standalone module maps the same desired
  objects to Docker resources.
- **Operators consume CRs one-way.** An operator reads `spec`, writes `status`, and never writes back to upstream PG; operators don't know about each other (tenant-operator never reads `MLRun`/`MLService`; compute-operator never reads `Tenant`/`ElasticQuota` — it only passes the quota name through).
- **Read/write paths converge through etcd.** cluster-manager writes ResourcePool / Tenant CRs; compute-service (via Informer) and tenant-operator read them directly — no direct service-to-service calls between them.
- **Tenant scope ≠ landing namespace.** The `namespace` field on compute/artifact records is the tenant scope; the K8s Namespace is `Tenant.spec.namespace.name`, which several Tenants may share.
- **All derived Pods route through axisml-scheduler** with an ElasticQuota label — there is no quota-bypassing path.

## Module and test boundaries

The shared API contract and each deployable component own their production
module. Integration tests, OpenAPI tooling and shared test utilities use nested
repository-local modules.

```
apis/go.mod
tenant-operator/go.mod   compute-operator/go.mod
cluster-manager/go.mod   compute-service/go.mod   artifact-hub/go.mod
  └── test/integration/  (one per component, separate module where applicable)
test/{testutil,crds/external,setup-envtest}/   Shared test infrastructure
docs/system_design/    Per-component design docs + database.md + overview.md
deploy/helm/           The "axisml-system" chart (CRDs + all five components)
```

Use the layer Makefile so the component modules and nested test/tool modules are
validated together.

## Build / test

The layer Makefile provides aggregate and per-component targets for the five
Kubernetes components.

```sh
make help                         # list all targets
make build                        # build all five component binaries
make test                         # unit tests across every component (no Docker)
make integration                  # integration tests (envtest + testcontainers; Docker required)
make doc-gen / make doc-test      # component OpenAPI specs
make image / make image-load      # build (and load into minikube) every component image

# Per-component shortcuts, e.g.:
make compute-service-test
make cluster-manager-doc-gen
make artifact-hub-integration
```

OpenAPI specs under `docs/apis/<component>.yaml` are **generated from Go DTOs, never hand-edited** — run `make <component>-doc-gen` after changing a handler signature or DTO.

## Deployment

Kubernetes uses the System chart after Infra and before Platform. The separate
[`axisml-standalone`](../axisml-standalone/) distribution is started from the
repository root with `make standalone-up`; its Compose stack contains
PostgreSQL, the composed System process, Platform and zot.

```sh
make helm-lint        # lint the chart
make helm-template    # render locally for review
make helm-install     # apply CRDs (via helm-crds) then install/upgrade the release
make helm-uninstall   # tear down
```

`make helm-install` depends on `helm-crds`, which `kubectl apply`s `deploy/helm/crds/` directly — Helm only installs `crds/` on first install, so this picks up CRD schema upgrades. The chart's `appVersion` is the **single image-tag authority** across all three layer charts.

## See also

- **[System design overview](docs/system_design/overview.md)** + per-component docs and **[database.md](docs/system_design/database.md)**
- **[Standalone distribution](../axisml-standalone/README.md)** — single-host module, runtime and deployment contract
- **[High-level design](../docs/high_level_design.md)** — system-level invariants and the three-layer architecture
- **[Development workflow](../docs/development_workflow.md)** — setup, build/test, testing layers
- **[Deployment manual](../docs/deployment.md)**
