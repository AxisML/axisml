# axisml-system

The **System layer** of AxisML — the **control plane**. 100% in-house domain logic that receives the Platform layer's internal calls and lands user intent as real Kubernetes resources. It sits in the middle of the three deployment layers (**infra → system → platform**).

System is **not externally exposed**: every component is a ClusterIP service or an operator. The REST services accept only internal calls and trust the `X-Axisml-User` identity header that the Platform layer propagates. The CRDs ship with this layer.

## Components

Three REST services and two operators, each its own Go module:

| Component | Kind | One-line responsibility | Key model |
| --- | --- | --- | --- |
| **[cluster-manager](cluster-manager/)** | REST (admin) | Stateless REST shell over K8s for the admin domain — `ResourcePool` + `Tenant` CRUD, quota folding, cluster capacity/metrics — so upstream never calls the K8s API directly | `ResourcePool` CR (inline `units[]`) + `Tenant` CR |
| **[compute-service](compute-service/)** | REST (business) | Business authority for Run / Service / Workspace / TrafficPolicy with **PG as the sole source of truth**; derives three CRs and reads their status back | `MLRun` + `MLService` + `MLTrafficPolicy`, PG, namespace-partitioned |
| **[artifact-hub](artifact-hub/)** | REST (business) | Artifact registry with a metadata (PG) / storage (zot · RustFS) split | Artifact tuple `(namespace, kind, name, version)` |
| **[tenant-operator](tenant-operator/)** | Operator | Reconciles the `Tenant` CR into Namespace / `ElasticQuota` / per-tenant Secret·CM·SA·RBAC; reflects `status.used` | `Tenant` CR (cluster-scoped) |
| **[compute-operator](compute-operator/)** | Operator | dispatcher + handler routing the three CRs by `spec.backend.{name,engine}` into K8s & gateway resources | three CRs + backend handler registry |

The compute-operator's backend handlers route each CR's `(name, engine)` tuple — `native` (Job / Deployment / StatefulSet + gang-scheduled `PodGroup`, `HTTPRoute`), `kubeflow-trainer` (PyTorchJob / TFJob / MPIJob), `kserve` (`InferenceService`), or a `custom` GVK.

## Layer invariants

- **PG is authoritative, CRs are derived.** compute-service / artifact-hub treat PostgreSQL as the business authority and emit CRs as derived artifacts (compute uses an embedded Outbox + reconciler for strong consistency). cluster-manager is the exception — it has no PG and reads/writes `Tenant` / `ResourcePool` CRs on etcd directly over REST.
- **Operators consume CRs one-way.** An operator reads `spec`, writes `status`, and never writes back to upstream PG; operators don't know about each other (tenant-operator never reads `MLRun`/`MLService`; compute-operator never reads `Tenant`/`ElasticQuota` — it only passes the quota name through).
- **Read/write paths converge through etcd.** cluster-manager writes ResourcePool / Tenant CRs; compute-service (via Informer) and tenant-operator read them directly — no direct service-to-service calls between them.
- **Tenant scope ≠ landing namespace.** The `namespace` field on compute/artifact records is the tenant scope; the K8s Namespace is `Tenant.spec.namespace.name`, which several Tenants may share.
- **All derived Pods route through koord-scheduler** with an ElasticQuota label — there is no quota-bypassing path.

## Multi-module workspace

Each component is its own Go module with a sibling `test/integration/` submodule (keeps test-only deps out of the production `go.mod` / Dockerfile). Shared helpers live in `test/testutil/`; vendored upstream CRDs in `test/crds/external/`; the envtest binary in `test/setup-envtest/`.

```
tenant-operator/   compute-operator/   cluster-manager/   compute-service/   artifact-hub/
  └── test/integration/  (one per component, separate module)
test/{testutil,crds/external,setup-envtest}/   Shared test infrastructure
docs/system_design/    Per-component design docs + database.md + overview.md
deploy/helm/           The "axisml-system" chart (CRDs + all five components)
```

`go test ./...` from this dir won't traverse every module — use the Makefile.

## Build / test

The layer Makefile generates per-component targets from its `COMPONENTS` list (`tenant-operator compute-operator cluster-manager compute-service artifact-hub`); the three API services in `DOC_COMPONENTS` (`cluster-manager compute-service artifact-hub`) also get doc targets.

```sh
make help                         # list all targets
make build                        # build all five component binaries
make test                         # unit tests across every component (no Docker)
make integration                  # integration tests (envtest + testcontainers; Docker required)
make doc-gen / make doc-test      # regenerate / verify every OpenAPI spec
make image / make image-load      # build (and load into minikube) every component image

# Per-component shortcuts, e.g.:
make compute-service-test
make cluster-manager-doc-gen
make artifact-hub-integration
```

OpenAPI specs under `docs/apis/<component>.yaml` are **generated from Go DTOs, never hand-edited** — run `make <component>-doc-gen` after changing a handler signature or DTO.

## Deployment

The System chart installs **after** infra and **before** platform. It consumes the infra DB cross-namespace (`axisml-database.axisml-infra:5432`) and ships no PostgreSQL of its own.

```sh
make helm-lint        # lint the chart
make helm-template    # render locally for review
make helm-install     # apply CRDs (via helm-crds) then install/upgrade the release
make helm-uninstall   # tear down
```

`make helm-install` depends on `helm-crds`, which `kubectl apply`s `deploy/helm/crds/` directly — Helm only installs `crds/` on first install, so this picks up CRD schema upgrades. The chart's `appVersion` is the **single image-tag authority** across all three layer charts.

## See also

- **[System design overview](docs/system_design/overview.md)** + per-component docs and **[database.md](docs/system_design/database.md)**
- **[High-level design](../docs/high_level_design.md)** — system-level invariants and the three-layer architecture
- **[Development workflow](../docs/development_workflow.md)** — setup, build/test, testing layers
- **[Deployment manual](../docs/deployment.md)**
