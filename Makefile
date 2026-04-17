# AxisML Makefile

# --- Cluster Configuration ---
export MINIKUBE_PROFILE ?= axisml
export MINIKUBE_CPUS    ?= 4
export MINIKUBE_MEMORY  ?= 4096
export MINIKUBE_DISK    ?= 20g
export K8S_VERSION      ?=

# --- Cluster Management ---

.PHONY: cluster-up cluster-down cluster-delete cluster-status

cluster-up: ## Create and start the local Kubernetes cluster
	@bash scripts/minikube.sh up

cluster-down: ## Stop the cluster (preserves state)
	@bash scripts/minikube.sh down

cluster-delete: ## Destroy the cluster entirely
	@bash scripts/minikube.sh delete

cluster-status: ## Show cluster status
	@bash scripts/minikube.sh status

# --- Help ---

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
