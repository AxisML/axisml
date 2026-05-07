# axisml-cluster-manager Makefile

IMAGE_REPO ?= ghcr.io/axisml/axisml-cluster-manager
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

BIN_NAME  ?= axisml-cluster-manager
CMD_PATH  ?= ./cmd/cluster-manager
REPO_ROOT := $(abspath $(CURDIR)/../..)

.DEFAULT_GOAL := build

.PHONY: help build test image image-load-minikube clean fmt vet tidy coverage coverage-html

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

build: ## Compile the cluster-manager binary into bin/
	@CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/$(BIN_NAME) $(CMD_PATH)

test: ## Run unit tests
	@go test ./...

image: ## Build the container image
	@docker build -f $(CURDIR)/Dockerfile -t $(IMAGE) $(REPO_ROOT)

image-load-minikube: image ## Build and load the image into the local minikube node
	@minikube image load $(IMAGE)

clean:
	@rm -rf bin/ coverage/

fmt:
	@go fmt ./...

vet:
	@go vet ./...

tidy:
	@go mod tidy

# --- OpenAPI ---
#
# Regenerates docs/openapi/cluster-manager.yaml from the Go request/response
# structs the runtime handlers use. The generator lives at cmd/openapi-gen
# and shares its reflection engine with compute and artifacts via
# pkg/openapigen.

OPENAPI_OUT ?= $(REPO_ROOT)/docs/openapi/cluster-manager.yaml

.PHONY: doc-gen doc-test
doc-gen: ## Regenerate $(OPENAPI_OUT) from the Go request/response types
	@go run ./cmd/openapi-gen -o $(OPENAPI_OUT) -version $(IMAGE_TAG)

doc-test: ## Fail if $(OPENAPI_OUT) is out of date with the Go types (CI guard)
	@$(MAKE) --no-print-directory doc-gen
	@if ! git -C $(REPO_ROOT) diff --quiet -- $(OPENAPI_OUT); then \
	  echo "ERROR: $(OPENAPI_OUT) is out of date. Run 'make doc-gen' and commit the result."; \
	  git -C $(REPO_ROOT) --no-pager diff -- $(OPENAPI_OUT); \
	  exit 1; \
	fi

ENVTEST             := $(REPO_ROOT)/test/setup-envtest/setup-envtest
ENVTEST_K8S_VERSION ?= 1.31.0
MODULE_PATH         := github.com/axisml/axisml/components/cluster-manager

.PHONY: integration integration-coverage
integration: ## Run L1 integration tests (envtest with Tenant CRD)
	@if [ ! -x "$(ENVTEST)" ]; then \
	  echo "ERROR: $(ENVTEST) not found. Run 'make setup-envtest' from $(REPO_ROOT)."; \
	  exit 1; \
	fi
	@cd test/integration && \
	  KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  go test -tags=integration -count=1 -timeout=10m ./...

integration-coverage:
	@mkdir -p $(COVERAGE_DIR)
	@if [ ! -x "$(ENVTEST)" ]; then \
	  echo "ERROR: $(ENVTEST) not found. Run 'make setup-envtest' from $(REPO_ROOT)."; \
	  exit 1; \
	fi
	@cd test/integration && \
	  KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  go test -tags=integration -count=1 -timeout=10m \
	    -coverpkg=$(MODULE_PATH)/... \
	    -coverprofile=$(COVERAGE_DIR)/integration-coverage.out -covermode=atomic ./...

COVERAGE_DIR  ?= $(CURDIR)/coverage
COVERAGE_OUT  ?= $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML ?= $(COVERAGE_DIR)/coverage.html

coverage:
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_OUT) -covermode=atomic -count=1 ./...

coverage-html:
	@if [ -f $(COVERAGE_OUT) ]; then \
	  go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML); \
	  echo "unit coverage: $(COVERAGE_HTML)"; \
	fi
