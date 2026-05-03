# tenant-operator Makefile

IMAGE_REPO ?= axisml/tenant-operator
# Must track deploy/helm/axisml-system Chart.appVersion: the Helm template
# defaults image.tag to .Chart.AppVersion when unset, so locally-built images
# need this tag for `minikube image load` to satisfy the rendered Deployment.
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: fmt
fmt: ## go fmt ./...
	@go fmt ./...

.PHONY: vet
vet: ## go vet ./...
	@go vet ./...

.PHONY: build
build: ## Compile the operator binary into bin/tenant-operator
	@CGO_ENABLED=0 go build -trimpath -o bin/tenant-operator ./cmd

.PHONY: test
test: ## Run unit tests (no cluster required)
	@go test ./...

.PHONY: test-integration
test-integration: ## Run envtest integration tests against the kubeconfig context (run `make cluster-up && make helm-install-infra` first; scale any in-cluster tenant-operator to 0)
	@go test -tags integration -count=1 -timeout=10m ./test/envtest/...

.PHONY: image
image: ## Build the operator container image
	@docker build -t $(IMAGE) .

.PHONY: image-load-minikube
image-load-minikube: image ## Load the image into the local minikube node
	@minikube image load $(IMAGE)

.PHONY: clean
clean: ## Remove build artifacts
	@rm -rf bin/
