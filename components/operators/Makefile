# operators Makefile
#
# Builds the shared axisml-operator image. The image contains the tenant,
# MLJob, and MLService operator binaries; entrypoint.sh selects which one to
# execute.

IMAGE_REPO ?= ghcr.io/axisml/axisml/axisml-operator
# Must track deploy/helm/axisml-system Chart.appVersion. The top-level
# Makefile exports IMAGE_TAG from the chart for repo-wide invocations.
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

REPO_ROOT := $(abspath $(CURDIR)/../..)
OPERATORS := tenant-operator mljob-operator mlservice-operator

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile all operator binaries
	@set -e; for op in $(OPERATORS); do \
	  printf '\n>>> components/operators/%s (build)\n' "$$op"; \
	  $(MAKE) -C $(CURDIR)/$$op build; \
	done

.PHONY: image
image: ## Build the shared operator container image ($(IMAGE))
	@docker build -f $(CURDIR)/Dockerfile -t $(IMAGE) $(REPO_ROOT)

.PHONY: image-load-minikube
image-load-minikube: image ## Build and load the shared image into the local minikube node
	@minikube image load $(IMAGE)

.PHONY: test
test: ## Run unit tests for all operators
	@set -e; for op in $(OPERATORS); do \
	  printf '\n>>> components/operators/%s (test)\n' "$$op"; \
	  $(MAKE) -C $(CURDIR)/$$op test; \
	done

.PHONY: envtest
envtest: ## Run L1 envtest for all operators
	@set -e; for op in $(OPERATORS); do \
	  printf '\n>>> components/operators/%s (envtest)\n' "$$op"; \
	  $(MAKE) -C $(CURDIR)/$$op envtest; \
	done

.PHONY: fmt vet tidy
fmt: ## Run go fmt for all operators
	@set -e; for op in $(OPERATORS); do $(MAKE) -C $(CURDIR)/$$op fmt; done

vet: ## Run go vet for all operators
	@set -e; for op in $(OPERATORS); do $(MAKE) -C $(CURDIR)/$$op vet; done

tidy: ## Run go mod tidy for all operators
	@set -e; for op in $(OPERATORS); do $(MAKE) -C $(CURDIR)/$$op tidy; done

.PHONY: coverage envtest-coverage coverage-html
coverage: ## Run unit coverage for all operators
	@set -e; for op in $(OPERATORS); do \
	  printf '\n>>> components/operators/%s (coverage)\n' "$$op"; \
	  $(MAKE) -C $(CURDIR)/$$op coverage; \
	done

envtest-coverage: ## Run L1 envtest coverage for all operators
	@set -e; for op in $(OPERATORS); do \
	  printf '\n>>> components/operators/%s (envtest-coverage)\n' "$$op"; \
	  $(MAKE) -C $(CURDIR)/$$op envtest-coverage; \
	done

coverage-html: ## Render coverage HTML for all operators
	@set -e; for op in $(OPERATORS); do $(MAKE) -C $(CURDIR)/$$op coverage-html; done

.PHONY: clean
clean: ## Remove shared and per-operator build artifacts
	@rm -rf coverage/
	@set -e; for op in $(OPERATORS); do $(MAKE) -C $(CURDIR)/$$op clean; done
