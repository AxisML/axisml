# AxisML Makefile

# --- Cluster Configuration ---
export MINIKUBE_PROFILE ?= axisml
export MINIKUBE_CPUS    ?= 4
export MINIKUBE_MEMORY  ?= 4096
export MINIKUBE_DISK    ?= 20g
export K8S_VERSION      ?=
export MINIKUBE_DRIVER  ?=

# --- Helm Configuration ---
HELM_RELEASE   ?= axisml
HELM_NAMESPACE ?= axisml-system
HELM_CHART     ?= deploy/helm/axisml

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

.PHONY: helm-install helm-upgrade helm-uninstall helm-template

helm-install: ## Install AxisML to the cluster
	@helm install $(HELM_RELEASE) $(HELM_CHART) -n $(HELM_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE)

helm-upgrade: ## Upgrade AxisML deployment
	@helm upgrade $(HELM_RELEASE) $(HELM_CHART) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-uninstall: ## Uninstall AxisML from the cluster
	@helm uninstall $(HELM_RELEASE) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-template: ## Render Helm templates locally
	@helm template $(HELM_RELEASE) $(HELM_CHART)

# --- Help ---

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
