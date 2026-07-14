"""Platform seeding helpers for the UI e2e tests.

The UI is the system under test, but its prerequisites (a tenant the admin can
scope into, a Job to view, a Run to display) are seeded through the Platform API
so the browser test asserts rendering/behaviour, not data entry. Admin-scoped.
"""

from __future__ import annotations

import httpx

from clients.platform import AuthenticatedClient
from clients.platform.api.jobs import create_job
from clients.platform.api.quotas import create_tenant_quota
from clients.platform.api.tenants import create_tenant, delete_tenant
from clients.platform.models import (
    Backend,
    BackendName,
    JobCreateRequest,
    JobSpec,
    MLRunRole,
    QuotaCreateRequest,
    QuotaUnit,
    RoleTemplate,
    TenantCreateRequest,
)
from lib import config, platform_helpers
from lib.polling import eventually


def admin_client(base_url: str, token: str) -> AuthenticatedClient:
    return AuthenticatedClient(
        base_url=base_url, token=token, raise_on_unexpected_status=False
    )


def create_tenant_with_quota(
    client: AuthenticatedClient,
    identifier: str,
    admin_username: str,
    cfg: config.Config,
) -> None:
    ct = create_tenant.sync_detailed(
        client=client,
        body=TenantCreateRequest(
            identifier=identifier,
            kubernetes_namespace=identifier,
            display_name=identifier,
            initial_admin=admin_username,
        ),
    )
    if ct.status_code not in (200, 201):
        raise AssertionError(
            f"seed tenant {identifier}: {ct.status_code}: {ct.content!r}"
        )

    # The tenant CR is still settling right after create; the quota write can hit
    # a transient OptimisticLockConflict (409). Retry until it lands.
    def put_quota():
        q = create_tenant_quota.sync_detailed(
            identifier,
            client=client,
            body=QuotaCreateRequest(
                pool=cfg.default_pool,
                units=[QuotaUnit(unit_name=cfg.default_unit, quantity=1)],
            ),
        )
        assert q.status_code in (200, 201), (
            f"seed quota for {identifier}: {q.status_code}: {q.content!r}"
        )

    eventually(put_quota, timeout=30.0, interval=2.0)


def delete_tenant_cascade(client: AuthenticatedClient, identifier: str) -> None:
    platform_helpers.remove_all_members(client, identifier)
    delete_tenant.sync_detailed(identifier, client=client)


def create_simple_job(
    client: AuthenticatedClient, tenant: str, name: str, cfg: config.Config
) -> None:
    spec = JobSpec(
        backend=Backend(name=BackendName.NATIVE, engine="job"),
        pool_name=cfg.default_pool,
        unit_name=cfg.default_unit,
        roles=[
            MLRunRole(
                name="worker",
                replicas=1,
                template=RoleTemplate(
                    image=cfg.mlrun_image, command=["sh", "-c", "echo hi"]
                ),
            )
        ],
    )
    r = create_job.sync_detailed(
        client=client,
        x_axisml_tenant=tenant,
        body=JobCreateRequest(name=name, display_name=name, spec=spec),
    )
    if r.status_code not in (200, 201):
        raise AssertionError(f"seed job {name}: {r.status_code}: {r.content!r}")


def trigger_job_run(
    base_url: str, token: str, tenant: str, job: str, cfg: config.Config
) -> None:
    # Raw POST: we only need the trigger to land; the typed Run response can carry
    # an empty enum field the generated client refuses to deserialize.
    r = httpx.post(
        f"{base_url}/api/v1/jobs/{job}/runs",
        headers={"Authorization": f"Bearer {token}", "X-Axisml-Tenant": tenant},
        json={"poolName": cfg.default_pool, "unitName": cfg.default_unit},
        timeout=15.0,
    )
    if r.status_code not in (200, 201, 202):
        raise AssertionError(f"trigger run for {job}: {r.status_code}: {r.text}")
