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

## Documentation

- [System Design Overview](docs/system_design/overview.md)
- [Local Development Setup](docs/development/local-setup.md)
