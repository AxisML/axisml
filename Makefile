# AxisML Makefile

# --- Cluster Configuration ---
export MINIKUBE_PROFILE ?= axisml
export MINIKUBE_CPUS    ?= 4
export MINIKUBE_MEMORY  ?= 4096
export MINIKUBE_DISK    ?= 20g
export K8S_VERSION      ?=
export MINIKUBE_DRIVER  ?=

# --- Helm Configuration ---
HELM_INFRA_RELEASE    ?= axisml-infra
HELM_INFRA_NAMESPACE  ?= axisml-infra
HELM_INFRA_CHART      ?= deploy/helm/axisml-infra

HELM_SYSTEM_RELEASE   ?= axisml
HELM_SYSTEM_NAMESPACE ?= axisml-system
HELM_SYSTEM_CHART     ?= deploy/helm/axisml-system

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

# --- Helm Management ---

.PHONY: helm-install helm-upgrade helm-uninstall helm-template helm-deps
.PHONY: helm-install-infra helm-install-system
.PHONY: helm-upgrade-infra helm-upgrade-system
.PHONY: helm-uninstall-infra helm-uninstall-system

helm-deps: ## Fetch sub-chart tarballs for both charts (run after clone / Chart.yaml change)
	@helm dependency update $(HELM_INFRA_CHART)
	@helm dependency update $(HELM_SYSTEM_CHART)

helm-install-infra: ## Install or upgrade AxisML infrastructure (idempotent)
	@helm upgrade --install $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE)

helm-install-system: ## Install or upgrade AxisML control plane (idempotent)
	@helm upgrade --install $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE)

helm-install: helm-install-infra helm-install-system ## Install or upgrade infra + control plane

helm-upgrade-infra: ## Upgrade AxisML infrastructure (must already be installed)
	@helm upgrade $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-upgrade-system: ## Upgrade AxisML control plane (must already be installed)
	@helm upgrade $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-upgrade: helm-upgrade-infra helm-upgrade-system ## Upgrade both

helm-uninstall-system: ## Uninstall AxisML control plane
	@helm uninstall $(HELM_SYSTEM_RELEASE) -n $(HELM_SYSTEM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-uninstall-infra: ## Uninstall AxisML infrastructure
	@helm uninstall $(HELM_INFRA_RELEASE) -n $(HELM_INFRA_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-uninstall: helm-uninstall-system helm-uninstall-infra ## Uninstall control plane then infra

helm-template: ## Render both charts locally
	@helm template $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE)
	@helm template $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE)

# --- Help ---

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
