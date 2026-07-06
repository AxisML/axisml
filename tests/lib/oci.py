"""Minimal OCI-distribution push for the artifact-hub two-phase upload test.

Ports the Go suite's ``oci_test.go``: push a tiny config-only image manifest to
the real zot registry (reached over a port-forward) and return the manifest
digest that the ``complete`` call records. Defensive about the upload
credential/URI shape (the one contract that can't be verified offline).
"""

from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass

import httpx


@dataclass
class OciCreds:
    username: str = ""
    password: str = ""
    token: str = ""


def parse_repo_ref(uri: str) -> tuple[str, str]:
    """Extract (repository, reference) from an upload URI, ignoring the host."""
    s = uri
    for p in ("oci://", "https://", "http://", "docker://"):
        if s.startswith(p):
            s = s[len(p):]
    if "/" in s:
        s = s[s.index("/") + 1:]
    ref = "latest"
    # A trailing :tag (no '/' after the colon) is the reference.
    if ":" in s:
        head, _, tail = s.rpartition(":")
        if "/" not in tail:
            ref, s = tail, head
    # The upload URI can carry a leading '//' after the host or a trailing '/';
    # strip them so the /v2/<repo>/blobs path has no double slashes (which the
    # registry 301-redirects to the canonical form).
    return s.strip("/"), ref


def sha256_digest(b: bytes) -> str:
    return "sha256:" + hashlib.sha256(b).hexdigest()


class OciClient:
    def __init__(self, base: str, creds: OciCreds):
        self.base = base.rstrip("/")
        self._auth = None
        self._headers: dict[str, str] = {}
        if creds.token:
            self._headers["Authorization"] = f"Bearer {creds.token}"
        elif creds.username:
            self._auth = (creds.username, creds.password)
        self._http = httpx.Client(timeout=30.0)

    def _req(self, method: str, path: str, *, content: bytes | None = None, content_type: str | None = None) -> httpx.Response:
        headers = dict(self._headers)
        if content_type:
            headers["Content-Type"] = content_type
        return self._http.request(method, self.base + path, content=content, headers=headers, auth=self._auth)

    def push_config_only_manifest(self, repo: str, ref: str) -> str:
        """Push an empty config blob + a manifest referencing it; return the manifest digest."""
        config = b"{}"
        cfg_digest = sha256_digest(config)
        self._push_blob(repo, cfg_digest, config)

        manifest = {
            "schemaVersion": 2,
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "config": {
                "mediaType": "application/vnd.oci.image.config.v1+json",
                "digest": cfg_digest,
                "size": len(config),
            },
            "layers": [],
        }
        mb = json.dumps(manifest).encode()
        m_digest = sha256_digest(mb)
        resp = self._req(
            "PUT", f"/v2/{repo}/manifests/{ref}", content=mb,
            content_type="application/vnd.oci.image.manifest.v1+json",
        )
        if resp.status_code // 100 != 2:
            raise RuntimeError(f"put manifest: {resp.status_code}: {resp.text}")
        return m_digest

    def _push_blob(self, repo: str, digest: str, blob: bytes) -> None:
        start = self._req("POST", f"/v2/{repo}/blobs/uploads/")
        loc = start.headers.get("Location", "")
        if start.status_code // 100 != 2 or not loc:
            raise RuntimeError(f"start upload: status {start.status_code} loc={loc!r}")
        # Normalize an absolute Location to a path on our base.
        for p in ("http://", "https://"):
            if loc.startswith(p):
                rest = loc[len(p):]
                loc = rest[rest.index("/"):] if "/" in rest else "/"
        sep = "&" if "?" in loc else "?"
        put = self._req("PUT", f"{loc}{sep}digest={digest}", content=blob, content_type="application/octet-stream")
        if put.status_code // 100 != 2:
            raise RuntimeError(f"put blob: {put.status_code}: {put.text}")

    def close(self) -> None:
        self._http.close()
