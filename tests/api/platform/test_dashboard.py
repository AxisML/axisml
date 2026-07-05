"""platform: dashboard aggregates (activity, cluster-usage, cluster-metrics) and
the workspace-image catalog.

Activity and the cluster folds are tenant-scoped, so they scaffold a tenant
(``MULTI_TENANT``). cluster-metrics additionally needs a metrics backend and is
``standard_only``. The catalog is unscoped.
"""

from __future__ import annotations

import pytest

from clients.platform.api.dashboard import get_cluster_metrics, get_cluster_usage, list_activity
from clients.platform.api.tenants import create_tenant, delete_tenant
from clients.platform.api.users import create_user
from clients.platform.api.workspaces import list_workspace_images
from clients.platform.models import GetClusterMetricsMetric, TenantCreateRequest, UserCreateRequest
from lib import platform_helpers
from lib.harness import Capability
from lib.naming import unique_name

OWNER_PASSWORD = "password123"


def test_workspace_images(harness):
    admin = harness.platform(harness.admin_token())
    resp = list_workspace_images.sync_detailed(client=admin)
    assert resp.status_code == 200, resp.content
    assert resp.parsed.items, "workspace-image catalog should not be empty"
    assert all(img.ref for img in resp.parsed.items), "each image needs a ref"


def _scaffold_tenant(harness):
    admin = harness.platform(harness.admin_token())
    owner = unique_name("dash-u")
    tenant = unique_name("dash-t")
    create_user.sync_detailed(client=admin, body=UserCreateRequest(username=owner, password=OWNER_PASSWORD, display_name=owner))
    ct = create_tenant.sync_detailed(
        client=admin,
        body=TenantCreateRequest(identifier=tenant, kubernetes_namespace=tenant, display_name="Dash E2E", initial_admin=owner),
    )
    assert ct.status_code in (200, 201), ct.content
    return admin, tenant, owner


def _teardown_tenant(admin, tenant, owner):
    platform_helpers.remove_all_members(admin, tenant)
    delete_tenant.sync_detailed(tenant, client=admin)
    platform_helpers.delete_user_by_name(admin, owner)


def test_dashboard_activity(harness):
    harness.skip_unless(Capability.MULTI_TENANT)
    admin, tenant, owner = _scaffold_tenant(harness)
    try:
        # The tenant-create just performed is itself an audited mutation.
        resp = list_activity.sync_detailed(client=admin, x_axisml_tenant=tenant, limit=20)
        assert resp.status_code == 200, resp.content
        assert isinstance(resp.parsed.items, list)
    finally:
        _teardown_tenant(admin, tenant, owner)


def test_dashboard_cluster_usage(harness):
    harness.skip_unless(Capability.MULTI_TENANT)
    admin, tenant, owner = _scaffold_tenant(harness)
    try:
        resp = get_cluster_usage.sync_detailed(client=admin, x_axisml_tenant=tenant)
        assert resp.status_code == 200, resp.content
        assert isinstance(resp.parsed.pools, list)
    finally:
        _teardown_tenant(admin, tenant, owner)


@pytest.mark.standard_only
def test_dashboard_cluster_metrics(harness, cfg):
    admin, tenant, owner = _scaffold_tenant(harness)
    try:
        m = get_cluster_metrics.sync_detailed(
            client=admin,
            x_axisml_tenant=tenant,
            pool=cfg.default_pool,
            metric=GetClusterMetricsMetric.CPU_UTIL,
            range_="15m",
        )
        assert m.status_code == 200, m.content
        assert m.parsed.metric == "cpu_util"
        assert isinstance(m.parsed.series, list)
    finally:
        _teardown_tenant(admin, tenant, owner)
