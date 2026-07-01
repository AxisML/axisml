"""UI e2e: the main routes render the app shell over the real backend.

Admin-scoped routes (no active tenant required) so the assertion is robust: each
route keeps the sidebar shell and lands on the right URL — proving auth + routing
+ backend wiring, without depending on seeded tenant data.
"""

from __future__ import annotations

import pytest

ROUTES = ["/", "/tenants", "/resource-pools"]


@pytest.mark.parametrize("route", ROUTES)
def test_route_renders_shell(logged_in_page, base_url, route):
    page = logged_in_page
    page.goto(f"{base_url}{route}")
    page.wait_for_url(lambda u: route.rstrip("/") in u or route == "/", timeout=10000)
    # The app shell stays mounted (no crash / error boundary swallowing the page).
    page.locator("aside").wait_for(state="visible", timeout=10000)
    assert "/login" not in page.url
