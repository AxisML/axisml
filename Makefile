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
.PHONY: helm-crds-system

helm-deps: ## Fetch sub-chart tarballs for both charts (run after clone / Chart.yaml change)
	@helm dependency update $(HELM_INFRA_CHART)
	@helm dependency update $(HELM_SYSTEM_CHART)

helm-install-infra: ## Install or upgrade AxisML infrastructure (idempotent)
	@helm upgrade --install $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE)

helm-crds-system: ## Apply axisml-system CRDs (Helm only installs files under crds/ once; this picks up schema upgrades)
	@kubectl --context $(MINIKUBE_PROFILE) apply -f $(HELM_SYSTEM_CHART)/crds/

helm-install-system: helm-crds-system ## Install or upgrade AxisML control plane (idempotent)
	@helm upgrade --install $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE)

helm-install: helm-install-infra helm-install-system ## Install or upgrade infra + control plane

helm-upgrade-infra: ## Upgrade AxisML infrastructure (must already be installed)
	@helm upgrade $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-upgrade-system: helm-crds-system ## Upgrade AxisML control plane (must already be installed)
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

# --- Operators ---

TENANT_OPERATOR_DIR ?= components/operators/tenant-operator
MLSERVICE_OPERATOR_DIR ?= components/operators/mlservice-operator
MLSERVICE_OPERATOR_IMG ?= axisml/mlservice-operator:dev

MLJOB_OPERATOR_DIR ?= components/operators/mljob-operator
# Default tag tracks the axisml-system chart's appVersion so the
# image we build locally matches the tag the Helm chart renders by
# default (Chart.AppVersion via the axisml.image helper). Override
# MLJOB_OPERATOR_IMAGE_TAG to push a non-release build.
MLJOB_OPERATOR_IMAGE_TAG ?= $(shell awk '/^appVersion:/{gsub(/"/,"",$$2);print $$2}' deploy/helm/axisml-system/Chart.yaml)
MLJOB_OPERATOR_IMAGE     ?= axisml/mljob-operator:$(MLJOB_OPERATOR_IMAGE_TAG)

.PHONY: tenant-operator-build tenant-operator-test tenant-operator-image tenant-operator-image-load
.PHONY: mlservice-operator-build mlservice-operator-test mlservice-operator-image mlservice-operator-image-load
.PHONY: mljob-operator-tidy mljob-operator-generate mljob-operator-build mljob-operator-test mljob-operator-image mljob-operator-image-load

tenant-operator-build: ## Compile the tenant-operator binary
	@$(MAKE) -C $(TENANT_OPERATOR_DIR) build

tenant-operator-test: ## Run tenant-operator unit tests (no cluster required)
	@$(MAKE) -C $(TENANT_OPERATOR_DIR) test

tenant-operator-image: ## Build the tenant-operator container image
	@$(MAKE) -C $(TENANT_OPERATOR_DIR) image

tenant-operator-image-load: ## Build and load the tenant-operator image into minikube
	@$(MAKE) -C $(TENANT_OPERATOR_DIR) image-load-minikube

mlservice-operator-build: ## Compile the mlservice-operator binary
	@cd $(MLSERVICE_OPERATOR_DIR) && go build ./...

mlservice-operator-test: ## Run mlservice-operator unit tests (no cluster required)
	@cd $(MLSERVICE_OPERATOR_DIR) && go test ./...

mlservice-operator-image: ## Build the mlservice-operator container image
	@docker build -t $(MLSERVICE_OPERATOR_IMG) $(MLSERVICE_OPERATOR_DIR)

mlservice-operator-image-load: mlservice-operator-image ## Build and load the mlservice-operator image into minikube
	@minikube -p $(MINIKUBE_PROFILE) image load $(MLSERVICE_OPERATOR_IMG)

mljob-operator-tidy: ## Resolve Go module deps for mljob-operator
	@$(MAKE) -C $(MLJOB_OPERATOR_DIR) tidy

mljob-operator-generate: ## Run controller-gen for deepcopy code
	@$(MAKE) -C $(MLJOB_OPERATOR_DIR) generate

mljob-operator-build: ## Compile the mljob-operator binary
	@$(MAKE) -C $(MLJOB_OPERATOR_DIR) build

mljob-operator-test: ## Run mljob-operator unit tests (no cluster required)
	@$(MAKE) -C $(MLJOB_OPERATOR_DIR) test

mljob-operator-image: ## Build the mljob-operator container image
	@docker build -t $(MLJOB_OPERATOR_IMAGE) -f $(MLJOB_OPERATOR_DIR)/Dockerfile $(MLJOB_OPERATOR_DIR)

mljob-operator-image-load: mljob-operator-image ## Build and load the mljob-operator image into minikube
	@minikube image load $(MLJOB_OPERATOR_IMAGE) -p $(MINIKUBE_PROFILE)

# --- Aggregated test targets ---

.PHONY: test integration-test

test: tenant-operator-test mlservice-operator-test mljob-operator-test ## Run unit tests across all operators (no cluster required)

integration-test: ## Run all integration tests against the running cluster (requires `make cluster-up && make helm-install-infra`; in-cluster operators must be scaled to 0)
	@$(MAKE) -C $(TENANT_OPERATOR_DIR) test-integration

# --- Help ---

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
