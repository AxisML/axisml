# mljob-operator Makefile
#
# Standard targets (per AxisML component conventions):
#   make / make build  Compile the binary into bin/
#   make image         Build the container image
#   make test          Run unit tests (no cluster required)
#   make clean         Remove build artifacts

IMAGE_REPO ?= axisml/mljob-operator
# Must track deploy/helm/axisml-system Chart.appVersion: the Helm template
# defaults image.tag to .Chart.AppVersion when unset, so locally-built images
# need this tag for `minikube image load` to satisfy the rendered Deployment.
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

BIN_NAME ?= mljob-operator
CMD_PATH ?= ./cmd/manager

CONTROLLER_GEN_VERSION ?= v0.15.0
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile the manager binary into bin/$(BIN_NAME)
	@CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/$(BIN_NAME) $(CMD_PATH)

.PHONY: test
test: ## Run unit tests (no cluster required)
	@go test ./... -count=1

.PHONY: image
image: ## Build the operator container image ($(IMAGE))
	@docker build -t $(IMAGE) .

.PHONY: image-load-minikube
image-load-minikube: image ## Build and load the image into the local minikube node
	@minikube image load $(IMAGE)

.PHONY: clean
clean: ## Remove build artifacts
	@rm -rf bin/

# --- Go convenience targets ---

.PHONY: fmt vet tidy generate
fmt: ## go fmt ./...
	@go fmt ./...

vet: ## go vet ./...
	@go vet ./...

tidy: ## go mod tidy
	@go mod tidy

generate: ## Run controller-gen for deepcopy code
	$(CONTROLLER_GEN) object:headerFile=hack/boilerplate.go.txt paths=./api/...

# --- L1 envtest ---
#
# Hermetic in-process reconciler tests. test/envtest/ is its own Go module
# so this operator's production go.mod stays free of test-only deps; the
# Dockerfile builds from this directory and a sibling-test replace would
# fall outside its build context.

REPO_ROOT           := $(abspath $(CURDIR)/../../..)
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
	  go test -tags=envtest -count=1 -timeout=10m ./...
