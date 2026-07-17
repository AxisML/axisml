# AxisML test-suite layer Makefile.
#
# Deliberately tiny: the only thing that genuinely belongs in make here is
# regenerating the typed Python clients from the OpenAPI specs (analogous to the
# old Go `e2e-client-gen`). Everything else is a `uv` command:
#
#   uv sync                         install deps + build the venv
#   uv run playwright install chromium
#   uv run test-setup
#   uv run test-teardown [--delete]
#   uv run pytest api
#   uv run pytest e2e
#
# The repo-root Makefile delegates `client-gen` here (`make -C tests client-gen`).

SPEC_DIR_SYSTEM   := ../axisml-system/docs/apis
SPEC_DIR_PLATFORM := ../axisml-platform/docs/apis

# (spec : output package under clients/)
CLIENTS := \
	$(SPEC_DIR_SYSTEM)/cluster-manager.yaml:clustermanager \
	$(SPEC_DIR_SYSTEM)/compute-service.yaml:computeservice \
	$(SPEC_DIR_SYSTEM)/artifact-hub.yaml:artifacthub \
	$(SPEC_DIR_PLATFORM)/platform.yaml:platform

.PHONY: help client-gen
help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n",$$1,$$2}'

client-gen: ## Regenerate the typed Python clients from the OpenAPI specs
	@for pair in $(CLIENTS); do \
		spec=$${pair%%:*}; out=clients/$${pair##*:}; \
		echo ">> $$spec -> $$out"; \
		rm -rf $$out; \
		uv run openapi-python-client generate --path $$spec --meta none --output-path $$out --overwrite || exit 1; \
		rm -rf $$out/.ruff_cache; \
	done
	@echo "client-gen: done"
