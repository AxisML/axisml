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

```bash
# Install AxisML to the cluster via Helm
make helm-install

# Upgrade an existing installation
make helm-upgrade

# Render Helm templates locally (dry run)
make helm-template

# Uninstall AxisML from the cluster
make helm-uninstall
```

## Documentation

- [System Design Overview](docs/system_design/overview.md)
- [Local Development Setup](docs/development/local-setup.md)
