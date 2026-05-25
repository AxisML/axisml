# compute-service Makefile
#
# Standard targets (per AxisML component conventions):
#   make / make build  Compile the binary into bin/
#   make image         Build the container image
#   make test          Run unit tests (no cluster required)
#   make clean         Remove build artifacts
#
# NOTE: compute-service imports operator API packages via local replace
# directives, so `image` builds with the repo root as build context.

IMAGE_REPO ?= ghcr.io/axisml/axisml-compute-service
# Must track deploy/helm/axisml-system Chart.appVersion: the Helm template
# defaults image.tag to .Chart.AppVersion when unset, so locally-built images
# need this tag for `minikube image load` to satisfy the rendered Deployment.
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

BIN_NAME ?= compute-service
CMD_PATH ?= ./cmd/compute-service
REPO_ROOT := $(abspath $(CURDIR)/../..)

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile the binary into bin/$(BIN_NAME)
	@CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/$(BIN_NAME) $(CMD_PATH)

.PHONY: test
test: ## Run unit tests (no cluster required)
	@go test ./...

.PHONY: image
image: ## Build the container image ($(IMAGE)) using repo root as build context
	@docker build -f $(CURDIR)/Dockerfile -t $(IMAGE) $(REPO_ROOT)

.PHONY: image-load-minikube
image-load-minikube: image ## Build and load the image into the local minikube node
	@minikube image load $(IMAGE)

.PHONY: clean
clean: ## Remove build artifacts
	@rm -rf bin/ coverage/

# --- OpenAPI ---
#
# Regenerates docs/openapi/compute-service.yaml from the Go request/response structs
# the runtime handlers use. The generator lives at cmd/openapi-gen and shares
# its reflection engine with artifact-hub and cluster-manager via pkg/openapigen.

OPENAPI_OUT ?= $(REPO_ROOT)/docs/openapi/compute-service.yaml

.PHONY: doc-gen
doc-gen: ## Regenerate $(OPENAPI_OUT) from the Go request/response types
	@go run ./cmd/openapi-gen -o $(OPENAPI_OUT) -version $(IMAGE_TAG)

.PHONY: doc-test
doc-test: ## Fail if $(OPENAPI_OUT) is out of date with the Go types (CI guard)
	@$(MAKE) --no-print-directory doc-gen
	@if ! git -C $(REPO_ROOT) diff --quiet -- $(OPENAPI_OUT); then \
	  echo "ERROR: $(OPENAPI_OUT) is out of date. Run 'make doc-gen' and commit the result."; \
	  git -C $(REPO_ROOT) --no-pager diff -- $(OPENAPI_OUT); \
	  exit 1; \
	fi

# --- Go convenience targets ---

.PHONY: fmt vet tidy
fmt: ## go fmt ./...
	@go fmt ./...

vet: ## go vet ./...
	@go vet ./...

tidy: ## go mod tidy
	@go mod tidy

# --- Integration ---
#
# Hermetic in-process tests against an embedded apiserver+etcd
# (controller-runtime envtest) plus a testcontainers-managed PostgreSQL. Uses
# the shared setup-envtest binary at $(REPO_ROOT)/test/setup-envtest/setup-envtest.

ENVTEST             := $(REPO_ROOT)/test/setup-envtest/setup-envtest
ENVTEST_K8S_VERSION ?= 1.31.0

.PHONY: integration
integration: ## Run integration tests (uses shared setup-envtest at test/setup-envtest/)
	@if [ ! -x "$(ENVTEST)" ]; then \
	  echo "ERROR: $(ENVTEST) not found. Run 'make setup-envtest' from $(REPO_ROOT)."; \
	  exit 1; \
	fi
	@cd test/integration && \
	  KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  go test -tags=integration -count=1 -timeout=15m ./...

# --- Coverage ---

COVERAGE_DIR              ?= $(CURDIR)/coverage
COVERAGE_OUT              ?= $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML             ?= $(COVERAGE_DIR)/coverage.html
INTEGRATION_COVERAGE_OUT  ?= $(COVERAGE_DIR)/integration-coverage.out
INTEGRATION_COVERAGE_HTML ?= $(COVERAGE_DIR)/integration-coverage.html
MODULE_PATH               := github.com/axisml/axisml/components/compute-service

.PHONY: coverage integration-coverage coverage-html

coverage: ## Run unit tests with coverage profile
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_OUT) -covermode=atomic -count=1 ./...

integration-coverage: ## Run integration tests with coverage profile (covers compute-service production packages)
	@mkdir -p $(COVERAGE_DIR)
	@if [ ! -x "$(ENVTEST)" ]; then \
	  echo "ERROR: $(ENVTEST) not found. Run 'make setup-envtest' from $(REPO_ROOT)."; \
	  exit 1; \
	fi
	@cd test/integration && \
	  KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  go test -tags=integration -count=1 -timeout=15m \
	    -coverpkg=$(MODULE_PATH)/... \
	    -coverprofile=$(INTEGRATION_COVERAGE_OUT) -covermode=atomic ./...

coverage-html: ## Render this component's coverage profile(s) as HTML
	@if [ ! -f $(COVERAGE_OUT) ] && [ ! -f $(INTEGRATION_COVERAGE_OUT) ]; then \
	  echo "no coverage profile found in $(COVERAGE_DIR); run 'make coverage' or 'make integration-coverage' first"; \
	  exit 1; \
	fi
	@if [ -f $(COVERAGE_OUT) ]; then \
	  go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML); \
	  echo "unit coverage: $(COVERAGE_HTML)"; \
	fi
	@if [ -f $(INTEGRATION_COVERAGE_OUT) ]; then \
	  go tool cover -html=$(INTEGRATION_COVERAGE_OUT) -o $(INTEGRATION_COVERAGE_HTML); \
	  echo "integration coverage: $(INTEGRATION_COVERAGE_HTML)"; \
	fi
