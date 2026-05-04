# platform-backend

Backend for AxisML Platform — the user-facing entry point that orchestrates business operations across Compute and Artifacts.

> **Status: scaffold.** The directory and Makefile are in place; the Go implementation is not yet committed. See [`docs/system_design/platform.md`](../../../docs/system_design/platform.md) for the design and the API surface this service will expose.

## Responsibilities

- **External API surface** — RESTful API consumed by the frontend and any external clients (via the Envoy Gateway).
- **Business orchestration** — coordinate Compute (jobs, services, tenants, pools, units, quotas) and Artifacts (models, images, datasets) to fulfill user-facing operations.
- **Auth entry point** — central place where user identity, role, and tenant access control are evaluated. Compute and Artifacts only accept internal calls and trust the identity Platform propagates.

The frontend (TypeScript + React) lives in [`../frontend/`](../frontend/) and consumes this backend's API.

## Planned layout

```
cmd/                Service entrypoint (HTTP server, downstream clients, auth)
internal/
  ├── api/             HTTP handlers (per resource: jobs, services, tenants, pools, ...)
  ├── orchestrator/    Cross-service orchestration logic
  ├── auth/            IdP integration, role model, tenant access control (TBD)
  └── client/          Typed clients for compute / artifacts
api/                  OpenAPI / proto contract definitions
deploy/Dockerfile     Container image build (to be added)
```

## Local development

```sh
make help            # list all targets
make / make build    # compile bin/platform-backend
make test            # unit tests
make image           # docker build -> ghcr.io/axisml/axisml/axisml-platform-backend:0.1.0
make clean           # remove build artifacts
```

`IMAGE_TAG` defaults to `0.1.0` and must track the `appVersion` in [`deploy/helm/axisml-system/Chart.yaml`](../../../deploy/helm/axisml-system/Chart.yaml).

## Deployment

Platform backend will ship as part of the `axisml-system` chart under `deploy/helm/axisml-system/templates/platform/`.
