# Local Development Environment Setup

This guide walks you through setting up a local Kubernetes cluster for AxisML development using minikube.

## 1. Prerequisites

### System Requirements

- macOS (Intel or Apple Silicon)
- At least 16 GB RAM recommended (8 GB allocated to minikube)
- 40 GB free disk space

### Required Tools

Install the following tools via [Homebrew](https://brew.sh/):

| Tool | Install Command | Verify |
|------|----------------|--------|
| Docker Desktop | `brew install --cask docker` | `docker --version` |
| minikube | `brew install minikube` | `minikube version` |
| kubectl | `brew install kubectl` | `kubectl version --client` |
| Helm | `brew install helm` | `helm version` |
| Go 1.22+ | `brew install go` | `go version` |

> **Note:** Make sure Docker Desktop is running before proceeding.

## 2. Starting the Local Kubernetes Cluster

### Quick Start

```bash
make cluster-up
```

This creates a minikube cluster named `axisml` with the following defaults:

| Setting | Default | Override |
|---------|---------|---------|
| Profile | `axisml` | `MINIKUBE_PROFILE` |
| CPUs | 4 | `MINIKUBE_CPUS` |
| Memory | 4096 MB | `MINIKUBE_MEMORY` |
| Disk | 20 GB | `MINIKUBE_DISK` |
| Kubernetes |  | `K8S_VERSION` |
| Driver | docker | `MINIKUBE_DRIVER` |

To customize, pass variables to `make`:

```bash
make cluster-up MINIKUBE_CPUS=6 MINIKUBE_MEMORY=16384
```

### Enabled Addons

The following minikube addons are automatically enabled:

- **ingress** — NGINX ingress controller for HTTP routing
- **metrics-server** — Enables `kubectl top` and HPA autoscaling
- **storage-provisioner** — Dynamic PV provisioning for StatefulSets
- **dashboard** — Kubernetes web UI for debugging

### Verifying the Cluster

```bash
# Check cluster status
make cluster-status

# Verify nodes are ready
kubectl get nodes

# Verify system pods are running
kubectl get pods -A
```

### Stopping and Deleting

```bash
# Stop the cluster (preserves all data)
make cluster-down

# Destroy the cluster entirely
make cluster-delete
```

### Accessing the Dashboard

```bash
minikube dashboard -p axisml
```

This opens the Kubernetes Dashboard in your browser.

## 3. Available Make Targets

Run `make help` to see all available targets:

```
cluster-delete       Destroy the cluster entirely
cluster-down         Stop the cluster (preserves state)
cluster-status       Show cluster status
cluster-up           Create and start the local Kubernetes cluster
help                 Show this help message
```

## 4. Troubleshooting

### Docker Desktop is not running

```
ERROR: Docker daemon is not running. Please start Docker Desktop first.
```

Open Docker Desktop and wait for it to fully start before running `make cluster-up`.

### Insufficient resources

If minikube fails to start or pods are stuck in `Pending`:

1. Open Docker Desktop → Settings → Resources
2. Ensure at least **10 GB memory** and **4 CPUs** are allocated
3. Restart Docker Desktop, then retry `make cluster-up`

### Apple Silicon compatibility

The `docker` driver works on both Intel and Apple Silicon Macs. If you encounter image pull issues, some ML-related images may only provide `amd64` builds. minikube will use Rosetta 2 emulation automatically in most cases.

### Cluster won't start after macOS update

```bash
make cluster-delete
make cluster-up
```

A fresh cluster usually resolves post-update issues.

### Port conflicts

If port 80/443 is already in use (preventing the ingress addon), stop any local web servers or proxies before starting the cluster.
