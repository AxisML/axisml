"""cluster-manager capability and health endpoints."""

from __future__ import annotations

from clients.clustermanager.api.capabilities import get_capabilities
from clients.clustermanager.api.health import healthz


def test_capabilities(harness):
    resp = get_capabilities.sync_detailed(client=harness.cluster_manager)
    assert resp.status_code == 200, resp.content
    caps = resp.parsed
    assert caps.multi_tenant is True
    assert caps.resource_pools_writable is True


def test_healthz(harness):
    resp = healthz.sync_detailed(client=harness.cluster_manager)
    assert resp.status_code == 200, resp.content
