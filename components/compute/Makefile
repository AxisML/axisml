# compute Makefile
#
# Standard targets (per AxisML component conventions):
#   make / make build  Compile the binary into bin/
#   make image         Build the container image
#   make test          Run unit tests (no cluster required)
#   make clean         Remove build artifacts
#
# NOTE: Compute imports operator API packages via local replace directives,
# so `image` builds with the repo root as build context.

IMAGE_REPO ?= axisml/compute
# Must track deploy/helm/axisml-system Chart.appVersion: the Helm template
# defaults image.tag to .Chart.AppVersion when unset, so locally-built images
# need this tag for `minikube image load` to satisfy the rendered Deployment.
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

BIN_NAME ?= compute
CMD_PATH ?= ./cmd/compute
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

# --- Go convenience targets ---

.PHONY: fmt vet tidy
fmt: ## go fmt ./...
	@go fmt ./...

vet: ## go vet ./...
	@go vet ./...

tidy: ## go mod tidy
	@go mod tidy

# --- L1 envtest ---
#
# Hermetic in-process tests against an embedded apiserver+etcd plus a
# testcontainers-managed PostgreSQL. Uses the shared setup-envtest binary at
# $(REPO_ROOT)/test/setup-envtest/setup-envtest.

ENVTEST             := $(REPO_ROOT)/test/setup-envtest/setup-envtest
ENVTEST_K8S_VERSION ?= 1.31.0

.PHONY: envtest
envtest: ## Run L1 envtest (uses shared setup-envtest at test/setup-envtest/)
	@if [ ! -x "$(ENVTEST)" ]; then \
	  echo "ERROR: $(ENVTEST) not found. Run 'make setup-envtest' from $(REPO_ROOT)."; \
	  exit 1; \
	fi
	@cd test/envtest && \
	  KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  go test -tags=envtest -count=1 -timeout=15m ./...

# --- Coverage ---

COVERAGE_DIR          ?= $(CURDIR)/coverage
COVERAGE_OUT          ?= $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML         ?= $(COVERAGE_DIR)/coverage.html
ENVTEST_COVERAGE_OUT  ?= $(COVERAGE_DIR)/envtest-coverage.out
ENVTEST_COVERAGE_HTML ?= $(COVERAGE_DIR)/envtest-coverage.html
MODULE_PATH           := github.com/axisml/axisml/components/compute

.PHONY: coverage envtest-coverage coverage-html

coverage: ## Run unit tests with coverage profile
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_OUT) -covermode=atomic -count=1 ./...

envtest-coverage: ## Run L1 envtest with coverage profile (covers compute production packages)
	@mkdir -p $(COVERAGE_DIR)
	@if [ ! -x "$(ENVTEST)" ]; then \
	  echo "ERROR: $(ENVTEST) not found. Run 'make setup-envtest' from $(REPO_ROOT)."; \
	  exit 1; \
	fi
	@cd test/envtest && \
	  KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	  go test -tags=envtest -count=1 -timeout=15m \
	    -coverpkg=$(MODULE_PATH)/... \
	    -coverprofile=$(ENVTEST_COVERAGE_OUT) -covermode=atomic ./...

coverage-html: ## Render this component's coverage profile(s) as HTML
	@if [ ! -f $(COVERAGE_OUT) ] && [ ! -f $(ENVTEST_COVERAGE_OUT) ]; then \
	  echo "no coverage profile found in $(COVERAGE_DIR); run 'make coverage' or 'make envtest-coverage' first"; \
	  exit 1; \
	fi
	@if [ -f $(COVERAGE_OUT) ]; then \
	  go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML); \
	  echo "unit coverage: $(COVERAGE_HTML)"; \
	fi
	@if [ -f $(ENVTEST_COVERAGE_OUT) ]; then \
	  go tool cover -html=$(ENVTEST_COVERAGE_OUT) -o $(ENVTEST_COVERAGE_HTML); \
	  echo "envtest coverage: $(ENVTEST_COVERAGE_HTML)"; \
	fi
