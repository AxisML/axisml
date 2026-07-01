# AxisML Lite layer Makefile. Builds the single axisml-core binary/image and
# runs the Lite unit + e2e suites. The Go module lives under axisml-core/; the
# image is built from the repo ROOT so the go.mod replace directives resolve the
# sibling System modules.

REPO_ROOT := $(abspath $(CURDIR)/..)
CORE_DIR := $(CURDIR)/axisml-core
IMAGE_TAG ?= dev
IMAGE ?= axisml-core:$(IMAGE_TAG)

# Compose stack: default services are PostgreSQL + axisml-core (published on
# :8090) + axisml-platform (API + UI on :8080). Extra services live behind
# profiles — add them with PROFILES="storage gateway". Requires registry access
# (image build + base-image / workload pulls).
COMPOSE := docker compose -f deploy/docker-compose.yaml
PROFILE_FLAGS := $(foreach p,$(PROFILES),--profile $(p))

.PHONY: help build test vet fmt tidy image doc-gen doc-test lite-up lite-down lite-delete e2e-test clean

# axisml-core's OpenAPI spec: the COMPLETE composite HTTP surface. openapi-gen
# folds the three System surfaces — built in-process from each System module's
# pkg/apidoc.Document builder (still owned by the System layer) — into the
# Lite-owned probes + aggregate capability endpoint, since axisml-core serves
# all three modules on one server. No YAML re-read, no ordering dependency on
# the System specs.
LITE_SPEC := docs/apis/axisml-core.yaml

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'

build: ## Build the axisml-core binary
	cd $(CORE_DIR) && go build -o bin/axisml-core ./cmd/axisml-core

test: ## Run unit tests
	cd $(CORE_DIR) && go test ./...

vet: ## go vet (incl. the centralized lite-form e2e build)
	cd $(CORE_DIR) && go vet ./...
	cd $(REPO_ROOT)/test/e2e && go vet -tags=lite ./...

fmt: ## Format
	cd $(CORE_DIR) && gofmt -w cmd internal pkg
	cd $(CORE_DIR) && go run golang.org/x/tools/cmd/goimports@latest -w cmd internal pkg 2>/dev/null || true
	$(MAKE) -C examples fmt

tidy: ## Tidy this module
	cd $(CORE_DIR) && go mod tidy

doc-gen: ## Regenerate the axisml-core OpenAPI spec (folds in the System surfaces)
	cd $(CORE_DIR) && go run ./cmd/openapi-gen -o $(REPO_ROOT)/axisml-lite/$(LITE_SPEC)

doc-test: doc-gen ## Verify the axisml-core spec is in sync with the code
	@cd $(REPO_ROOT) && if ! git diff --quiet -- axisml-lite/$(LITE_SPEC); then \
	  echo "ERROR: axisml-lite/$(LITE_SPEC) is out of date. Run 'make -C axisml-lite doc-gen' and commit."; \
	  git --no-pager diff -- axisml-lite/$(LITE_SPEC); exit 1; \
	fi

image: ## Build the axisml-core image (context = repo root)
	docker build -f $(CORE_DIR)/Dockerfile -t $(IMAGE) $(REPO_ROOT)

lite-up: ## Bring up the Lite stack (db + axisml-core on :8090 + platform UI on :8080; PROFILES="storage gateway" for more)
	$(COMPOSE) $(PROFILE_FLAGS) up -d --build

lite-down: ## Tear down the Lite stack (CLEAN=1 also removes the data volumes)
	$(COMPOSE) $(PROFILE_FLAGS) down $(if $(CLEAN),--volumes)

# Full purge: the Compose stack + its volumes/networks AND every resource the
# Standalone runtime spawned itself (workload containers + workspace
# volumes labeled io.axisml.managed=true), which `lite-down` does not touch.
lite-delete: ## Purge the Lite stack + all axisml-managed workload containers & volumes
	-@docker rm -f $$(docker ps -aq --filter "label=io.axisml.managed=true") 2>/dev/null || true
	$(COMPOSE) down --volumes --remove-orphans
	-@docker volume rm $$(docker volume ls -q --filter "label=io.axisml.managed=true") 2>/dev/null || true

# Bring the stack up first (`make lite-up`), then run the suite against it.
e2e-test: ## Run the centralized e2e suite (lite form) against $(LITE_CORE_URL) (default http://localhost:8090)
	cd $(REPO_ROOT)/test/e2e && go test -tags=lite -v -timeout 15m ./...

clean: ## Remove build artifacts
	rm -rf bin
