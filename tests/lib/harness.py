"""The single environment seam between the black-box tests and a deployment form.

A System/CORE test drives only the typed clients exposed here plus ``tenant`` /
``supports`` — never a Kubernetes client — so the same test runs against both the
Standard form (real cluster reached over port-forwards) and the Lite form (one
axisml-core process). Mirrors the Go suite's ``Harness`` interface.
"""

from __future__ import annotations

import enum
from abc import ABC, abstractmethod

import httpx
import pytest

from clients.artifacthub import Client as ArtifactHubClient
from clients.clustermanager import Client as ClusterManagerClient
from clients.computeservice import Client as ComputeServiceClient
from clients.platform import AuthenticatedClient as PlatformClient
from lib import config
from lib.naming import unique_name
from lib.portforward import PortForward

USER_HEADER = "X-Axisml-User"


class Capability(str, enum.Enum):
    """A feature whose presence differs by deployment form."""

    MULTI_TENANT = "multiTenant"
    RESOURCE_POOL_WRITE = "resourcePoolsWritable"
    QUOTA_ENFORCEMENT = "quotaEnforcement"
    ARTIFACT_UPLOAD = "artifactUpload"


def form_supports(mode: str, cap: Capability) -> bool:
    """Per-form capability matrix — the single source of truth, usable without a
    harness instance (e.g. from a collection-time fixture that only knows --mode).

    Standard (full Kubernetes) backs every capability; Lite is fixed to a single
    static tenant with no ElasticQuota, so it only serves artifact upload.
    """
    if mode == "standard":
        return True
    return cap == Capability.ARTIFACT_UPLOAD  # lite


class Harness(ABC):
    def __init__(self, cfg: config.Config):
        self.cfg = cfg
        # Memoize session tokens per credential pair: /auth/login is rate-limited
        # per client IP (burst), and httptest/port-forward makes every request
        # share one IP, so re-authenticating on every call would exhaust the bucket
        # and 429. Reusing the token also mirrors a real client (log in once).
        self._tokens: dict[tuple[str, str], str] = {}

    @property
    @abstractmethod
    def cluster_manager(self) -> ClusterManagerClient: ...

    @property
    @abstractmethod
    def compute_service(self) -> ComputeServiceClient: ...

    @property
    @abstractmethod
    def artifact_hub(self) -> ArtifactHubClient: ...

    @abstractmethod
    def supports(self, cap: Capability) -> bool: ...

    def skip_unless(self, cap: Capability) -> None:
        if not self.supports(cap):
            pytest.skip(f"form does not support {cap.value}")

    # --- platform (both forms serve axisml-platform; only the base URL differs) ---
    @property
    @abstractmethod
    def platform_base_url(self) -> str:
        """Base URL of the Platform API + SPA (JWT-authenticated)."""

    def admin_token(self) -> str:
        return self.login(self.cfg.admin_username, self.cfg.admin_password)

    def login(self, username: str, password: str, *, cache: bool = True) -> str:
        """Log in and return a JWT. Memoized per (username, password) by default;
        pass cache=False for a throwaway session a test intends to revoke (logout /
        refresh) so it does not invalidate the shared cached token."""
        key = (username, password)
        if cache:
            cached = self._tokens.get(key)
            if cached is not None:
                return cached
        r = httpx.post(
            f"{self.platform_base_url}/api/v1/auth/login",
            json={"username": username, "password": password},
            timeout=15.0,
        )
        r.raise_for_status()
        token = r.json()["jwt"]
        if cache:
            self._tokens[key] = token
        return token

    def platform(self, token: str | None = None) -> PlatformClient:
        return PlatformClient(
            base_url=self.platform_base_url,
            token=token or self.admin_token(),
            raise_on_unexpected_status=False,
        )

    # --- artifact registry (zot) reachable from the test host ---
    @abstractmethod
    def oci_endpoint(self) -> str:
        """Base URL of the OCI registry (zot) the two-phase upload pushes to."""

    # --- tenant lifecycle (Standard provisions; Lite uses the static default) ---
    @abstractmethod
    def create_tenant(
        self, name: str, *, pool: str | None = None, quantity: int = 4
    ) -> None:
        """Provision a tenant with one quota for the selected resource pool."""

    @abstractmethod
    def delete_tenant(self, name: str) -> None: ...

    def new_tenant_name(self) -> str:
        return unique_name("e2e")

    def close(self) -> None:  # overridden by Standard to stop port-forwards
        pass


def _system_client(cls, base_url: str, user: str):
    return cls(
        base_url=base_url, headers={USER_HEADER: user}, raise_on_unexpected_status=False
    )


class StandardHarness(Harness):
    """Real Kubernetes cluster reached over per-service port-forwards."""

    def __init__(self, cfg: config.Config):
        super().__init__(cfg)
        self._forwards: list[PortForward] = []
        self._cm = self._build(cfg.cluster_manager, ClusterManagerClient)
        self._cs = self._build(cfg.compute_service, ComputeServiceClient)
        self._ah = self._build(cfg.artifact_hub, ArtifactHubClient)
        # Platform: JWT auth. Reuse the admin seeded by env-setup.
        self._platform_pf = PortForward(
            cfg.platform.name, cfg.platform.namespace, cfg.platform.port
        ).start()
        self._forwards.append(self._platform_pf)
        self._platform_base = self._platform_pf.local_url
        self._platform: PlatformClient | None = None
        self._zot_pf: PortForward | None = None

    def _build(self, svc: config.Service, cls):
        pf = PortForward(svc.name, svc.namespace, svc.port).start()
        self._forwards.append(pf)
        return _system_client(cls, pf.local_url, self.cfg.user)

    @property
    def cluster_manager(self) -> ClusterManagerClient:
        return self._cm

    @property
    def compute_service(self) -> ComputeServiceClient:
        return self._cs

    @property
    def artifact_hub(self) -> ArtifactHubClient:
        return self._ah

    # --- platform ---
    @property
    def platform_base_url(self) -> str:
        return self._platform_base

    def oci_endpoint(self) -> str:
        # zot is ClusterIP-only; forward lazily and reuse (closed with the rest).
        if self._zot_pf is None:
            self._zot_pf = PortForward("zot", self.cfg.infra_namespace, 5000).start()
            self._forwards.append(self._zot_pf)
        return self._zot_pf.local_url

    def supports(self, cap: Capability) -> bool:
        return form_supports("standard", cap)

    def create_tenant(
        self, name: str, *, pool: str | None = None, quantity: int = 4
    ) -> None:
        from clients.clustermanager.api.tenants import create_tenant
        from clients.clustermanager.models import (
            CreateTenantRequest,
            ServerQuota,
            ServerQuotaUnit,
            Tenantv1Alpha1NamespaceSpec,
        )

        pool = pool or self.cfg.default_pool
        body = CreateTenantRequest(
            name=name,
            namespace=Tenantv1Alpha1NamespaceSpec(name=name),
            quotas=[
                ServerQuota(
                    pool=pool,
                    units=[
                        ServerQuotaUnit(
                            unit_name=self.cfg.default_unit, quantity=quantity
                        )
                    ],
                )
            ],
        )
        resp = create_tenant.sync_detailed(client=self._cm, body=body)
        if resp.status_code not in (200, 201):
            raise AssertionError(
                f"create tenant {name}: {resp.status_code}: {resp.content!r}"
            )

    def delete_tenant(self, name: str) -> None:
        from clients.clustermanager.api.tenants import delete_tenant

        delete_tenant.sync_detailed(name, client=self._cm)

    def close(self) -> None:
        for pf in self._forwards:
            pf.stop()


class LiteHarness(Harness):
    """One axisml-core process serving all three System modules at a single URL,
    fronted by axisml-platform (API + SPA) as a separate container.

    Lite ships the Platform layer and the UI, so Platform-API and UI tests run
    here too. It is fixed to a single static tenant (``default``) and refuses
    tenant/pool writes (409): multi-tenant tests gate on ``MULTI_TENANT`` (or stay
    ``standard_only``) and are skipped under ``--mode lite``.
    """

    LITE_TENANT = "default"

    def __init__(self, cfg: config.Config):
        super().__init__(cfg)
        base = cfg.lite_base_url
        self._cm = _system_client(ClusterManagerClient, base, cfg.user)
        self._cs = _system_client(ComputeServiceClient, base, cfg.user)
        self._ah = _system_client(ArtifactHubClient, base, cfg.user)
        self._platform_base = cfg.lite_platform_url

    @property
    def cluster_manager(self) -> ClusterManagerClient:
        return self._cm

    @property
    def compute_service(self) -> ComputeServiceClient:
        return self._cs

    @property
    def artifact_hub(self) -> ArtifactHubClient:
        return self._ah

    @property
    def platform_base_url(self) -> str:
        return self._platform_base

    def oci_endpoint(self) -> str:
        return self.cfg.lite_oci_url

    def supports(self, cap: Capability) -> bool:
        return form_supports("lite", cap)

    def create_tenant(
        self, name: str, *, pool: str | None = None, quantity: int = 4
    ) -> None:
        # Lite serves a single static tenant; there is nothing to provision.
        return None

    def delete_tenant(self, name: str) -> None:
        pass

    def new_tenant_name(self) -> str:
        return self.LITE_TENANT
