# artifacts Makefile
#
# Standard targets (per AxisML component conventions):
#   make / make build  Compile the binary into bin/
#   make image         Build the container image
#   make test          Run unit tests
#   make clean         Remove build artifacts
#
# NOTE: This component is not yet implemented. The targets below assume a
# Go service with a `./cmd` entrypoint and a co-located Dockerfile, matching
# the layout planned in docs/system_design/artifacts.md.

IMAGE_REPO ?= ghcr.io/axisml/axisml-artifacts
# Must track deploy/helm/axisml-system Chart.appVersion (the Helm template
# defaults image.tag to .Chart.AppVersion when unset).
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

BIN_NAME ?= artifacts
CMD_PATH ?= ./cmd

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile the binary into bin/$(BIN_NAME)
	@CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/$(BIN_NAME) $(CMD_PATH)

.PHONY: test
test: ## Run unit tests
	@go test ./...

.PHONY: image
image: ## Build the container image ($(IMAGE))
	@docker build -t $(IMAGE) .

.PHONY: image-load-minikube
image-load-minikube: image ## Build and load the image into the local minikube node
	@minikube image load $(IMAGE)

.PHONY: clean
clean: ## Remove build artifacts
	@rm -rf bin/

# --- Go convenience targets ---

.PHONY: fmt vet tidy
fmt: ## go fmt ./...
	@go fmt ./...

vet: ## go vet ./...
	@go vet ./...

tidy: ## go mod tidy
	@go mod tidy
