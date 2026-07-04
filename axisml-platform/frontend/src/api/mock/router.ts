// Mock request router for VITE_USE_MOCK_API mode. It is installed as the custom
// `fetch` of the generated @hey-api/client-fetch client (see api/client.ts), so
// every SDK call resolves here without any network I/O. The router returns real
// Response objects; the client wrapper handles parsing + the {data,error} unwrap
// exactly as it would for a live backend.
import * as db from "./data";

interface Result {
  status?: number;
  body?: unknown;
}

// A route is a method + path-pattern (":param" segments) + handler.
type Handler = (params: Record<string, string>, query: URLSearchParams, body: unknown) => Result;
interface Route {
  method: string;
  pattern: string;
  handler: Handler;
}

const routes: Route[] = [];
const on = (method: string, pattern: string, handler: Handler) =>
  routes.push({ method, pattern: `/api/v1${pattern}`, handler });

const list = (items: unknown[]): Result => ({ body: { count: items.length, items } });
const ok = (body: unknown = {}): Result => ({ body });
const created = (body: unknown): Result => ({ status: 201, body });
const noContent = (): Result => ({ status: 204 });

// Server-style list: honor the shared filter params (q / owner / phase /
// poolName / purpose / mode) + limit/continue pagination, so mock filtering &
// "load more" behave like the real backend instead of returning everything.
type Item = Record<string, unknown>;
function matchesQuery(it: Item, q: URLSearchParams): boolean {
  const kw = (q.get("q") ?? "").toLowerCase();
  if (kw) {
    const hay = [it.name, it.identifier, it.displayName, it.description, it.image]
      .filter((s): s is string => typeof s === "string")
      .join(" ")
      .toLowerCase();
    if (!hay.includes(kw)) return false;
  }
  for (const [param, field] of [
    ["owner", "owner"],
    ["phase", "phase"],
    ["poolName", "poolName"],
    ["mode", "mode"],
  ] as const) {
    const v = q.get(param);
    if (v && it[field] !== v) return false;
  }
  const purpose = q.get("purpose");
  if (purpose && (it.spec as Item | undefined)?.purpose !== purpose) return false;
  return true;
}
function paged(items: unknown[], q: URLSearchParams): Result {
  const filtered = (items as Item[]).filter((it) => matchesQuery(it, q));
  const start = Number(q.get("continue")) || 0;
  const limit = Number(q.get("limit")) || 50;
  const page = filtered.slice(start, start + limit);
  const next = start + limit;
  return {
    body: {
      count: page.length,
      items: page,
      continueToken: next < filtered.length ? String(next) : undefined,
    },
  };
}

// ── auth ─────────────────────────────────────────────────────────────────────
on("POST", "/auth/login", () => ok({ jwt: "mock.jwt.token", expiresAt: db.ago(-24) }));
on("POST", "/auth/logout", () => noContent());
on("POST", "/auth/refresh", () => ok({ jwt: "mock.jwt.token", expiresAt: db.ago(-24) }));
on("GET", "/auth/me", () => ok(db.me));

// ── jobs + runs ───────────────────────────────────────────────────────────────
on("GET", "/jobs", (_p, q) => paged(db.jobs, q));
on("POST", "/jobs", (_p, _q, body) => created({ ...db.jobs[0], ...(body as object) }));
on("GET", "/jobs/:name", (p) => ok(db.jobs.find((j) => j.name === p.name) ?? db.jobs[0]));
on("PATCH", "/jobs/:name", (p, _q, body) => ok({ ...(db.jobs.find((j) => j.name === p.name) ?? db.jobs[0]), ...(body as object) }));
on("DELETE", "/jobs/:name", () => noContent());
on("GET", "/jobs/:name/runs", (p) => list(db.runsFor(p.name, "job")));
on("POST", "/jobs/:name/runs", (p) => created(db.runsFor(p.name, "job")[0]));
on("GET", "/jobs/:name/runs/:run", (p) => ok(db.runsFor(p.name, "job").find((r) => r.name === p.run) ?? db.runsFor(p.name, "job")[0]));
on("DELETE", "/jobs/:name/runs/:run", () => noContent());
on("POST", "/jobs/:name/runs/:run/cancel", (p) => ok(db.runsFor(p.name, "job")[0]));
on("GET", "/jobs/:name/runs/:run/events", (p) => list(db.eventsFor(p.run)));
on("GET", "/jobs/:name/runs/:run/metrics", (_p, q) => ok(db.metricSeries(q.get("metric") ?? "gpu_util")));
on("GET", "/jobs/:name/runs/:run/pods", (p) => list(db.podsFor(p.run)));
on("GET", "/jobs/:name/runs/:run/pods/:pod/events", (p) => list(db.eventsFor(p.pod)));
on("GET", "/jobs/:name/runs/:run/pods/:pod/logs", (p) => ({ body: db.podLogs(p.pod) }));

// ── experiments + runs (mirror jobs) ────────────────────────────────────────────
on("GET", "/experiments", (_p, q) => paged(db.experiments, q));
on("POST", "/experiments", (_p, _q, body) => created({ ...db.experiments[0], ...(body as object) }));
on("GET", "/experiments/:name", (p) => ok(db.experiments.find((e) => e.name === p.name) ?? db.experiments[0]));
on("PATCH", "/experiments/:name", (p, _q, body) => ok({ ...(db.experiments.find((e) => e.name === p.name) ?? db.experiments[0]), ...(body as object) }));
on("DELETE", "/experiments/:name", () => noContent());
on("GET", "/experiments/:name/runs", (p) => list(db.runsFor(p.name, "experiment")));
on("POST", "/experiments/:name/runs", (p) => created(db.runsFor(p.name, "experiment")[0]));
on("GET", "/experiments/:name/runs/:run", (p) => ok(db.runsFor(p.name, "experiment").find((r) => r.name === p.run) ?? db.runsFor(p.name, "experiment")[0]));
on("DELETE", "/experiments/:name/runs/:run", () => noContent());
on("POST", "/experiments/:name/runs/:run/cancel", (p) => ok(db.runsFor(p.name, "experiment")[0]));
on("GET", "/experiments/:name/runs/:run/events", (p) => list(db.eventsFor(p.run)));
on("GET", "/experiments/:name/runs/:run/metrics", (_p, q) => ok(db.metricSeries(q.get("metric") ?? "gpu_util")));
on("GET", "/experiments/:name/runs/:run/pods", (p) => list(db.podsFor(p.run)));
on("GET", "/experiments/:name/runs/:run/pods/:pod/events", (p) => list(db.eventsFor(p.pod)));
on("GET", "/experiments/:name/runs/:run/pods/:pod/logs", (p) => ({ body: db.podLogs(p.pod) }));
on("GET", "/experiments/:name/tensorboard", (p) => ok({ name: `tb-${p.name}`, phase: "Ready", url: `https://tb-${p.name}.llm-lab.axisml.io`, createdAt: db.ago(2) }));
on("POST", "/experiments/:name/tensorboard", (p) => ok({ name: `tb-${p.name}`, phase: "Ready", url: `https://tb-${p.name}.llm-lab.axisml.io`, createdAt: db.now() }));
on("DELETE", "/experiments/:name/tensorboard", () => noContent());

// ── workspaces ──────────────────────────────────────────────────────────────────
on("GET", "/workspaces", (_p, q) => paged(db.workspaces, q));
on("POST", "/workspaces", (_p, _q, body) => created({ ...db.workspaces[0], ...(body as object) }));
on("GET", "/workspaces/:name", (p) => ok(db.workspaces.find((w) => w.name === p.name) ?? db.workspaces[0]));
on("PATCH", "/workspaces/:name", (p, _q, body) => ok({ ...(db.workspaces.find((w) => w.name === p.name) ?? db.workspaces[0]), ...(body as object) }));
on("DELETE", "/workspaces/:name", () => noContent());
on("POST", "/workspaces/:name/start", (p) => ok({ ...(db.workspaces.find((w) => w.name === p.name) ?? db.workspaces[0]), phase: "Starting" }));
on("POST", "/workspaces/:name/stop", (p) => ok({ ...(db.workspaces.find((w) => w.name === p.name) ?? db.workspaces[0]), phase: "Stopped" }));
on("GET", "/workspaces/:name/events", (p) => list(db.eventsFor(p.name)));
on("GET", "/workspaces/:name/pods", (p) => list(db.podsFor(p.name).slice(0, 1)));
on("GET", "/workspaces/:name/pods/:pod/events", (p) => list(db.eventsFor(p.pod)));
on("GET", "/workspaces/:name/pods/:pod/logs", (p) => ({ body: db.podLogs(p.pod) }));
on("GET", "/workspace-images", () => list(db.workspaceImages()));

// ── services ────────────────────────────────────────────────────────────────────
on("GET", "/mlservices", (_p, q) => paged(db.services, q));
on("POST", "/mlservices", (_p, _q, body) => created({ ...db.services[0], ...(body as object) }));
on("GET", "/mlservices/:name", (p) => ok(db.services.find((s) => s.name === p.name) ?? db.services[0]));
on("PATCH", "/mlservices/:name", (p, _q, body) => ok({ ...(db.services.find((s) => s.name === p.name) ?? db.services[0]), ...(body as object) }));
on("DELETE", "/mlservices/:name", () => noContent());
on("POST", "/mlservices/:name/scale", (p, _q, body) => ok({ ...(db.services.find((s) => s.name === p.name) ?? db.services[0]), ...(body as object) }));
on("POST", "/mlservices/:name/start", (p) => ok({ ...(db.services.find((s) => s.name === p.name) ?? db.services[0]), phase: "Ready" }));
on("POST", "/mlservices/:name/stop", (p) => ok({ ...(db.services.find((s) => s.name === p.name) ?? db.services[0]), phase: "Stopped" }));
on("GET", "/mlservices/:name/events", (p) => list(db.eventsFor(p.name)));
on("GET", "/mlservices/:name/metrics", (_p, q) => ok(db.metricSeries(q.get("metric") ?? "request_rate")));
on("GET", "/mlservices/:name/pods", (p) => list(db.podsFor(p.name).slice(0, 2)));
on("GET", "/mlservices/:name/pods/:pod/events", (p) => list(db.eventsFor(p.pod)));
on("GET", "/mlservices/:name/pods/:pod/logs", (p) => ({ body: db.podLogs(p.pod) }));

// ── traffic policies ──────────────────────────────────────────────────────────
on("GET", "/trafficpolicies", (_p, q) => paged(db.trafficPolicies, q));
on("POST", "/trafficpolicies", (_p, _q, body) => created({ ...db.trafficPolicies[0], ...(body as object) }));
on("GET", "/trafficpolicies/:name", (p) => ok(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]));
on("PATCH", "/trafficpolicies/:name", (p, _q, body) => ok({ ...(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]), ...(body as object) }));
on("DELETE", "/trafficpolicies/:name", () => noContent());
on("GET", "/trafficpolicies/:name/events", (p) => list(db.eventsFor(p.name)));
on("GET", "/trafficpolicies/:name/metrics", (_p, q) => ok(db.metricSeries(q.get("metric") ?? "request_rate")));
on("POST", "/trafficpolicies/:name/promote", (p) => ok(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]));
on("POST", "/trafficpolicies/:name/rollback", (p) => ok(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]));
on("POST", "/trafficpolicies/:name/split", (p, _q, body) => ok({ ...(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]), ...(body as object) }));

// ── models ──────────────────────────────────────────────────────────────────────
on("GET", "/models", (_p, q) => paged(db.models, q));
on("GET", "/models/:tenant/:name", (p) => ok(db.models.find((m) => m.name === p.name) ?? db.models[0]));
on("POST", "/models/:tenant/:name", (_p, _q, body) => created({ ...db.models[0], ...(body as object) }));
on("PATCH", "/models/:tenant/:name", (p, _q, body) => ok({ ...(db.models.find((m) => m.name === p.name) ?? db.models[0]), ...(body as object) }));
on("DELETE", "/models/:tenant/:name", () => noContent());
on("GET", "/models/:tenant/:name/versions", (p) => list(db.modelVersions(p.name)));
on("POST", "/models/:tenant/:name/versions", (p, _q, body) => created({ id: `${p.name}-init`, uri: `s3://axisml-models/${p.tenant}/${p.name}`, storageKind: "s3", uploadCredentials: {}, ...(body as object) }));
on("GET", "/models/:tenant/:name/versions/:version", (p) => ok(db.modelVersions(p.name).find((v) => v.version === p.version) ?? db.modelVersions(p.name)[0]));
on("DELETE", "/models/:tenant/:name/versions/:version", () => noContent());
on("POST", "/models/:tenant/:name/versions/:version/complete", (p) => ok(db.modelVersions(p.name)[0]));
on("GET", "/models/:tenant/:name/versions/:version/resolve", (p) => ok({
  storageKind: "oci",
  uri: `zot.axisml.internal/${p.tenant}/${p.name}:${p.version}`,
  digest: "sha256:9b0d5a2c7f3148e1f4a6c8e3d2b4a6c8e1f9b0d5a2c7f3148e1f4a6c8e3d2b4a",
  pullCredentials: { username: "pull-token", password: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.mock" },
  expiresAt: db.ago(-1),
}));

// ── images ────────────────────────────────────────────────────────────────────
on("GET", "/images", (_p, q) => paged(db.images, q));
on("GET", "/images/:tenant/:name", (p) => ok(db.images.find((m) => m.name === p.name) ?? db.images[0]));
on("POST", "/images/:tenant/:name", (_p, _q, body) => created({ ...db.images[0], ...(body as object) }));
on("PATCH", "/images/:tenant/:name", (p, _q, body) => ok({ ...(db.images.find((m) => m.name === p.name) ?? db.images[0]), ...(body as object) }));
on("DELETE", "/images/:tenant/:name", () => noContent());
on("GET", "/images/:tenant/:name/versions", (p) => list(db.imageVersions(p.name)));
on("POST", "/images/:tenant/:name/versions", (p, _q, body) => created({ id: `${p.name}-init`, uri: `harbor.axisml.io/${p.tenant}/${p.name}`, storageKind: "oci", uploadCredentials: {}, ...(body as object) }));
on("GET", "/images/:tenant/:name/versions/:version", (p) => ok(db.imageVersions(p.name).find((v) => v.version === p.version) ?? db.imageVersions(p.name)[0]));
on("DELETE", "/images/:tenant/:name/versions/:version", () => noContent());
on("POST", "/images/:tenant/:name/versions/:version/complete", (p) => ok(db.imageVersions(p.name)[0]));
on("GET", "/images/:tenant/:name/versions/:version/resolve", (p) => ok({
  storageKind: "oci",
  uri: `zot.axisml.internal/${p.tenant}/${p.name}:${p.version}`,
  digest: "sha256:a1b2c3d4e5f60718293a4b5c6d7e8f9012a3b4c5d6e7f8091a2b3c4d5e6f70819",
  pullCredentials: { username: "pull-token", password: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.mock" },
  expiresAt: db.ago(-1),
}));

// ── resource pools + units ──────────────────────────────────────────────────────
on("GET", "/resourcepools", () => list(db.pools));
on("POST", "/resourcepools", (_p, _q, body) => created({ ...db.pools[0], ...(body as object) }));
on("GET", "/resourcepools/:pool", (p) => ok(db.pools.find((x) => x.name === p.pool) ?? db.pools[0]));
on("PATCH", "/resourcepools/:pool", (p, _q, body) => ok({ ...(db.pools.find((x) => x.name === p.pool) ?? db.pools[0]), ...(body as object) }));
on("DELETE", "/resourcepools/:pool", () => noContent());
on("GET", "/resourcepools/:pool/units", (p) => list(db.pools.find((x) => x.name === p.pool)?.units ?? []));
on("POST", "/resourcepools/:pool/units", (_p, _q, body) => created(body));
on("GET", "/resourcepools/:pool/units/:unit", (p) => ok((db.pools.find((x) => x.name === p.pool)?.units ?? []).find((u) => u.name === p.unit) ?? {}));
on("PATCH", "/resourcepools/:pool/units/:unit", (_p, _q, body) => ok(body));
on("DELETE", "/resourcepools/:pool/units/:unit", () => noContent());

// ── data volumes ────────────────────────────────────────────────────────────────
on("GET", "/datavolumes", () => list(db.dataVolumes));
on("POST", "/datavolumes", (_p, _q, body) => created({ ...db.dataVolumes[0], ...(body as object) }));
on("GET", "/datavolumes/:name", (p) => ok(db.dataVolumes.find((x) => x.name === p.name) ?? db.dataVolumes[0]));
on("PATCH", "/datavolumes/:name", (p, _q, body) => ok({ ...(db.dataVolumes.find((x) => x.name === p.name) ?? db.dataVolumes[0]), ...(body as object) }));
on("DELETE", "/datavolumes/:name", () => noContent());
on("GET", "/storageclasses", () => list(db.storageClasses));

// ── users (directory for the add-member typeahead) ──────────────────────────────
on("GET", "/users", (_p, q) => {
  const needle = (q.get("q") ?? "").toLowerCase();
  const items = db.users.filter(
    (u) =>
      !needle ||
      u.username.toLowerCase().includes(needle) ||
      (u.email ?? "").toLowerCase().includes(needle) ||
      (u.displayName ?? "").toLowerCase().includes(needle),
  );
  return list(items);
});

// ── tenants, members, quotas ────────────────────────────────────────────────────
on("GET", "/tenants", (_p, q) => paged(db.tenants, q));
on("POST", "/tenants", (_p, _q, body) => created({ ...db.tenants[0], ...(body as object) }));
on("GET", "/tenants/:name", (p) => ok(db.tenants.find((t) => t.identifier === p.name) ?? db.tenants[0]));
on("PATCH", "/tenants/:name", (p, _q, body) => ok({ ...(db.tenants.find((t) => t.identifier === p.name) ?? db.tenants[0]), ...(body as object) }));
on("DELETE", "/tenants/:name", () => noContent());
on("POST", "/tenants/:name/suspend", (p) => ok({ ...(db.tenants.find((t) => t.identifier === p.name) ?? db.tenants[0]), suspended: true, phase: "Suspended" }));
on("POST", "/tenants/:name/resume", (p) => ok({ ...(db.tenants.find((t) => t.identifier === p.name) ?? db.tenants[0]), suspended: false, phase: "Active" }));
on("GET", "/tenants/:name/members", (p) => list(db.membersByTenant[p.name] ?? []));
on("POST", "/tenants/:name/members", (_p, _q, body) => created(body));
on("PATCH", "/tenants/:name/members/:userId", (_p, _q, body) => ok(body));
on("DELETE", "/tenants/:name/members/:userId", () => noContent());
on("GET", "/tenants/:name/quotas", (p) => ({ body: { count: (db.quotasByTenant[p.name] ?? []).length, items: db.quotasByTenant[p.name] ?? [] } }));
on("POST", "/tenants/:name/quotas", (_p, _q, body) => created(body));
on("PATCH", "/tenants/:name/quotas/:pool", (_p, _q, body) => ok(body));
on("DELETE", "/tenants/:name/quotas/:pool", () => noContent());

// ── dashboard aggregates ──────────────────────────────────────────────────────
on("GET", "/dashboard/cluster-usage", (_p, q) => ok(db.clusterUsage(q.get("pool") ?? undefined)));
on("GET", "/dashboard/cluster-metrics", (_p, q) => ok(db.clusterMetric(q.get("metric") ?? "gpu_util")));
on("GET", "/dashboard/activity", () => list(db.activityFeed()));

// ── health ──────────────────────────────────────────────────────────────────────
on("GET", "/healthz", () => ok({ status: "ok" }));
on("GET", "/readyz", () => ok({ status: "ok" }));

// ── matcher ─────────────────────────────────────────────────────────────────────
function match(routePattern: string, path: string): Record<string, string> | null {
  const rp = routePattern.split("/");
  const pp = path.split("/");
  if (rp.length !== pp.length) return null;
  const params: Record<string, string> = {};
  for (let i = 0; i < rp.length; i++) {
    if (rp[i].startsWith(":")) params[rp[i].slice(1)] = decodeURIComponent(pp[i]);
    else if (rp[i] !== pp[i]) return null;
  }
  return params;
}

export function route(method: string, path: string, query: URLSearchParams, body: unknown): Result {
  for (const r of routes) {
    if (r.method !== method) continue;
    const params = match(r.pattern, path);
    if (params) return r.handler(params, query, body);
  }
  // Unknown route: stay safe rather than hang. GETs look like an empty list;
  // writes succeed with an empty body.
  if (method === "GET") return { body: { count: 0, items: [] } };
  return { body: {} };
}
