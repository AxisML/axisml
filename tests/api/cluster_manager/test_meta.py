"""cluster-manager: the capability document must mirror the harness form matrix.

The harness gates writes off a static ``form_supports`` matrix; nothing otherwise
checks that the real ``/capabilities`` endpoint agrees with it. This closes that
drift risk directly.
"""

from __future__ import annotations

import pytest

from clients.clustermanager.api.capabilities import get_capabilities
from clients.clustermanager.api.health import healthz
from lib.harness import Capability


# Lite embeds all System modules in one axisml-core process and serves an
# aggregated {components: {...}} capabilities doc at the shared /capabilities
# path; the per-component flat client (multiTenant/resourcePoolsWritable) can't
# parse it. The flat per-service contract is a Standard-form concept.
@pytest.mark.standard_only
def test_capabilities_match_form(harness):
    resp = get_capabilities.sync_detailed(client=harness.cluster_manager)
    assert resp.status_code == 200, resp.content
    caps = resp.parsed
    assert caps.multi_tenant == harness.supports(Capability.MULTI_TENANT)
    assert caps.resource_pools_writable == harness.supports(Capability.RESOURCE_POOL_WRITE)


def test_healthz(harness):
    resp = healthz.sync_detailed(client=harness.cluster_manager)
    assert resp.status_code == 200, resp.content
