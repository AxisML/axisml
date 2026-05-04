# artifacts

AxisML Artifacts is the artifact management service: it owns metadata, reference resolution, and storage credential issuance for the platform's main artifact types.

> **Status: scaffold.** The directory and Makefile are in place; the Go implementation is not yet committed. See [`docs/system_design/artifacts.md`](../../docs/system_design/artifacts.md) for the design and the API surface this service will expose.

## Responsibilities

- **Models** — version, metadata, immutable references, and storage URIs.
- **Images** — register, version, and resolve training / inference container images.
- **Datasets** — metadata, storage location, version, and access credentials.

Artifacts uses a metadata-service / storage-backend split:

| Artifact | Metadata | Storage |
| --- | --- | --- |
| Model | PostgreSQL | zot (OCI Distribution) |
| Image | PostgreSQL | zot (OCI Distribution) |
| Dataset | PostgreSQL | RustFS (S3) |

Uploads and downloads go directly between the CLI / consumer and the storage backend via signed credentials — Artifacts does not proxy bulk file bytes.

## Planned layout

```
cmd/                Service entrypoint (HTTP / gRPC server, DB + storage clients)
internal/
  ├── model/           Model metadata + zot reference resolution
  ├── image/           Image metadata + zot reference resolution
  ├── dataset/         Dataset metadata + RustFS S3 credential issuance
  └── auth/            Internal-call auth (trust identity propagated by Platform)
api/                  HTTP / gRPC contract definitions
deploy/Dockerfile     Container image build (to be added)
```

## Local development

```sh
make help            # list all targets
make / make build    # compile bin/artifacts
make test            # unit tests
make image           # docker build -> ghcr.io/axisml/axisml-artifacts:0.1.0
make clean           # remove build artifacts
```

`IMAGE_TAG` defaults to `0.1.0` and must track the `appVersion` in [`deploy/helm/axisml-system/Chart.yaml`](../../deploy/helm/axisml-system/Chart.yaml).

## Deployment

Artifacts will ship as part of the `axisml-system` chart under `deploy/helm/axisml-system/templates/artifacts/`.
