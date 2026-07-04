"""cluster-manager: the ResourcePool + ResourceUnit REST shell over the CRD.

The default pool is read-only on both forms; pool/unit writes only exist on the
Standard form (``RESOURCE_POOL_WRITE``) and are skipped under ``--mode lite``.
"""

from __future__ import annotations

from clients.clustermanager.api.resource_pools import (
    create_resource_pool,
    delete_resource_pool,
    get_resource_pool,
    list_resource_pools,
    update_resource_pool,
)
from clients.clustermanager.api.resource_units import (
    create_resource_unit,
    delete_resource_unit,
    get_resource_unit,
    list_resource_units,
)
from clients.clustermanager.models import (
    CreateResourcePoolRequest,
    CreateResourceUnitRequest,
    CreateResourceUnitRequestLimits,
    CreateResourceUnitRequestRequests,
    PatchResourcePoolRequest,
    ServerCreateResourceUnitRequest,
    ServerCreateResourceUnitRequestLimits,
    ServerCreateResourceUnitRequestRequests,
)
from lib.harness import Capability
from lib.naming import unique_name


def test_default_pool_readable(harness, cfg):
    resp = get_resource_pool.sync_detailed(cfg.default_pool, client=harness.cluster_manager)
    assert resp.status_code == 200, resp.content
    pool = resp.parsed
    assert pool.name == cfg.default_pool
    assert pool.units, "default pool should expose at least one unit"


def test_get_unknown_pool_returns_404(harness):
    """The negative contract: an absent pool is a clean 404, not a 5xx."""
    resp = get_resource_pool.sync_detailed(unique_name("e2e-nopool"), client=harness.cluster_manager)
    assert resp.status_code == 404, resp.content


def test_resource_pool_and_unit_crud(harness):
    harness.skip_unless(Capability.RESOURCE_POOL_WRITE)
    pool = unique_name("e2e-pool")

    r = create_resource_pool.sync_detailed(
        client=harness.cluster_manager,
        body=CreateResourcePoolRequest(
            name=pool,
            description="e2e pool",
            units=[
                ServerCreateResourceUnitRequest(
                    name="small",
                    limits=ServerCreateResourceUnitRequestLimits.from_dict({"cpu": "2", "memory": "4Gi"}),
                    requests=ServerCreateResourceUnitRequestRequests.from_dict({"cpu": "2", "memory": "4Gi"}),
                )
            ],
        ),
    )
    assert r.status_code in (200, 201), r.content
    try:
        # get + list project the new pool with its inline unit.
        g = get_resource_pool.sync_detailed(pool, client=harness.cluster_manager)
        assert g.status_code == 200, g.content
        assert g.parsed.name == pool
        assert any(u.name == "small" for u in g.parsed.units), "inline unit absent from pool"

        lst = list_resource_pools.sync_detailed(client=harness.cluster_manager)
        assert lst.status_code == 200, lst.content
        assert any(p.name == pool for p in lst.parsed.items), "created pool absent from list"

        # patch the pool description.
        up = update_resource_pool.sync_detailed(
            pool, client=harness.cluster_manager, body=PatchResourcePoolRequest(description="updated")
        )
        assert up.status_code == 200, up.content

        # add a second unit, read it back, list, then remove it.
        cu = create_resource_unit.sync_detailed(
            pool,
            client=harness.cluster_manager,
            body=CreateResourceUnitRequest(
                name="large",
                limits=CreateResourceUnitRequestLimits.from_dict({"cpu": "8", "memory": "16Gi"}),
                requests=CreateResourceUnitRequestRequests.from_dict({"cpu": "8", "memory": "16Gi"}),
            ),
        )
        assert cu.status_code in (200, 201), cu.content

        gu = get_resource_unit.sync_detailed(pool, "large", client=harness.cluster_manager)
        assert gu.status_code == 200, gu.content
        assert gu.parsed.name == "large"

        ul = list_resource_units.sync_detailed(pool, client=harness.cluster_manager)
        assert ul.status_code == 200, ul.content
        assert {u.name for u in ul.parsed.items} >= {"small", "large"}

        du = delete_resource_unit.sync_detailed(pool, "large", client=harness.cluster_manager)
        assert du.status_code in (200, 202, 204), du.content
    finally:
        delete_resource_pool.sync_detailed(pool, client=harness.cluster_manager)
