# artifacts Makefile
#
# Standard targets (per AxisML component conventions):
#   make / make build       Compile the binary into bin/
#   make image              Build the container image
#   make test               Run unit tests (no cluster, no testcontainers)
#   make integration        Run integration tests (PG + zot via testcontainers)
#   make clean              Remove build artifacts

IMAGE_REPO ?= ghcr.io/axisml/axisml-artifacts
# Must track deploy/helm/axisml-system Chart.appVersion. The top-level Makefile
# exports IMAGE_TAG from Chart.appVersion; this default applies only to direct
# per-component invocations.
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

BIN_NAME ?= artifacts
CMD_PATH ?= ./cmd/artifacts
REPO_ROOT := $(abspath $(CURDIR)/../..)

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile the binary into bin/$(BIN_NAME)
	@CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/$(BIN_NAME) $(CMD_PATH)

.PHONY: test
test: ## Run unit tests (no cluster, no testcontainers)
	@go test ./...

.PHONY: integration
integration: ## Run integration tests (PG + zot via testcontainers)
	@go test -tags=integration -count=1 -timeout=10m ./internal/integration/...

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

# --- Coverage ---

COVERAGE_DIR  ?= $(CURDIR)/coverage
COVERAGE_OUT  ?= $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML ?= $(COVERAGE_DIR)/coverage.html
MODULE_PATH   := github.com/axisml/axisml/components/artifacts

.PHONY: coverage coverage-html

coverage: ## Run unit tests with coverage profile
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_OUT) -covermode=atomic -count=1 ./...

coverage-html: ## Render this component's coverage profile as HTML
	@if [ ! -f $(COVERAGE_OUT) ]; then \
	  echo "no coverage profile found in $(COVERAGE_DIR); run 'make coverage' first"; \
	  exit 1; \
	fi
	@go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "unit coverage: $(COVERAGE_HTML)"

# --- L1 envtest stubs ---
#
# Artifacts watches no CRDs and uses build-tagged integration tests instead
# of an envtest sub-module. The top-level `envtest-test` and
# `coverage-envtest` targets fan out to every COMPONENT, so we expose
# these as no-ops to keep the fan-out contract.
#
# Run real integration tests with `make integration` (requires Docker).

.PHONY: envtest envtest-coverage
envtest: ## (artifacts has no envtest module; use `make integration` instead)
	@echo "artifacts: no envtest module; run 'make integration' for testcontainer-backed integration tests"

envtest-coverage: ## (no-op for artifacts)
	@echo "artifacts: no envtest module; coverage-envtest is a no-op"
