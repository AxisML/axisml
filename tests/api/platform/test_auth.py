"""platform: authentication (shared — both forms serve the Platform layer)."""

from __future__ import annotations

import httpx
import pytest

from clients.platform.api.auth import get_current_user, logout, refresh_token


def test_admin_me(harness):
    client = harness.platform(harness.admin_token())
    r = get_current_user.sync_detailed(client=client)
    assert r.status_code == 200, r.content
    assert r.parsed.is_system_admin is True


def test_refresh_token_issues_working_jwt(harness):
    # Refresh revokes the presenting session, so use a dedicated (uncached) token
    # rather than the shared admin session the rest of the suite reuses.
    token = harness.login(harness.cfg.admin_username, harness.cfg.admin_password, cache=False)
    r = refresh_token.sync_detailed(client=harness.platform(token))
    assert r.status_code == 200, r.content
    assert r.parsed.jwt, "refresh must return a new JWT"
    # The refreshed token authenticates.
    refreshed = harness.platform(r.parsed.jwt)
    assert get_current_user.sync_detailed(client=refreshed).status_code == 200


def test_logout(harness):
    # Logout revokes the session — use a dedicated one so the shared admin token
    # the rest of the suite reuses stays valid.
    token = harness.login(harness.cfg.admin_username, harness.cfg.admin_password, cache=False)
    r = logout.sync_detailed(client=harness.platform(token))
    assert r.status_code in (200, 204), r.content


def test_invalid_token_401(harness):
    client = harness.platform(token="not-a-valid-jwt")
    r = get_current_user.sync_detailed(client=client)
    assert r.status_code == 401, r.content


def test_bad_credentials_rejected(harness):
    with pytest.raises(httpx.HTTPStatusError):
        harness.login(harness.cfg.admin_username, "definitely-wrong")
