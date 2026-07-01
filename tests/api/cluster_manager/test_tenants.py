"""cluster-manager: tenant provisioning (gated on multi-tenant support).

Standard backs tenant writes with the Tenant CR + tenant-operator; Lite serves a
single static tenant and refuses writes — so these are skipped under --mode lite.
"""

from __future__ import annotations

from clients.clustermanager.api.tenants import get_tenant
from lib.harness import Capability
from lib.naming import unique_name


def test_tenant_create_and_read(harness):
    harness.skip_unless(Capability.MULTI_TENANT)
    name = unique_name("e2e-apitenant")
    harness.create_tenant(name)
    try:
        resp = get_tenant.sync_detailed(name, client=harness.cluster_manager)
        assert resp.status_code == 200, resp.content
        assert resp.parsed.name == name
    finally:
        harness.delete_tenant(name)
