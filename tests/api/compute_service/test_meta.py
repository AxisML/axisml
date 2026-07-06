"""compute-service: deployment-form capability probe + liveness endpoint.

The /capabilities document is what the harness capability matrix
(``lib.harness.form_supports``) is supposed to mirror, yet the tests otherwise
consult the static matrix and never hit the real endpoint. This closes that gap.
"""

from __future__ import annotations

import pytest

from clients.computeservice.api.capabilities import get_capabilities
from clients.computeservice.api.health import healthz
from lib.harness import Capability


# Lite embeds all System modules in one axisml-core process and serves an
# aggregated {components: {...}} capabilities doc at the shared /capabilities
# path; the per-component flat client (runtime/quotaEnforcement) can't parse it.
# The flat per-service contract is a Standard-form concept.
@pytest.mark.standard_only
def test_capabilities_match_form(harness):
    resp = get_capabilities.sync_detailed(client=harness.compute_service)
    assert resp.status_code == 200, resp.content
    caps = resp.parsed
    assert caps.runtime in ("kubernetes", "standalone"), caps.runtime
    # The advertised quota enforcement must agree with the harness form matrix.
    assert caps.quota_enforcement == harness.supports(Capability.QUOTA_ENFORCEMENT)


def test_healthz(harness):
    resp = healthz.sync_detailed(client=harness.compute_service)
    assert resp.status_code == 200, resp.content
