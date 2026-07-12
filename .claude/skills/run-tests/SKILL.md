---
name: run-tests
description: Run the AxisML test suites — Go unit, Go integration, and the black-box tests (API in Lite + Standard modes, plus the Playwright UI e2e). Use when the user asks to run tests, "跑测试 / 跑一下完整的测试", verify a change, check CI locally, or bring up a test environment. Covers every layer, environment bring-up/teardown, and the known bring-up gotchas.
user-invocable: true
---

# Running AxisML tests

Test layers, in run order. Pick by what the user asked for; if they say "完整的测试 / full tests" run **all of them** in the order below (unit → integration → Lite API → Standard API → UI e2e), stopping to fix the first failure. Lite API and Standard API are the same black-box API suite in two `--mode`s; the UI e2e suite (Playwright, `tests/e2e/`) is `standard_only`.

Run each long suite in the background and log to the scratchpad, then monitor for pass/fail lines rather than blocking. Report failures with the actual output; never claim green without seeing the summary line.

| Layer | Command (from repo root) | Backing | Notes |
|---|---|---|---|
| Go unit | `make test` | none (fake client) | fast, always run first |
| Go integration | `make integration-test` | envtest + testcontainers Postgres | needs Docker running |
| Lite API | `cd tests && uv run test-setup --mode lite` then `uv run pytest --mode lite api` | axisml-core + platform in Docker Compose | System CORE tests; platform/e2e/artifact_hub skip |
| Standard API | `cd tests && uv run test-setup` then `uv run pytest --mode standard api` | real `axisml` minikube cluster + helm | full coverage |
| UI e2e | `cd tests && uv run test-setup` then `uv run pytest e2e` | real `axisml` minikube cluster + helm (Standard) | Playwright UI e2e; `standard_only`; needs `uv run playwright install chromium` once |

Prerequisites: Docker (integration + both API layers), `minikube` (Standard + UI e2e), `uv` + `cd tests && uv sync` (API + e2e layers), plus `uv run playwright install chromium` once for the UI e2e. Install envtest once with `make setup-envtest` if integration complains about a missing binary.

## Go unit + integration

```sh
make test               # unit across every layer — expect no FAIL/panic
make integration-test   # integration across 6 components — needs Docker
```

Both are hermetic. A single module/test: `cd <component> && go test -run TestX ./internal/...` (add `-tags=integration` inside a `test/integration/` submodule).

## Black-box tests (API + UI e2e) — need a live environment

Provisioning is **not** a pytest fixture: bring an env up once, then run pytest against it many times. The readiness gate is read-only and aborts with guidance if the env isn't up.

```sh
cd tests
uv sync                                   # one-time: build venv
uv run test-setup --mode lite             # Lite: build images + compose up (core :8090, platform :8080, zot :5001)
uv run pytest --mode lite api             # → System CORE tests; standard_only + unsupported-capability tests skip

uv run test-setup                         # Standard: minikube (profile "axisml") → image-load → helm → seed admin
uv run pytest --mode standard api         # → full API coverage (port-forward, dynamic local ports)

uv run playwright install chromium        # one-time: browser for the UI e2e
uv run pytest e2e                         # → Playwright UI e2e over the Platform SPA (Standard-only)
```

Teardown when done:

```sh
cd tests
uv run test-teardown --mode lite [--clean]        # compose down (+ remove data volumes)
uv run test-teardown --mode standard [--delete]   # helm uninstall + cluster-down (--delete = cluster-delete)
```

Run a slice: `uv run pytest --mode standard api/compute_service -k mlrun -v`.

## Known bring-up gotchas (hit these before — check here first when setup or a suite fails)

1. **Frontend build break blocks BOTH API envs.** `test-setup` builds the `axisml-platform` image, whose Dockerfile runs `pnpm run build` (tsc). A TypeScript error there fails the image build → the whole env won't come up. The frontend is NOT in the default Go build aggregate, so `make build`/CI never catches it. Diagnose with `cd axisml-platform/frontend && pnpm run build`; fix the TS errors (usually stale hand-written code vs the generated `src/api/generated/types.gen.ts` contract), then re-run setup.

2. **Lite platform wants host :8080.** If another process holds :8080 (`lsof -nP -iTCP:8080 -sTCP:LISTEN`), the platform container fails to bind and `lite-up` errors — but core/db/zot are already up. Do NOT kill the other process. Remap the platform port with a scratchpad compose override and point pytest at it:
   ```yaml
   # override.yaml — !override replaces the base ports list (plain merge appends, keeping the 8080 conflict)
   services:
     axisml-platform:
       ports: !override ["8081:8080"]
   ```
   ```sh
   cd axisml-lite/deploy
   docker compose -f docker-compose.yaml -f /path/to/override.yaml up -d axisml-platform
   AXISML_LITE_PLATFORM_URL=http://localhost:8081 uv run pytest --mode lite api   # (run from tests/)
   ```
   The Lite readiness gate requires **both** core (:8090) and platform `/readyz` = 200, so platform must be up even though Lite-only runs skip the platform tests.

3. **Stale committed Python clients** (`tests/clients/`). Symptom: a client model `from_dict` throws `KeyError`/parse error (e.g. `ServerQuota.units` treated as required). The clients drift from the OpenAPI specs. Regenerate: `make -C tests client-gen`. If a schema was renamed, update the matching reference in `tests/lib/harness.py`.

4. **`<component>-image-load` uses the wrong minikube profile.** The system-layer make target defaults to profile `minikube`, but the cluster is profile `axisml` → "cluster 'minikube' does not exist". After changing a System component's Go code, load its rebuilt image and restart the deployment manually:
   ```sh
   make -C axisml-system <component>-image-load   # builds the image (the load step may error on the profile — image is still tagged)
   minikube image load ghcr.io/axisml/axisml-<component>:0.1.0 -p axisml
   kubectl rollout restart deploy/axisml-<component> -n axisml-system
   kubectl rollout status  deploy/axisml-<component> -n axisml-system --timeout=120s
   ```
   The image tag is `Chart.appVersion` from `axisml-system/deploy/helm/Chart.yaml` (currently `0.1.0`).

## Reporting

- Grep the log for the summary: `N passed`, `N failed`, `--- FAIL`, `ok  `, `panic`.
- After fixing a failure, re-run the **whole package/suite** (not just the one test) — these suites share state (one Postgres per Go package; one env per API mode), so an isolated pass can still collide in the full run.
- Conventions live in `docs/development_workflow.md` and `tests/README.md`.

## After the run: ask whether to tear down

If the run brought up a live environment (Lite compose or a Standard minikube cluster — i.e. any API/UI-e2e layer ran), **do not leave it dangling and do not tear it down unprompted**. Once you've reported the results, ask the user whether to close the environment, e.g. "测试已完成，是否关闭测试环境？". Then act on the answer:

- **Yes** → run the matching teardown (see the Teardown section above): `uv run test-teardown --mode lite [--clean]` for Lite, `uv run test-teardown --mode standard [--delete]` for Standard. Use `--clean` / `--delete` only if the user also wants the data volumes / cluster removed; otherwise keep them for a faster next bring-up.
- **No / keep it** → leave it running and remind them of the teardown command so they can close it later.

Skip this step when the run was Go-only (unit/integration) — those are hermetic and leave nothing to tear down.
