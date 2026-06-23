# axisml-platform

The **Platform layer** of AxisML — the **only layer that directly faces users and the only one exposed externally**. It owns identity/authentication, business orchestration, and the view-layer mapping, translating user actions into internal calls to the System layer (cluster-manager / compute-service / artifact-hub). It is the top of the three deployment layers (**infra → system → platform**).

External traffic enters Platform through the Envoy Gateway; everything downstream is ClusterIP and trusts the `X-Axisml-User` header Platform injects.

## Components

| Component | What it does |
| --- | --- |
| **[backend](backend/)** | Go BFF: identity / JWT auth, cross-service orchestration, and the authority for durable tenant records + the four name-level definitions (Job / Experiment / Model / Image) + view-layer mapping. Holds **no** downstream instance state — Run / Service / Workspace / artifact-version / quota-folding all live in the System layer and are fetched live. |
| **[frontend](frontend/)** | React + Vite + TypeScript SPA: routing, data fetching, i18n, data-plane access. Consumes the backend REST API through a typed client generated from the OpenAPI spec; never talks to System / Infra directly. |

Authentication exists **only** at this layer — System / Infra components do no user auth, they only trust the propagated `X-Axisml-User`. See [`backend`](backend/) and the design docs for the RBAC model and downstream identity-propagation contract.

## Boundary

- **Owns:** durable tenant records (`tenants` table — tenant scope, K8s-Namespace mapping, suspend, hard-delete), the four definitions, users / roles / sessions.
- **Does not own:** any K8s resource or CR, runtime instances (Run / Service / Workspace), artifact versions, quota folding, or namespace resolution — all pushed down to the System layer.

## Layout

```
backend/                 Go BFF (own module)
  ├── cmd/platform-backend/   Server entrypoint
  ├── cmd/openapi-gen/        DTOs → docs/apis/platform.yaml
  ├── internal/server/        API DTO structs — the contract surface
  └── test/integration/       httptest-driven suite (separate module)
frontend/                Vite + React + TS SPA (pnpm; own toolchain)
  └── src/api/generated/      Typed client GENERATED from platform.yaml (never hand-edited)
docs/
  ├── apis/platform.yaml      Generated OpenAPI spec (single source of truth for the HTTP contract)
  ├── system_design/          overview · backend · frontend · auth · database
  └── product_design/         PRD + interactive prototype/ (page design reference)
deploy/helm/             The "axisml-platform" chart (frontend + backend; the only exposed layer)
```

The OpenAPI spec drives both directions: the **frontend's** typed client (`make frontend-gen-api`) and — downstream of the System specs — the **backend's** typed System clients (`make client-gen`).

## Build / test

```sh
make help              # list all targets
make build             # build the backend binary
make test              # backend unit tests
make integration       # backend httptest suite (in-process gin; no envtest/Docker)
make doc-gen           # regenerate docs/apis/platform.yaml from backend DTOs
make doc-test          # CI guard: fail if the committed spec is stale vs the Go types
make client-gen        # regenerate backend's typed System clients from axisml-system/docs/apis

# Frontend (pnpm; NOT in the default build aggregate):
make frontend-install
make frontend-dev          # dev server (proxies /api → backend)
make frontend-build
make frontend-gen-api      # regenerate the typed client from docs/apis/platform.yaml
make frontend-lint
```

`docs/apis/platform.yaml` is **generated, never hand-edited** — edit the DTOs under `backend/internal/server/` and regenerate. The pre-commit `doc-test` hook does **not** watch Platform backend DTOs, so run `make doc-gen` yourself after changing them.

## Deployment

The Platform chart installs **last** (infra → system → platform) and is the **only externally-exposed** layer — in production it is fronted by the Envoy Gateway via an `HTTPRoute`. It depends on the System-layer services being up.

```sh
make helm-lint        # lint the chart
make helm-template    # render locally for review
make helm-install     # install or upgrade the axisml-platform release (idempotent)
make helm-uninstall   # tear down
```

`IMAGE_TAG` defaults to the `appVersion` in `axisml-system/deploy/helm/Chart.yaml` — the single image-tag authority across all three charts.

## See also

- **[Platform design overview](docs/system_design/overview.md)** + **[backend](docs/system_design/backend.md)** · **[frontend](docs/system_design/frontend.md)** · **[auth](docs/system_design/auth.md)** · **[database](docs/system_design/database.md)**
- **[PRD](docs/product_design/prd.md)** + the interactive **[prototype/](docs/product_design/prototype)** (the page-design reference)
- **[High-level design](../docs/high_level_design.md)** — where Platform sits in the three-layer architecture
- **[DESIGN.md](../DESIGN.md)** — the frontend visual design system (Vercel Geist style)
