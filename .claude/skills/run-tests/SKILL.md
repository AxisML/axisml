---
name: run-tests
description: Run the AxisML test suites — Go unit, Go integration, black-box API, and Playwright UI e2e. Use when the user asks to run tests, "跑测试 / 跑一下完整的测试", verify a change, check CI locally, or bring up a test environment. Covers every layer, environment bring-up/teardown, and known bring-up gotchas.
user-invocable: true
---

# Running AxisML tests

Run test layers in this order: unit → integration → API → UI e2e. If the user asks for the full suite, run all four and stop to fix the first failure. Run long suites in the background with scratchpad logs, monitor their summaries, and report actual output.

| Layer | Command (from repo root) | Backing | Notes |
|---|---|---|---|
| Go unit | `make test` | none (fake client) | fast, always run first |
| Go integration | `make integration-test` | envtest + testcontainers Postgres | needs Docker |
| API | `cd tests && uv run test-setup` then `uv run pytest api` | real `axisml` minikube cluster + Helm | black-box HTTP coverage |
| UI e2e | `cd tests && uv run test-setup` then `uv run pytest e2e` | the same minikube cluster + Helm | needs Chromium installed once |

Prerequisites: Docker, `minikube`, `kubectl`, `helm`, and `uv`. Run `cd tests && uv sync` before black-box tests, `uv run playwright install chromium` before UI e2e, and `make setup-envtest` if integration reports a missing envtest binary.

## Go unit and integration

```sh
make test
make integration-test
```

Both are hermetic. To run one Go test, use `cd <component> && go test -run TestX ./internal/...`; add `-tags=integration` inside a `test/integration/` module.

## Black-box API and UI tests

Provisioning is not a pytest fixture. Bring the environment up once, then run pytest against it repeatedly:

```sh
cd tests
uv sync
uv run test-setup
uv run pytest api
uv run playwright install chromium
uv run pytest e2e
```

Run a slice with `uv run pytest api/compute_service -k mlrun -v`. Tear down with `uv run test-teardown`; add `--delete` only when the user also wants the minikube cluster removed.

## Known bring-up gotchas

1. The Platform image build runs `pnpm run build`, so a frontend TypeScript error blocks environment setup even though the frontend is not part of the default Go build. Diagnose with `cd axisml-platform/frontend && pnpm run build`.
2. If generated Python clients under `tests/clients/` drift from OpenAPI, parsing can fail with `KeyError` or model errors. Regenerate with `make -C tests client-gen` and update any matching references in `tests/lib/harness.py`.
3. A component image-load target may use minikube profile `minikube` while this repo uses `axisml`. Load the built image explicitly with `minikube image load <image> -p axisml`, then restart and wait for the affected deployment.

## Reporting

- Confirm the summary (`N passed`, `N failed`, `--- FAIL`, `ok`, or `panic`) before reporting results.
- After fixing a failure, rerun the whole package or suite because tests may share state.
- Conventions live in `docs/development_workflow.md` and `tests/README.md`.

If API or UI testing brought up a live environment, ask whether to close it after reporting results. On approval, run `cd tests && uv run test-teardown`; use `--delete` only if requested. Skip this for hermetic Go-only runs.
