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
  e2e/          UI end-to-end (pytest-playwright)
    auth/  navigation/
```

## Deployment

The suite targets a real AxisML Kubernetes cluster through `kubectl port-forward`.

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
uv run test-setup                    # images -> minikube -> helm -> admin

uv run test-teardown                 # helm uninstall + cluster-down
uv run test-teardown --delete        # ... cluster-delete instead
```

`test-setup` resets the Platform `admin` password in the metadata DB to a known
suite value (clearing the forced first-login change), so the tests log in with a
deterministic credential regardless of the cluster's prior state.

## Run

```sh
uv run pytest api                    # all API tests
uv run pytest e2e                    # UI end-to-end
uv run pytest api/compute_service -k mlrun -v   # a slice
```

If the environment isn't ready, the session aborts with guidance to run
`uv run test-setup` first (a read-only gate — it never provisions).

## Configuration

Every knob has a stock-install default; override via env (see `lib/config.py`):
`AXISML_DEFAULT_POOL`, `AXISML_DEFAULT_UNIT`, `AXISML_MLRUN_IMAGE`,
`AXISML_ADMIN_PASSWORD`, the per-service `*_SVC`/`*_NS`, and the
timeout budgets.
