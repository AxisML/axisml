"""Form-neutral request builders for the compute black-box tests.

Mirror the Go suite's ``core_actions_test.go``: minimal native/job MLRun,
native/deployment MLService, and a canary traffic policy — using only the
generated client models and literal role/engine strings.
"""

from __future__ import annotations

from clients.computeservice.models import (
    MLRunBackendSpec,
    MLRunCreateRequest,
    MLRunPodTemplateSubset,
    MLRunRoleSpec,
    MLServiceBackend,
    MLServiceCreateRequest,
    MLServicePodPort,
    MLServicePodTemplate,
    MLServiceRoleSpec,
    MLServiceRoute,
    MLTrafficPolicyBackendMember,
    MLTrafficPolicyEndpoint,
    TrafficPolicyCreateRequest,
)
from lib import config


def busybox_mlrun(cfg: config.Config, name: str) -> MLRunCreateRequest:
    """A native/job MLRun that runs to completion fast, echoing a log marker."""
    return MLRunCreateRequest(
        name=name,
        pool_name=cfg.default_pool,
        unit_name=cfg.default_unit,
        backend=MLRunBackendSpec(name="native", engine="job"),
        roles=[
            MLRunRoleSpec(
                name="worker",
                replicas=1,
                restart_policy="Never",
                template=MLRunPodTemplateSubset(
                    image=cfg.mlrun_image,
                    image_pull_policy="IfNotPresent",
                    command=["sh", "-c", "echo hello"],
                ),
            )
        ],
    )


def nginx_mlservice(
    cfg: config.Config, name: str, *, route: MLServiceRoute | None = None
) -> MLServiceCreateRequest:
    """A native/deployment MLService serving nginx on :80."""
    kwargs = dict(
        name=name,
        kind="service",
        pool_name=cfg.default_pool,
        unit_name=cfg.default_unit,
        backend=MLServiceBackend(name="native", engine="deployment"),
        roles=[
            MLServiceRoleSpec(
                name="predictor",
                replicas=1,
                template=MLServicePodTemplate(
                    image=cfg.service_image,
                    image_pull_policy="IfNotPresent",
                    ports=[
                        MLServicePodPort(name="http", container_port=80, protocol="TCP")
                    ],
                ),
            )
        ],
    )
    if route is not None:
        kwargs["route"] = route
    return MLServiceCreateRequest(**kwargs)


def canary_traffic(
    name: str, stable_svc: str, canary_svc: str
) -> TrafficPolicyCreateRequest:
    """A canary policy fronting two member services (stable @90 / canary @10)."""
    return TrafficPolicyCreateRequest(
        name=name,
        mode="canary",
        endpoint=MLTrafficPolicyEndpoint(hostname=f"{name}.e2e.local"),
        backends=[
            MLTrafficPolicyBackendMember(
                service_name=stable_svc, role="stable", weight=90
            ),
            MLTrafficPolicyBackendMember(
                service_name=canary_svc, role="canary", weight=10
            ),
        ],
    )
