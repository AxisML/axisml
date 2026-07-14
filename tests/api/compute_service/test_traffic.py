"""compute-service: MLTrafficPolicy canary lifecycle (create -> split -> promote -> delete).

Black-box over the HTTP contract only. The white-box assertion that the gateway
HTTPRoute carries the weighted backendRefs lives at the compute-operator
integration layer, not here.
"""

from __future__ import annotations

import pytest

from clients.computeservice.api.ml_services import (
    create_ml_service,
    delete_ml_service,
    get_ml_service,
)
from clients.computeservice.api.traffic_policies import (
    create_traffic_policy,
    delete_traffic_policy,
    get_traffic_policy,
    list_traffic_policies,
    promote_traffic_policy,
    rollback_traffic_policy,
    split_traffic_policy,
)
from clients.computeservice.models import (
    TrafficPolicySplitRequest,
    TrafficPolicyWeightUpdate,
)
from lib import builders
from lib.naming import unique_name
from lib.polling import eventually


@pytest.fixture
def canary_pair(harness, cfg, tenant):
    """Two member MLServices (stable + canary) for a traffic policy to front."""
    ns = tenant
    stable = unique_name("e2e-stable")
    canary = unique_name("e2e-canary")
    for svc in (stable, canary):
        r = create_ml_service.sync_detailed(
            ns, client=harness.compute_service, body=builders.nginx_mlservice(cfg, svc)
        )
        assert r.status_code in (200, 201), r.content

    # A traffic policy refuses members that aren't Ready yet (412), so wait.
    def ready(svc: str):
        g = get_ml_service.sync_detailed(ns, svc, client=harness.compute_service)
        assert g.status_code == 200, g.content
        assert g.parsed.phase in ("Running", "Available", "Ready"), (
            f"{svc} phase={g.parsed.phase!r}"
        )

    for svc in (stable, canary):
        eventually(
            lambda s=svc: ready(s),
            timeout=cfg.pod_ready_timeout,
            interval=cfg.poll_interval,
        )
    try:
        yield ns, stable, canary
    finally:
        for svc in (stable, canary):
            delete_ml_service.sync_detailed(ns, svc, client=harness.compute_service)


def test_traffic_canary_lifecycle(harness, cfg, canary_pair):
    ns, stable, canary = canary_pair
    name = unique_name("e2e-tp")

    # create
    r = create_traffic_policy.sync_detailed(
        ns,
        client=harness.compute_service,
        body=builders.canary_traffic(name, stable, canary),
    )
    assert r.status_code in (200, 201), r.content
    try:

        def programmed():
            g = get_traffic_policy.sync_detailed(
                ns, name, client=harness.compute_service
            )
            assert g.status_code == 200, g.content
            assert g.parsed.phase in ("Programmed", "Ready", "Active"), (
                f"phase={g.parsed.phase!r}"
            )

        eventually(
            programmed, timeout=cfg.cr_provision_timeout, interval=cfg.poll_interval
        )

        # split: shift weights 50/50
        sp = split_traffic_policy.sync_detailed(
            ns,
            name,
            client=harness.compute_service,
            body=TrafficPolicySplitRequest(
                backends=[
                    TrafficPolicyWeightUpdate(service_name=stable, weight=50),
                    TrafficPolicyWeightUpdate(service_name=canary, weight=50),
                ]
            ),
        )
        assert sp.status_code in (200, 202), sp.content

        # promote: canary becomes stable (100%)
        pr = promote_traffic_policy.sync_detailed(
            ns, name, client=harness.compute_service
        )
        assert pr.status_code in (200, 202), pr.content
    finally:
        delete_traffic_policy.sync_detailed(ns, name, client=harness.compute_service)


def test_traffic_list_and_rollback(harness, cfg, canary_pair):
    """List projection + the rollback safety valve (canary shifted, then rolled back to 0)."""
    ns, stable, canary = canary_pair
    name = unique_name("e2e-tp-rb")

    r = create_traffic_policy.sync_detailed(
        ns,
        client=harness.compute_service,
        body=builders.canary_traffic(name, stable, canary),
    )
    assert r.status_code in (200, 201), r.content
    try:

        def programmed():
            g = get_traffic_policy.sync_detailed(
                ns, name, client=harness.compute_service
            )
            assert g.status_code == 200, g.content
            assert g.parsed.phase in ("Programmed", "Ready", "Active"), (
                f"phase={g.parsed.phase!r}"
            )

        eventually(
            programmed, timeout=cfg.cr_provision_timeout, interval=cfg.poll_interval
        )

        # List projects the freshly created policy.
        lst = list_traffic_policies.sync_detailed(ns, client=harness.compute_service)
        assert lst.status_code == 200, lst.content
        assert any(it.name == name for it in lst.parsed.items), (
            "created policy absent from list"
        )

        # Shift some traffic onto the canary, then roll it back to 0.
        sp = split_traffic_policy.sync_detailed(
            ns,
            name,
            client=harness.compute_service,
            body=TrafficPolicySplitRequest(
                backends=[
                    TrafficPolicyWeightUpdate(service_name=stable, weight=50),
                    TrafficPolicyWeightUpdate(service_name=canary, weight=50),
                ]
            ),
        )
        assert sp.status_code in (200, 202), sp.content

        rb = rollback_traffic_policy.sync_detailed(
            ns, name, client=harness.compute_service
        )
        assert rb.status_code in (200, 202), rb.content
    finally:
        delete_traffic_policy.sync_detailed(ns, name, client=harness.compute_service)
