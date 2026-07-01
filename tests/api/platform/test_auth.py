"""platform: authentication (shared — both forms serve the Platform layer)."""

from __future__ import annotations

import httpx
import pytest

from clients.platform.api.auth import get_current_user


def test_admin_me(harness):
    client = harness.platform(harness.admin_token())
    r = get_current_user.sync_detailed(client=client)
    assert r.status_code == 200, r.content
    assert r.parsed.is_system_admin is True


def test_invalid_token_401(harness):
    client = harness.platform(token="not-a-valid-jwt")
    r = get_current_user.sync_detailed(client=client)
    assert r.status_code == 401, r.content


def test_bad_credentials_rejected(harness):
    with pytest.raises(httpx.HTTPStatusError):
        harness.login(harness.cfg.admin_username, "definitely-wrong")
