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

# --- Image Tag (shared across components) ---
#
# Each component's Makefile defaults IMAGE_TAG to 0.1.0 and notes that it
# must track Chart.appVersion. We export the chart's appVersion here so any
# top-level invocation overrides that default and keeps every component's
# image tag aligned with what the Helm chart will pull. Override on the
# command line for non-release builds:
#   make image IMAGE_TAG=dev
export IMAGE_TAG ?= $(shell awk '/^appVersion:/{gsub(/"/,"",$$2);print $$2}' $(HELM_SYSTEM_CHART)/Chart.yaml)

##@ Cluster Management

.PHONY: cluster-up cluster-down cluster-delete cluster-status

cluster-up: ## Create and start the local Kubernetes cluster
	@bash scripts/minikube.sh up

cluster-down: ## Stop the cluster (preserves state)
	@bash scripts/minikube.sh down

cluster-delete: ## Destroy the cluster entirely
	@bash scripts/minikube.sh delete

cluster-status: ## Show cluster status
	@bash scripts/minikube.sh status

##@ Helm Management

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

# --- Components ---
#
# Components implementing the standard Makefile contract:
#   make build / make image / make image-load / make test / make clean
#
# Each entry is a directory that contains a Makefile honouring the contract.
# Add scaffolded components here as they ship working build targets.
COMPONENTS := \
  components/operators/tenant-operator \
  components/operators/mljob-operator \
  components/operators/mlservice-operator
# Scaffolded components (uncomment as they ship code):
# COMPONENTS += components/compute
# COMPONENTS += components/artifacts
# COMPONENTS += components/platform/backend
# COMPONENTS += components/platform/frontend

##@ Components — aggregate targets (fan out to every COMPONENT)
#
# `make build` / `make image` / `make image-load` / `make test` / `make clean`
# fan out to each component's sub-make. `image-load` maps to the per-component
# `image-load-minikube` target (the explicit name component Makefiles use).

# Helper: run a sub-make target across every COMPONENT, surfacing failures.
_RUN_COMPONENTS = set -e; for c in $(COMPONENTS); do \
	printf '\n>>> %s (%s)\n' "$$c" "$(1)"; \
	$(MAKE) -C $$c $(1); \
done

.PHONY: build image image-load test clean

build: ## Build every component (delegates to each component's `make build`)
	@$(call _RUN_COMPONENTS,build)

image: ## Build container images for every component
	@$(call _RUN_COMPONENTS,image)

image-load: ## Build images and load them into the local minikube node
	@$(call _RUN_COMPONENTS,image-load-minikube)

test: ## Run unit tests across every component (no cluster required)
	@$(call _RUN_COMPONENTS,test)

clean: ## Remove build artifacts across every component
	@$(call _RUN_COMPONENTS,clean)

# --- Per-component shortcut targets ---
#
# Generate `<basename>-build`, `<basename>-image`, `<basename>-image-load`,
# `<basename>-test`, `<basename>-clean` for each component listed above. For
# example: `make tenant-operator-image`, `make mljob-operator-test`.
#
# COMPONENT basenames must be unique. If you add a component whose basename
# would collide (e.g., `components/platform/backend` would clash with any
# other `backend`), give it a distinct directory name or rework the mapping.
define _COMPONENT_SHORTCUTS
.PHONY: $(notdir $1)-build $(notdir $1)-image $(notdir $1)-image-load $(notdir $1)-test $(notdir $1)-clean
$(notdir $1)-build:
	@$$(MAKE) -C $1 build
$(notdir $1)-image:
	@$$(MAKE) -C $1 image
$(notdir $1)-image-load:
	@$$(MAKE) -C $1 image-load-minikube
$(notdir $1)-test:
	@$$(MAKE) -C $1 test
$(notdir $1)-clean:
	@$$(MAKE) -C $1 clean
endef
$(foreach c,$(COMPONENTS),$(eval $(call _COMPONENT_SHORTCUTS,$(c))))

##@ Integration tests

.PHONY: integration-test

integration-test: ## Run all integration tests against the running cluster (requires `make cluster-up && make helm-install-infra`; in-cluster operators must be scaled to 0)
	@$(MAKE) -C components/operators/tenant-operator test-integration

##@ Help

.PHONY: help
help: ## Show this help message
	@awk 'BEGIN { \
	    FS = ":.*?##"; \
	    printf "\nUsage: make \033[36m<target>\033[0m\n"; \
	  } \
	  /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next } \
	  /^[a-zA-Z][a-zA-Z0-9_-]*:.*##/ { printf "  \033[36m%-26s\033[0m %s\n", $$1, $$2 }' \
	  $(MAKEFILE_LIST)
	@printf "\n\033[1mPer-component shortcuts (auto-generated)\033[0m\n"
	@printf "  Pattern : <component>-{build,image,image-load,test,clean}\n"
	@printf "  Active  : %s\n" "$(notdir $(COMPONENTS))"
	@printf "  Example : make tenant-operator-image  |  make mljob-operator-test\n\n"

.DEFAULT_GOAL := build
