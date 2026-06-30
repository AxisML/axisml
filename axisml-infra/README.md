# axisml-infra

The **Infra layer** of AxisML — the third-party infrastructure substrate the rest of the platform stands on. It is the bottom of the three deployment layers (**infra → system → platform**) and the only one composed entirely of upstream open-source components plus the glue resources (`Gateway`, `HTTPRoute`, `Secret`, `ConfigMap`, `ServiceAccount`) that wire them together.

The design unit here is a **capability the platform needs**, each satisfied by a mature OSS component that is a replaceable implementation detail — not the design itself. Infra carries no business logic.

## Capabilities

| Capability | Implementation | Notes |
| --- | --- | --- |
| Service gateway | [Envoy Gateway](https://gateway.envoyproxy.io/) | Single `axisml-gateway`; CRD-driven `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy`; native gRPC/HTTP2; the platform's only north-south ingress |
| Object storage | [RustFS](https://rustfs.dev/) | S3-compatible (Apache 2.0); clients read/write directly via short-lived creds — never proxied through a service |
| OCI registry | [zot](https://zotregistry.dev/) | OCI Distribution v2 + artifact manifests for model weights & images; content-addressable `@digest` refs |
| Database | **PostgreSQL** (bitnami sub-chart) | The single metadata DB for the whole platform; lives here, consumed cross-namespace by the System layer |
| Cache | Redis (bitnami sub-chart) | Optional hot-read acceleration; **non-authoritative** — callers fall back to the source DB when it's unreachable |
| Accelerator management | [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) | Exposes `nvidia.com/gpu`, node labels, DCGM Exporter |
| Scheduling & quota | axisml-scheduler (self-built, on [scheduler-plugins](https://github.com/kubernetes-sigs/scheduler-plugins)) | `axisml-scheduler` + `ElasticQuota` (elastic multi-tenant quota) + `PodGroup` (gang scheduling) |
| Monitoring | [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) | Prometheus + Grafana + AlertManager; auto-discovers `ServiceMonitor` / `PodMonitor` |

Infra exposes standard protocols and does **not** embed a tenant model — isolation (bucket/prefix, repo path, table prefix, key prefix) is the caller's responsibility.

## Key invariants

- **No quota-bypassing scheduling path.** Any workload Pod admitted onto this infra MUST set `schedulerName: axisml-scheduler` and carry the label `scheduling.axisml.io/quota=<elastic-quota-name>`. A Pod without both is a quota-bypass bug. Third-party controllers that derive Pods (e.g. KServe) must pass both through.
- **Infra owns no quota CRs.** It ships zero `ElasticQuota` / `PodGroup` objects and holds no mutation rights over them — those are owned by the CR owners in the System layer.
- **PostgreSQL lives here.** The System layer reaches it cross-namespace at `axisml-database.axisml-infra:5432`; Redis at `axisml-redis-master.axisml-infra:6379`. The shared `database.auth.password` / `cache.auth.password` inputs must match what each consuming service renders into its own namespace-scoped Secret.
- **Infra's own Pods use the default scheduler** — they do not set `schedulerName` and do not consume any ElasticQuota.

## Layout

```
deploy/helm/                Umbrella chart "axisml-infra" (version 0.1.0)
  ├── Chart.yaml            Sub-chart dependencies (gateway-helm, rustfs, zot,
  │                         gpu-operator, axisml-scheduler, kube-prometheus-stack)
  ├── values.yaml           Capability toggles + shared DB/cache credentials
  └── templates/
      ├── gateway.yaml      The single axisml-gateway + GatewayClass
      └── NOTES.txt
docs/system_design/overview.md   Capability-by-capability design (contracts & rationale)
scripts/minikube.sh        Local cluster lifecycle helper
```

> PostgreSQL and Redis are pulled in as bitnami sub-charts; the GPU Operator, Envoy Gateway, RustFS, zot, and kube-prometheus-stack come in as their upstream charts pinned in `Chart.yaml`. axisml-scheduler is a first-party component templated directly (not a sub-chart).

## Local cluster

```sh
make help            # list all targets
make cluster-up      # create + start the local minikube cluster (profile "axisml")
make cluster-status  # show cluster status
make cluster-down    # stop the cluster (preserves state)
make cluster-delete  # destroy the cluster entirely
```

## Deployment

This is the **first** chart installed (infra → system → platform) and the **last** uninstalled.

```sh
make helm-deps       # fetch sub-chart tarballs (run once / after Chart.yaml changes)
make helm-lint       # lint the chart
make helm-template   # render locally for review
make helm-install    # install or upgrade the axisml-infra release (idempotent)
make helm-uninstall  # tear down
```

The release installs into the `axisml-infra` namespace. The root `make cluster-up` / `make helm-install` orchestrate this layer together with System and Platform in the correct order.

## See also

- **[Infra design](docs/system_design/overview.md)** — the per-capability contracts, technology selection, and deployment modes
- **[Deployment manual](../docs/deployment.md)** — chart organization, namespaces, install order, external-DB/RDS modes
- **[High-level design](../docs/high_level_design.md)** — where Infra sits in the three-layer architecture
