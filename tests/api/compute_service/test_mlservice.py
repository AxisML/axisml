"""compute-service: MLService lifecycle (create -> running -> scale -> delete)."""

from __future__ import annotations

from clients.computeservice.api.ml_services import (
    create_ml_service,
    delete_ml_service,
    get_ml_service,
    list_ml_service_pods,
    scale_ml_service,
)
from clients.computeservice.models import MLServiceScaleRequest
from lib import builders
from lib.naming import unique_name
from lib.polling import eventually


def test_mlservice_lifecycle(harness, cfg, tenant):
    ns, quota = tenant
    name = unique_name("e2e-svc")

    r = create_ml_service.sync_detailed(ns, client=harness.compute_service, body=builders.nginx_mlservice(cfg, name, quota))
    assert r.status_code in (200, 201), r.content
    try:
        # Becomes Running/Available.
        def ready():
            g = get_ml_service.sync_detailed(ns, name, client=harness.compute_service)
            assert g.status_code == 200, g.content
            assert g.parsed.phase in ("Running", "Available", "Ready"), f"phase={g.parsed.phase!r}"

        eventually(ready, timeout=cfg.pod_ready_timeout, interval=cfg.poll_interval)

        pods = list_ml_service_pods.sync_detailed(ns, name, client=harness.compute_service)
        assert pods.status_code == 200, pods.content
        assert pods.parsed.items, "expected at least one pod"

        # Scale to 2 replicas.
        s = scale_ml_service.sync_detailed(ns, name, client=harness.compute_service, body=MLServiceScaleRequest(replicas=2))
        assert s.status_code in (200, 202), s.content
    finally:
        delete_ml_service.sync_detailed(ns, name, client=harness.compute_service)
