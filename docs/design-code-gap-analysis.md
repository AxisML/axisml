# Design Doc ↔ Code Gap Analysis

> Generated 2026-06-28. Audit of every component's `system_design` doc against its
> actual implementation (code, generated API specs, Helm charts, migrations).
>
> **Bias:** code is treated as the newer truth; the default action is to update the
> design doc to match. Each row carries a **Recommendation**:
> - **Doc → match code** — code is the current behavior; rewrite the doc.
> - **Code gap** — the doc describes intended state that is *not yet* implemented; the
>   doc may be correct as a target. Decide whether to implement or to demote to "future work".
> - **Fix both** — doc and code agree with each other but a third artifact (e.g. a
>   `main.go` default, an API spec) is the odd one out.
>
> **Status (2026-06-28): RESOLVED — all rows below have been applied.** Docs were updated
> to match the code; the operator port defaults were fixed in code; three OpenAPI specs were
> regenerated (never hand-edited). The per-row text below is retained as the historical audit
> record. One row turned out to be a false finding on re-verification:
>
> - **axisml-lite cancel → CR condition:** the audit claimed the code didn't write the
>   `Suspended=True, reason=CancelRequested` condition. On re-check, the code **does** write it
>   (`run.go:194-202`), so the doc (§6.6) was already correct and was left unchanged.
>
> Additionally, the platform-backend fix was extended beyond the audit to `auth.md` §7 (the
> canonical middleware contract) and the backend.md §4.x endpoint-flow tables, which carried the
> same fictional `Require*Owner` shorthand.

---

## Cross-cutting theme: operator metrics/probe port defaults

Both operators have the *same* discrepancy and it's worth deciding once:

| Component | Doc says | Helm deployment passes | `main.go` flag default |
|---|---|---|---|
| tenant-operator | metrics `:8081`, probes `:8082` | `--metrics-bind-address=:8081`, `--health-probe-bind-address=:8082` | `:8080` / `:8081` |
| compute-operator | metrics `:8081`, probes `:8082` | (same intent) | `:8080` / `:8081` |

The **deployed behavior matches the doc** (Helm overrides the flags). The odd one out is the
hardcoded default in `cmd/main.go`. **Recommendation: Fix both** — align the `main.go` defaults
to `:8081`/`:8082` so the doc, the chart, and the binary defaults all agree. (Alternatively, if
the bare-binary default is intentional, add a one-line note to the doc.)

---

## tenant-operator

| Topic | Doc says | Code does | Recommendation |
|---|---|---|---|
| Metrics/probe ports | §8 (L172): metrics `:8081`, probes `:8082` | `main.go:39-41` defaults `:8080`/`:8081`; Helm `deployment.yaml:29-30` overrides to `:8081`/`:8082` | **Fix both** — align `main.go` defaults (see cross-cutting note) |
| ElasticQuota naming | §3 table (L49): `axisml-<tenant>-<pool>` | `internal/reconcile/naming.go:27-28`: `axisml-<tenant>-<pool>-<quota>` (3 segments). §6.2 (L77) already states it correctly | **Doc → match code** — fix the incomplete table entry at L49 to include `-<quota>` |
| Namespace RBAC verbs | §8 (L173): `namespaces` get `update` (no delete) | `clusterrole.yaml:29-31`: grants `patch`, **not** `update`; no delete | **Doc → match code** — code's narrower `patch` is correct; update the doc's verb list |
| RBAC `bind`/`escalate` | §8 (L173): not mentioned | `clusterrole.yaml:45-54`: grants `bind` on roles and `bind`+`escalate` (needed to create per-tenant Roles) | **Doc → match code** — document the escalation verbs and why they're required |

---

## compute-operator

| Topic | Doc says | Code does | Recommendation |
|---|---|---|---|
| Metrics/probe ports | §8 (L227): metrics `:8081`, probes `:8082` | `cmd/main.go:51,53` defaults `:8080`/`:8081` | **Fix both** (see cross-cutting note) |

Everything else verified as **matching**: three dispatchers + enable flags, the `(native,job)` /
`(native,deployment|statefulset)` / `(native,httproute)` handlers, the mandatory koord-scheduler
+ quota label injection on all backend Pods, phase/status fields, immutability enforcement, the
handler-registry `init()` pattern, no-PodGroup / gang-scheduling-out-of-scope, and KServe/inference
left as reserved extension points.

---

## cluster-manager

| Topic | Doc says | Code does | Recommendation |
|---|---|---|---|
| `GET /api/v1/storageclasses` | §6 interface contract lists only resourcepools / tenants / volumes | Implemented `volume/handler.go:40`, registered `module.go:46`; present in `cluster-manager.yaml` | **Doc → match code** — add storageclasses to the §6 endpoint list |
| `updatedAt` on ResourcePool | Not mentioned (§3.1/§4.1/§6 only document `createdAt`) | Response type declares `UpdatedAt` (`types.go:31`, in API spec L639) but `PoolToAPI()` (`conversion.go:11-27`) never populates it — always zero | **Code gap** — either wire `UpdatedAt` from K8s metadata or drop the field from the response type/spec. Then reflect the decision in the doc |

Otherwise all CRUD ops, HTTP methods, DTO fields, CRD spec/status, and error codes align.

---

## compute-service

| Topic | Doc says | Code does | Recommendation |
|---|---|---|---|
| Quota assembly | §5.4 (compute-service.md L188): compute auto-assembles `axisml-<identifier>-<pool>` into `spec.scheduling.quota` | `service.go:45-46` *requires* a non-empty caller-provided `Quota`; passes it through (`service.go:94`) | **Decide intent.** If caller-provided is the design, **Doc → match code**. If auto-assembly is still wanted, **Code gap** |
| MLRun filter index | database.md §2.1 (L68-69): `mlruns_namespace_job_created` on `labels->>'axisml.io/job'` | migration `0001_init.up.sql:48-50`: `mlruns_namespace_project_created` on `labels->>'axisml.io/project'` | **Doc → match code** — name + label key both differ; confirm `project` is the intended scope key |
| MLService filter index | database.md §2.2 (L103): symmetric `axisml.io/job` | migration L85-87: `mlservices_namespace_project_created` on `axisml.io/project` | **Doc → match code** (same as above) |
| List response shape | API spec `MLRunList`/`TrafficPolicyList`: `{items, total}` | handlers return `{items, count, total, continueToken}` (`mlrun/handler.go:144-149`, mlservice + `trafficpolicy/handler.go:62-67`) | **Doc → match code** — regenerate the spec so `count`/`continueToken` are documented (run doc-gen) |

---

## artifact-hub

| Topic | Doc says | Code does | Recommendation |
|---|---|---|---|
| Artifact status enum | API spec (`artifact-hub.yaml:96,274-275`): "Pending, Ready, Failed, Deleting" | Code (`artifact/model.go:5-9`) + design doc §3/§6 use `Uploading → Ready/Failed → Deleting → Deleted`; initiate sets `StatusUploading` (`service.go:107`) | **Doc → match code** — the *generated spec* and the `server/artifact.go:27` docstring are stale ("Pending", missing "Deleted"). Fix the docstring + regenerate spec |
| Table prefix | §7 (L142): "表前缀 `artifact_*`" | Table is `artifacts` (no prefix); database.md §3 agrees | **Doc → match code** — remove the bogus prefix claim in §7 |
| Index names | database.md §3 (L164-168): `artifacts_nknv_uniq`, `artifacts_namespace_kind`, `artifacts_visibility_public`, `artifacts_labels_gin` | migration: `uq_artifacts_coord`, `idx_artifacts_namespace_kind`, `idx_artifacts_visibility_public`, `artifacts_labels_gin` + extra `idx_artifacts_workset`, `idx_artifacts_uploading_ttl` | **Doc → match code** — update index names and add the two GC-supporting indexes |
| `owner` column | database.md §3 (L153): `owner text` | migration `0001_init.up.sql:21`: `owner_user text` (API still exposes `owner`) | **Doc → match code** — rename to `owner_user` in the schema doc |
| Nullable columns | database.md §3 (L150): `display_name text`, `description text` (nullable) | migration L17-18: both `NOT NULL DEFAULT ''` | **Doc → match code** — note the NOT NULL + default constraints |

---

## platform-backend

| Topic | Doc says | Code does | Recommendation |
|---|---|---|---|
| System-admin storage | database.md §2: roles "不入表"; omits where system-admin lives | migration `0001_init.up.sql:38` adds `users.is_system_admin boolean`; used in `store/identity.go:44`. Migration comment itself flags the doc gap | **Doc → match code** — document the `is_system_admin` column |
| Owner middlewares | backend.md §6 (L302): `RequireJobOwner` / `RequireExperimentOwner` / `RequireServiceOwner` / `RequireTrafficPolicyOwner` / `RequireWorkspaceOwner` | No such middlewares exist. Uses `RequireActiveTenantRole(RoleUser)` at route level + per-handler ownership checks in the service layer (e.g. `job/handler.go:25-44`) | **Doc → match code** — rewrite §6 to describe tenant-role middleware + service-layer ownership validation |
| `GET /api/v1/storageclasses` | Not in any system-design doc | Implemented `datavolume/handler.go:6`, system-admin gated, proxies cluster-manager | **Doc → match code** — add to backend doc |
| Dashboard | §4.7: explicitly deferred to future design | No dashboard endpoints — matches | **No action** (intentional) |
| Workspace/TensorBoard data-plane access | auth.md §5: "规划，当前 fail-closed"; API KEY "本版本不实现" | No access-JWT/API-KEY endpoints; route creation fails closed | **No action** (intentional, matches) |

---

## platform-frontend

| Topic | Doc says | Code does | Recommendation |
|---|---|---|---|
| Tenant switching | frontend.md: "切换即整页刷新" (full page refresh on switch) | `app/store.tsx:131` soft-switches: `setTenant` only writes localStorage + state; TanStack Query refetches via the updated `axisml.tenant` cookie (`api/setup.ts:22`). No `window.location.reload()` | **Doc → match code** — soft switch appears intentional (better UX); update the doc |

Everything else verified **matching**: shadcn/Radix + Tailwind (no AntD), recharts + sonner +
Radix layers, lucide-react, react-i18next + dayjs (zh-CN/en-US, localStorage-only lang), @hey-api
generated client, CSS-variable design tokens + `data-theme` toggle, Geist font, the 5-group
sidebar, cookie-first tenant header, and the JWT/401-redirect session flow.

> Note: CLAUDE.md / project overview still describe the stack as "AntD + Tailwind". The frontend
> doc (frontend.md) is already correct (shadcn/Radix). Consider correcting CLAUDE.md separately.

---

## axisml-lite

| Topic | Doc says | Code does | Recommendation |
|---|---|---|---|
| Observability compose file | §3.2 (L123): `deploy/docker-compose.observability.yaml` | File does not exist; no observability profile | **Code gap** — implement the file or remove the reference (demote to future work) |
| `io.axisml.instance-id` label | §5.2 (L407): required on managed containers | `baseLabels()` (`runtime.go:129-137`) omits it — only managed/resource-*/replica-index/role/tenant | **Code gap** — add the label or drop it from the doc |
| Cancel → CR status | §6.6 (L548): cancelled Run adds `Suspended=True, reason=CancelRequested` condition; pod surfaces Suspended | Cancellation tracked in an internal map only (`runtime.go`); no condition written; `pod.go:59-82` has no Suspended phase | **Code gap** — surface the condition or simplify the doc to describe internal-only cancel |
| ResourcePool example | §5.1.1 (L298-312): 2 units (`cpu-small`, `gpu-1x`) | `resource-pool.yaml:14-20` adds `cpu-medium` | **Doc → match code** — minor; update the example |
| Event ring capacity | §6.6 (L553): "bounded" (no number) | Fixed 512 (`runtime.go:86`) | **Doc → match code** — optional; state the concrete cap |

All other contract mappings (Docker DeviceRequest GPU, named volumes, Traefik network/weighted
routing, restart policies, backend tuple validation, no-Docker-socket on platform backend) align.

---

## axisml-infra

| Topic | Doc says | Code/chart does | Recommendation |
|---|---|---|---|
| Gateway listeners | overview.md L49: HTTP(80) + HTTPS(443) | `templates/gateway.yaml:30-32`: only HTTP/80 listener | **Code gap** — add the HTTPS/443 listener or mark it future work in the doc |
| RustFS service DNS | values.yaml L13 comment: `rustfs-svc.axisml-infra:9000` | `fullnameOverride: rustfs` → actual `rustfs.axisml-infra:9000` (values.yaml:45) | **Doc → match code** — fix the stale inline comment |
| Koordinator descheduler | overview.md L163: "暂不启用" | `values.yaml:149-150`: `descheduler.replicas: 1` | **Decide intent** — confirm whether the descheduler is meant to run; align doc or values accordingly |

All 8 infra components in the §1 table are present in `Chart.yaml` with pinned versions; PostgreSQL
and cross-namespace DNS conventions match.

---

## Suggested order of operations

1. **Pure doc-staleness rewrites (safe, code is truth):** tenant-operator naming/RBAC, cluster-manager
   storageclasses, compute-service indexes + list shape, artifact-hub status/schema/indexes,
   platform-backend middleware/storageclasses/is_system_admin, frontend tenant-switch, lite
   resource-pool/event-ring, infra RustFS comment. These just need doc edits (+ a `make doc-gen`
   for the regenerated API specs).
2. **Decisions needed (doc could be a real target):** compute-service quota auto-assembly,
   cluster-manager `updatedAt`, lite observability file / instance-id label / cancel condition,
   infra HTTPS listener, infra descheduler.
3. **Fix-both:** operator `main.go` metrics/probe port defaults.

Tell me which rows (or groups) to apply and I'll make the edits.
