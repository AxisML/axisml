"""UI e2e: login against the real Platform backend (drives the real form)."""

from __future__ import annotations

from lib import config


def test_valid_login_reaches_app(page, base_url, cfg: config.Config, login_via_form):
    login_via_form(page, base_url, cfg.admin_username, cfg.admin_password)
    # Post-login redirect is client-side; wait for the authenticated shell.
    page.locator("aside").wait_for(state="visible", timeout=30000)
    assert "/login" not in page.url


def test_invalid_login_shows_error(page, base_url, login_via_form):
    login_via_form(page, base_url, "admin", "definitely-wrong")
    # Stays on /login and surfaces an error (does not silently proceed).
    page.wait_for_selector('[role="alert"]', timeout=10000)
    assert "/login" in page.url
