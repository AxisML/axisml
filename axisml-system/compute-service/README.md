# compute-service

AxisML Compute Service is the internal compute service: it owns the business metadata for jobs, services, tenants, resource pools, resource units, and quotas, and translates user intent into `MLRun` / `MLService` / `Tenant` CRs for the operators to act on.

> **Status: scaffold.** The directory and Makefile are in place; the Go implementation is not yet committed. See [`axisml-system/docs/system_design/compute-service.md`](../docs/system_design/compute-service.md) for the design and the API surface this service will expose.

## Responsibilities

- **Compute workloads** — maintain Job / Service business metadata; create `MLRun` / `MLService` CRs; consume status reflowed by the operators via informer.
- **Tenants** — maintain tenant metadata; emit `Tenant` CRs for tenant-operator to land Namespace / ResourceQuota / init resources.
- **Resource pools** — resolve `ResourcePool` node selectors, optional capacity overrides, and cluster mapping.
- **Resource units** — maintain reusable resource templates (CPU/GPU/memory recipes); inject `requests`/`limits` and node-matching at workload submit time.
- **Quotas** — flat per-`(tenant, pool)` quota CRUD with `min`/`max`; sync 1:1 to namespace-scoped `ElasticQuota` CRs; cache observed usage.

compute-service does not directly create Pods, Deployments, or PodGroups — those are produced by operators or downstream controllers from the CRs it emits.

## Planned layout

```
cmd/                Service entrypoint (HTTP / gRPC server, informer, K8s client wiring)
internal/
  ├── tenant/          Tenant CRUD + Tenant CR sync
  ├── resourcepool/    ResourcePool metadata
  ├── resourceunit/    ResourceUnit templates + injection helpers
  ├── quota/           Quota model + ElasticQuota sync + usage reflow
  ├── mlrun/           MLRun CRUD + status informer
  ├── mlservice/       MLService CRUD + status informer
  └── volume/          Data volume management (TBD)
api/                  HTTP / gRPC contract definitions
deploy/Dockerfile     Container image build (to be added)
```

## Local development

```sh
make help            # list all targets
make / make build    # compile bin/compute-service
make test            # unit tests
make image           # docker build -> ghcr.io/axisml/axisml-compute-service:0.1.0
make openapi         # regenerate axisml-system/docs/apis/compute-service.yaml
make clean           # remove build artifacts
```

The OpenAPI 3.0 description of the HTTP API lives at [`axisml-system/docs/apis/compute-service.yaml`](../docs/apis/compute-service.yaml). It is generated from the same Go request/response structs the runtime handlers use; regenerate via `make openapi` after touching any handler signature, route, or `*Input` / `View` struct.

`IMAGE_TAG` defaults to `0.1.0` and must track the `appVersion` in [`axisml-system/deploy/helm/Chart.yaml`](../../axisml-system/deploy/helm/Chart.yaml).

## Deployment

compute-service ships as part of the `axisml-system` chart under `axisml-system/deploy/helm/templates/compute-service/`.
