# compute

AxisML Compute is the internal compute service: it owns the business metadata for jobs, services, tenants, resource pools, resource units, and quotas, and translates user intent into `MLJob` / `MLService` / `Tenant` CRs for the operators to act on.

> **Status: scaffold.** The directory and Makefile are in place; the Go implementation is not yet committed. See [`docs/system_design/compute.md`](../../docs/system_design/compute.md) for the design and the API surface this service will expose.

## Responsibilities

- **Compute workloads** — maintain Job / Service business metadata; create `MLJob` / `MLService` CRs; consume status reflowed by the operators via informer.
- **Tenants** — maintain tenant metadata; emit `Tenant` CRs for tenant-operator to land Namespace / ResourceQuota / init resources.
- **Resource pools** — maintain `ResourcePool` metadata: node selectors, tolerations, cluster mapping.
- **Resource units** — maintain reusable resource templates (CPU/GPU/memory recipes); inject `requests`/`limits` and node-matching at workload submit time.
- **Quotas** — flat per-`(tenant, pool)` quota CRUD with `min`/`max`; sync 1:1 to namespace-scoped Koordinator `ElasticQuota` CRs; cache observed usage.

Compute does not directly create Pods, Deployments, or PodGroups — those are produced by operators or downstream controllers from the CRs it emits.

## Planned layout

```
cmd/                Service entrypoint (HTTP / gRPC server, informer, K8s client wiring)
internal/
  ├── tenant/          Tenant CRUD + Tenant CR sync
  ├── resourcepool/    ResourcePool metadata
  ├── resourceunit/    ResourceUnit templates + injection helpers
  ├── quota/           Quota model + ElasticQuota sync + usage reflow
  ├── job/             MLJob CRUD + status informer
  ├── service/         MLService CRUD + status informer
  └── volume/          Data volume management (TBD)
api/                  HTTP / gRPC contract definitions
deploy/Dockerfile     Container image build (to be added)
```

## Local development

```sh
make help            # list all targets
make / make build    # compile bin/compute
make test            # unit tests
make image           # docker build → axisml/compute:0.1.0
make clean           # remove build artifacts
```

`IMAGE_TAG` defaults to `0.1.0` and must track the `appVersion` in [`deploy/helm/axisml-system/Chart.yaml`](../../deploy/helm/axisml-system/Chart.yaml).

## Deployment

Compute will ship as part of the `axisml-system` chart under `deploy/helm/axisml-system/templates/compute/`.
