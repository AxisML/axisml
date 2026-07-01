"""UI e2e fixtures (pytest-playwright). Runs under both forms.

The Platform serves the SPA at / and the API at /api/v1. Under Standard a session
port-forward to the Platform service backs ``base_url``; under Lite it is the
axisml-platform container's published URL directly (no port-forward).
``logged_in_page`` signs in as the admin seeded by env-setup. We reuse
pytest-playwright's ``page`` fixture as-is.
"""

from __future__ import annotations

import json

import httpx
import pytest

from lib import config, seeding
from lib.harness import Capability, form_supports
from lib.naming import unique_name
from lib.portforward import PortForward


@pytest.fixture(scope="session")
def _platform_forward():
    c = config.load()
    pf = PortForward(c.platform.name, c.platform.namespace, c.platform.port).start()
    try:
        yield pf
    finally:
        pf.stop()


@pytest.fixture(scope="session")
def base_url(mode: str, cfg: config.Config, request: pytest.FixtureRequest) -> str:
    # Lite reaches the Platform container directly; Standard tunnels via port-forward.
    if mode == "lite":
        return cfg.lite_platform_url
    return request.getfixturevalue("_platform_forward").local_url


@pytest.fixture
def login_via_form():
    """Drive the real login form (used by the explicit login test)."""
    def _login(page, base_url: str, username: str, password: str) -> None:
        page.goto(f"{base_url}/login", wait_until="domcontentloaded")
        page.locator("#username").wait_for(state="visible", timeout=30000)
        page.fill("#username", username)
        page.fill("#password", password)
        page.locator("button[type=submit]").click()

    return _login


@pytest.fixture
def logged_in_page(page, base_url: str, admin_token: str):
    """An authenticated page via token injection (the SPA reads the JWT from
    localStorage). Avoids driving the login form per test — the backend throttles
    /auth/login per IP, so repeated UI logins across the suite would trip it. The
    real form is exercised once, in e2e/auth/test_login.py."""
    page.add_init_script(f"localStorage.setItem('axisml.token', {json.dumps(admin_token)})")
    page.goto(f"{base_url}/")
    page.locator("aside").wait_for(state="visible", timeout=30000)
    assert "/login" not in page.url
    return page


# --- API seeding (prerequisites for the tenant/jobs UI flows) ----------------- #
@pytest.fixture(scope="session")
def admin_token(base_url: str, cfg: config.Config) -> str:
    r = httpx.post(
        f"{base_url}/api/v1/auth/login",
        json={"username": cfg.admin_username, "password": cfg.admin_password},
        timeout=15.0,
    )
    r.raise_for_status()
    return r.json()["jwt"]


@pytest.fixture
def seeded_tenant(mode: str, base_url: str, admin_token: str, cfg: config.Config):
    """A tenant (admin is its initial member) + quota, removed after the test.

    Provisioning an extra tenant is a MULTI_TENANT operation, so tests that
    depend on this fixture skip on forms fixed to the static default (Lite)."""
    if not form_supports(mode, Capability.MULTI_TENANT):
        pytest.skip("form does not support multiTenant")
    client = seeding.admin_client(base_url, admin_token)
    identifier = unique_name("ui-t")
    seeding.create_tenant_with_quota(client, identifier, cfg.admin_username, cfg)
    try:
        yield identifier
    finally:
        seeding.delete_tenant_cascade(client, identifier)


@pytest.fixture
def select_tenant():
    """Persist the active tenant the way the app does (localStorage), then reload."""
    def _select(page, identifier: str) -> None:
        page.evaluate("(id) => localStorage.setItem('axisml.tenant', id)", identifier)
        page.reload()

    return _select
