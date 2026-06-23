# axisml-compute-operator Makefile

IMAGE_REPO ?= ghcr.io/axisml/axisml-compute-operator
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

BIN_NAME  ?= axisml-compute-operator
CMD_PATH  ?= ./cmd
REPO_ROOT := $(abspath $(CURDIR)/../..)

.DEFAULT_GOAL := build

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile the operator binary into bin/$(BIN_NAME)
	@CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/$(BIN_NAME) $(CMD_PATH)

.PHONY: test
test: ## Run unit tests
	@go test ./...

.PHONY: image
image: ## Build the operator container image
	@docker build -f $(CURDIR)/Dockerfile -t $(IMAGE) $(REPO_ROOT)

.PHONY: image-load-minikube
image-load-minikube: image
	@minikube image load $(IMAGE)

.PHONY: clean
clean:
	@rm -rf bin/ coverage/

CONTROLLER_GEN_VERSION ?= v0.16.5
CONTROLLER_GEN         ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

.PHONY: fmt vet tidy generate
fmt:
	@go fmt ./...

vet:
	@go vet ./...

tidy:
	@go mod tidy

generate: ## Run controller-gen for deepcopy code (mlrun; mlservice deepcopy is hand-maintained)
	@$(CONTROLLER_GEN) object paths=./api/mlrun/...

# --- Integration ---

ENVTEST             := $(REPO_ROOT)/test/setup-envtest/setup-envtest
ENVTEST_K8S_VERSION ?= 1.31.0

.PHONY: integration
integration:
	@if [ ! -x "$(ENVTEST)" ]; then \
	  echo "ERROR: $(ENVTEST) not found. Run 'make setup-envtest' from $(REPO_ROOT)."; \
	  exit 1; \
	fi
	@cd test/integration && \
	  KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  go test -tags=integration -count=1 -timeout=10m ./...

# --- Coverage ---

COVERAGE_DIR              ?= $(CURDIR)/coverage
COVERAGE_OUT              ?= $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML             ?= $(COVERAGE_DIR)/coverage.html
INTEGRATION_COVERAGE_OUT  ?= $(COVERAGE_DIR)/integration-coverage.out
INTEGRATION_COVERAGE_HTML ?= $(COVERAGE_DIR)/integration-coverage.html
MODULE_PATH               := github.com/axisml/axisml/components/compute-operator

.PHONY: coverage integration-coverage coverage-html

coverage:
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_OUT) -covermode=atomic -count=1 ./...

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
	    -coverprofile=$(INTEGRATION_COVERAGE_OUT) -covermode=atomic ./...

coverage-html:
	@if [ -f $(COVERAGE_OUT) ]; then \
	  go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML); \
	fi
	@if [ -f $(INTEGRATION_COVERAGE_OUT) ]; then \
	  go tool cover -html=$(INTEGRATION_COVERAGE_OUT) -o $(INTEGRATION_COVERAGE_HTML); \
	fi
