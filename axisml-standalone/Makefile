# AxisML Standalone — single-host distribution, public Go module and Docker image.

REPO_ROOT := $(abspath $(CURDIR)/..)

IMAGE_REPO ?= ghcr.io/axisml/axisml-standalone
IMAGE_TAG  ?= $(shell awk '/^appVersion:/{gsub(/"/,"",$$2);print $$2}' $(REPO_ROOT)/axisml-system/deploy/helm/Chart.yaml)
ENVOY_GATEWAY_VERSION ?= v$(shell awk '/alias: envoy-gateway/{found=1;next} found && /version:/{gsub(/"/,"",$$2);print $$2;exit}' $(REPO_ROOT)/axisml-infra/deploy/helm/Chart.yaml)
COMPOSE    := AXISML_VERSION=$(IMAGE_TAG) ENVOY_GATEWAY_VERSION=$(ENVOY_GATEWAY_VERSION) docker compose -f $(CURDIR)/compose.yaml
PROFILE_FLAGS := $(foreach p,$(PROFILES),--profile $(p))

.DEFAULT_GOAL := build

.PHONY: build test image image-load coverage coverage-html fmt vet tidy clean doc-gen doc-test ensure-workloads-network up down delete help
build: ## Build the axisml-standalone binary
	@CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/axisml-standalone ./cmd/axisml-standalone
test: ## Run standalone unit tests
	@go test ./...
image: ## Build the standalone image
	@docker build -f $(CURDIR)/Dockerfile -t $(IMAGE_REPO):$(IMAGE_TAG) $(REPO_ROOT)
image-load: image ## Build and load the standalone image into minikube
	@minikube image load $(IMAGE_REPO):$(IMAGE_TAG)
coverage: ## Write standalone unit coverage
	@mkdir -p coverage && go test -coverprofile=coverage/coverage.out -covermode=atomic -count=1 ./...
coverage-html: ## Render standalone unit coverage HTML
	@if [ -f coverage/coverage.out ]; then go tool cover -html=coverage/coverage.out -o coverage/coverage.html && echo "standalone unit: axisml-standalone/coverage/coverage.html"; fi
fmt: ## Format standalone Go packages
	@go fmt ./...
vet: ## Vet standalone Go packages
	@go vet ./...
tidy: ## Tidy standalone module dependencies
	@./scripts/tidy.sh
clean: ## Remove standalone build and coverage artifacts
	@rm -rf bin coverage
doc-gen: ## Regenerate the aggregate standalone OpenAPI spec
	@go run ./cmd/openapi-gen \
	  -o $(CURDIR)/docs/apis/standalone.yaml \
	  -embed $(CURDIR)/openapi.gen.yaml \
	  -version $(IMAGE_TAG)
doc-test: doc-gen ## Verify the aggregate OpenAPI spec is in sync
	@if ! git -C $(REPO_ROOT) diff --quiet -- \
	  axisml-standalone/docs/apis/standalone.yaml axisml-standalone/openapi.gen.yaml; then \
	  echo "ERROR: standalone OpenAPI artifacts are out of date. Run 'make -C axisml-standalone doc-gen' and commit."; \
	  git -C $(REPO_ROOT) --no-pager diff -- \
	    axisml-standalone/docs/apis/standalone.yaml axisml-standalone/openapi.gen.yaml; exit 1; fi
ensure-workloads-network:
	@docker network inspect axisml-workloads >/dev/null 2>&1 || docker network create axisml-workloads >/dev/null
up: ensure-workloads-network ## Build and start the standalone distribution
	@$(COMPOSE) $(PROFILE_FLAGS) up -d --build --remove-orphans
down: ## Stop the stack; CLEAN=1 also removes data volumes
	@$(COMPOSE) $(PROFILE_FLAGS) down $(if $(CLEAN),--volumes)
delete: ## Purge the stack and standalone-managed resources
	-@docker rm -f $$(docker ps -aq --filter "label=io.axisml.managed=true") 2>/dev/null || true
	@$(COMPOSE) down --volumes --remove-orphans
	-@docker volume rm $$(docker volume ls -q --filter "label=io.axisml.managed=true") 2>/dev/null || true
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'
