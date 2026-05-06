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

# cluster-manager has no L1 integration suite (no reconciler); the integration
# target is a no-op so the top-level fan-out doesn't fail.
.PHONY: integration integration-coverage
integration:
	@echo "(cluster-manager has no L1 integration suite; skipping)"

integration-coverage:
	@echo "(cluster-manager has no L1 integration coverage; skipping)"

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
