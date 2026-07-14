"""compute-service: MLService lifecycle (create -> running -> scale -> delete)."""

from __future__ import annotations

from clients.computeservice.api.ml_services import (
    create_ml_service,
    delete_ml_service,
    get_ml_service,
    get_ml_service_pod_logs,
    list_ml_service_pods,
    list_ml_services,
    patch_ml_service,
    scale_ml_service,
)
from clients.computeservice.models import MLServicePatchRequest, MLServiceScaleRequest
from lib import builders
from lib.naming import unique_name
from lib.polling import eventually


def test_mlservice_lifecycle(harness, cfg, tenant):
    ns = tenant
    name = unique_name("e2e-svc")

    r = create_ml_service.sync_detailed(
        ns, client=harness.compute_service, body=builders.nginx_mlservice(cfg, name)
    )
    assert r.status_code in (200, 201), r.content
    try:
        # Becomes Running/Available.
        def ready():
            g = get_ml_service.sync_detailed(ns, name, client=harness.compute_service)
            assert g.status_code == 200, g.content
            assert g.parsed.phase in ("Running", "Available", "Ready"), (
                f"phase={g.parsed.phase!r}"
            )

        eventually(ready, timeout=cfg.pod_ready_timeout, interval=cfg.poll_interval)

        pods = list_ml_service_pods.sync_detailed(
            ns, name, client=harness.compute_service
        )
        assert pods.status_code == 200, pods.content
        assert pods.parsed.items, "expected at least one pod"

        # Scale to 2 replicas.
        s = scale_ml_service.sync_detailed(
            ns,
            name,
            client=harness.compute_service,
            body=MLServiceScaleRequest(replicas=2),
        )
        assert s.status_code in (200, 202), s.content
    finally:
        delete_ml_service.sync_detailed(ns, name, client=harness.compute_service)


def test_mlservice_list_patch_logs(harness, cfg, tenant):
    """List projection, display-field patch round-trip, and pod-log reachability."""
    ns = tenant
    name = unique_name("e2e-svc-lpl")

    r = create_ml_service.sync_detailed(
        ns, client=harness.compute_service, body=builders.nginx_mlservice(cfg, name)
    )
    assert r.status_code in (200, 201), r.content
    try:

        def ready():
            g = get_ml_service.sync_detailed(ns, name, client=harness.compute_service)
            assert g.status_code == 200, g.content
            assert g.parsed.phase in ("Running", "Available", "Ready"), (
                f"phase={g.parsed.phase!r}"
            )

        eventually(ready, timeout=cfg.pod_ready_timeout, interval=cfg.poll_interval)

        # List projects the freshly created service.
        lst = list_ml_services.sync_detailed(ns, client=harness.compute_service)
        assert lst.status_code == 200, lst.content
        assert any(it.name == name for it in lst.parsed.items), (
            "created service absent from list"
        )

        # Patch display fields, then read them back.
        p = patch_ml_service.sync_detailed(
            ns,
            name,
            client=harness.compute_service,
            body=MLServicePatchRequest(display_name="Renamed Svc"),
        )
        assert p.status_code == 200, p.content
        assert p.parsed.display_name == "Renamed Svc", p.content

        # A service pod's container log is reachable.
        pods = list_ml_service_pods.sync_detailed(
            ns, name, client=harness.compute_service
        )
        assert pods.status_code == 200 and pods.parsed.items, pods.content
        logs = get_ml_service_pod_logs.sync_detailed(
            ns, name, pods.parsed.items[0].name, client=harness.compute_service
        )
        assert logs.status_code == 200, logs.content
    finally:
        delete_ml_service.sync_detailed(ns, name, client=harness.compute_service)
