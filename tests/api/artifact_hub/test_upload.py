"""artifact-hub: two-phase model upload against the REAL zot registry.

initiate -> push a config-only manifest to zot -> complete -> resolve, asserting
the resolved digest equals what was pushed. Metadata-only lifecycle is covered by
the hermetic integration suite, so it isn't duplicated here. Runs under both forms
(``ARTIFACT_UPLOAD``): the harness resolves a host-visible zot endpoint per form.

The API exposes a single /artifacts resource for every kind; kind travels in the
initiate body.
"""

from __future__ import annotations

from clients.artifacthub.api.artifacts import (
    complete_artifact,
    delete_artifact,
    get_artifact,
    initiate_artifact,
    resolve_artifact,
)
from clients.artifacthub.models import (
    ArtifactCompleteRequest,
    ArtifactInitiateRequest,
    ArtifactInitiateRequestSpec,
)
from lib import oci
from lib.harness import Capability
from lib.naming import unique_name
from lib.polling import eventually


def test_model_two_phase_upload_resolve(harness, cfg, tenant):
    harness.skip_unless(Capability.ARTIFACT_UPLOAD)
    ns, _ = tenant
    name = unique_name("e2e-2phase")
    version = "1.0.0"

    spec = ArtifactInitiateRequestSpec.from_dict({"framework": "onnx", "format": "onnx"})
    init = initiate_artifact.sync_detailed(
        ns, name, client=harness.artifact_hub, body=ArtifactInitiateRequest(kind="model", version=version, spec=spec)
    )
    assert init.status_code in (200, 201), init.content
    upload = init.parsed.upload
    try:
        # Push a minimal manifest to the host-visible registry (zot).
        client = oci.OciClient(harness.oci_endpoint(), oci.OciCreds(username=upload.credentials.username, password=upload.credentials.password))
        try:
            repo, ref = oci.parse_repo_ref(upload.uri)
            digest = client.push_config_only_manifest(repo, ref)
        finally:
            client.close()

        # Complete with the pushed digest.
        c = complete_artifact.sync_detailed(ns, name, version, client=harness.artifact_hub, body=ArtifactCompleteRequest(digest=digest))
        assert c.status_code in (200, 201, 202), c.content

        # Status becomes Ready.
        def ready():
            g = get_artifact.sync_detailed(ns, name, version, client=harness.artifact_hub)
            assert g.status_code == 200, g.content
            assert g.parsed.status.lower() == "ready", f"status={g.parsed.status!r}"

        eventually(ready, timeout=cfg.cr_provision_timeout, interval=cfg.poll_interval)

        # Resolve echoes the completed digest.
        r = resolve_artifact.sync_detailed(ns, name, version, client=harness.artifact_hub)
        assert r.status_code == 200, r.content
        assert r.parsed.digest == digest, f"resolve digest {r.parsed.digest!r} != pushed {digest!r}"
    finally:
        delete_artifact.sync_detailed(ns, name, version, client=harness.artifact_hub)
