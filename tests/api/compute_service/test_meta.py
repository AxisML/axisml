"""compute-service health endpoint."""

from __future__ import annotations

from clients.computeservice.api.health import healthz


def test_healthz(harness):
    resp = healthz.sync_detailed(client=harness.compute_service)
    assert resp.status_code == 200, resp.content
