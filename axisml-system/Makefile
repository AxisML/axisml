# axisml-system Makefile — the whole System layer in one file.
#
# Owns build / test / image / integration / OpenAPI / coverage for all five
# System components, the System Helm chart, and the shared envtest binary.
# Pattern rules give both aggregates (`make test`) and per-component shortcuts
# (`make compute-service-test`). The repo-root Makefile delegates here.

REPO_ROOT := $(abspath $(CURDIR)/..)
CHART     := $(CURDIR)/deploy/helm

COMPONENTS     := tenant-operator compute-operator cluster-manager compute-service artifact-hub
DOC_COMPONENTS := cluster-manager compute-service artifact-hub

IMAGE_REPO_PREFIX ?= ghcr.io/axisml/axisml
IMAGE_TAG         ?= $(shell awk '/^appVersion:/{gsub(/"/,"",$$2);print $$2}' $(CHART)/Chart.yaml)
MINIKUBE_PROFILE  ?= axisml

# Shared envtest binary — this layer owns it; the integration suites consume it.
ENVTEST_BIN_DIR       := $(CURDIR)/test/setup-envtest
ENVTEST               := $(ENVTEST_BIN_DIR)/setup-envtest
ENVTEST_K8S_VERSION   ?= 1.31.0
SETUP_ENVTEST_VERSION ?= release-0.19

# Per-component conventions: operators build ./cmd, services ./cmd/<name>.
# Binaries: operators + cluster-manager carry the axisml- prefix; the two
# services keep their bare name (matching their published image entrypoints).
cmdpath = $(if $(filter $1,tenant-operator compute-operator),./cmd,./cmd/$1)
binname = $(if $(filter $1,compute-service artifact-hub),$1,axisml-$1)
module  = github.com/axisml/axisml/axisml-system/$1

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

# --- Aggregate targets (fan out over every component) ---
.PHONY: build test image image-load integration doc-gen doc-test coverage integration-coverage coverage-html fmt vet tidy clean
build:                $(COMPONENTS:%=%-build)                  ## Build every component binary
test:                 apis-test $(COMPONENTS:%=%-test)        ## Unit-test APIs and every component
image:                $(COMPONENTS:%=%-image)                 ## Build every component image
image-load:           $(COMPONENTS:%=%-image-load)           ## Build + load every image into minikube
integration:          $(COMPONENTS:%=%-integration)          ## Integration-test every component (envtest)
doc-gen:              $(DOC_COMPONENTS:%=%-doc-gen)           ## Regenerate every OpenAPI spec
doc-test:            $(DOC_COMPONENTS:%=%-doc-test)           ## Verify every OpenAPI spec is in sync
coverage:             $(COMPONENTS:%=%-coverage)              ## Unit coverage for every component
integration-coverage: $(COMPONENTS:%=%-integration-coverage) ## Integration coverage for every component
coverage-html:        $(COMPONENTS:%=%-coverage-html)        ## Render HTML coverage per component
fmt:                  apis-fmt $(COMPONENTS:%=%-fmt)         ## go fmt APIs and every component
vet:                  apis-vet $(COMPONENTS:%=%-vet)         ## go vet APIs and every component
tidy:                 apis-tidy $(COMPONENTS:%=%-tidy)       ## go mod tidy APIs and every component
clean:                $(COMPONENTS:%=%-clean)                ## Remove build/coverage artifacts

.PHONY: apis-test apis-fmt apis-vet apis-tidy
apis-test:
	@echo ">>> apis test" && cd apis && go test ./...
apis-fmt:
	@cd apis && go fmt ./...
apis-vet:
	@cd apis && go vet ./...
apis-tidy:
	@cd apis && go mod tidy

# --- Per-component rules (explicit, generated; e.g. `make compute-service-test`) ---
# Generated as explicit targets (not pattern rules) so overlapping suffixes like
# `-test` / `-doc-test` and `-coverage` / `-integration-coverage` never collide.
define COMPONENT_template
.PHONY: $(1)-build $(1)-test $(1)-image $(1)-image-load $(1)-integration $(1)-coverage $(1)-integration-coverage $(1)-coverage-html $(1)-fmt $(1)-vet $(1)-tidy $(1)-clean
$(1)-build:
	@echo ">>> $(1) build" && cd $(1) && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/$(call binname,$(1)) $(call cmdpath,$(1))
$(1)-test:
	@echo ">>> $(1) test" && cd $(1) && go test ./...
	@if [ -f $(1)/pkg/apidoc/go.mod ]; then cd $(1)/pkg/apidoc && go test ./...; fi
$(1)-image:
	@docker build -f $(CURDIR)/$(1)/Dockerfile -t $(IMAGE_REPO_PREFIX)-$(1):$(IMAGE_TAG) $(REPO_ROOT)
$(1)-image-load: $(1)-image
	@minikube image load $(IMAGE_REPO_PREFIX)-$(1):$(IMAGE_TAG)
$(1)-integration: $(ENVTEST)
	@echo ">>> $(1) integration" && cd $(1)/test/integration && \
	  KUBEBUILDER_ASSETS="$$$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  go test -tags=integration -count=1 -timeout=15m ./...
$(1)-coverage:
	@mkdir -p $(1)/coverage && cd $(1) && go test -coverprofile=coverage/coverage.out -covermode=atomic -count=1 ./...
$(1)-integration-coverage: $(ENVTEST)
	@mkdir -p $(1)/coverage && cd $(1)/test/integration && \
	  KUBEBUILDER_ASSETS="$$$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  go test -tags=integration -count=1 -timeout=15m \
	    -coverpkg=$(call module,$(1))/... \
	    -coverprofile=$(CURDIR)/$(1)/coverage/integration-coverage.out -covermode=atomic ./...
$(1)-coverage-html:
	@if [ -f $(1)/coverage/coverage.out ]; then cd $(1) && go tool cover -html=coverage/coverage.out -o coverage/coverage.html && echo "$(1) unit: $(1)/coverage/coverage.html"; fi
	@if [ -f $(1)/coverage/integration-coverage.out ]; then cd $(1) && go tool cover -html=coverage/integration-coverage.out -o coverage/integration-coverage.html && echo "$(1) integration: $(1)/coverage/integration-coverage.html"; fi
$(1)-fmt:
	@cd $(1) && go fmt ./...
	@if [ -f $(1)/pkg/apidoc/go.mod ]; then cd $(1)/pkg/apidoc && go fmt ./...; fi
$(1)-vet:
	@cd $(1) && go vet ./...
	@if [ -f $(1)/pkg/apidoc/go.mod ]; then cd $(1)/pkg/apidoc && go vet ./...; fi
$(1)-tidy:
	@cd $(1) && go mod tidy
	@if [ -f $(1)/pkg/apidoc/go.mod ]; then cd $(1)/pkg/apidoc && go mod tidy; fi
$(1)-clean:
	@rm -rf $(1)/bin $(1)/coverage
endef
$(foreach c,$(COMPONENTS),$(eval $(call COMPONENT_template,$(c))))

define DOC_template
.PHONY: $(1)-doc-gen $(1)-doc-test
$(1)-doc-gen:
	@cd $(1)/pkg/apidoc && go run ./cmd/openapi-gen \
	  -o $(REPO_ROOT)/axisml-system/docs/apis/$(1).yaml \
	  -embed $(REPO_ROOT)/axisml-system/$(1)/pkg/apispec/openapi.gen.yaml \
	  -version $(IMAGE_TAG)
$(1)-doc-test: $(1)-doc-gen
	@if ! git -C $(REPO_ROOT) diff --quiet -- \
	  axisml-system/docs/apis/$(1).yaml axisml-system/$(1)/pkg/apispec/openapi.gen.yaml; then \
	  echo "ERROR: $(1) OpenAPI artifacts are out of date. Run 'make $(1)-doc-gen' and commit."; \
	  git -C $(REPO_ROOT) --no-pager diff -- \
	    axisml-system/docs/apis/$(1).yaml axisml-system/$(1)/pkg/apispec/openapi.gen.yaml; exit 1; fi
endef
$(foreach c,$(DOC_COMPONENTS),$(eval $(call DOC_template,$(c))))

# --- Shared envtest binary ---
.PHONY: setup-envtest
setup-envtest: $(ENVTEST) ## Install setup-envtest into test/setup-envtest/
$(ENVTEST):
	@mkdir -p $(ENVTEST_BIN_DIR)
	@GOBIN=$(ENVTEST_BIN_DIR) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)

# --- Helm (system layer) ---
HELM_RELEASE    ?= axisml
HELM_NAMESPACE  ?= axisml-system
HELM_EXTRA_ARGS ?=
HELM_IMAGE_SET  := \
  --set clusterManager.image.tag=$(IMAGE_TAG) \
  --set computeService.image.tag=$(IMAGE_TAG) \
  --set artifactHub.image.tag=$(IMAGE_TAG) \
  --set tenantOperator.image.tag=$(IMAGE_TAG) \
  --set computeOperator.image.tag=$(IMAGE_TAG)
HELM_DEV_DEFAULTS := --set artifactHub.storage.oci.adminSecretRef.password=admin

.PHONY: helm-deps helm-lint helm-template helm-crds helm-install helm-upgrade helm-uninstall
helm-deps: ## Fetch sub-chart tarballs
	@helm dependency update $(CHART)
helm-lint: ## Lint the system chart
	@helm lint $(CHART)
helm-template: ## Render the system chart locally
	@helm template $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) $(HELM_IMAGE_SET) $(HELM_DEV_DEFAULTS) $(HELM_EXTRA_ARGS)
helm-crds: ## Apply system CRDs (Helm only installs crds/ on first install)
	@kubectl --context $(MINIKUBE_PROFILE) apply -f $(CHART)/crds/
helm-install: helm-crds ## Install or upgrade the system release (idempotent)
	@helm upgrade --install $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE) --timeout 10m $(HELM_IMAGE_SET) $(HELM_DEV_DEFAULTS) $(HELM_EXTRA_ARGS)
helm-upgrade: helm-crds ## Upgrade the system release (must already be installed)
	@helm upgrade $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE) --timeout 10m $(HELM_IMAGE_SET) $(HELM_DEV_DEFAULTS) $(HELM_EXTRA_ARGS)
helm-uninstall: ## Uninstall the system release
	@helm uninstall $(HELM_RELEASE) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)
