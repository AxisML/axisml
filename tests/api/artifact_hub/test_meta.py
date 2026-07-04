"""artifact-hub: list projection, capability document, and liveness."""

from __future__ import annotations

from clients.artifacthub.api.artifacts import (
    complete_model,
    delete_model,
    initiate_model,
    list_models,
)
from clients.artifacthub.api.capabilities import get_capabilities
from clients.artifacthub.api.health import healthz
from clients.artifacthub.models import (
    ArtifactCompleteRequest,
    ArtifactInitiateRequest,
    ArtifactInitiateRequestSpec,
)
from lib import oci
from lib.harness import Capability
from lib.naming import unique_name
from lib.polling import eventually


def test_capabilities_and_health(harness):
    caps = get_capabilities.sync_detailed(client=harness.artifact_hub)
    assert caps.status_code == 200, caps.content
    assert "model" in caps.parsed.kinds
    # Advertised upload availability must agree with the harness form matrix.
    assert caps.parsed.upload == harness.supports(Capability.ARTIFACT_UPLOAD)

    h = healthz.sync_detailed(client=harness.artifact_hub)
    assert h.status_code == 200, h.content


def test_list_models_projects_uploaded(harness, cfg, tenant):
    """A completed model must surface in the namespace's model listing."""
    harness.skip_unless(Capability.ARTIFACT_UPLOAD)
    ns, _ = tenant
    name = unique_name("e2e-list")
    version = "1.0.0"

    spec = ArtifactInitiateRequestSpec.from_dict({"framework": "onnx", "format": "onnx"})
    init = initiate_model.sync_detailed(
        ns, name, client=harness.artifact_hub, body=ArtifactInitiateRequest(version=version, spec=spec)
    )
    assert init.status_code in (200, 201), init.content
    upload = init.parsed.upload
    try:
        client = oci.OciClient(
            harness.oci_endpoint(),
            oci.OciCreds(username=upload.credentials.username, password=upload.credentials.password),
        )
        try:
            repo, ref = oci.parse_repo_ref(upload.uri)
            digest = client.push_config_only_manifest(repo, ref)
        finally:
            client.close()
        c = complete_model.sync_detailed(ns, name, version, client=harness.artifact_hub, body=ArtifactCompleteRequest(digest=digest))
        assert c.status_code in (200, 201, 202), c.content

        def listed():
            lst = list_models.sync_detailed(ns, client=harness.artifact_hub)
            assert lst.status_code == 200, lst.content
            assert any(a.name == name for a in lst.parsed.items), "uploaded model absent from list"

        eventually(listed, timeout=cfg.cr_provision_timeout, interval=cfg.poll_interval)
    finally:
        delete_model.sync_detailed(ns, name, version, client=harness.artifact_hub)
