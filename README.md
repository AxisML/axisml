# AxisML

A unified machine learning platform with native support for distributed training, intelligent resource scheduling, and elastic scaling. Manages the full model lifecycle from development and training to inference and operations.

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
make helm-install-infra    # helm-install-system, helm-upgrade-infra, ...

# Render both charts locally (dry run)
make helm-template

# Uninstall (system first, then infra)
make helm-uninstall
```

## Documentation

- [System Design Overview](docs/system_design/overview.md)
- [Local Development Setup](docs/development/local-setup.md)
