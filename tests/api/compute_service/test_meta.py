"""compute-service capability and health endpoints."""

from __future__ import annotations

from clients.computeservice.api.capabilities import get_capabilities
from clients.computeservice.api.health import healthz


def test_capabilities(harness):
    resp = get_capabilities.sync_detailed(client=harness.compute_service)
    assert resp.status_code == 200, resp.content
    caps = resp.parsed
    assert caps.runtime == "kubernetes", caps.runtime
    assert caps.quota_enforcement is True


def test_healthz(harness):
    resp = healthz.sync_detailed(client=harness.compute_service)
    assert resp.status_code == 200, resp.content
