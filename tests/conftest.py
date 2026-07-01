"""Root pytest configuration: deployment-form selection, readiness gate, fixtures.

Form selection is pytest-native — a ``--mode {standard,lite}`` option (default
standard) plus ``standard``/``lite`` markers that gate collection. Heavy env
provisioning is NOT here (run ``uv run test-setup`` first); this conftest only
performs a read-only readiness check and wires the typed clients + tenants.
"""

from __future__ import annotations

import httpx
import pytest

from lib import config
from lib.harness import Harness, LiteHarness, StandardHarness


# --------------------------------------------------------------------------- #
# Form selection (pytest-native): --mode option + marker gating
# --------------------------------------------------------------------------- #
def pytest_addoption(parser: pytest.Parser) -> None:
    parser.addoption(
        "--mode",
        action="store",
        default="standard",
        choices=("standard", "lite"),
        help="deployment form under test (default: standard)",
    )


def pytest_collection_modifyitems(config: pytest.Config, items: list[pytest.Item]) -> None:
    """Shared-by-default form gating.

    Unmarked tests run under any --mode. A test marked ``<form>_only`` is skipped
    when a different form is selected. (Finer capability differences within a
    shared test are handled inline by ``harness.skip_unless``.)
    """
    mode = config.getoption("--mode")
    other = "lite" if mode == "standard" else "standard"
    skip = pytest.mark.skip(reason=f"{other}_only: not applicable to --mode={mode}")
    for item in items:
        if f"{other}_only" in item.keywords:
            item.add_marker(skip)


# --------------------------------------------------------------------------- #
# Core fixtures
# --------------------------------------------------------------------------- #
@pytest.fixture(scope="session")
def mode(pytestconfig: pytest.Config) -> str:
    return pytestconfig.getoption("--mode")


@pytest.fixture(scope="session")
def cfg() -> config.Config:
    return config.load()


@pytest.fixture(scope="session")
def harness(mode: str, cfg: config.Config) -> Harness:
    """Build the form's harness, gate on readiness, and tear down on exit."""
    h: Harness = StandardHarness(cfg) if mode == "standard" else LiteHarness(cfg)
    try:
        _gate_ready(h, mode, cfg)
    except Exception as e:  # noqa: BLE001 — surface a clear, actionable message
        h.close()
        pytest.exit(f"environment not ready ({e}). Run `uv run test-setup --mode {mode}` first.", returncode=3)
    yield h
    h.close()


def _gate_ready(h: Harness, mode: str, cfg: config.Config) -> None:
    """Read-only readiness gate: the default pool must exist (Standard) / readyz (Lite)."""
    from clients.clustermanager.api.resource_pools import get_resource_pool

    if mode == "lite":
        # axisml-core (System modules) and axisml-platform (API + SPA) are separate
        # containers — both must be up for the shared Platform/UI tests to run.
        for label, base in (("core", cfg.lite_base_url), ("platform", cfg.lite_platform_url)):
            r = httpx.get(f"{base}/readyz", timeout=10.0)
            if r.status_code != 200:
                raise RuntimeError(f"{label} GET {base}/readyz = {r.status_code}")
        return
    # Standard: the port-forwards (built in the harness) already prove HTTP
    # reachability; assert the default ResourcePool the tenants' quotas fold into.
    resp = get_resource_pool.sync_detailed(cfg.default_pool, client=h.cluster_manager)
    if resp.status_code != 200:
        raise RuntimeError(f"default ResourcePool '{cfg.default_pool}' not found ({resp.status_code})")


@pytest.fixture(scope="module")
def tenant(harness: Harness):
    """A fresh tenant (Standard) / the static default (Lite) shared by a module.

    Yields (namespace, quota). The namespace == tenant name; quota is the
    ElasticQuota workloads schedule under ('' when the form has none).
    """
    name = harness.new_tenant_name()
    quota = harness.create_tenant(name)
    try:
        yield name, quota
    finally:
        harness.delete_tenant(name)
