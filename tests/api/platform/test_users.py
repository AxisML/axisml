"""platform: user account CRUD + admin password reset (system-admin scoped)."""

from __future__ import annotations

from clients.platform.api.users import (
    create_user,
    get_user,
    list_users,
    set_user_password,
    update_user,
)
from clients.platform.models import SetPasswordRequest, UserCreateRequest, UserPatchRequest
from lib import platform_helpers
from lib.naming import unique_name

INITIAL_PASSWORD = "password123"
NEW_PASSWORD = "password456"


def test_user_crud(harness):
    admin = harness.platform(harness.admin_token())
    username = unique_name("plat-user")

    cu = create_user.sync_detailed(
        client=admin, body=UserCreateRequest(username=username, password=INITIAL_PASSWORD, display_name=username)
    )
    assert cu.status_code in (200, 201), cu.content
    try:
        # Resolve the id via search, then read it back by id.
        found = list_users.sync_detailed(client=admin, q=username)
        assert found.status_code == 200, found.content
        user = next(u for u in found.parsed.items if u.username == username)

        g = get_user.sync_detailed(user.id, client=admin)
        assert g.status_code == 200, g.content
        assert g.parsed.username == username

        # Patch the display name and read it back.
        up = update_user.sync_detailed(user.id, client=admin, body=UserPatchRequest(display_name="Renamed User"))
        assert up.status_code == 200, up.content
        assert up.parsed.display_name == "Renamed User"

        # Admin resets the account password (black-box: the endpoint accepts it).
        sp = set_user_password.sync_detailed(user.id, client=admin, body=SetPasswordRequest(new_password=NEW_PASSWORD))
        assert sp.status_code in (200, 204), sp.content
    finally:
        platform_helpers.delete_user_by_name(admin, username)
