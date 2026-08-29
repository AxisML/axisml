"""Root pytest configuration shared by both AxisML deployment forms."""

from __future__ import annotations

import httpx
import pytest

from lib import config
from lib.harness import Harness, KubernetesHarness, StandaloneHarness


def pytest_addoption(parser: pytest.Parser) -> None:
    parser.addoption(
        "--mode",
        action="store",
        default="kubernetes",
        choices=("kubernetes", "standalone"),
        help="deployment form under test",
    )


def pytest_collection_modifyitems(
    config: pytest.Config, items: list[pytest.Item]
) -> None:
    mode = config.getoption("--mode")
    other = "standalone" if mode == "kubernetes" else "kubernetes"
    skip = pytest.mark.skip(reason=f"{other}_only: not applicable to --mode={mode}")
    for item in items:
        if f"{other}_only" in item.keywords:
            item.add_marker(skip)


# --------------------------------------------------------------------------- #
# Core fixtures
# --------------------------------------------------------------------------- #
@pytest.fixture(scope="session")
def cfg() -> config.Config:
    return config.load()


@pytest.fixture(scope="session")
def mode(pytestconfig: pytest.Config) -> str:
    return pytestconfig.getoption("--mode")


@pytest.fixture(scope="session")
def harness(mode: str, cfg: config.Config) -> Harness:
    """Build the selected harness, gate on readiness, and close it on exit."""
    h: Harness = (
        KubernetesHarness(cfg) if mode == "kubernetes" else StandaloneHarness(cfg)
    )
    try:
        _gate_ready(h, mode, cfg)
    except Exception as e:  # noqa: BLE001 — surface a clear, actionable message
        h.close()
        pytest.exit(
            f"environment not ready ({e}). Run `uv run test-setup --mode {mode}` first.",
            returncode=3,
        )
    yield h
    h.close()


def _gate_ready(h: Harness, mode: str, cfg: config.Config) -> None:
    """Read-only readiness gate for the selected deployment form."""
    from clients.clustermanager.api.resource_pools import get_resource_pool

    if mode == "standalone":
        for label, base in (
            ("system", cfg.standalone_system_url),
            ("platform", cfg.standalone_platform_url),
        ):
            response = httpx.get(f"{base}/readyz", timeout=10.0)
            if response.status_code != 200:
                raise RuntimeError(f"{label} GET {base}/readyz = {response.status_code}")
        return

    resp = get_resource_pool.sync_detailed(cfg.default_pool, client=h.cluster_manager)
    if resp.status_code != 200:
        raise RuntimeError(
            f"default ResourcePool '{cfg.default_pool}' not found ({resp.status_code})"
        )


@pytest.fixture(scope="module")
def tenant(harness: Harness):
    """A fresh tenant on Kubernetes or the static tenant on standalone.

    Yields the tenant scope used as the compute namespace path parameter.
    """
    name = harness.new_tenant_name()
    harness.create_tenant(name)
    try:
        yield name
    finally:
        harness.delete_tenant(name)
