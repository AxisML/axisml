# AxisML black-box test suite (Python + pytest)

Treats the system as a black box: **API tests** drive each component's HTTP
contract through clients generated from its OpenAPI spec, and **UI e2e** drives
the Platform SPA with Playwright. White-box assertions (CR/ElasticQuota/HTTPRoute
shapes) live at each Go component's `test/integration/` layer, not here.

Dependencies are managed with [uv](https://docs.astral.sh/uv/); the suite is its
own project (`pyproject.toml`), independent of the Go modules.

## Layout

```
tests/
  setup/        env lifecycle (test-setup / test-teardown console scripts)
  lib/          harness, config, port-forward, polling, OCI push, builders
  clients/      generated OpenAPI clients (committed) — make -C tests client-gen
  api/          API tests, one dir per component (black-box HTTP)
    cluster_manager/  compute_service/  artifact_hub/  platform/  golden_path/
  e2e/          UI end-to-end (pytest-playwright, Standard-only)
    auth/  navigation/
```

## Deployment forms (`--mode`) — shared by default

The form is chosen with a pytest-native option (default `standard`):

- `--mode standard` — a real `axisml` Kubernetes cluster, reached over
  `kubectl port-forward`. Backs every capability.
- `--mode lite` — one `axisml-core` process (`:9080`).

**Tests are shared by default.** An *unmarked* test runs under whichever `--mode`
is selected — this is most of the suite (the System-layer lifecycle tests in
`cluster_manager` / `compute_service` run unchanged against both forms). Two
narrower mechanisms handle differences:

- **`@pytest.mark.standard_only` / `lite_only`** — for a test that needs a whole
  layer/capability the other form lacks. Lite has no Platform layer, no UI, no
  registry in the default compose, and no multi-tenant, so `platform/`, `e2e/`,
  `artifact_hub/` (real registry round-trip) and `golden_path/` are
  `standard_only`. These *skip* under the other `--mode`.
- **`harness.skip_unless(Capability.X)`** — for a *shared* test with a
  form-specific sub-behaviour (e.g. the cluster-manager tenant test runs on both
  forms but skips its write path on Lite via `MULTI_TENANT`). Prefer this over a
  whole-test marker whenever only part of a test is form-specific.

Capabilities are read from each service's `/api/v1/capabilities` document.

## One-time setup

```sh
cd tests
uv sync                              # build the venv + install deps
uv run playwright install chromium   # browser for the UI e2e
make -C . client-gen                 # (re)generate clients after a spec change
```

## Bring an environment up / down

Provisioning is **not** a pytest fixture — bring an environment up once, then run
pytest against it many times.

```sh
uv run test-setup                    # Standard: images -> minikube -> helm -> admin
uv run test-setup --mode lite        # Lite: build axisml-core + compose up (:9080)

uv run test-teardown                 # Standard: helm uninstall + cluster-down
uv run test-teardown --mode standard --delete   # ... cluster-delete instead
uv run test-teardown --mode lite [--clean]
```

`test-setup` resets the Platform `admin` password in the metadata DB to a known
suite value (clearing the forced first-login change), so the tests log in with a
deterministic credential regardless of the cluster's prior state.

## Run

```sh
uv run pytest --mode standard api    # all API tests against Standard
uv run pytest --mode lite api        # System CORE tests against Lite
uv run pytest e2e                    # UI end-to-end (Standard)
uv run pytest api/compute_service -k mlrun -v   # a slice
```

If the environment isn't ready, the session aborts with guidance to run
`uv run test-setup` first (a read-only gate — it never provisions).

## Configuration

Every knob has a stock-install default; override via env (see `lib/config.py`):
`AXISML_DEFAULT_POOL`, `AXISML_DEFAULT_UNIT`, `AXISML_MLRUN_IMAGE`,
`AXISML_ADMIN_PASSWORD`, `AXISML_LITE_URL`, the per-service `*_SVC`/`*_NS`, and the
timeout budgets.
