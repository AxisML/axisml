"""cluster-manager health endpoint."""

from __future__ import annotations

from clients.clustermanager.api.health import healthz


def test_healthz(harness):
    resp = healthz.sync_detailed(client=harness.cluster_manager)
    assert resp.status_code == 200, resp.content
