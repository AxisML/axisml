"""artifact-hub: list projection, capability document, and liveness."""

from __future__ import annotations

import pytest

from clients.artifacthub.api.artifacts import (
    complete_artifact,
    delete_artifact,
    initiate_artifact,
    list_artifacts,
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


# Lite embeds all System modules in one axisml-core process and serves an
# aggregated {components: {...}} capabilities doc at the shared /capabilities
# path; the per-component flat client (kinds/upload) can't parse it. The flat
# per-service contract is a Standard-form concept — health runs on both forms.
@pytest.mark.standard_only
def test_capabilities(harness):
    caps = get_capabilities.sync_detailed(client=harness.artifact_hub)
    assert caps.status_code == 200, caps.content
    assert "model" in caps.parsed.kinds
    # Advertised upload availability must agree with the harness form matrix.
    assert caps.parsed.upload == harness.supports(Capability.ARTIFACT_UPLOAD)


def test_health(harness):
    h = healthz.sync_detailed(client=harness.artifact_hub)
    assert h.status_code == 200, h.content


def test_list_artifacts_projects_uploaded(harness, cfg, tenant):
    """A completed model must surface in the namespace's artifact listing."""
    harness.skip_unless(Capability.ARTIFACT_UPLOAD)
    ns, _ = tenant
    name = unique_name("e2e-list")
    version = "1.0.0"

    spec = ArtifactInitiateRequestSpec.from_dict({"framework": "onnx", "format": "onnx"})
    init = initiate_artifact.sync_detailed(
        ns, name, client=harness.artifact_hub, body=ArtifactInitiateRequest(kind="model", version=version, spec=spec)
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
        c = complete_artifact.sync_detailed(ns, name, version, client=harness.artifact_hub, body=ArtifactCompleteRequest(digest=digest))
        assert c.status_code in (200, 201, 202), c.content

        def listed():
            lst = list_artifacts.sync_detailed(ns, client=harness.artifact_hub, kind="model")
            assert lst.status_code == 200, lst.content
            assert any(a.name == name for a in lst.parsed.items), "uploaded model absent from list"

        eventually(listed, timeout=cfg.cr_provision_timeout, interval=cfg.poll_interval)
    finally:
        delete_artifact.sync_detailed(ns, name, version, client=harness.artifact_hub)
