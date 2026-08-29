"""platform: tenant member add + role update (requires MULTI_TENANT).

The existing quota test covers listing/removing members; this closes the write
paths — adding a member and promoting their role.
"""

from __future__ import annotations

from clients.platform.api.members import (
    add_tenant_member,
    list_tenant_members,
    update_tenant_member,
)
from clients.platform.api.tenants import create_tenant, delete_tenant
from clients.platform.api.users import create_user
from clients.platform.models import (
    MemberCreateRequest,
    MemberCreateRequestRoleName,
    MemberPatchRequest,
    MemberPatchRequestRoleName,
    TenantCreateRequest,
    UserCreateRequest,
)
from lib import platform_helpers
from lib.harness import Capability
from lib.naming import unique_name

PASSWORD = "password123"


def test_add_and_update_member(harness):
    harness.skip_unless(Capability.MULTI_TENANT)
    admin = harness.platform(harness.admin_token())
    owner = unique_name("mem-owner")
    extra = unique_name("mem-extra")
    tenant = unique_name("mem-t")

    for u in (owner, extra):
        cu = create_user.sync_detailed(client=admin, body=UserCreateRequest(username=u, password=PASSWORD, display_name=u))
        assert cu.status_code in (200, 201), cu.content
    try:
        ct = create_tenant.sync_detailed(
            client=admin,
            body=TenantCreateRequest(identifier=tenant, kubernetes_namespace=tenant, display_name="Member E2E", initial_admin=owner),
        )
        assert ct.status_code in (200, 201), ct.content
        try:
            # Add the second user as a plain member.
            am = add_tenant_member.sync_detailed(
                tenant, client=admin, body=MemberCreateRequest(account=extra, role_name=MemberCreateRequestRoleName.USER)
            )
            assert am.status_code in (200, 201), am.content

            ml = list_tenant_members.sync_detailed(tenant, client=admin)
            assert ml.status_code == 200, ml.content
            assert ml.parsed.count == 2, ml.parsed
            added = next(m for m in ml.parsed.items if m.username == extra)

            # Promote them to tenant-admin.
            um = update_tenant_member.sync_detailed(
                tenant, added.user_id, client=admin, body=MemberPatchRequest(role_name=MemberPatchRequestRoleName.TENANT_ADMIN)
            )
            assert um.status_code == 200, um.content
        finally:
            platform_helpers.remove_all_members(admin, tenant)
            delete_tenant.sync_detailed(tenant, client=admin)
    finally:
        for u in (owner, extra):
            platform_helpers.delete_user_by_name(admin, u)
