# platform-frontend

Frontend for AxisML Platform — the Web UI delivered to users.

> **Status: scaffold.** The directory and Makefile are in place; the TypeScript + React implementation is not yet committed. See [`docs/system_design/platform.md`](../../../docs/system_design/platform.md) for the design.

## Tech stack

- **Language**: TypeScript
- **Framework**: React
- **Build**: Vite (planned)

The backend (Go) lives in [`../backend/`](../backend/) and provides the REST API this frontend consumes.

## Functional surface (planned)

- Job management — create / monitor / cancel `MLJob`s, view logs and metrics.
- Service management — deploy / scale / route `MLService`s, view inference traffic.
- Artifact center — browse models, images, datasets; manage versions and references.
- System management — tenants, resource pools, resource units, quotas, data volumes.

## Planned layout

```
src/
  ├── pages/           Top-level routes (Job, Service, Artifact, System)
  ├── components/      Shared UI components
  ├── api/             Typed clients for the platform-backend API
  ├── hooks/           React hooks (data fetching, auth, tenant context)
  └── styles/          Global styles / theme
public/                Static assets
package.json           npm scripts: build, test, dev, lint
Dockerfile             Container image build (to be added)
```

## Local development

The Makefile expects a `package.json` with the following npm scripts: `build`, `test`, `dev`, `lint`.

```sh
make help            # list all targets
make install         # npm install
make / make build    # build the production bundle into dist/
make test            # unit tests
make dev             # start the dev server
make lint            # lint the codebase
make image           # docker build -> ghcr.io/axisml/axisml-platform-frontend:0.1.0
make clean           # remove dist/, build/, and the npm cache
```

`IMAGE_TAG` defaults to `0.1.0` and must track the `appVersion` in [`deploy/helm/axisml-system/Chart.yaml`](../../../deploy/helm/axisml-system/Chart.yaml).

## Deployment

Platform frontend will ship as part of the `axisml-system` chart under `deploy/helm/axisml-system/templates/platform/`. In production it is fronted by the Envoy Gateway.
