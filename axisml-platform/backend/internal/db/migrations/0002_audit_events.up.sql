CREATE TABLE IF NOT EXISTS audit_events (
  id         uuid PRIMARY KEY,
  tenant     text NOT NULL,                 -- tenant identifier scope (= Tenant CR name); '' for global/admin actions
  kind       text NOT NULL,                 -- subject kind: job, run, service, workspace, trafficpolicy, tenant, quota, member, model, image, ...
  name       text NOT NULL,                 -- subject resource name
  action     text NOT NULL,                 -- created, updated, deleted, scaled, stopped, started, canceled, split, promoted, ...
  actor      text NOT NULL DEFAULT '',      -- username that triggered the mutation (from the request identity)
  phase      text NOT NULL DEFAULT '',      -- subject phase at record time, when known
  created_at timestamptz NOT NULL DEFAULT now()
);

-- Activity feed reads are per-tenant, newest first.
CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_created ON audit_events (tenant, created_at DESC);
