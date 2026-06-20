# AxisML Makefile

# --- Cluster Configuration ---
export MINIKUBE_PROFILE ?= axisml
export MINIKUBE_CPUS    ?= 4
export MINIKUBE_MEMORY  ?= 8192
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

HELM_PLATFORM_RELEASE   ?= axisml-platform
HELM_PLATFORM_NAMESPACE ?= axisml-platform
HELM_PLATFORM_CHART     ?= deploy/helm/axisml-platform

# --- Image Tag (shared across components) ---
#
# Each component's Makefile defaults IMAGE_TAG to 0.1.0 and notes that it
# must track Chart.appVersion. The axisml-system chart is the single version
# authority across all three charts (infra / system / platform): we export its
# appVersion here so any top-level invocation overrides that default and keeps
# every component's image tag aligned with what the charts will pull. Override
# on the command line for non-release builds:
#   make image IMAGE_TAG=dev
export IMAGE_TAG ?= $(shell awk '/^appVersion:/{gsub(/"/,"",$$2);print $$2}' $(HELM_SYSTEM_CHART)/Chart.yaml)

# --- Helm image-tag overrides ---
#
# Propagate IMAGE_TAG into AxisML component image refs so any
# `helm-install` / `helm-upgrade` / `helm-template` invocation honours
# overrides like `make helm-install IMAGE_TAG=latest` without editing
# values.yaml. The default IMAGE_TAG is Chart.appVersion (above), so
# omitting the override matches the chart's built-in default.
#
# HELM_EXTRA_ARGS is an escape hatch for ad-hoc `--set` / `-f` flags;
# it's empty by default and appended last (highest precedence).
HELM_SYSTEM_IMAGE_SET := \
  --set clusterManager.image.tag=$(IMAGE_TAG) \
  --set computeService.image.tag=$(IMAGE_TAG) \
  --set artifactHub.image.tag=$(IMAGE_TAG) \
  --set tenantOperator.image.tag=$(IMAGE_TAG) \
  --set computeOperator.image.tag=$(IMAGE_TAG)

# Platform still ships an nginx placeholder image (see axisml-platform values),
# which does not track Chart.appVersion — so IMAGE_TAG is NOT forced onto it yet.
# Uncomment once Platform publishes a real image tagged to appVersion.
HELM_PLATFORM_IMAGE_SET :=
# HELM_PLATFORM_IMAGE_SET := \
#   --set platform.image.tag=$(IMAGE_TAG)

# Dev-only defaults the chart `required` gates demand. Production installs
# should override these via HELM_EXTRA_ARGS or a values file with real
# secrets — never ship `axisml` to a real cluster.
HELM_SYSTEM_DEV_DEFAULTS := \
  --set artifactHub.storage.oci.adminSecretRef.password=admin

HELM_EXTRA_ARGS ?=

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

.PHONY: helm-install helm-upgrade helm-uninstall helm-template helm-lint helm-deps
.PHONY: helm-install-infra helm-install-system helm-install-platform
.PHONY: helm-upgrade-infra helm-upgrade-system helm-upgrade-platform
.PHONY: helm-uninstall-infra helm-uninstall-system helm-uninstall-platform
.PHONY: helm-crds-system

helm-deps: ## Fetch sub-chart tarballs for all charts (run after clone / Chart.yaml change)
	@helm dependency update $(HELM_INFRA_CHART)
	@helm dependency update $(HELM_SYSTEM_CHART)
	@helm dependency update $(HELM_PLATFORM_CHART)

helm-lint: ## Lint all Helm charts (no dep fetching — missing-dep warning is harmless)
	@helm lint $(HELM_INFRA_CHART)
	@helm lint $(HELM_SYSTEM_CHART)
	@helm lint $(HELM_PLATFORM_CHART)

helm-install-infra: ## Install or upgrade AxisML infrastructure (idempotent)
	@kubectl --context $(MINIKUBE_PROFILE) create namespace $(HELM_INFRA_NAMESPACE) --dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -
	@kubectl --context $(MINIKUBE_PROFILE) label namespace $(HELM_INFRA_NAMESPACE) app.kubernetes.io/managed-by=Helm --overwrite
	@kubectl --context $(MINIKUBE_PROFILE) annotate namespace $(HELM_INFRA_NAMESPACE) meta.helm.sh/release-name=$(HELM_INFRA_RELEASE) meta.helm.sh/release-namespace=$(HELM_INFRA_NAMESPACE) --overwrite
	@helm upgrade --install $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-crds-system: ## Apply axisml-system CRDs (Helm only installs files under crds/ once; this picks up schema upgrades)
	@kubectl --context $(MINIKUBE_PROFILE) apply -f $(HELM_SYSTEM_CHART)/crds/

helm-install-system: helm-crds-system ## Install or upgrade AxisML control plane (idempotent)
	@helm upgrade --install $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE) --timeout 10m $(HELM_SYSTEM_IMAGE_SET) $(HELM_SYSTEM_DEV_DEFAULTS) $(HELM_EXTRA_ARGS)

helm-install-platform: ## Install or upgrade AxisML platform (idempotent)
	@helm upgrade --install $(HELM_PLATFORM_RELEASE) $(HELM_PLATFORM_CHART) -n $(HELM_PLATFORM_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE) $(HELM_PLATFORM_IMAGE_SET) $(HELM_EXTRA_ARGS)

helm-install: helm-install-infra helm-install-system helm-install-platform ## Install or upgrade infra + control plane + platform

helm-upgrade-infra: ## Upgrade AxisML infrastructure (must already be installed)
	@helm upgrade $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-upgrade-system: helm-crds-system ## Upgrade AxisML control plane (must already be installed)
	@helm upgrade $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE) --timeout 10m $(HELM_SYSTEM_IMAGE_SET) $(HELM_SYSTEM_DEV_DEFAULTS) $(HELM_EXTRA_ARGS)

helm-upgrade-platform: ## Upgrade AxisML platform (must already be installed)
	@helm upgrade $(HELM_PLATFORM_RELEASE) $(HELM_PLATFORM_CHART) -n $(HELM_PLATFORM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE) $(HELM_PLATFORM_IMAGE_SET) $(HELM_EXTRA_ARGS)

helm-upgrade: helm-upgrade-infra helm-upgrade-system helm-upgrade-platform ## Upgrade all three

helm-uninstall-platform: ## Uninstall AxisML platform
	@helm uninstall $(HELM_PLATFORM_RELEASE) -n $(HELM_PLATFORM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-uninstall-system: ## Uninstall AxisML control plane
	@helm uninstall $(HELM_SYSTEM_RELEASE) -n $(HELM_SYSTEM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-uninstall-infra: ## Uninstall AxisML infrastructure
	@helm uninstall $(HELM_INFRA_RELEASE) -n $(HELM_INFRA_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-uninstall: helm-uninstall-platform helm-uninstall-system helm-uninstall-infra ## Uninstall platform then control plane then infra

helm-template: ## Render all charts locally
	@helm template $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE)
	@helm template $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE) $(HELM_SYSTEM_IMAGE_SET) $(HELM_SYSTEM_DEV_DEFAULTS) $(HELM_EXTRA_ARGS)
	@helm template $(HELM_PLATFORM_RELEASE) $(HELM_PLATFORM_CHART) -n $(HELM_PLATFORM_NAMESPACE) $(HELM_PLATFORM_IMAGE_SET) $(HELM_EXTRA_ARGS)

# --- Components ---
#
# Components implementing the standard Makefile contract:
#   make build / make image / make image-load / make test / make clean
#
# Each entry is a directory that contains a Makefile honouring the contract.
# Add scaffolded components here as they ship working build targets.
COMPONENTS := \
  components/tenant-operator \
  components/compute-operator \
  components/cluster-manager \
  components/compute-service \
  components/artifact-hub \
  components/platform-backend
# platform-backend runs serve/migrate/bootstrap + the real HTTP API (Auth /
# Tenants / Quotas / Members wired today; the rest of the declared surface 501s
# until its handlers land), with the full contract declared as DTOs that
# generate docs/openapi/platform.yaml. It builds/tests/images like a sibling.
# Scaffolded components (uncomment as they ship code):
# COMPONENTS += components/platform-frontend

# Coverage profiles are produced by each Go module under COMPONENTS.
COVERAGE_COMPONENTS := \
  components/tenant-operator \
  components/compute-operator \
  components/cluster-manager \
  components/compute-service \
  components/artifact-hub \
  components/platform-backend

# Components participating in `make integration-test` and the matching
# CI integration job. Most suites need either envtest (kubebuilder assets
# cached via `make setup-envtest`) or testcontainers (Docker daemon), both of
# which CI provides; platform-backend's suite needs only testcontainers Postgres
# (it drives the in-process gin engine via httptest, no envtest/K8s).
INTEGRATION_COMPONENTS := \
  components/tenant-operator \
  components/compute-operator \
  components/cluster-manager \
  components/compute-service \
  components/artifact-hub \
  components/platform-backend

# Components that ship a public REST API and therefore an OpenAPI spec under
# docs/openapi/. The two operators have no HTTP surface (they reconcile CRs),
# so they're excluded. platform-backend owns docs/openapi/platform.yaml, so it
# participates in doc-gen.
DOC_COMPONENTS := \
  components/cluster-manager \
  components/compute-service \
  components/artifact-hub \
  components/platform-backend

# Every Go module in the repo (each component + its integration sub-module
# + shared test/testutil). `go fmt ./...` does not cross module boundaries,
# so `make fmt` iterates these explicitly. Sorted for stable output; bin/
# excluded so we never recurse into build artifacts.
GO_MODULES := $(sort $(shell find . -name go.mod -not -path '*/bin/*' -exec dirname {} \;))

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

# Same shape as _RUN_COMPONENTS but iterates the integration subset.
_RUN_INTEGRATION_COMPONENTS = set -e; for c in $(INTEGRATION_COMPONENTS); do \
	printf '\n>>> %s (%s)\n' "$$c" "$(1)"; \
	$(MAKE) -C $$c $(1); \
done

.PHONY: build image image-load test fmt tidy clean

build: ## Build every component (delegates to each component's `make build`)
	@$(call _RUN_COMPONENTS,build)

image: ## Build container images for every component
	@$(call _RUN_COMPONENTS,image)

image-load: ## Build images and load them into the local minikube node
	@$(call _RUN_COMPONENTS,image-load-minikube)

test: ## Run unit tests across every component (no cluster required)
	@$(call _RUN_COMPONENTS,test)

fmt: ## Run gofmt across every Go module (covers files behind build tags too)
	@set -e; for d in $(GO_MODULES); do \
	  printf '\n>>> %s (fmt)\n' "$$d"; \
	  (cd $$d && gofmt -w -l .); \
	done

tidy: ## Run `go mod tidy` across every component
	@$(call _RUN_COMPONENTS,tidy)

clean: ## Remove build artifacts across every component
	@$(call _RUN_COMPONENTS,clean)

.PHONY: doc-gen doc-test
doc-gen: ## Regenerate OpenAPI specs (docs/openapi/*.yaml) for every API service
	@set -e; for c in $(DOC_COMPONENTS); do \
	  printf '\n>>> %s (doc-gen)\n' "$$c"; \
	  $(MAKE) -C $$c doc-gen; \
	done

doc-test: ## Verify OpenAPI specs are in sync with Go request/response types (CI guard)
	@set -e; for c in $(DOC_COMPONENTS); do \
	  printf '\n>>> %s (doc-test)\n' "$$c"; \
	  $(MAKE) -C $$c doc-test; \
	done

# --- Per-component shortcut targets ---
#
# Generate `<basename>-build`, `<basename>-image`, `<basename>-image-load`,
# `<basename>-test`, `<basename>-clean` for each component listed above. For
# example: `make operator-image`, `make compute-test`.
#
# COMPONENT basenames must be unique. If you add a component whose basename
# would collide (e.g., `components/platform-backend` would clash with any
# other `backend`), give it a distinct directory name or rework the mapping.
define _COMPONENT_SHORTCUTS
.PHONY: $(notdir $1)-build $(notdir $1)-image $(notdir $1)-image-load $(notdir $1)-test $(notdir $1)-integration $(notdir $1)-coverage $(notdir $1)-integration-coverage $(notdir $1)-coverage-html $(notdir $1)-fmt $(notdir $1)-tidy $(notdir $1)-clean
$(notdir $1)-build:
	@$$(MAKE) -C $1 build
$(notdir $1)-image:
	@$$(MAKE) -C $1 image
$(notdir $1)-image-load:
	@$$(MAKE) -C $1 image-load-minikube
$(notdir $1)-test:
	@$$(MAKE) -C $1 test
$(notdir $1)-integration:
	@$$(MAKE) -C $1 integration
$(notdir $1)-coverage:
	@$$(MAKE) -C $1 coverage
$(notdir $1)-integration-coverage:
	@$$(MAKE) -C $1 integration-coverage
$(notdir $1)-coverage-html:
	@$$(MAKE) -C $1 coverage-html
$(notdir $1)-fmt:
	@$$(MAKE) -C $1 fmt
$(notdir $1)-tidy:
	@$$(MAKE) -C $1 tidy
$(notdir $1)-clean:
	@$$(MAKE) -C $1 clean
endef
$(foreach c,$(COMPONENTS),$(eval $(call _COMPONENT_SHORTCUTS,$(c))))

# Per-component shortcuts for the OpenAPI doc targets. Only the API services
# (DOC_COMPONENTS) carry these, since the operators have no HTTP surface.
define _DOC_COMPONENT_SHORTCUTS
.PHONY: $(notdir $1)-doc-gen $(notdir $1)-doc-test
$(notdir $1)-doc-gen:
	@$$(MAKE) -C $1 doc-gen
$(notdir $1)-doc-test:
	@$$(MAKE) -C $1 doc-test
endef
$(foreach c,$(DOC_COMPONENTS),$(eval $(call _DOC_COMPONENT_SHORTCUTS,$(c))))

##@ Test infrastructure

# Shared setup-envtest binary location. Each component's `integration`
# Makefile target invokes $(REPO_ROOT)/test/setup-envtest/setup-envtest.
ENVTEST_BIN_DIR       ?= $(CURDIR)/test/setup-envtest
ENVTEST               ?= $(ENVTEST_BIN_DIR)/setup-envtest
ENVTEST_K8S_VERSION   ?= 1.31.0
SETUP_ENVTEST_VERSION ?= release-0.19

.PHONY: setup-envtest

setup-envtest: $(ENVTEST) ## Install setup-envtest binary into test/setup-envtest/
$(ENVTEST):
	@mkdir -p $(ENVTEST_BIN_DIR)
	@GOBIN=$(ENVTEST_BIN_DIR) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)

##@ Test execution

.PHONY: integration-test

# Integration: hermetic, in-process reconciler tests against an embedded
# apiserver+etcd (controller-runtime envtest) plus testcontainers-managed
# PostgreSQL where needed. Each component's `integration` target boots its
# own dependencies and runs `go test -tags=integration ./test/integration/...`.
integration-test: setup-envtest ## Integration tests across every component (hermetic, CI-friendly)
	@$(call _RUN_INTEGRATION_COMPONENTS,integration)

.PHONY: e2e-test e2e-clean

# E2E: real-cluster system-layer tests. NOT hermetic and NOT part of CI — it
# assumes the `axisml` minikube cluster is up with infra + system installed
# (`make cluster-up && make helm-install`) and that the workload images are
# preloaded (`minikube image load busybox:latest`, `... nginx:1.27`). The suite
# reaches the in-cluster HTTP services via `kubectl port-forward`.
# See test/e2e/README.md.
e2e-test: ## End-to-end tests against the running axisml minikube cluster (manual)
	@cd test/e2e && go test -tags=e2e -count=1 -timeout=30m -v ./...

# Remove the shared e2e tenant left behind by an interrupted run (E2E_KEEP_TENANT).
e2e-clean: ## Delete the shared e2e tenant namespace
	@kubectl --context $(MINIKUBE_PROFILE) delete tenant e2e --ignore-not-found

##@ Coverage
#
# Each component's Makefile produces coverage profiles under <component>/coverage/:
#   coverage.out              (unit, from `go test -coverprofile`)
#   integration-coverage.out  (integration, with -coverpkg pointed at the
#                              component's production module so cross-module
#                              integration hits count)
# Top-level targets fan out via _RUN_COMPONENTS, then merge with a tiny shell
# script (atomic-mode profiles concatenate cleanly without gocovmerge).

COVERAGE_DIR  ?= $(CURDIR)/coverage
COVERAGE_FILE ?= $(COVERAGE_DIR)/coverage.out

.PHONY: coverage coverage-unit coverage-integration coverage-merge coverage-html coverage-clean

coverage-unit: ## Run unit tests with coverage profile across all components
	@$(call _RUN_COMPONENTS,coverage)

coverage-integration: setup-envtest ## Run integration tests with coverage across every component
	@$(call _RUN_INTEGRATION_COMPONENTS,integration-coverage)

coverage-merge: ## Merge per-component profiles into $(COVERAGE_FILE)
	@bash scripts/merge-coverage.sh $(COVERAGE_FILE) $(COVERAGE_COMPONENTS)

coverage: coverage-unit coverage-integration coverage-merge ## Run unit + integration with coverage and produce a merged report

# HTML rendering is per-component because `go tool cover -html` resolves source
# paths against the current Go module and the repo root has no go.mod. See
# docs/development/testing.md for details.
coverage-html: ## Render per-component HTML coverage reports
	@$(call _RUN_COMPONENTS,coverage-html)
	@printf "\nMerged profile (for Codecov / external tools): %s\n" "$(COVERAGE_FILE)"

coverage-clean: ## Remove all coverage artifacts (root + per-component)
	@rm -rf $(COVERAGE_DIR)
	@for c in $(COVERAGE_COMPONENTS); do rm -rf $$c/coverage; done

##@ Git hooks

# Hook orchestration is delegated to the `pre-commit` framework
# (https://pre-commit.com); see .pre-commit-config.yaml for the hook list.
# These targets are thin wrappers so contributors don't need to remember
# the underlying CLI invocation.

.PHONY: install-hooks uninstall-hooks pre-commit-run

install-hooks: ## Install pre-commit + pre-push hooks into .git/hooks/ (requires `pre-commit` on PATH)
	@command -v pre-commit >/dev/null || { \
	  echo "pre-commit not found. Install: brew install pre-commit  (or pipx install pre-commit)"; \
	  exit 1; }
	@pre-commit install
	@pre-commit install --hook-type pre-push

uninstall-hooks: ## Remove pre-commit + pre-push hooks from .git/hooks/
	@command -v pre-commit >/dev/null && { \
	  pre-commit uninstall; \
	  pre-commit uninstall --hook-type pre-push; \
	} || true

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
	  /^[a-zA-Z][a-zA-Z0-9_-]*:.*##/ { printf "  \033[36m%-26s\033[0m %s\n", $$1, $$2 }' \
	  $(MAKEFILE_LIST)
	@printf "\n\033[1mPer-component shortcuts (auto-generated)\033[0m\n"
	@printf "  Pattern : <component>-{build,image,image-load,test,integration,coverage,integration-coverage,coverage-html,fmt,tidy,clean}\n"
	@printf "  Active  : %s\n" "$(notdir $(COMPONENTS))"
	@printf "  Example : make tenant-operator-image  |  make compute-service-test\n"
	@printf "\n\033[1mDoc shortcuts (API services only)\033[0m\n"
	@printf "  Pattern : <component>-{doc-gen,doc-test}\n"
	@printf "  Active  : %s\n" "$(notdir $(DOC_COMPONENTS))"
	@printf "  Example : make compute-service-doc-gen  |  make doc-test\n\n"

.DEFAULT_GOAL := build
