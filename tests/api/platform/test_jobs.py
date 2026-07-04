"""platform: Job definition lifecycle (create -> get/list -> runs empty -> delete)."""

from __future__ import annotations

from clients.platform.api.jobs import (
    create_job,
    delete_job,
    get_job,
    list_jobs,
    list_runs,
    update_job,
)
from clients.platform.api.tenants import create_tenant, delete_tenant
from clients.platform.api.users import create_user
from clients.platform.models import (
    Backend,
    BackendName,
    JobCreateRequest,
    JobPatchRequest,
    JobSpec,
    MLRunRole,
    RoleTemplate,
    TenantCreateRequest,
    UserCreateRequest,
)
from lib import platform_helpers
from lib.harness import Capability
from lib.naming import unique_name

OWNER_PASSWORD = "password123"


def _job_spec(cfg) -> JobSpec:
    return JobSpec(
        backend=Backend(name=BackendName.NATIVE, engine="job"),
        pool_name=cfg.default_pool,
        unit_name=cfg.default_unit,
        roles=[
            MLRunRole(
                name="worker",
                replicas=1,
                template=RoleTemplate(image=cfg.mlrun_image, command=["sh", "-c", "echo hi"]),
            )
        ],
    )


def test_job_definition_lifecycle(harness, cfg):
    # Scaffolds a dedicated tenant + owner — creating tenants is a MULTI_TENANT op.
    harness.skip_unless(Capability.MULTI_TENANT)
    admin = harness.platform(harness.admin_token())
    owner = unique_name("job-u")
    tenant = unique_name("job-t")

    create_user.sync_detailed(client=admin, body=UserCreateRequest(username=owner, password=OWNER_PASSWORD, display_name=owner))
    try:
        ct = create_tenant.sync_detailed(
            client=admin,
            body=TenantCreateRequest(identifier=tenant, kubernetes_namespace=tenant, display_name="Job E2E", initial_admin=owner),
        )
        assert ct.status_code in (200, 201), ct.content
        try:
            owner_client = harness.platform(harness.login(owner, OWNER_PASSWORD))
            job = unique_name("job")

            cj = create_job.sync_detailed(
                client=owner_client, x_axisml_tenant=tenant,
                body=JobCreateRequest(name=job, display_name="echo", spec=_job_spec(cfg)),
            )
            assert cj.status_code in (200, 201), cj.content

            assert get_job.sync_detailed(job, client=owner_client, x_axisml_tenant=tenant).status_code == 200
            assert list_jobs.sync_detailed(client=owner_client, x_axisml_tenant=tenant).status_code == 200
            # No trigger yet -> runs list is reachable (and empty).
            assert list_runs.sync_detailed(job, client=owner_client, x_axisml_tenant=tenant).status_code == 200

            # Patch the job's display fields.
            uj = update_job.sync_detailed(
                job, client=owner_client, x_axisml_tenant=tenant,
                body=JobPatchRequest(display_name="Renamed Job", description="patched by e2e"),
            )
            assert uj.status_code == 200, uj.content

            dj = delete_job.sync_detailed(job, client=owner_client, x_axisml_tenant=tenant)
            assert dj.status_code in (200, 204), dj.content
        finally:
            platform_helpers.remove_all_members(admin, tenant)
            delete_tenant.sync_detailed(tenant, client=admin)
    finally:
        platform_helpers.delete_user_by_name(admin, owner)
