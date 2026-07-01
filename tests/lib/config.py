"""Environment-driven knobs shared by the suite and the env-setup scripts.

Ports the Go suite's ``config_test.go`` envConfig: every value has a default that
matches a stock ``test-setup`` (``cluster-up`` + ``helm-install``), so a developer
runs the suite with no extra env. Override only when an install differs.
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field


def _env(key: str, default: str) -> str:
    v = os.environ.get(key)
    return v if v else default


def _env_int(key: str, default: int) -> int:
    v = os.environ.get(key)
    return int(v) if v else default


def _env_float(key: str, default: float) -> float:
    v = os.environ.get(key)
    return float(v) if v else default


@dataclass(frozen=True)
class Service:
    """A ClusterIP HTTP service reachable via ``kubectl port-forward``."""

    name: str
    namespace: str
    port: int = 8080


@dataclass(frozen=True)
class Config:
    # Namespaces the Helm layers install into.
    infra_namespace: str = field(default_factory=lambda: _env("AXISML_INFRA_NS", "axisml-infra"))
    system_namespace: str = field(default_factory=lambda: _env("AXISML_SYSTEM_NS", "axisml-system"))
    platform_namespace: str = field(default_factory=lambda: _env("AXISML_PLATFORM_NS", "axisml-platform"))

    # The four HTTP-surface components (System trio + Platform). All on :8080.
    cluster_manager: Service = field(
        default_factory=lambda: Service(_env("AXISML_CLUSTER_MANAGER_SVC", "axisml-cluster-manager"), _env("AXISML_SYSTEM_NS", "axisml-system"))
    )
    compute_service: Service = field(
        default_factory=lambda: Service(_env("AXISML_COMPUTE_SERVICE_SVC", "axisml-compute-service"), _env("AXISML_SYSTEM_NS", "axisml-system"))
    )
    artifact_hub: Service = field(
        default_factory=lambda: Service(_env("AXISML_ARTIFACT_HUB_SVC", "axisml-artifact-hub"), _env("AXISML_SYSTEM_NS", "axisml-system"))
    )
    platform: Service = field(
        default_factory=lambda: Service(_env("AXISML_PLATFORM_SVC", "axisml-platform"), _env("AXISML_PLATFORM_NS", "axisml-platform"))
    )
    # Envoy gateway fronting the Platform SPA (UI e2e base URL is derived from it).
    gateway_name: str = field(default_factory=lambda: _env("AXISML_GATEWAY", "axisml-gateway"))

    # Lite serves all three System modules from one axisml-core process at this
    # base URL; axisml-platform (API + SPA) runs as a separate container.
    lite_base_url: str = field(default_factory=lambda: _env("AXISML_LITE_URL", "http://localhost:8090"))
    lite_platform_url: str = field(default_factory=lambda: _env("AXISML_LITE_PLATFORM_URL", "http://localhost:8080"))
    # zot registry published by the Lite compose stack (two-phase artifact upload).
    lite_oci_url: str = field(default_factory=lambda: _env("AXISML_LITE_OCI_URL", "http://localhost:5001"))

    # Identity stamped on every System call (X-Axisml-User). Platform uses JWT.
    user: str = field(default_factory=lambda: _env("AXISML_TEST_USER", "axisml-tester"))

    # Default ResourcePool + unit each provisioned tenant draws quota from.
    default_pool: str = field(default_factory=lambda: _env("AXISML_DEFAULT_POOL", "default"))
    default_unit: str = field(default_factory=lambda: _env("AXISML_DEFAULT_UNIT", "cpu-small"))

    # Workload images (preloaded into minikube; imagePullPolicy IfNotPresent).
    mlrun_image: str = field(default_factory=lambda: _env("AXISML_MLRUN_IMAGE", "busybox:latest"))
    service_image: str = field(default_factory=lambda: _env("AXISML_SERVICE_IMAGE", "nginx:1.27"))

    # --- Platform admin credentials (single source of truth) ---
    # The suite uses the default `admin`; env-setup resets its password in the DB
    # to admin_password (clearing any forced-change), so the suite logs in with a
    # deterministic credential regardless of the cluster's prior state.
    admin_username: str = field(default_factory=lambda: _env("AXISML_ADMIN_USER", "admin"))
    admin_password: str = field(default_factory=lambda: _env("AXISML_ADMIN_PASSWORD", "axisml-admin-test"))

    # --- Platform metadata DB (env-setup admin reset only; tests never touch it) ---
    db_pod: str = field(default_factory=lambda: _env("AXISML_DB_POD", "axisml-database-0"))
    db_name: str = field(default_factory=lambda: _env("AXISML_DB_NAME", "axisml"))
    db_user: str = field(default_factory=lambda: _env("AXISML_DB_USER", "axisml"))
    db_password: str = field(default_factory=lambda: _env("AXISML_DB_PASSWORD", "axisml"))

    # Timeout budgets (seconds) — much larger than integration: real scheduling,
    # image pulls and kubelet startup take real time.
    cr_provision_timeout: float = field(default_factory=lambda: _env_float("AXISML_CR_TIMEOUT", 120.0))
    http_ready_timeout: float = field(default_factory=lambda: _env_float("AXISML_HTTP_TIMEOUT", 90.0))
    pod_ready_timeout: float = field(default_factory=lambda: _env_float("AXISML_POD_TIMEOUT", 180.0))
    mlrun_complete_timeout: float = field(default_factory=lambda: _env_float("AXISML_MLRUN_TIMEOUT", 300.0))
    poll_interval: float = field(default_factory=lambda: _env_float("AXISML_POLL_INTERVAL", 2.0))


def load() -> Config:
    """Build a Config from the current environment."""
    return Config()
