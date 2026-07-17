# AxisML root Makefile — thin orchestrator over the per-layer Makefiles.
#
# Each layer owns one Makefile with the real build/test/helm logic:
#   axisml-infra/Makefile      cluster lifecycle + infra Helm chart
#   axisml-system/Makefile     the 5 System components + System Helm + envtest
#   axisml-platform/Makefile   backend + frontend + Platform Helm
#
# This file delegates aggregate targets to those layers, enforces cross-layer
# install ordering, and keeps repo-level concerns (e2e, coverage merge, hooks).

# Single version authority: the System chart's appVersion. Exported so every
# layer's image tags line up with what the charts pull. Override per build:
#   make image IMAGE_TAG=dev
export IMAGE_TAG ?= $(shell awk '/^appVersion:/{gsub(/"/,"",$$2);print $$2}' axisml-system/deploy/helm/Chart.yaml)

# Cluster config (consumed by axisml-infra/scripts/minikube.sh and Helm).
export MINIKUBE_PROFILE ?= axisml
export MINIKUBE_CPUS    ?= 4
export MINIKUBE_MEMORY  ?= 8192
export MINIKUBE_DISK    ?= 20g
export K8S_VERSION      ?=
export MINIKUBE_DRIVER  ?=

# Layers that contain Go modules / build artifacts. axisml-infra hosts the
# first-party axisml-scheduler component; system + platform host the rest.
GO_LAYERS := axisml-infra axisml-system axisml-platform

# Layers that own generated OpenAPI specs (doc-gen/doc-test). axisml-infra is
# excluded: the axisml-scheduler is a controller/plugin with no HTTP surface.
DOC_LAYERS := axisml-system axisml-platform

# Component dirs that emit coverage profiles (for the merged report).
COVERAGE_COMPONENTS := \
  axisml-infra/axisml-scheduler \
  axisml-system/tenant-operator axisml-system/compute-operator \
  axisml-system/cluster-manager axisml-system/compute-service \
  axisml-system/artifact-hub axisml-platform/backend

COVERAGE_DIR  ?= $(CURDIR)/coverage
COVERAGE_FILE ?= $(COVERAGE_DIR)/coverage.out

.DEFAULT_GOAL := build

##@ Build & test (delegated to layer Makefiles)
.PHONY: build test image image-load fmt vet tidy clean docs-gen docs-test api-docs-gen api-docs-test config-docs-gen config-docs-test
build: ## Build every layer's components
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l build; done
test: ## Unit tests across every layer
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l test; done
image: ## Build container images across every layer
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l image; done
image-load: ## Build + load images into minikube across every layer
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l image-load; done
fmt: ## go fmt across every layer
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l fmt; done
vet: ## go vet across every layer
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l vet; done
tidy: ## go mod tidy across every layer
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l tidy; done
clean: ## Remove build + coverage artifacts across every layer
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l clean; done
	@rm -rf $(COVERAGE_DIR)
docs-gen: api-docs-gen config-docs-gen ## Regenerate all generated docs (OpenAPI specs + config manual)
docs-test: api-docs-test config-docs-test ## Verify all generated docs are in sync (CI guard)
api-docs-gen: ## Regenerate all OpenAPI specs
	@set -e; for l in $(DOC_LAYERS); do $(MAKE) -C $$l doc-gen; done
api-docs-test: ## Verify all OpenAPI specs are in sync
	@set -e; for l in $(DOC_LAYERS); do $(MAKE) -C $$l doc-test; done
config-docs-gen: ## Regenerate docs/configuration.md from the service Config structs
	@./scripts/gen-config-doc.sh
config-docs-test: ## Verify docs/configuration.md is in sync with the Config structs
	@tmp=$$(mktemp); ./scripts/gen-config-doc.sh $$tmp >/dev/null; \
	if ! diff -q docs/configuration.md $$tmp >/dev/null 2>&1; then \
	  echo "ERROR: docs/configuration.md is out of date. Run 'make config-docs-gen' and commit."; \
	  rm -f $$tmp; exit 1; \
	fi; rm -f $$tmp; echo "docs/configuration.md is in sync"

##@ Test execution
.PHONY: setup-envtest integration-test client-gen
setup-envtest: ## Install the shared envtest binary (axisml-system/test/setup-envtest/)
	@$(MAKE) -C axisml-system setup-envtest
integration-test: ## Integration tests across every layer (hermetic, CI-friendly)
	@$(MAKE) -C axisml-system integration
	@$(MAKE) -C axisml-platform integration
# Black-box test suite (Python + pytest) lives in tests/. Its dependencies, env
# lifecycle, and runs are uv commands (see tests/README.md):
#   cd tests && uv sync && uv run playwright install chromium
#   uv run test-setup                          # bring a Standard environment up
#   uv run pytest api                          # API tests (per component)
#   uv run pytest e2e                          # UI end-to-end (Playwright)
# Only client generation is a make target, delegated to the suite's Makefile:
client-gen: ## Regenerate the test suite's typed Python clients from the OpenAPI specs
	@$(MAKE) -C tests client-gen

##@ Coverage
.PHONY: coverage coverage-unit coverage-integration coverage-merge coverage-html coverage-clean
coverage-unit: ## Unit coverage across every layer
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l coverage; done
coverage-integration: ## Integration coverage across every layer
	@$(MAKE) -C axisml-system integration-coverage
	@$(MAKE) -C axisml-platform integration-coverage
coverage-merge: ## Merge per-component profiles into $(COVERAGE_FILE)
	@mkdir -p $(COVERAGE_DIR)
	@bash scripts/merge-coverage.sh $(COVERAGE_FILE) $(COVERAGE_COMPONENTS)
coverage: coverage-unit coverage-integration coverage-merge ## Unit + integration coverage, merged
coverage-html: ## Render per-layer HTML coverage reports
	@set -e; for l in $(GO_LAYERS); do $(MAKE) -C $$l coverage-html; done
	@printf "\nMerged profile (for Codecov / external tools): %s\n" "$(COVERAGE_FILE)"
coverage-clean: ## Remove all coverage artifacts (root + per-component)
	@rm -rf $(COVERAGE_DIR)
	@set -e; for c in $(COVERAGE_COMPONENTS); do rm -rf $$c/coverage; done

##@ Cluster (delegated to axisml-infra)
.PHONY: cluster-up cluster-down cluster-delete cluster-status
cluster-up: ## Create and start the local minikube cluster
	@$(MAKE) -C axisml-infra cluster-up
cluster-down: ## Stop the cluster (preserves state)
	@$(MAKE) -C axisml-infra cluster-down
cluster-delete: ## Destroy the cluster entirely
	@$(MAKE) -C axisml-infra cluster-delete
cluster-status: ## Show cluster status
	@$(MAKE) -C axisml-infra cluster-status

##@ Helm (cross-layer ordering: infra -> system -> platform)
.PHONY: helm-deps helm-lint helm-template helm-install helm-upgrade helm-uninstall
helm-deps: ## Fetch sub-chart tarballs for all charts
	@$(MAKE) -C axisml-infra helm-deps
	@$(MAKE) -C axisml-system helm-deps
	@$(MAKE) -C axisml-platform helm-deps
helm-lint: ## Lint all Helm charts
	@$(MAKE) -C axisml-infra helm-lint
	@$(MAKE) -C axisml-system helm-lint
	@$(MAKE) -C axisml-platform helm-lint
helm-template: ## Render all charts locally
	@$(MAKE) -C axisml-infra helm-template
	@$(MAKE) -C axisml-system helm-template
	@$(MAKE) -C axisml-platform helm-template
helm-install: ## Install or upgrade infra -> system -> platform
	@$(MAKE) -C axisml-infra helm-install
	@$(MAKE) -C axisml-system helm-install
	@$(MAKE) -C axisml-platform helm-install
helm-upgrade: ## Upgrade infra -> system -> platform
	@$(MAKE) -C axisml-infra helm-upgrade
	@$(MAKE) -C axisml-system helm-upgrade
	@$(MAKE) -C axisml-platform helm-upgrade
helm-uninstall: ## Uninstall platform -> system -> infra (reverse order)
	@$(MAKE) -C axisml-platform helm-uninstall
	@$(MAKE) -C axisml-system helm-uninstall
	@$(MAKE) -C axisml-infra helm-uninstall

##@ Git hooks
.PHONY: install-hooks uninstall-hooks pre-commit-run pre-push-run
install-hooks: ## Install pre-commit + pre-push hooks (requires `pre-commit` on PATH)
	@command -v pre-commit >/dev/null || { echo "pre-commit not found. Install: brew install pre-commit"; exit 1; }
	@pre-commit install
	@pre-commit install --hook-type pre-push
	@pre-commit install --hook-type commit-msg
uninstall-hooks: ## Remove pre-commit + pre-push hooks
	@command -v pre-commit >/dev/null && { pre-commit uninstall; pre-commit uninstall --hook-type pre-push; pre-commit uninstall --hook-type commit-msg; } || true
pre-commit-run: ## Run all pre-commit hooks against every tracked file
	@pre-commit run --all-files
pre-push-run: ## Run all pre-push hooks against every tracked file
	@pre-commit run --hook-stage pre-push --all-files

##@ Help
.PHONY: help
help: ## Show this help message
	@awk 'BEGIN { \
	    FS = ":.*?##"; \
	    printf "\nUsage: make \033[36m<target>\033[0m\n"; \
	  } \
	  /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next } \
	  /^[a-zA-Z][a-zA-Z0-9_-]*:.*##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' \
	  $(MAKEFILE_LIST)
	@printf "\nPer-layer Makefiles own component targets. Examples:\n"
	@printf "  make -C axisml-system compute-service-test   (per-component shortcut)\n"
	@printf "  make -C axisml-system build                   (whole layer)\n"
	@printf "  make -C axisml-platform frontend-dev          (frontend)\n\n"
