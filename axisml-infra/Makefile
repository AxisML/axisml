# axisml-infra Makefile — local cluster lifecycle + infra-layer Helm chart.
#
# Owns the minikube cluster (scripts/minikube.sh) and the infra Helm release
# (third-party infrastructure + PostgreSQL). Cross-layer install ordering is
# orchestrated by the repo-root Makefile.

export MINIKUBE_PROFILE ?= axisml
export MINIKUBE_CPUS    ?= 4
export MINIKUBE_MEMORY  ?= 8192
export MINIKUBE_DISK    ?= 20g
export K8S_VERSION      ?=
export MINIKUBE_DRIVER  ?=

CHART           := $(CURDIR)/deploy/helm
MINIKUBE        := $(CURDIR)/scripts/minikube.sh
HELM_RELEASE    ?= axisml-infra
HELM_NAMESPACE  ?= axisml-infra
HELM_EXTRA_ARGS ?=

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Cluster ---
.PHONY: cluster-up cluster-down cluster-delete cluster-status
cluster-up: ## Create and start the local minikube cluster
	@bash $(MINIKUBE) up
cluster-down: ## Stop the cluster (preserves state)
	@bash $(MINIKUBE) down
cluster-delete: ## Destroy the cluster entirely
	@bash $(MINIKUBE) delete
cluster-status: ## Show cluster status
	@bash $(MINIKUBE) status

# --- Helm (infra layer) ---
.PHONY: helm-deps helm-lint helm-template helm-install helm-upgrade helm-uninstall
helm-deps: ## Fetch sub-chart tarballs
	@helm dependency update $(CHART)
helm-lint: ## Lint the infra chart
	@helm lint $(CHART)
helm-template: ## Render the infra chart locally
	@helm template $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE)
helm-install: ## Install or upgrade the infra release (idempotent)
	@kubectl --context $(MINIKUBE_PROFILE) create namespace $(HELM_NAMESPACE) --dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -
	@kubectl --context $(MINIKUBE_PROFILE) label namespace $(HELM_NAMESPACE) app.kubernetes.io/managed-by=Helm --overwrite
	@kubectl --context $(MINIKUBE_PROFILE) annotate namespace $(HELM_NAMESPACE) meta.helm.sh/release-name=$(HELM_RELEASE) meta.helm.sh/release-namespace=$(HELM_NAMESPACE) --overwrite
	@helm upgrade --install $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE) $(HELM_EXTRA_ARGS)
helm-upgrade: ## Upgrade the infra release (must already be installed)
	@helm upgrade $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE) $(HELM_EXTRA_ARGS)
helm-uninstall: ## Uninstall the infra release
	@helm uninstall $(HELM_RELEASE) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)
