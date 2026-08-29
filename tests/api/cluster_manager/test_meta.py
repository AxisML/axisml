"""cluster-manager capability and health endpoints."""

from __future__ import annotations

import pytest

from clients.clustermanager.api.capabilities import get_capabilities
from clients.clustermanager.api.health import healthz
from lib.harness import Capability


@pytest.mark.kubernetes_only
def test_capabilities(harness):
    resp = get_capabilities.sync_detailed(client=harness.cluster_manager)
    assert resp.status_code == 200, resp.content
    caps = resp.parsed
    assert caps.multi_tenant == harness.supports(Capability.MULTI_TENANT)
    assert caps.resource_pools_writable == harness.supports(Capability.RESOURCE_POOL_WRITE)


def test_healthz(harness):
    resp = healthz.sync_detailed(client=harness.cluster_manager)
    assert resp.status_code == 200, resp.content
