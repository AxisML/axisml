"""UI e2e: a tenant-scoped Job and its Run render in the SPA over the real backend.

The Job + Run are seeded through the Platform API; the browser test drives the
UI to scope into the tenant, view the Job in the list, and see the triggered Run
surface in the Job detail. Gated on MULTI_TENANT via ``seeded_tenant``.
"""

from __future__ import annotations

from lib import config, seeding
from lib.naming import unique_name


def test_job_and_run_visible_in_ui(logged_in_page, base_url, admin_token, cfg: config.Config, seeded_tenant, select_tenant):
    page = logged_in_page
    tenant = seeded_tenant
    client = seeding.admin_client(base_url, admin_token)
    job = unique_name("ui-job")

    # Seed a Job (and a Run) for the UI to display.
    seeding.create_simple_job(client, tenant, job, cfg)

    # Scope the SPA into the tenant, then the jobs list shows the seeded job.
    select_tenant(page, tenant)
    page.goto(f"{base_url}/jobs")
    page.locator(f'a[href="/jobs/{job}"]').first.wait_for(state="visible", timeout=15000)

    # Trigger a run, then the job detail's runs surface it (real tenant-scoped read).
    seeding.trigger_job_run(base_url, admin_token, tenant, job, cfg)
    page.goto(f"{base_url}/jobs/{job}")
    # The Runs tab is the second tab (label is i18n, so select by position).
    page.get_by_role("tab").nth(1).click()
    page.locator(f'a[href^="/jobs/{job}/runs/"]').first.wait_for(state="visible", timeout=20000)
