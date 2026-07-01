"""cluster-manager: the default ResourcePool both forms serve."""

from __future__ import annotations

from clients.clustermanager.api.resource_pools import get_resource_pool


def test_default_pool_readable(harness, cfg):
    resp = get_resource_pool.sync_detailed(cfg.default_pool, client=harness.cluster_manager)
    assert resp.status_code == 200, resp.content
    pool = resp.parsed
    assert pool.name == cfg.default_pool
    assert pool.units, "default pool should expose at least one unit"
