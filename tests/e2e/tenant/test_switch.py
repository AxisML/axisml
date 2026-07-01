"""UI e2e: the tenant switcher selects an active scope (mandatory tenant scope).

Gated on MULTI_TENANT via the ``seeded_tenant`` fixture (Lite has one tenant)."""

from __future__ import annotations


def test_switch_active_tenant(logged_in_page, seeded_tenant):
    page = logged_in_page
    tenant = seeded_tenant

    # The switcher reads the tenant list lazily; reload so the seeded tenant shows.
    page.reload()
    page.locator("aside").wait_for(state="visible", timeout=15000)

    # Open the account menu (avatar = the only button carrying a title attribute).
    page.locator("button[title]").click()
    # Hover the tenant sub-menu trigger (the menu row that opens a submenu) and
    # pick the seeded tenant from its radio group.
    page.locator('[role="menuitem"][aria-haspopup="menu"]').hover()
    page.get_by_role("menuitemradio", name=tenant, exact=False).click()

    # The selection persisted the way the app scopes every request.
    assert page.evaluate("() => localStorage.getItem('axisml.tenant')") == tenant
