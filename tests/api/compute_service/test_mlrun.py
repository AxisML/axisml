"""compute-service: MLRun lifecycle over the HTTP contract (black-box).

Drives only the typed compute-service client and asserts over phase / pod
projection / logs, so the same test validates Standard (operators) and Lite
(in-process runtime). White-box CR/Job assertions live at the integration layer.
"""

from __future__ import annotations

from clients.computeservice.api.ml_runs import (
    cancel_ml_run,
    create_ml_run,
    delete_ml_run,
    get_ml_run,
    get_ml_run_pod_logs,
    list_ml_run_pods,
)
from lib import builders
from lib.naming import unique_name
from lib.polling import eventually


def test_mlrun_lifecycle(harness, cfg, tenant):
    ns, quota = tenant
    name = unique_name("e2e-run")

    r = create_ml_run.sync_detailed(ns, client=harness.compute_service, body=builders.busybox_mlrun(cfg, name, quota))
    assert r.status_code in (200, 201), r.content
    try:
        # Runs to Succeeded.
        def succeeded():
            g = get_ml_run.sync_detailed(ns, name, client=harness.compute_service)
            assert g.status_code == 200, g.content
            assert g.parsed.phase == "Succeeded", f"phase={g.parsed.phase!r}"

        eventually(succeeded, timeout=cfg.mlrun_complete_timeout, interval=cfg.poll_interval)

        # Pod projection is reachable and non-empty.
        pods = list_ml_run_pods.sync_detailed(ns, name, client=harness.compute_service)
        assert pods.status_code == 200, pods.content
        assert pods.parsed.items, "expected at least one pod"

        # Log streaming carries the container output.
        logs = get_ml_run_pod_logs.sync_detailed(ns, name, pods.parsed.items[0].name, client=harness.compute_service)
        assert logs.status_code == 200, logs.content
        assert "hello" in logs.content.decode(), logs.content
    finally:
        delete_ml_run.sync_detailed(ns, name, client=harness.compute_service)


def test_mlrun_cancel(harness, cfg, tenant):
    ns, quota = tenant
    name = unique_name("e2e-cancel")
    # A long-running command so there's a window to cancel.
    body = builders.busybox_mlrun(cfg, name, quota)
    body.roles[0].template.command = ["sh", "-c", "sleep 600"]

    r = create_ml_run.sync_detailed(ns, client=harness.compute_service, body=body)
    assert r.status_code in (200, 201), r.content
    try:
        # Cancel is rejected (412) while the job is still being created; wait until
        # it's actually Running before issuing the cancel.
        def running():
            g = get_ml_run.sync_detailed(ns, name, client=harness.compute_service)
            assert g.status_code == 200, g.content
            assert g.parsed.phase == "Running", f"phase={g.parsed.phase!r}"

        eventually(running, timeout=cfg.pod_ready_timeout, interval=cfg.poll_interval)

        c = cancel_ml_run.sync_detailed(ns, name, client=harness.compute_service)
        assert c.status_code in (200, 202), c.content

        def terminal():
            g = get_ml_run.sync_detailed(ns, name, client=harness.compute_service)
            assert g.status_code == 200, g.content
            assert g.parsed.phase in ("Cancelled", "Canceled", "Failed"), f"phase={g.parsed.phase!r}"

        eventually(terminal, timeout=cfg.mlrun_complete_timeout, interval=cfg.poll_interval)
    finally:
        delete_ml_run.sync_detailed(ns, name, client=harness.compute_service)
