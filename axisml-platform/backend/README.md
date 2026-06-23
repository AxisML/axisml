# platform-backend

Backend for AxisML Platform — the user-facing entry point that orchestrates business operations across Compute and Artifacts.

> **Status: contract-only shell.** The directory layout, entrypoint, and build
> tooling match the sibling services, but the server is not implemented: the
> entrypoint serves health probes and a `501` fallback for every API route. What
> *is* committed is the API contract — the HTTP surface is declared as Go DTOs
> under [`internal/server/`](internal/server/) and rendered to
> [`axisml-platform/docs/apis/platform.yaml`](../docs/apis/platform.yaml) by
> [`cmd/openapi-gen`](cmd/openapi-gen/) via the shared `pkg/openapigen` engine —
> the same code-first flow used by cluster-manager / compute-service /
> artifact-hub. See [`docs/system_design/platform.md`](../../../docs/system_design/platform.md)
> for the design. When the handlers land, they reuse these same DTOs, so the
> spec stays in lock-step automatically.

## API contract

```sh
make doc-gen     # regenerate axisml-platform/docs/apis/platform.yaml from internal/server DTOs
make doc-test    # CI guard: fail if the committed spec is stale vs the Go types
make test        # spec integrity tests (refs resolve, operationIds unique, coverage)
```

`axisml-platform/docs/apis/platform.yaml` is **generated, never hand-edited** — edit the DTOs
in `internal/server/` (and routes in `cmd/openapi-gen/paths.go`) and regenerate.
The OpenAPI engine renders `components/schemas` only; shared parameters/responses
are inlined per operation and the bearer-JWT security scheme is documented in the
design doc rather than the generated spec (matching the sibling services).

## Responsibilities

- **External API surface** — RESTful API consumed by the frontend and any external clients (via the Envoy Gateway).
- **Business orchestration** — coordinate Compute (jobs, services, tenants, pools, units, quotas) and Artifacts (models, images, datasets) to fulfill user-facing operations.
- **Auth entry point** — central place where user identity, role, and tenant access control are evaluated. Compute and Artifacts only accept internal calls and trust the identity Platform propagates.

The frontend (TypeScript + React) lives in [`../frontend/`](../frontend/) and consumes this backend's API.

## Layout

```
cmd/platform-backend/  Service entrypoint (flags + signals -> internal/app)
cmd/openapi-gen/       OpenAPI generator: DTOs -> axisml-platform/docs/apis/platform.yaml
internal/
  ├── app/             Process wiring: HTTP servers, graceful shutdown, routers
  └── server/          API DTO structs — the contract surface
Dockerfile             Container image build (repo root build context)
test/integration/      httptest-driven integration suite (separate go.mod)

# Planned (land with the server implementation):
internal/
  ├── handler/         HTTP handlers (per resource: jobs, services, tenants, pools, ...)
  ├── orchestrator/    Cross-service orchestration logic
  ├── auth/            IdP integration, role model, tenant access control (TBD)
  └── client/          Typed clients for compute / artifacts
```

## Local development

```sh
make help            # list all targets
make / make build    # compile bin/platform-backend
make test            # unit tests
make image           # docker build -> ghcr.io/axisml/axisml-platform-backend:0.1.0
make clean           # remove build artifacts
```

`IMAGE_TAG` defaults to `0.1.0` and must track the `appVersion` in [`axisml-system/deploy/helm/Chart.yaml`](../../../axisml-system/deploy/helm/Chart.yaml).

## Deployment

Platform backend will ship as part of the `axisml-system` chart under `axisml-system/deploy/helm/templates/platform/`.
