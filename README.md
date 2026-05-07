<p align="center">
  <img src="docs/assets/banner.png" alt="AxisML" width="720">
</p>

AxisML is a machine learning platform with native support for distributed training, intelligent resource scheduling, and elastic scaling. Manages the full model lifecycle from development and training to inference and operations.

## Quick Start

### Set up local development cluster

```bash
# Start a local Kubernetes cluster (requires Docker Desktop and minikube)
make cluster-up

# Check cluster status
make cluster-status

# See all available commands
make help
```

For detailed setup instructions, see [Local Development Environment Setup](docs/development/local-setup.md).

### Install AxisML services

AxisML ships as two Helm charts: `axisml-infra` (third-party infrastructure) and
`axisml-system` (control plane + metadata DB). Install infra first.

```bash
# Install both (infra then system)
make helm-install

# Or install/upgrade one layer at a time
make helm-install-infra    # also: helm-install-system / helm-upgrade-infra / helm-upgrade-system

# Render both charts locally (dry run)
make helm-template

# Uninstall (system first, then infra)
make helm-uninstall
```

## Components

AxisML is a monorepo split into independent Go modules, deployed together via the `axisml-system` Helm chart.

**Control plane — admin domain**
- **[cluster-manager](docs/system_design/cluster-manager.md)** — stateless REST shell over the `Tenant` CR. No PG; Kubernetes etcd is the source of truth.
- **[tenant-operator](docs/system_design/tenant-operator.md)** — reconciles `Tenant` into Namespace, Koordinator `ElasticQuota`, and per-tenant Secret / ConfigMap / ServiceAccount / RBAC.

**Control plane — workload domain**
- **[compute](docs/system_design/compute.md)** — REST service for Job / Service / ResourcePool / ResourceUnit. Holds business metadata in PG and emits `MLJob` / `MLService` CRs. Partitioned by bare `namespace` string; does not know about tenants.
- **[compute-operator](docs/system_design/compute-operator.md)** — reconciles `MLJob` and `MLService` via a dispatcher + handler model. Backends include `native` (Job / Deployment / StatefulSet + scheduler-plugins `PodGroup`), `kubeflow-trainer` (PyTorchJob / TFJob / MPIJob / ...), `kserve` (`InferenceService`), and `custom`. All backend-derived Pods route through `koord-scheduler` for ElasticQuota enforcement.
- **[artifacts](docs/system_design/artifacts.md)** — registry for models, datasets, images, evaluation reports. Addressed by `(namespace, kind, name, version)`. PG is the metadata authority; bytes live in zot (OCI) and RustFS (S3) and are accessed directly by clients, not proxied.

**User-facing layer**
- **[platform](docs/system_design/platform.md)** — Go backend + React frontend. The only externally exposed entry point; orchestrates calls to cluster-manager / compute / artifacts. Currently a scaffold.

**Infrastructure** (separate `axisml-infra` chart): Envoy Gateway, RustFS, zot, Koordinator (ElasticQuota + koord-scheduler), GPU Operator, kube-prometheus-stack. See [infra design](docs/system_design/infra.md).

See the [System Design Overview](docs/system_design/overview.md) for how these fit together.

### Run tests

```bash
make test                # unit tests across all components
make integration-test    # integration tests (envtest + testcontainers, needs Docker)
```

See [Testing Guide](docs/development/testing.md) for the two-layer test pyramid.

## Documentation

- [System Design Overview](docs/system_design/overview.md)
- Component designs: [tenant-operator](docs/system_design/tenant-operator.md) · [compute-operator](docs/system_design/compute-operator.md) · [cluster-manager](docs/system_design/cluster-manager.md) · [compute](docs/system_design/compute.md) · [artifacts](docs/system_design/artifacts.md) · [platform](docs/system_design/platform.md) · [infra](docs/system_design/infra.md)
- [Local Development Setup](docs/development/local-setup.md)
- [Testing Guide](docs/development/testing.md)
