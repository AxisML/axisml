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

# Image tag authority mirrors the System chart's appVersion so infra images line
# up with what the charts pull, even when this layer is built standalone.
IMAGE_TAG       ?= $(shell awk '/^appVersion:/{gsub(/"/,"",$$2);print $$2}' $(CURDIR)/../axisml-system/deploy/helm/Chart.yaml)
IMAGE_REGISTRY  ?=
REPO_ROOT       := $(CURDIR)/..
SCHED_DIR       := $(CURDIR)/axisml-scheduler
SCHED_IMAGE     := $(IMAGE_REGISTRY)axisml/axisml-scheduler

.DEFAULT_GOAL := help

# --- axisml-scheduler (first-party Go component) ---
# The scheduler is a controller/plugin binary with no HTTP API surface, so this
# layer has no doc-gen/doc-test targets (the root doc aggregates skip infra).
.PHONY: build test coverage coverage-html fmt vet tidy clean image image-load
build: ## Build the axisml-scheduler binaries
	@cd $(SCHED_DIR) && go build ./...
test: ## Unit-test the axisml-scheduler module
	@cd $(SCHED_DIR) && go test ./...
coverage: ## axisml-scheduler unit coverage
	@mkdir -p $(SCHED_DIR)/coverage && cd $(SCHED_DIR) && go test -coverprofile=coverage/coverage.out -covermode=atomic -count=1 ./...
coverage-html: ## Render axisml-scheduler HTML coverage
	@if [ -f $(SCHED_DIR)/coverage/coverage.out ]; then cd $(SCHED_DIR) && go tool cover -html=coverage/coverage.out -o coverage/coverage.html && echo "axisml-scheduler unit: $(SCHED_DIR)/coverage/coverage.html"; fi
fmt: ## gofmt the axisml-scheduler module
	@cd $(SCHED_DIR) && gofmt -w .
vet: ## go vet the axisml-scheduler module
	@cd $(SCHED_DIR) && go vet ./...
tidy: ## go mod tidy the axisml-scheduler module
	@cd $(SCHED_DIR) && go mod tidy
clean: ## Remove build + coverage artifacts
	@cd $(SCHED_DIR) && rm -rf bin coverage
image: ## Build the axisml-scheduler image (scheduler + controller)
	@docker build -f $(SCHED_DIR)/Dockerfile -t $(SCHED_IMAGE):$(IMAGE_TAG) $(REPO_ROOT)
image-load: image ## Build + load the image into minikube
	@minikube -p $(MINIKUBE_PROFILE) image load $(SCHED_IMAGE):$(IMAGE_TAG)

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
.PHONY: helm-deps helm-lint helm-template helm-crds helm-install helm-upgrade helm-uninstall
helm-deps: ## Fetch sub-chart tarballs
	@helm dependency update $(CHART)
helm-lint: ## Lint the infra chart
	@helm lint $(CHART)
helm-template: ## Render the infra chart locally
	@helm template $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE)
helm-crds: ## Apply infra CRDs (Helm only installs crds/ on first install, not upgrade)
	@kubectl --context $(MINIKUBE_PROFILE) apply -f $(CHART)/crds/
helm-install: helm-crds ## Install or upgrade the infra release (idempotent)
	@kubectl --context $(MINIKUBE_PROFILE) create namespace $(HELM_NAMESPACE) --dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -
	@kubectl --context $(MINIKUBE_PROFILE) label namespace $(HELM_NAMESPACE) app.kubernetes.io/managed-by=Helm --overwrite
	@kubectl --context $(MINIKUBE_PROFILE) annotate namespace $(HELM_NAMESPACE) meta.helm.sh/release-name=$(HELM_RELEASE) meta.helm.sh/release-namespace=$(HELM_NAMESPACE) --overwrite
	@helm upgrade --install $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE) $(HELM_EXTRA_ARGS)
helm-upgrade: helm-crds ## Upgrade the infra release (must already be installed)
	@helm upgrade $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE) $(HELM_EXTRA_ARGS)
helm-uninstall: ## Uninstall the infra release
	@helm uninstall $(HELM_RELEASE) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)
