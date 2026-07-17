"""Root pytest configuration for the Kubernetes Standard deployment."""

from __future__ import annotations

import pytest

from lib import config
from lib.harness import Harness, StandardHarness


# --------------------------------------------------------------------------- #
# Core fixtures
# --------------------------------------------------------------------------- #
@pytest.fixture(scope="session")
def cfg() -> config.Config:
    return config.load()


@pytest.fixture(scope="session")
def harness(cfg: config.Config) -> Harness:
    """Build the Standard harness, gate on readiness, and tear down on exit."""
    h: Harness = StandardHarness(cfg)
    try:
        _gate_ready(h, cfg)
    except Exception as e:  # noqa: BLE001 — surface a clear, actionable message
        h.close()
        pytest.exit(
            f"environment not ready ({e}). Run `uv run test-setup` first.",
            returncode=3,
        )
    yield h
    h.close()


def _gate_ready(h: Harness, cfg: config.Config) -> None:
    """Read-only readiness gate: the default ResourcePool must exist."""
    from clients.clustermanager.api.resource_pools import get_resource_pool

    # The port-forwards (built in the harness) already prove HTTP
    # reachability; assert the default ResourcePool the tenants' quotas fold into.
    resp = get_resource_pool.sync_detailed(cfg.default_pool, client=h.cluster_manager)
    if resp.status_code != 200:
        raise RuntimeError(
            f"default ResourcePool '{cfg.default_pool}' not found ({resp.status_code})"
        )


@pytest.fixture(scope="module")
def tenant(harness: Harness):
    """A fresh tenant shared by a module.

    Yields the tenant scope used as the compute namespace path parameter.
    """
    name = harness.new_tenant_name()
    harness.create_tenant(name)
    try:
        yield name
    finally:
        harness.delete_tenant(name)
