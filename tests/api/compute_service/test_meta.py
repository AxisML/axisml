"""compute-service capability and health endpoints."""

from __future__ import annotations

import pytest

from clients.computeservice.api.capabilities import get_capabilities
from clients.computeservice.api.health import healthz
from lib.harness import Capability


@pytest.mark.kubernetes_only
def test_capabilities(harness):
    resp = get_capabilities.sync_detailed(client=harness.compute_service)
    assert resp.status_code == 200, resp.content
    caps = resp.parsed
    assert caps.runtime == "kubernetes", caps.runtime
    assert caps.quota_enforcement == harness.supports(Capability.QUOTA_ENFORCEMENT)


def test_healthz(harness):
    resp = healthz.sync_detailed(client=harness.compute_service)
    assert resp.status_code == 200, resp.content
