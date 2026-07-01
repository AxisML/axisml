"""Best-effort teardown helpers for the platform tests (admin-scoped)."""

from __future__ import annotations

from clients.platform.api.members import list_tenant_members, remove_tenant_member
from clients.platform.api.users import delete_user, list_users


def delete_user_by_name(admin_client, username: str) -> None:
    r = list_users.sync_detailed(client=admin_client, q=username)
    if r.status_code != 200 or not r.parsed:
        return
    for u in r.parsed.items:
        if u.username == username:
            delete_user.sync_detailed(u.id, client=admin_client)


def remove_all_members(admin_client, tenant: str) -> None:
    r = list_tenant_members.sync_detailed(tenant, client=admin_client)
    if r.status_code != 200 or not r.parsed:
        return
    for m in r.parsed.items:
        remove_tenant_member.sync_detailed(tenant, m.user_id, client=admin_client)
