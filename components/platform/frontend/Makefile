# platform-frontend Makefile
#
# Standard targets (per AxisML component conventions):
#   make / make build  Build the production bundle into dist/
#   make image         Build the container image
#   make test          Run unit tests
#   make clean         Remove build artifacts (dist/, cache)
#
# NOTE: This component is not yet implemented. The targets below assume a
# TypeScript + React project (Vite-style) with package.json scripts:
#   build, test, dev, lint
# matching the layout planned in docs/system_design/platform.md.

IMAGE_REPO ?= ghcr.io/axisml/axisml/axisml-platform-frontend
# Must track deploy/helm/axisml-system Chart.appVersion (the Helm template
# defaults image.tag to .Chart.AppVersion when unset).
IMAGE_TAG  ?= 0.1.0
IMAGE      := $(IMAGE_REPO):$(IMAGE_TAG)

NPM ?= npm

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: install
install: ## Install npm dependencies
	@$(NPM) install

.PHONY: build
build: ## Build the production bundle into dist/
	@$(NPM) run build

.PHONY: test
test: ## Run unit tests
	@$(NPM) test

.PHONY: image
image: ## Build the container image ($(IMAGE))
	@docker build -t $(IMAGE) .

.PHONY: image-load-minikube
image-load-minikube: image ## Build and load the image into the local minikube node
	@minikube image load $(IMAGE)

.PHONY: clean
clean: ## Remove build artifacts (dist/, build/, cache)
	@rm -rf dist/ build/ node_modules/.cache

# --- Frontend convenience targets ---

.PHONY: dev lint
dev: ## Start the dev server
	@$(NPM) run dev

lint: ## Lint the codebase
	@$(NPM) run lint
