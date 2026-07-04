"""platform: liveness probe (both forms serve the Platform layer)."""

from __future__ import annotations

from clients.platform.api.health import get_healthz


def test_healthz(harness):
    resp = get_healthz.sync_detailed(client=harness.platform(harness.admin_token()))
    assert resp.status_code == 200, resp.content
