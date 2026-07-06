"""artifact-hub: two-phase upload for the dataset & image kinds.

The model kind is covered in test_upload.py. dataset and image route through the
same two-phase handler but a different kind branch (spec storage / provenance),
which was never exercised black-box. initiate -> push manifest -> complete ->
resolve, asserting the resolved digest matches. Runs under both forms
(``ARTIFACT_UPLOAD``).
"""

from __future__ import annotations

import pytest

from clients.artifacthub.api.artifacts import (
    complete_dataset,
    complete_image,
    delete_dataset,
    delete_image,
    get_dataset,
    get_image,
    initiate_dataset,
    initiate_image,
    resolve_dataset,
    resolve_image,
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

# kind -> (initiate, complete, get, resolve, delete, spec fields)
_KINDS = {
    "dataset": (initiate_dataset, complete_dataset, get_dataset, resolve_dataset, delete_dataset, {"format": "parquet"}),
    "image": (initiate_image, complete_image, get_image, resolve_image, delete_image, {"format": "oci", "purpose": "training"}),
}


@pytest.mark.parametrize("kind", list(_KINDS))
def test_two_phase_upload_resolve(harness, cfg, tenant, kind):
    harness.skip_unless(Capability.ARTIFACT_UPLOAD)
    initiate, complete, get, resolve, delete, spec_fields = _KINDS[kind]
    ns, _ = tenant
    name = unique_name(f"e2e-{kind}")
    version = "1.0.0"

    spec = ArtifactInitiateRequestSpec.from_dict(spec_fields)
    init = initiate.sync_detailed(
        ns, name, client=harness.artifact_hub, body=ArtifactInitiateRequest(version=version, spec=spec)
    )
    assert init.status_code in (200, 201), init.content
    upload = init.parsed.upload
    try:
        if upload.uri.startswith("s3://"):
            # S3/RustFS-backed kind (dataset). The MVP has no live STS push and
            # VerifyComplete accepts the claimed digest verbatim, so record a
            # synthetic content digest rather than pushing to the OCI registry.
            digest = oci.sha256_digest(b"e2e-dataset:" + name.encode())
        else:
            # OCI-registry-backed kind (image / model): push a real manifest.
            client = oci.OciClient(
                harness.oci_endpoint(),
                oci.OciCreds(username=upload.credentials.username, password=upload.credentials.password),
            )
            try:
                repo, ref = oci.parse_repo_ref(upload.uri)
                digest = client.push_config_only_manifest(repo, ref)
            finally:
                client.close()

        c = complete.sync_detailed(ns, name, version, client=harness.artifact_hub, body=ArtifactCompleteRequest(digest=digest))
        assert c.status_code in (200, 201, 202), c.content

        def ready():
            g = get.sync_detailed(ns, name, version, client=harness.artifact_hub)
            assert g.status_code == 200, g.content
            assert g.parsed.status.lower() == "ready", f"status={g.parsed.status!r}"

        eventually(ready, timeout=cfg.cr_provision_timeout, interval=cfg.poll_interval)

        r = resolve.sync_detailed(ns, name, version, client=harness.artifact_hub)
        assert r.status_code == 200, r.content
        assert r.parsed.digest == digest, f"resolve digest {r.parsed.digest!r} != pushed {digest!r}"
    finally:
        delete.sync_detailed(ns, name, version, client=harness.artifact_hub)
