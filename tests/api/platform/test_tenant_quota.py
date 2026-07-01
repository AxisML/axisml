"""platform: tenant + quota + member lifecycle (requires MULTI_TENANT)."""

from __future__ import annotations

from clients.platform.api.members import list_tenant_members, remove_tenant_member
from clients.platform.api.quotas import create_tenant_quota
from clients.platform.api.tenants import (
    create_tenant,
    delete_tenant,
    get_tenant,
    resume_tenant,
    suspend_tenant,
)
from clients.platform.api.users import create_user
from clients.platform.models import (
    QuotaCreateRequest,
    QuotaUnit,
    TenantCreateRequest,
    UserCreateRequest,
)
from lib import platform_helpers
from lib.harness import Capability
from lib.naming import unique_name

MEMBER_PASSWORD = "password123"


def test_tenant_quota_member_lifecycle(harness, cfg):
    harness.skip_unless(Capability.MULTI_TENANT)
    admin = harness.platform(harness.admin_token())
    member = unique_name("plat-u")
    tenant = unique_name("plat-t")

    cu = create_user.sync_detailed(client=admin, body=UserCreateRequest(username=member, password=MEMBER_PASSWORD, display_name=member))
    assert cu.status_code in (200, 201), cu.content
    try:
        ct = create_tenant.sync_detailed(
            client=admin,
            body=TenantCreateRequest(identifier=tenant, kubernetes_namespace=tenant, display_name="Platform E2E", initial_admin=member),
        )
        assert ct.status_code in (200, 201), ct.content
        try:
            assert get_tenant.sync_detailed(tenant, client=admin).status_code == 200

            # Quota: assign the default pool / cpu-small unit.
            q = create_tenant_quota.sync_detailed(
                tenant, client=admin,
                body=QuotaCreateRequest(pool=cfg.default_pool, units=[QuotaUnit(unit_name=cfg.default_unit, quantity=1)]),
            )
            assert q.status_code in (200, 201), q.content

            # The initial admin is the sole member; removing the last admin is blocked (409).
            member_client = harness.platform(harness.login(member, MEMBER_PASSWORD))
            ml = list_tenant_members.sync_detailed(tenant, client=member_client)
            assert ml.status_code == 200, ml.content
            assert ml.parsed.count == 1, ml.parsed
            rm = remove_tenant_member.sync_detailed(tenant, ml.parsed.items[0].user_id, client=admin)
            assert rm.status_code == 409, rm.content

            # Suspend / resume.
            assert suspend_tenant.sync_detailed(tenant, client=admin).status_code == 200
            assert resume_tenant.sync_detailed(tenant, client=admin).status_code == 200
        finally:
            platform_helpers.remove_all_members(admin, tenant)
            delete_tenant.sync_detailed(tenant, client=admin)
    finally:
        platform_helpers.delete_user_by_name(admin, member)
