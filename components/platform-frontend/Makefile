# platform-frontend Makefile
#
# Standard targets (per AxisML component conventions):
#   make / make build  Build the production bundle into dist/
#   make image         Build the container image
#   make test          Run unit tests
#   make clean         Remove build artifacts (dist/, cache)
#
# TypeScript + React (Vite) project managed with pnpm. package.json scripts:
#   build, test, dev, lint, gen:api

IMAGE_REPO ?= ghcr.io/axisml/axisml-platform-frontend
# Must track deploy/helm/axisml-system Chart.appVersion (the Helm template
# defaults image.tag to .Chart.AppVersion when unset).
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

PNPM ?= pnpm

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: install
install: ## Install dependencies (pnpm)
	@$(PNPM) install

.PHONY: gen-api
gen-api: ## Regenerate the typed API client from docs/openapi/platform.yaml
	@$(PNPM) run gen:api

.PHONY: build
build: ## Build the production bundle into dist/
	@$(PNPM) run build

.PHONY: test
test: ## Run unit tests
	@$(PNPM) test

.PHONY: image
image: ## Build the container image ($(IMAGE))
	@docker build -t $(IMAGE) .

.PHONY: image-load-minikube
image-load-minikube: image ## Build and load the image into the local minikube node
	@minikube image load $(IMAGE)

.PHONY: clean
clean: ## Remove build artifacts (dist/, build/, cache)
	@rm -rf dist/ build/ node_modules/.cache tsconfig.tsbuildinfo

# --- Frontend convenience targets ---

.PHONY: dev lint
dev: ## Start the dev server
	@$(PNPM) run dev

lint: ## Lint the codebase
	@$(PNPM) run lint
