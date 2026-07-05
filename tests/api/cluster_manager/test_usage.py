"""cluster-manager: per-(tenant, pool) usage (N2) and metrics (N3).

Usage folds the tenant's quota ceiling against the reflected ElasticQuota Used
and needs no metrics backend, so it runs on both forms. Metrics query Prometheus
(Standard only).
"""

from __future__ import annotations

import pytest

from clients.clustermanager.api.resource_pools import get_resource_pool_metrics, get_resource_pool_usage


def test_pool_usage(harness, cfg, tenant):
    ns, _ = tenant
    u = get_resource_pool_usage.sync_detailed(cfg.default_pool, client=harness.cluster_manager, tenant=ns)
    assert u.status_code == 200, u.content
    assert u.parsed.pool == cfg.default_pool
    assert u.parsed.tenant == ns
    assert isinstance(u.parsed.meters, list)


def test_pool_usage_requires_tenant(harness, cfg):
    """tenant is a required query param; an empty value is a clean 400."""
    u = get_resource_pool_usage.sync_detailed(cfg.default_pool, client=harness.cluster_manager, tenant="")
    assert u.status_code == 400, u.content


@pytest.mark.standard_only
def test_pool_metrics(harness, cfg, tenant):
    ns, _ = tenant
    m = get_resource_pool_metrics.sync_detailed(
        cfg.default_pool, client=harness.cluster_manager, tenant=ns, metric="cpu_util", range_="15m"
    )
    assert m.status_code == 200, m.content
    assert m.parsed.metric == "cpu_util"
    assert isinstance(m.parsed.series, list)
