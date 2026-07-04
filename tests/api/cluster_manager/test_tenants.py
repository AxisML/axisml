"""cluster-manager: tenant provisioning + per-pool quota folding (multi-tenant only).

Standard backs tenant writes with the Tenant CR + tenant-operator; Lite serves a
single static tenant and refuses writes — so these are skipped under --mode lite.
"""

from __future__ import annotations

from clients.clustermanager.api.tenants import (
    delete_tenant_quota,
    get_tenant,
    list_tenant_quotas,
    list_tenants,
    set_tenant_quota,
    update_tenant,
    update_tenant_quota,
)
from clients.clustermanager.models import (
    PatchQuotaRequest,
    PatchTenantRequest,
    PatchTenantRequestLabels,
    ServerQuotaUnit,
    SetQuotaRequest,
)
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


def test_tenant_list_and_update(harness):
    harness.skip_unless(Capability.MULTI_TENANT)
    name = unique_name("e2e-tenant-lu")
    harness.create_tenant(name)
    try:
        lst = list_tenants.sync_detailed(client=harness.cluster_manager)
        assert lst.status_code == 200, lst.content
        assert any(t.name == name for t in lst.parsed.items), "created tenant absent from list"

        up = update_tenant.sync_detailed(
            name,
            client=harness.cluster_manager,
            body=PatchTenantRequest(labels=PatchTenantRequestLabels.from_dict({"team": "e2e"})),
        )
        assert up.status_code == 200, up.content
    finally:
        harness.delete_tenant(name)


def test_tenant_quota_crud(harness, cfg):
    harness.skip_unless(Capability.MULTI_TENANT)
    name = unique_name("e2e-tenant-q")
    harness.create_tenant(name)
    pool = cfg.default_pool
    try:
        # set (create/replace) a per-pool quota.
        s = set_tenant_quota.sync_detailed(
            name,
            client=harness.cluster_manager,
            body=SetQuotaRequest(pool=pool, units=[ServerQuotaUnit(unit_name=cfg.default_unit, quantity=2)]),
        )
        assert s.status_code in (200, 201), s.content

        # list quotas projects it.
        ql = list_tenant_quotas.sync_detailed(name, client=harness.cluster_manager)
        assert ql.status_code == 200, ql.content
        assert any(q.pool == pool for q in ql.parsed.items), "set quota absent from list"

        # patch the granted quantity.
        up = update_tenant_quota.sync_detailed(
            name,
            pool,
            client=harness.cluster_manager,
            body=PatchQuotaRequest(units=[ServerQuotaUnit(unit_name=cfg.default_unit, quantity=3)]),
        )
        assert up.status_code == 200, up.content

        # delete the quota.
        d = delete_tenant_quota.sync_detailed(name, pool, client=harness.cluster_manager)
        assert d.status_code in (200, 202, 204), d.content
    finally:
        harness.delete_tenant(name)
