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

# --- Image Tag (shared across components) ---
#
# Each component's Makefile defaults IMAGE_TAG to 0.1.0 and notes that it
# must track Chart.appVersion. We export the chart's appVersion here so any
# top-level invocation overrides that default and keeps every component's
# image tag aligned with what the Helm chart will pull. Override on the
# command line for non-release builds:
#   make image IMAGE_TAG=dev
export IMAGE_TAG ?= $(shell awk '/^appVersion:/{gsub(/"/,"",$$2);print $$2}' $(HELM_SYSTEM_CHART)/Chart.yaml)

# --- Helm image-tag overrides ---
#
# Propagate IMAGE_TAG into the four AxisML component image refs so any
# `helm-install` / `helm-upgrade` / `helm-template` invocation honours
# overrides like `make helm-install IMAGE_TAG=latest` without editing
# values.yaml. The default IMAGE_TAG is Chart.appVersion (above), so
# omitting the override matches the chart's built-in default.
#
# HELM_EXTRA_ARGS is an escape hatch for ad-hoc `--set` / `-f` flags;
# it's empty by default and appended last (highest precedence).
HELM_SYSTEM_IMAGE_SET := \
  --set compute.image.tag=$(IMAGE_TAG) \
  --set operators.tenantOperator.image.tag=$(IMAGE_TAG) \
  --set operators.mljobOperator.image.tag=$(IMAGE_TAG) \
  --set operators.mlserviceOperator.image.tag=$(IMAGE_TAG)
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

.PHONY: helm-install helm-upgrade helm-uninstall helm-template helm-deps
.PHONY: helm-install-infra helm-install-system
.PHONY: helm-upgrade-infra helm-upgrade-system
.PHONY: helm-uninstall-infra helm-uninstall-system
.PHONY: helm-crds-system

helm-deps: ## Fetch sub-chart tarballs for both charts (run after clone / Chart.yaml change)
	@helm dependency update $(HELM_INFRA_CHART)
	@helm dependency update $(HELM_SYSTEM_CHART)

helm-install-infra: ## Install or upgrade AxisML infrastructure (idempotent)
	@kubectl --context $(MINIKUBE_PROFILE) create namespace $(HELM_INFRA_NAMESPACE) --dry-run=client -o yaml | kubectl --context $(MINIKUBE_PROFILE) apply -f -
	@kubectl --context $(MINIKUBE_PROFILE) label namespace $(HELM_INFRA_NAMESPACE) app.kubernetes.io/managed-by=Helm --overwrite
	@kubectl --context $(MINIKUBE_PROFILE) annotate namespace $(HELM_INFRA_NAMESPACE) meta.helm.sh/release-name=$(HELM_INFRA_RELEASE) meta.helm.sh/release-namespace=$(HELM_INFRA_NAMESPACE) --overwrite
	@helm upgrade --install $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-crds-system: ## Apply axisml-system CRDs (Helm only installs files under crds/ once; this picks up schema upgrades)
	@kubectl --context $(MINIKUBE_PROFILE) apply -f $(HELM_SYSTEM_CHART)/crds/

helm-install-system: helm-crds-system ## Install or upgrade AxisML control plane (idempotent)
	@helm upgrade --install $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE) $(HELM_SYSTEM_IMAGE_SET) $(HELM_EXTRA_ARGS)

helm-install: helm-install-infra helm-install-system ## Install or upgrade infra + control plane

helm-upgrade-infra: ## Upgrade AxisML infrastructure (must already be installed)
	@helm upgrade $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-upgrade-system: helm-crds-system ## Upgrade AxisML control plane (must already be installed)
	@helm upgrade $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE) $(HELM_SYSTEM_IMAGE_SET) $(HELM_EXTRA_ARGS)

helm-upgrade: helm-upgrade-infra helm-upgrade-system ## Upgrade both

helm-uninstall-system: ## Uninstall AxisML control plane
	@helm uninstall $(HELM_SYSTEM_RELEASE) -n $(HELM_SYSTEM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-uninstall-infra: ## Uninstall AxisML infrastructure
	@helm uninstall $(HELM_INFRA_RELEASE) -n $(HELM_INFRA_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)

helm-uninstall: helm-uninstall-system helm-uninstall-infra ## Uninstall control plane then infra

helm-template: ## Render both charts locally
	@helm template $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) -n $(HELM_INFRA_NAMESPACE)
	@helm template $(HELM_SYSTEM_RELEASE) $(HELM_SYSTEM_CHART) -n $(HELM_SYSTEM_NAMESPACE) $(HELM_SYSTEM_IMAGE_SET) $(HELM_EXTRA_ARGS)

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
  components/operators/mlservice-operator \
  components/compute
# Scaffolded components (uncomment as they ship code):
# COMPONENTS += components/artifacts
# COMPONENTS += components/platform/backend
# COMPONENTS += components/platform/frontend

# Operator basenames, derived from COMPONENTS. Used by e2e targets that
# act on every operator Deployment (scale-to-zero, rollout wait).
OPERATORS := $(notdir $(filter components/operators/%,$(COMPONENTS)))

# All component basenames. Used by e2e-pre-image-load / e2e-wait — every
# component whose image just got loaded into minikube needs to be scaled to
# zero first (otherwise `minikube image load` silently no-ops while the old
# pod still references :TAG) and then waited on after helm-install.
DEPLOYMENTS := $(notdir $(COMPONENTS))

# Every Go module in the repo (operators + their envtest sub-modules +
# shared test/testutil + test/e2e). `go fmt ./...` does not cross module
# boundaries, so `make fmt` iterates these explicitly. Sorted for stable
# output; bin/ excluded so we never recurse into build artifacts.
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
.PHONY: $(notdir $1)-build $(notdir $1)-image $(notdir $1)-image-load $(notdir $1)-test $(notdir $1)-envtest $(notdir $1)-coverage $(notdir $1)-envtest-coverage $(notdir $1)-coverage-html $(notdir $1)-fmt $(notdir $1)-tidy $(notdir $1)-clean
$(notdir $1)-build:
	@$$(MAKE) -C $1 build
$(notdir $1)-image:
	@$$(MAKE) -C $1 image
$(notdir $1)-image-load:
	@$$(MAKE) -C $1 image-load-minikube
$(notdir $1)-test:
	@$$(MAKE) -C $1 test
$(notdir $1)-envtest:
	@$$(MAKE) -C $1 envtest
$(notdir $1)-coverage:
	@$$(MAKE) -C $1 coverage
$(notdir $1)-envtest-coverage:
	@$$(MAKE) -C $1 envtest-coverage
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

##@ Test infrastructure

# Shared setup-envtest binary location. Each operator's `envtest` Makefile
# target invokes $(REPO_ROOT)/test/setup-envtest/setup-envtest, so all three
# operators reuse one binary instead of vendoring their own copies.
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

.PHONY: envtest-test e2e-test e2e-wait e2e-pre-image-load

# L1 envtest: hermetic, in-process reconciler tests against an embedded
# apiserver+etcd. Each operator's `envtest` target boots its own envtest with
# the right CRDs and runs `go test -tags=envtest ./test/envtest/...`.
envtest-test: setup-envtest ## L1 envtest across all operators (hermetic, CI-friendly)
	@$(call _RUN_COMPONENTS,envtest)

# L2 e2e: full-stack tests against a real minikube cluster running helm-installed
# infra + system. Operators run as deployed (NOT scaled to zero); tests act as
# external clients via client-go and (for service tests) port-forward to
# in-cluster Services.
e2e-test: cluster-up e2e-pre-image-load image-load helm-install e2e-wait ## L2 minikube e2e (operators + services)
	@if [ -d test/e2e ]; then \
	  cd test/e2e && go test -tags=e2e -count=1 -timeout=20m ./... ; \
	else \
	  echo "(no e2e tests yet — orchestration complete; add Go tests under test/e2e/ behind build tag e2e)"; \
	fi

# Scale axisml-system component deployments to zero before `image-load`.
# minikube's `image load --overwrite=true` is silently a no-op when a
# container in the minikube node still references the existing image:tag, so
# subsequent runs would deploy stale code. helm-install scales them back up to
# the chart's replicas (=1), at which point the freshly-loaded image is
# pulled. The `kubectl get` guard makes scale a no-op on a fresh cluster
# (`kubectl scale` doesn't accept `--ignore-not-found` until v1.31). Per-
# component wait selectors (`app.kubernetes.io/name=$$c`) stay scoped so we
# don't widen to unrelated axisml-system Pods.
e2e-pre-image-load:
	@for c in $(DEPLOYMENTS); do \
	  if kubectl --context $(MINIKUBE_PROFILE) -n $(HELM_SYSTEM_NAMESPACE) \
	      get deploy/$(HELM_SYSTEM_RELEASE)-$$c >/dev/null 2>&1; then \
	    kubectl --context $(MINIKUBE_PROFILE) -n $(HELM_SYSTEM_NAMESPACE) \
	      scale deploy/$(HELM_SYSTEM_RELEASE)-$$c --replicas=0; \
	  fi; \
	done
	@for c in $(DEPLOYMENTS); do \
	  kubectl --context $(MINIKUBE_PROFILE) -n $(HELM_SYSTEM_NAMESPACE) \
	    wait --for=delete pod -l app.kubernetes.io/name=$$c --timeout=60s; \
	done

e2e-wait: ## Wait for axisml component Deployments to become ready (used by e2e-test)
	@for c in $(DEPLOYMENTS); do \
	  printf '>>> waiting for %s\n' "$$c"; \
	  kubectl --context $(MINIKUBE_PROFILE) -n $(HELM_SYSTEM_NAMESPACE) \
	    rollout status deploy/$(HELM_SYSTEM_RELEASE)-$$c --timeout=180s; \
	done

##@ Coverage
#
# Each component's Makefile produces coverage profiles under <component>/coverage/:
#   coverage.out          (unit, from `go test -coverprofile`)
#   envtest-coverage.out  (envtest, with -coverpkg pointed at the operator's
#                          production module so cross-module envtest hits count)
# Top-level targets fan out via _RUN_COMPONENTS, then merge with a tiny shell
# script (atomic-mode profiles concatenate cleanly without gocovmerge).

COVERAGE_DIR  ?= $(CURDIR)/coverage
COVERAGE_FILE ?= $(COVERAGE_DIR)/coverage.out

.PHONY: coverage coverage-unit coverage-envtest coverage-merge coverage-html coverage-clean

coverage-unit: ## Run unit tests with coverage profile across all components
	@$(call _RUN_COMPONENTS,coverage)

coverage-envtest: setup-envtest ## Run L1 envtest with coverage across all operators
	@$(call _RUN_COMPONENTS,envtest-coverage)

coverage-merge: ## Merge per-component profiles into $(COVERAGE_FILE)
	@bash scripts/merge-coverage.sh $(COVERAGE_FILE) $(COMPONENTS)

coverage: coverage-unit coverage-envtest coverage-merge ## Run unit + envtest with coverage and produce a merged report

# HTML rendering is per-component because `go tool cover -html` resolves source
# paths against the current Go module and the repo root has no go.mod. See
# docs/development/testing.md for details.
coverage-html: ## Render per-component HTML coverage reports
	@$(call _RUN_COMPONENTS,coverage-html)
	@printf "\nMerged profile (for Codecov / external tools): %s\n" "$(COVERAGE_FILE)"

coverage-clean: ## Remove all coverage artifacts (root + per-component)
	@rm -rf $(COVERAGE_DIR)
	@for c in $(COMPONENTS); do rm -rf $$c/coverage; done

##@ Git hooks

# Hook orchestration is delegated to the `pre-commit` framework
# (https://pre-commit.com); see .pre-commit-config.yaml for the hook list.
# These targets are thin wrappers so contributors don't need to remember
# the underlying CLI invocation.

.PHONY: install-hooks uninstall-hooks pre-commit-run

install-hooks: ## Install the pre-commit hook into .git/hooks/ (requires `pre-commit` on PATH)
	@command -v pre-commit >/dev/null || { \
	  echo "pre-commit not found. Install: brew install pre-commit  (or pipx install pre-commit)"; \
	  exit 1; }
	@pre-commit install

uninstall-hooks: ## Remove the pre-commit hook from .git/hooks/
	@command -v pre-commit >/dev/null && pre-commit uninstall || true

pre-commit-run: ## Run all pre-commit hooks against every tracked file
	@pre-commit run --all-files

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
	@printf "  Pattern : <component>-{build,image,image-load,test,envtest,coverage,envtest-coverage,coverage-html,fmt,tidy,clean}\n"
	@printf "  Active  : %s\n" "$(notdir $(COMPONENTS))"
	@printf "  Example : make tenant-operator-image  |  make mljob-operator-envtest-coverage\n\n"

.DEFAULT_GOAL := build
