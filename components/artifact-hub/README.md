# artifact-hub

AxisML Artifact Hub is the artifact management service: it owns metadata, reference resolution, and storage credential issuance for the platform's main artifact types.

> **Status: MVP.** The model Kind end-to-end happy path is implemented:
> create repo → initiate (POST /artifacts) → cli `oras push` → complete →
> resolve → DELETE → GC. Other Kinds (`dataset`, `image`), `auth_hint`,
> public space, and full GC predicates are tracked under
> `docs/system_design/components/artifact-hub.md` §8.3.

## Responsibilities

- **Models** — version, metadata, immutable references, and storage URIs (MVP).
- **Images** — register, version, and resolve container images (Phase 2).
- **Datasets** — metadata, storage location, and access credentials (Phase 2).

artifact-hub uses a metadata-service / storage-backend split:

| Artifact     | Metadata    | Storage              | Phase |
| ------------ | ----------- | -------------------- | ----- |
| Model        | PostgreSQL  | zot (OCI)            | MVP   |
| Image        | PostgreSQL  | zot (OCI)            | 2     |
| Dataset      | PostgreSQL  | RustFS (S3)          | 2     |

Uploads and downloads go directly between the cli/consumer and the storage backend via signed credentials — artifact-hub does not proxy bulk file bytes.

## MVP simplifications (acknowledged tech debt)

- **OCI auth is admin-passthrough.** zot uses htpasswd; artifact-hub holds the
  admin user/password and returns those creds verbatim from `initiate` and
  `resolve?usage=download`. **NOT scope-limited, NOT TTL-bounded** —
  any holder can push to any zot repo. Acceptable only because zot is
  ClusterIP-scoped inside `axisml-infra`. Phase 2 replaces this with a
  JWT-issuing bearer-token realm.
- **Tenant resolver shares compute-service's PG.** artifact-hub reads the `tenants`
  table directly (read-only) via `internal/tenantresolver/`. The dependency
  is encapsulated in one package so a future cross-org split swaps in an
  HTTP client to compute-service.
- **No `auth_hint` on `resolve?usage=inspect`.** Operators rely on the
  convention-named imagePullSecret (`axisml-tenant-<tenant>-zot-pull`) per
  design §8.2.

## Layout

```
cmd/artifact-hub/        Service entrypoint (Cobra: serve, migrate)
internal/
  ├── app/              Boot wiring (Serve, Migrate, BuildModules)
  ├── config/           Env-driven Config
  ├── db/               GORM client + golang-migrate (migrations/*.sql)
  ├── server/           Gin engine, middleware, RFC7807 error rendering
  ├── auth/             X-Axisml-User parser
  ├── metrics/          Prometheus + Gin middleware
  ├── k8sclient/        controller-runtime Manager (leader election + probes)
  ├── tenantresolver/   Read-only `tenants` lookup (shared PG)
  ├── repo/             ArtifactRepo CRUD + state
  ├── artifact/         Artifact CRUD + state machine + initiate/complete/resolve
  │   └── handler/      Kind handler registry; only `model` in MVP
  ├── storage/oci/      zot client (HEAD / DELETE manifest, credential issuance)
  ├── gc/               Leader-only GC worker (Uploading TTL, Deleting cleanup)
  └── integration/      Build-tag `integration` end-to-end tests (testcontainers)
pkg/{logging,errors,strutil}/   Shared utilities (mirror compute-service/pkg)
deploy/Dockerfile     Container image build (multi-stage, repo-root context)
```

## Local development

```sh
make help                # list all targets
make / make build        # compile bin/artifact-hub
make test                # unit tests (no Docker required)
make integration         # integration tests (PG + zot via testcontainers; Docker required)
make image               # docker build -> ghcr.io/axisml/axisml-artifact-hub:<IMAGE_TAG>
make image-load-minikube # build and load into the local minikube node
make clean               # remove build artifacts
```

`IMAGE_TAG` defaults to `0.1.0`. The top-level `Makefile` overrides it from
`deploy/helm/axisml-system/Chart.yaml` `appVersion` so locally-built images
match what Helm will pull.

## Deployment

artifact-hub ships as part of the `axisml-system` chart at
[`deploy/helm/axisml-system/templates/artifact-hub/`](../../deploy/helm/axisml-system/templates/artifact-hub/).
The chart provisions a `Deployment`, `Service`, `ConfigMap`, two `Secret`s
(`-artifact-hub-db` for PG password, `-artifact-hub-zot` for OCI admin creds),
`ServiceAccount`, leader-election `Role`/`RoleBinding`, and an optional
`ServiceMonitor`.

For end-to-end deploy:

```sh
make image-load IMAGE_TAG=dev
make helm-install IMAGE_TAG=dev
kubectl -n axisml-system rollout status deploy/axisml-artifact-hub --timeout=120s
```
