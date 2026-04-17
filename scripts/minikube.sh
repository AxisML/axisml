#!/usr/bin/env bash
set -euo pipefail

# AxisML local Kubernetes cluster management via minikube.
# Usage: minikube.sh {up|down|delete|status}

MINIKUBE_PROFILE="${MINIKUBE_PROFILE:-axisml}"
MINIKUBE_CPUS="${MINIKUBE_CPUS:-4}"
MINIKUBE_MEMORY="${MINIKUBE_MEMORY:-4096}"
MINIKUBE_DISK="${MINIKUBE_DISK:-20g}"
K8S_VERSION="${K8S_VERSION:-v1.32.3}"
MINIKUBE_DRIVER="${MINIKUBE_DRIVER:-docker}"

ADDONS=(ingress metrics-server storage-provisioner dashboard)

info()  { echo "==> $*"; }
error() { echo "ERROR: $*" >&2; exit 1; }

check_prerequisites() {
    local missing=()
    for cmd in minikube kubectl; do
        if ! command -v "$cmd" &>/dev/null; then
            missing+=("$cmd")
        fi
    done

    if [[ "$MINIKUBE_DRIVER" == "docker" ]]; then
        if ! command -v docker &>/dev/null; then
            missing+=("docker")
        fi
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        error "Missing required tools: ${missing[*]}. Please install them first (see docs/development/local-setup.md)."
    fi

    if [[ "$MINIKUBE_DRIVER" == "docker" ]] && ! docker info &>/dev/null; then
        error "Docker daemon is not running. Please start Docker Desktop first."
    fi
}

cluster_exists() {
    minikube profile list -o json 2>/dev/null | grep -q "\"Name\":\"${MINIKUBE_PROFILE}\"" 2>/dev/null
}

cmd_up() {
    check_prerequisites

    if cluster_exists; then
        local status
        status=$(minikube status -p "$MINIKUBE_PROFILE" -f '{{.Host}}' 2>/dev/null || true)
        if [[ "$status" == "Running" ]]; then
            info "Cluster '${MINIKUBE_PROFILE}' is already running."
            return 0
        fi
        info "Starting existing cluster '${MINIKUBE_PROFILE}'..."
        minikube start \
            -p "$MINIKUBE_PROFILE" \
            --cpus="$MINIKUBE_CPUS" \
            --memory="$MINIKUBE_MEMORY" \
            --disk-size="$MINIKUBE_DISK"
    else
        info "Creating cluster '${MINIKUBE_PROFILE}'..."
        minikube start \
            -p "$MINIKUBE_PROFILE" \
            --driver="$MINIKUBE_DRIVER" \
            --cpus="$MINIKUBE_CPUS" \
            --memory="$MINIKUBE_MEMORY" \
            --disk-size="$MINIKUBE_DISK" \
            --kubernetes-version="$K8S_VERSION"

        info "Enabling addons: ${ADDONS[*]}"
        for addon in "${ADDONS[@]}"; do
            minikube addons enable "$addon" -p "$MINIKUBE_PROFILE"
        done
    fi

    echo ""
    info "Cluster '${MINIKUBE_PROFILE}' is ready!"
    echo "  kubectl context : ${MINIKUBE_PROFILE}"
    echo "  Kubernetes      : ${K8S_VERSION}"
    echo "  Dashboard       : Run 'minikube dashboard -p ${MINIKUBE_PROFILE}' to open"
    echo ""
}

cmd_down() {
    if ! cluster_exists; then
        info "Cluster '${MINIKUBE_PROFILE}' does not exist. Nothing to stop."
        return 0
    fi
    info "Stopping cluster '${MINIKUBE_PROFILE}'..."
    minikube stop -p "$MINIKUBE_PROFILE"
    info "Cluster stopped. Data is preserved. Run 'make cluster-up' to restart."
}

cmd_delete() {
    if ! cluster_exists; then
        info "Cluster '${MINIKUBE_PROFILE}' does not exist. Nothing to delete."
        return 0
    fi
    info "Deleting cluster '${MINIKUBE_PROFILE}'..."
    minikube delete -p "$MINIKUBE_PROFILE"
    info "Cluster deleted."
}

cmd_status() {
    if ! cluster_exists; then
        info "Cluster '${MINIKUBE_PROFILE}' does not exist."
        return 0
    fi

    echo "--- Cluster Status ---"
    minikube status -p "$MINIKUBE_PROFILE" || true
    echo ""
    echo "--- Enabled Addons ---"
    minikube addons list -p "$MINIKUBE_PROFILE" | grep STATUS && echo "" || true
    minikube addons list -p "$MINIKUBE_PROFILE" | grep "enabled" || true
    echo ""
    echo "--- Nodes ---"
    kubectl --context "$MINIKUBE_PROFILE" get nodes 2>/dev/null || echo "(cluster not reachable)"
}

case "${1:-}" in
    up)     cmd_up ;;
    down)   cmd_down ;;
    delete) cmd_delete ;;
    status) cmd_status ;;
    *)
        echo "Usage: $0 {up|down|delete|status}"
        echo ""
        echo "Commands:"
        echo "  up      Create and start the minikube cluster"
        echo "  down    Stop the cluster (preserves state)"
        echo "  delete  Destroy the cluster entirely"
        echo "  status  Show cluster status"
        exit 1
        ;;
esac
