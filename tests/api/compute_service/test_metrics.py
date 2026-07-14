"""compute-service: Prometheus-backed workload metrics (N1).

The metrics endpoints require a metrics backend (Prometheus), which only the
Standard form provides, so these are marked ``standard_only``. The sampled
series may legitimately be empty (a short-lived run, or no scraped data yet), so
the tests assert the contract shape rather than concrete values.
"""

from __future__ import annotations

import pytest

from clients.computeservice.api.ml_runs import (
    create_ml_run,
    delete_ml_run,
    get_ml_run_metrics,
)
from clients.computeservice.api.ml_services import (
    create_ml_service,
    delete_ml_service,
    get_ml_service_metrics,
)
from lib import builders
from lib.naming import unique_name


@pytest.mark.standard_only
def test_mlrun_metrics(harness, cfg, tenant):
    ns = tenant
    name = unique_name("e2e-run-metrics")

    r = create_ml_run.sync_detailed(
        ns, client=harness.compute_service, body=builders.busybox_mlrun(cfg, name)
    )
    assert r.status_code in (200, 201), r.content
    try:
        m = get_ml_run_metrics.sync_detailed(
            ns, name, client=harness.compute_service, metric="cpu_util", range_="15m"
        )
        assert m.status_code == 200, m.content
        assert m.parsed.metric == "cpu_util"
        assert isinstance(m.parsed.series, list)

        # Serving metrics have no scraped source → rejected as a client error.
        bad = get_ml_run_metrics.sync_detailed(
            ns,
            name,
            client=harness.compute_service,
            metric="request_rate",
            range_="15m",
        )
        assert bad.status_code in (400, 422), bad.content
    finally:
        delete_ml_run.sync_detailed(ns, name, client=harness.compute_service)


@pytest.mark.standard_only
def test_mlservice_metrics(harness, cfg, tenant):
    ns = tenant
    name = unique_name("e2e-svc-metrics")

    r = create_ml_service.sync_detailed(
        ns, client=harness.compute_service, body=builders.nginx_mlservice(cfg, name)
    )
    assert r.status_code in (200, 201), r.content
    try:
        m = get_ml_service_metrics.sync_detailed(
            ns, name, client=harness.compute_service, metric="mem_util", range_="15m"
        )
        assert m.status_code == 200, m.content
        assert m.parsed.metric == "mem_util"
        assert isinstance(m.parsed.series, list)
    finally:
        delete_ml_service.sync_detailed(ns, name, client=harness.compute_service)
