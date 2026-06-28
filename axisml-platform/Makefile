# axisml-platform Makefile — the whole Platform layer in one file.
#
# Backend (Go) is the layer's build/test aggregate; the frontend (pnpm) is a
# scaffold exposed via frontend-* targets (not part of the default aggregate).
# Also owns the Platform Helm chart. The repo-root Makefile delegates here.

REPO_ROOT := $(abspath $(CURDIR)/..)
CHART     := $(CURDIR)/deploy/helm
IMAGE_TAG ?= $(shell awk '/^appVersion:/{gsub(/"/,"",$$2);print $$2}' $(REPO_ROOT)/axisml-system/deploy/helm/Chart.yaml)
MINIKUBE_PROFILE ?= axisml

BACKEND_DIR    := backend
FRONTEND_DIR   := frontend
WEB_DIR        := $(BACKEND_DIR)/internal/web/dist
# Single Platform image: backend binary that also serves the frontend SPA.
PLATFORM_IMAGE ?= ghcr.io/axisml/axisml-platform:$(IMAGE_TAG)
BACKEND_MODULE := github.com/axisml/axisml/components/platform
OAPI_CODEGEN   ?= go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1
SPEC_DIR       := $(REPO_ROOT)/axisml-system/docs/apis
PNPM           ?= pnpm

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

# --- Backend (the layer's build/test aggregate) ---
.PHONY: build test image image-load integration coverage integration-coverage coverage-html doc-gen doc-test client-gen fmt vet tidy clean
build: ## Build the backend binary
	@cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/axisml-platform-backend ./cmd/platform-backend
test: ## Unit-test the backend
	@cd $(BACKEND_DIR) && go test ./...
image: ## Build the Platform image — frontend SPA + backend in one (repo root as build context)
	@docker build -f $(CURDIR)/Dockerfile -t $(PLATFORM_IMAGE) $(REPO_ROOT)
image-load: image ## Build + load the Platform image into minikube
	@minikube image load $(PLATFORM_IMAGE)
integration: ## Backend integration tests (in-process httptest)
	@cd $(BACKEND_DIR)/test/integration && go test -tags=integration -count=1 -timeout=10m ./...
coverage: ## Backend unit coverage
	@mkdir -p $(BACKEND_DIR)/coverage && cd $(BACKEND_DIR) && go test -coverprofile=coverage/coverage.out -covermode=atomic -count=1 ./...
integration-coverage: ## Backend integration coverage
	@mkdir -p $(BACKEND_DIR)/coverage && cd $(BACKEND_DIR)/test/integration && \
	  go test -tags=integration -count=1 -timeout=10m \
	    -coverpkg=$(BACKEND_MODULE)/... \
	    -coverprofile=$(CURDIR)/$(BACKEND_DIR)/coverage/integration-coverage.out -covermode=atomic ./...
coverage-html: ## Render backend HTML coverage
	@if [ -f $(BACKEND_DIR)/coverage/coverage.out ]; then cd $(BACKEND_DIR) && go tool cover -html=coverage/coverage.out -o coverage/coverage.html && echo "backend unit: $(BACKEND_DIR)/coverage/coverage.html"; fi
	@if [ -f $(BACKEND_DIR)/coverage/integration-coverage.out ]; then cd $(BACKEND_DIR) && go tool cover -html=coverage/integration-coverage.out -o coverage/integration-coverage.html && echo "backend integration: $(BACKEND_DIR)/coverage/integration-coverage.html"; fi
doc-gen: ## Regenerate docs/apis/platform.yaml from backend DTOs
	@cd $(BACKEND_DIR) && go run ./cmd/openapi-gen -o $(CURDIR)/docs/apis/platform.yaml -version $(IMAGE_TAG)
doc-test: doc-gen ## Verify docs/apis/platform.yaml is in sync
	@if ! git -C $(REPO_ROOT) diff --quiet -- axisml-platform/docs/apis/platform.yaml; then \
	  echo "ERROR: axisml-platform/docs/apis/platform.yaml is out of date. Run 'make -C axisml-platform doc-gen' and commit."; \
	  git -C $(REPO_ROOT) --no-pager diff -- axisml-platform/docs/apis/platform.yaml; exit 1; fi
client-gen: ## Regenerate backend's typed System clients from axisml-system/docs/apis
	@cd $(BACKEND_DIR) && $(OAPI_CODEGEN) -config internal/clients/clustermanager/generated/oapi-codegen.yaml $(SPEC_DIR)/cluster-manager.yaml
	@cd $(BACKEND_DIR) && $(OAPI_CODEGEN) -config internal/clients/computeservice/generated/oapi-codegen.yaml $(SPEC_DIR)/compute-service.yaml
	@cd $(BACKEND_DIR) && $(OAPI_CODEGEN) -config internal/clients/artifacthub/generated/oapi-codegen.yaml $(SPEC_DIR)/artifact-hub.yaml
	@cd $(BACKEND_DIR) && go mod tidy
fmt: ## go fmt the backend
	@cd $(BACKEND_DIR) && go fmt ./...
vet: ## go vet the backend
	@cd $(BACKEND_DIR) && go vet ./...
tidy: ## go mod tidy the backend
	@cd $(BACKEND_DIR) && go mod tidy
clean: ## Remove backend + frontend build artifacts
	@rm -rf $(BACKEND_DIR)/bin $(BACKEND_DIR)/coverage
	@rm -rf $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/build $(FRONTEND_DIR)/node_modules/.cache $(FRONTEND_DIR)/tsconfig.tsbuildinfo
	@find $(WEB_DIR) -mindepth 1 ! -name index.html ! -name .gitignore -delete 2>/dev/null || true
	@git -C $(REPO_ROOT) checkout -- $(WEB_DIR)/index.html 2>/dev/null || true

# --- Frontend (pnpm; built into the Platform image via the Dockerfile) ---
.PHONY: frontend-install frontend-build frontend-stage frontend-test frontend-gen-api frontend-gen-mock frontend-lint frontend-dev
frontend-install: ## pnpm install
	@cd $(FRONTEND_DIR) && $(PNPM) install
frontend-build: ## Build the frontend bundle
	@cd $(FRONTEND_DIR) && $(PNPM) run build
frontend-stage: frontend-build ## Build the SPA and stage it into the backend's static dir (for a local `make build` that serves the real frontend)
	@cp -r $(FRONTEND_DIR)/dist/. $(WEB_DIR)/
frontend-test: ## Frontend unit tests
	@cd $(FRONTEND_DIR) && $(PNPM) test
frontend-gen-api: ## Regenerate the typed API client + mock fixtures from docs/apis/platform.yaml
	@cd $(FRONTEND_DIR) && $(PNPM) run gen:api
frontend-gen-mock: ## Regenerate the in-browser mock fixtures from the spec's examples
	@cd $(FRONTEND_DIR) && $(PNPM) run gen:mock
frontend-lint: ## Lint the frontend
	@cd $(FRONTEND_DIR) && $(PNPM) run lint
frontend-dev: ## Start the frontend dev server
	@cd $(FRONTEND_DIR) && $(PNPM) run dev

# --- Helm (platform layer) ---
HELM_RELEASE    ?= axisml-platform
HELM_NAMESPACE  ?= axisml-platform
HELM_EXTRA_ARGS ?=
HELM_IMAGE_SET  := --set platform.image.tag=$(IMAGE_TAG)

.PHONY: helm-deps helm-lint helm-template helm-install helm-upgrade helm-uninstall
helm-deps: ## Fetch sub-chart tarballs (none today; no-op)
	@helm dependency update $(CHART) 2>/dev/null || true
helm-lint: ## Lint the platform chart
	@helm lint $(CHART)
helm-template: ## Render the platform chart locally
	@helm template $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) $(HELM_IMAGE_SET) $(HELM_EXTRA_ARGS)
helm-install: ## Install or upgrade the platform release (idempotent)
	@helm upgrade --install $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) --create-namespace --kube-context $(MINIKUBE_PROFILE) $(HELM_IMAGE_SET) $(HELM_EXTRA_ARGS)
helm-upgrade: ## Upgrade the platform release (must already be installed)
	@helm upgrade $(HELM_RELEASE) $(CHART) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE) $(HELM_IMAGE_SET) $(HELM_EXTRA_ARGS)
helm-uninstall: ## Uninstall the platform release
	@helm uninstall $(HELM_RELEASE) -n $(HELM_NAMESPACE) --kube-context $(MINIKUBE_PROFILE)
