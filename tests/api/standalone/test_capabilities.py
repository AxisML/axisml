"""Standalone publishes one aggregate capability contract for all modules."""

from __future__ import annotations

import httpx
import pytest


@pytest.mark.standalone_only
def test_aggregate_capabilities(cfg):
    response = httpx.get(
        f"{cfg.standalone_system_url}/api/v1/capabilities", timeout=10.0
    )
    assert response.status_code == 200, response.content
    components = response.json()["components"]
    assert components["cluster-manager"]["multiTenant"] is False
    assert components["cluster-manager"]["resourcePoolsWritable"] is False
    assert components["compute-service"]["runtime"] == "standalone"
    assert components["compute-service"]["quotaEnforcement"] is False
    assert components["artifact-hub"]["upload"] is True
