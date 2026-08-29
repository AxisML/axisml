"""Cross-service golden path: one train-and-serve journey through the System layer.

A single self-contained test (no inter-test ordering): a tenant created through
cluster-manager materializes a namespace + quota that compute-service schedules
into; a model uploaded to artifact-hub resolves back to the pushed digest; a job
runs to completion and a service comes up — all asserted over the HTTP contract.
"""

from __future__ import annotations

from clients.artifacthub.api.artifacts import (
    complete_artifact,
    get_artifact,
    initiate_artifact,
    resolve_artifact,
)
from clients.artifacthub.models import (
    ArtifactCompleteRequest,
    ArtifactInitiateRequest,
    ArtifactInitiateRequestSpec,
)
from clients.computeservice.api.ml_runs import create_ml_run, delete_ml_run, get_ml_run
from clients.computeservice.api.ml_services import (
    create_ml_service,
    delete_ml_service,
    get_ml_service,
)
from lib import builders, oci
from lib.harness import Capability
from lib.naming import unique_name
from lib.polling import eventually


def test_train_and_serve_journey(harness, cfg):
    harness.skip_unless(Capability.MULTI_TENANT)
    tenant = unique_name("e2e-golden")
    harness.create_tenant(tenant)
    model = unique_name("golden-model")
    version = "1.0.0"
    job = unique_name("golden-job")
    svc = unique_name("golden-svc")

    try:
        # --- model: artifact-hub two-phase upload resolves to the pushed digest.
        spec = ArtifactInitiateRequestSpec.from_dict(
            {"framework": "onnx", "format": "onnx"}
        )
        init = initiate_artifact.sync_detailed(
            tenant,
            model,
            client=harness.artifact_hub,
            body=ArtifactInitiateRequest(kind="model", version=version, spec=spec),
        )
        assert init.status_code in (200, 201), init.content
        upload = init.parsed.upload
        oc = oci.OciClient(
            harness.oci_endpoint(),
            oci.OciCreds(
                username=upload.credentials.username,
                password=upload.credentials.password,
            ),
        )
        try:
            repo, ref = oci.parse_repo_ref(upload.uri)
            digest = oc.push_config_only_manifest(repo, ref)
        finally:
            oc.close()
        assert complete_artifact.sync_detailed(
            tenant,
            model,
            version,
            client=harness.artifact_hub,
            body=ArtifactCompleteRequest(digest=digest),
        ).status_code in (200, 201, 202)

        def model_ready():
            g = get_artifact.sync_detailed(
                tenant, model, version, client=harness.artifact_hub
            )
            assert g.status_code == 200 and g.parsed.status.lower() == "ready", (
                g.content
            )

        eventually(
            model_ready, timeout=cfg.cr_provision_timeout, interval=cfg.poll_interval
        )
        r = resolve_artifact.sync_detailed(
            tenant, model, version, client=harness.artifact_hub
        )
        assert r.status_code == 200 and r.parsed.digest == digest, r.content

        # --- train: a job runs to completion in the tenant's quota.
        cr = create_ml_run.sync_detailed(
            tenant,
            client=harness.compute_service,
            body=builders.busybox_mlrun(cfg, job),
        )
        assert cr.status_code in (200, 201), cr.content

        def job_succeeded():
            g = get_ml_run.sync_detailed(tenant, job, client=harness.compute_service)
            assert g.status_code == 200 and g.parsed.phase == "Succeeded", (
                f"phase={g.parsed.phase!r}"
            )

        eventually(
            job_succeeded,
            timeout=cfg.mlrun_complete_timeout,
            interval=cfg.poll_interval,
        )

        # --- serve: a service comes up in the same tenant.
        cs = create_ml_service.sync_detailed(
            tenant,
            client=harness.compute_service,
            body=builders.nginx_mlservice(cfg, svc),
        )
        assert cs.status_code in (200, 201), cs.content

        def svc_ready():
            g = get_ml_service.sync_detailed(
                tenant, svc, client=harness.compute_service
            )
            assert g.status_code == 200 and g.parsed.phase in (
                "Running",
                "Available",
                "Ready",
            ), f"phase={g.parsed.phase!r}"

        eventually(svc_ready, timeout=cfg.pod_ready_timeout, interval=cfg.poll_interval)

        # Best-effort workload teardown before the tenant is removed.
        delete_ml_service.sync_detailed(tenant, svc, client=harness.compute_service)
        delete_ml_run.sync_detailed(tenant, job, client=harness.compute_service)
    finally:
        harness.delete_tenant(tenant)
