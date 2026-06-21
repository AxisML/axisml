import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTenants } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { FieldsetTitle } from "@/components/forms";
import { TableState, errorText } from "@/components/states";

// Page-scoped styles ported verbatim from prototype/tenants.html <style>: the
// multi-tab quota editor (one tab per resource pool → resource-unit cards ×
// quantity). Kept here because they are tenant-page-specific, not part of the
// shared design system in app.css.
const QUOTA_STYLES = `
.ptab-nav { display:flex; align-items:center; gap:var(--space-5); border-bottom:1px solid var(--border); margin-bottom:var(--space-5); overflow-x:auto; }
.ptab { flex:none; background:transparent; border:0; padding:var(--space-3) 0; font-size:14px; font-family:var(--font-mono); font-weight:600; color:var(--muted); cursor:pointer; position:relative; white-space:nowrap; transition:color var(--motion-fast) var(--ease-standard); }
.ptab:hover { color:var(--fg-2); }
.ptab.on { color:var(--fg); }
.ptab.on::after { content:""; position:absolute; left:0; right:0; bottom:-1px; height:2px; background:var(--accent); border-radius:2px; }
.ptab-pane { display:none; }
.ptab-pane.on { display:block; animation:fadein var(--motion-base) var(--ease-standard); }
.ptab-meta { font-size:12px; color:var(--muted); margin-bottom:var(--space-4); }
.qp-units { display:flex; flex-direction:column; gap:var(--space-3); }
.q-row { display:flex; align-items:center; gap:var(--space-4); }
.q-card { flex:1; min-width:0; padding:10px 14px; border:1px solid var(--border); border-radius:var(--radius-md); background:var(--bg); transition:border-color var(--motion-fast) var(--ease-standard), background var(--motion-fast) var(--ease-standard); }
.q-card:hover { border-color:color-mix(in oklab, var(--accent) 35%, var(--border)); }
.q-row.is-zero .q-card { background:var(--surface); border-color:var(--border-soft); }
.q-row.is-zero .q-card:hover { border-color:var(--border); }
.q-row.is-zero .uc-name, .q-row.is-zero .uc-spec { color:var(--muted); }
.qu-qty { display:flex; align-items:center; gap:var(--space-3); flex:none; }
.qu-qty::before { content:"×"; font-size:16px; color:var(--muted); line-height:1; }
.step-val { width:56px; height:32px; text-align:center; border:1px solid var(--border); border-radius:var(--radius-sm); background:var(--bg); font:inherit; font-size:13px; font-family:var(--font-mono); color:var(--fg); transition:border-color var(--motion-fast) var(--ease-standard), box-shadow var(--motion-fast) var(--ease-standard); -moz-appearance:textfield; }
.step-val::-webkit-outer-spin-button, .step-val::-webkit-inner-spin-button { -webkit-appearance:none; margin:0; }
.step-val:focus { outline:none; border-color:var(--accent); box-shadow:var(--focus-ring); }
@media (max-width:720px){
  .q-row { flex-wrap:wrap; }
  .qu-qty { margin-left:auto; }
}
/* list-row resource quota: one line per pool (pool + 用量 / 水位) */
.q-meters { display:flex; flex-direction:column; gap:8px; }
.qm-row { display:grid; grid-template-columns:92px 60px 1fr; align-items:center; gap:10px; }
.qm-num { font-family:var(--font-mono); font-size:12px; color:var(--fg-2); text-align:right; white-space:nowrap; }
`;

interface PoolQuota {
  pool: string;
  allocated: number; // sum of unit quantities granted in this pool
}
interface TenantRow {
  ident: string;
  display: string;
  active: boolean;
  pools: PoolQuota[];
  members: number;
  activeTasks: number; // active job runs + active experiment runs
  services: number; // online services
  created: string;
}

type DrawerKind =
  | { kind: "tenant" }
  | { kind: "quota"; ident: string; display: string }
  | { kind: "members"; ident: string; display: string }
  | { kind: "member"; ident: string };

export default function Tenants() {
  const q = useTenants();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<DrawerKind | null>(null);

  const delTenant = useApiMutation(
    (name: string) => sdk.deleteTenant({ path: { name } }),
    { invalidate: [["tenants"]], success: "租户已删除" },
  );
  const suspend = useApiMutation(
    (name: string) => sdk.suspendTenant({ path: { name } }),
    { invalidate: [["tenants"]], success: "租户已禁用" },
  );
  const resume = useApiMutation(
    (name: string) => sdk.resumeTenant({ path: { name } }),
    { invalidate: [["tenants"]], success: "租户已启用" },
  );

  const rows: TenantRow[] =
    q.data?.items?.map((t) => ({
      ident: t.identifier,
      display: t.displayName,
      active: !t.suspended,
      // listTenants?stats=true enriches each row with live roll-ups. The quota
      // payload carries the allocated unit total per pool; live usage (水位) has
      // no metrics source yet, so the meter renders against allocated with 0 used.
      pools: (t.quotas ?? []).map((quota) => ({
        pool: quota.pool,
        allocated: (quota.units ?? []).reduce((sum, u) => sum + (u.quantity ?? 0), 0),
      })),
      members: t.memberCount ?? 0,
      activeTasks: (t.activeJobRuns ?? 0) + (t.activeExperimentRuns ?? 0),
      services: t.onlineServices ?? 0,
      created: t.createdAt,
    })) ?? [];

  return (
    <main className="page">
      <style dangerouslySetInnerHTML={{ __html: QUOTA_STYLES }} />
      <div className="breadcrumb">
        <span>系统管理</span>
        <span className="sep">/</span>
        <span>租户管理</span>
      </div>
      <div className="page-head">
        <div>
          <h1>租户管理</h1>
          <p className="sub">
            支持多租户体系下的组织、用户与权限管理，满足不同团队的隔离与协作需求。通过精细化权限控制，提升平台运营与资源治理效率。
          </p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setDrawer({ kind: "tenant" })}>
            <Icon name="plus" />
            创建租户
          </button>
        </div>
      </div>

      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="租户名搜索" />
        </div>
        <select className="select">
          <option>状态：全部</option>
          <option>Active</option>
          <option>已禁用</option>
        </select>
        <button className="btn btn-ghost">重置</button>
      </div>

      <div className="panel">
        <div className="table-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>租户</th>
                <th>状态</th>
                <th style={{ width: 300 }}>资源配额（用量 / 水位）</th>
                <th className="num-col">成员</th>
                <th className="num-col">活跃任务</th>
                <th className="num-col">在线服务</th>
                <th>创建时间</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.ident}>
                  <td>
                    <span className="t-name mono">{r.ident}</span>
                    <div className="t-sub">{r.display}</div>
                  </td>
                  <td>
                    {r.active ? (
                      <span className="status status-running">
                        <span className="dot" />
                        已激活
                      </span>
                    ) : (
                      <span className="status status-stopped">
                        <span className="dot" />
                        已禁用
                      </span>
                    )}
                  </td>
                  <td>
                    {r.pools.length === 0 ? (
                      <span className="muted">—</span>
                    ) : (
                      <div className="q-meters">
                        {r.pools.map((p) => (
                          <div className="qm-row" key={p.pool}>
                            <span className="tag mono">{p.pool}</span>
                            <span className="qm-num">0 / {p.allocated}</span>
                            <div className="qbar">
                              <span className="used" style={{ width: "0%" }} />
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </td>
                  <td className="num-col">{r.members}</td>
                  <td className="num-col">{r.activeTasks}</td>
                  <td className="num-col">{r.services}</td>
                  <td className="muted">{r.created}</td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="act"
                        title="资源配额"
                        aria-label="资源配额"
                        onClick={() => setDrawer({ kind: "quota", ident: r.ident, display: r.display })}
                      >
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
                          <path d="M12 2 2 7l10 5 10-5-10-5z" />
                          <path d="m2 17 10 5 10-5" />
                          <path d="m2 12 10 5 10-5" />
                        </svg>
                      </button>
                      <button
                        className="act"
                        title="成员管理"
                        aria-label="成员管理"
                        onClick={() => setDrawer({ kind: "members", ident: r.ident, display: r.display })}
                      >
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
                          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                          <circle cx="9" cy="7" r="4" />
                          <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                          <path d="M16 3.13a4 4 0 0 1 0 7.75" />
                        </svg>
                      </button>
                      {r.active ? (
                        <button
                          className="act"
                          title="禁用"
                          aria-label="禁用"
                          onClick={() =>
                            confirm({
                              title: `禁用租户 ${r.ident}？`,
                              desc: "禁用后该租户成员将无法登录控制台或提交新任务，运行中的任务会被暂停。",
                              info: "如需彻底清理，可在禁用后再执行删除。",
                              okLabel: "确认禁用",
                              onConfirm: () => suspend.mutate(r.ident),
                            })
                          }
                        >
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <circle cx="12" cy="12" r="9" />
                            <path d="M5.6 5.6l12.8 12.8" />
                          </svg>
                        </button>
                      ) : (
                        <>
                          <button
                            className="act"
                            title="启用"
                            aria-label="启用"
                            onClick={() => resume.mutate(r.ident)}
                          >
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                              <path d="M6 4l14 8-14 8z" />
                            </svg>
                          </button>
                          <button
                            className="act act-danger"
                            title="删除"
                            aria-label="删除"
                            onClick={() =>
                              confirm({
                                title: `删除租户 ${r.ident}？`,
                                desc: "将删除该租户的全部资源与配额，该操作不可恢复。",
                                info: "建议先确认租户内已无活跃任务 / 服务 / 工作区。",
                                okLabel: "确认删除租户",
                                danger: true,
                                onConfirm: () => delTenant.mutate(r.ident),
                              })
                            }
                          >
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                              <path d="M3 6h18" />
                              <path d="M8 6V4h8v2" />
                              <path d="M19 6l-1 14H6L5 6" />
                              <path d="M10 11v6M14 11v6" />
                            </svg>
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
              <TableState q={q} cols={8} isEmpty={rows.length === 0} />
            </tbody>
          </table>
        </div>
        <div className="pagination">
          <span>共 {rows.length} 个租户</span>
          <div className="pages">
            <span className="pg">‹</span>
            <span className="pg on">1</span>
            <span className="pg">›</span>
          </div>
          <span>每页 20 条</span>
        </div>
      </div>

      {drawer?.kind === "tenant" && <TenantDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "quota" && (
        <QuotaDrawer ident={drawer.ident} display={drawer.display} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "members" && (
        <MembersDrawer
          ident={drawer.ident}
          display={drawer.display}
          onAddMember={() => setDrawer({ kind: "member", ident: drawer.ident })}
          onClose={() => setDrawer(null)}
        />
      )}
      {drawer?.kind === "member" && <MemberDrawer ident={drawer.ident} onClose={() => setDrawer(null)} />}
    </main>
  );
}

// ── Create-tenant drawer (controlled form → createTenant) ─────────────────────
function TenantDrawer({ onClose }: { onClose: () => void }) {
  const [displayName, setDisplayName] = useState("");
  const [identifier, setIdentifier] = useState("");
  const [admin, setAdmin] = useState("");

  const create = useApiMutation(
    (body: sdk.TenantCreateRequest) => sdk.createTenant({ body }),
    { invalidate: [["tenants"]], success: "租户已创建，正在初始化基础资源…" },
  );

  const valid = displayName.trim() && identifier.trim() && admin.trim();

  const submit = () => {
    const ident = identifier.trim();
    create.mutate(
      {
        displayName: displayName.trim(),
        identifier: ident,
        initialAdmin: admin.trim(),
        // The tenant identifier is a dns1123-valid slug; reuse it as the
        // physical namespace (tenant name = namespace convention).
        kubernetesNamespace: ident,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="创建租户"
      sub="提交后平台自动完成资源初始化与权限配置"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" disabled={!valid || create.isPending} onClick={submit}>
            {create.isPending ? "创建中…" : "创建租户"}
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>
            租户名称 <span className="req">*</span>
          </label>
          <input
            className="input"
            placeholder="大模型研究院"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
          <span className="help">用于在列表与详情中展示</span>
        </div>
        <div className="field">
          <label>
            租户标识 <span className="req">*</span>
          </label>
          <input
            className="input mono"
            placeholder="llm-lab"
            value={identifier}
            onChange={(e) => setIdentifier(e.target.value)}
          />
          <span className="help">小写字母、数字、连字符（同时作为 Kubernetes 命名空间）</span>
        </div>
        <div className="field full">
          <label>
            初始管理员 <span className="req">*</span>
          </label>
          <input
            className="input"
            placeholder="zhangwei@corp.com"
            value={admin}
            onChange={(e) => setAdmin(e.target.value)}
          />
        </div>
      </div>

      <FieldsetTitle n={2}>初始配额</FieldsetTitle>
      <div className="field">
        <span className="help">
          租户创建后，可在列表的「资源配额」操作中按资源池逐项分配资源单元数量。
        </span>
      </div>
    </Drawer>
  );
}

// ── Quota editor drawer (live quotas → create / update / delete) ──────────────
function QuotaDrawer({ ident, display, onClose }: { ident: string; display: string; onClose: () => void }) {
  const { confirm } = useUI();
  const quotasQ = useQuery({
    queryKey: ["tenant-quotas", ident],
    queryFn: async () => {
      const { data, error } = await sdk.listTenantQuotas({ path: { name: ident } });
      if (error) throw new Error(errorText(error));
      return data;
    },
  });

  const updateQuota = useApiMutation(
    (arg: { pool: string; units: sdk.QuotaUnit[] }) =>
      sdk.updateTenantQuota({ path: { name: ident, pool: arg.pool }, body: { units: arg.units } }),
    { invalidate: [["tenant-quotas", ident], ["tenants"]], success: "配额已保存" },
  );
  const delQuota = useApiMutation(
    (pool: string) => sdk.deleteTenantQuota({ path: { name: ident, pool } }),
    { invalidate: [["tenant-quotas", ident], ["tenants"]], success: "配额已移除" },
  );

  const quotas = quotasQ.data?.items ?? [];

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title={<span className="mono">{ident}</span>}
      sub={`${display} · 资源配额`}
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            关闭
          </button>
        </>
      }
    >
      {quotasQ.isLoading && <p className="muted">加载中…</p>}
      {quotasQ.isError && <p className="muted">{errorText(quotasQ.error)}</p>}
      {!quotasQ.isLoading && !quotasQ.isError && quotas.length === 0 && (
        <p className="muted">该租户暂无资源配额。</p>
      )}
      {quotas.length > 0 && (
        <QuotaPoolTabs
          quotas={quotas}
          saving={updateQuota.isPending}
          onSave={(pool, units) => updateQuota.mutate({ pool, units })}
          onDelete={(pool) =>
            confirm({
              title: `移除资源池 ${pool} 的配额？`,
              desc: "移除后该租户将不再拥有此资源池的可用资源单元。",
              okLabel: "确认移除",
              danger: true,
              onConfirm: () => delQuota.mutate(pool),
            })
          }
        />
      )}
    </Drawer>
  );
}

// Tabbed quota editor over live Quota[]: one tab per pool, each unit editable.
function QuotaPoolTabs({
  quotas,
  saving,
  onSave,
  onDelete,
}: {
  quotas: sdk.Quota[];
  saving: boolean;
  onSave: (pool: string, units: sdk.QuotaUnit[]) => void;
  onDelete: (pool: string) => void;
}) {
  const [active, setActive] = useState(quotas[0]?.pool);
  return (
    <div className="pool-tabs">
      <div className="ptab-nav">
        {quotas.map((p) => (
          <button
            key={p.pool}
            type="button"
            className={"ptab" + (p.pool === active ? " on" : "")}
            onClick={() => setActive(p.pool)}
          >
            {p.pool}
          </button>
        ))}
      </div>
      <div className="ptab-panes">
        {quotas.map((p) => (
          <div key={p.pool} className={"ptab-pane" + (p.pool === active ? " on" : "")}>
            <QuotaPoolPane quota={p} saving={saving} onSave={onSave} onDelete={onDelete} />
          </div>
        ))}
      </div>
    </div>
  );
}

function QuotaPoolPane({
  quota,
  saving,
  onSave,
  onDelete,
}: {
  quota: sdk.Quota;
  saving: boolean;
  onSave: (pool: string, units: sdk.QuotaUnit[]) => void;
  onDelete: (pool: string) => void;
}) {
  const [units, setUnits] = useState<sdk.QuotaUnit[]>(() =>
    (quota.units ?? []).map((u) => ({ unitName: u.unitName, quantity: u.quantity })),
  );
  const setQty = (unitName: string, qty: number) =>
    setUnits((arr) => arr.map((u) => (u.unitName === unitName ? { ...u, quantity: qty } : u)));

  return (
    <>
      <div className="ptab-meta">{quota.pool}</div>
      <div className="qp-units">
        {units.map((u) => (
          <div className={"q-row" + (u.quantity === 0 ? " is-zero" : "")} key={u.unitName}>
            <div className="q-card">
              <div className="uc-name">{u.unitName}</div>
            </div>
            <label className="qu-qty">
              <input
                className="step-val"
                type="number"
                min="0"
                step="1"
                inputMode="numeric"
                value={u.quantity}
                aria-label="配额数量"
                onChange={(e) => setQty(u.unitName, Math.max(0, Number(e.target.value) || 0))}
              />
            </label>
          </div>
        ))}
        {units.length === 0 && <p className="muted">该资源池暂无资源单元。</p>}
      </div>
      <div className="row-actions" style={{ marginTop: "var(--space-5)" }}>
        <button
          className="btn btn-sm btn-primary"
          disabled={saving}
          onClick={() => onSave(quota.pool, units)}
        >
          {saving ? "保存中…" : "保存配额"}
        </button>
        <button className="btn btn-sm btn-danger" onClick={() => onDelete(quota.pool)}>
          移除资源池配额
        </button>
      </div>
    </>
  );
}

// ── Members drawer (live members → add / remove / update role) ────────────────
function MembersDrawer({
  ident,
  display,
  onAddMember,
  onClose,
}: {
  ident: string;
  display: string;
  onAddMember: () => void;
  onClose: () => void;
}) {
  const { confirm } = useUI();
  const membersQ = useQuery({
    queryKey: ["tenant-members", ident],
    queryFn: async () => {
      const { data, error } = await sdk.listTenantMembers({ path: { name: ident } });
      if (error) throw new Error(errorText(error));
      return data;
    },
  });

  const removeMember = useApiMutation(
    (userId: string) => sdk.removeTenantMember({ path: { name: ident, userId } }),
    { invalidate: [["tenant-members", ident]], success: "成员已移除" },
  );
  const updateRole = useApiMutation(
    (arg: { userId: string; roleName: "tenant-admin" | "user" }) =>
      sdk.updateTenantMember({ path: { name: ident, userId: arg.userId }, body: { roleName: arg.roleName } }),
    { invalidate: [["tenant-members", ident]], success: "成员角色已更新" },
  );

  const members = membersQ.data?.items ?? [];
  const roleLabel = (r: string) => (r === "tenant-admin" ? "租户管理员" : "普通用户");

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title={<span className="mono">{ident}</span>}
      sub={`${display} · 成员管理`}
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            关闭
          </button>
        </>
      }
    >
      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="成员搜索" />
        </div>
        <div className="grow" />
        <button className="btn btn-sm btn-primary" onClick={onAddMember}>
          + 添加成员
        </button>
      </div>
      <div className="panel">
        <div className="table-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>成员</th>
                <th>角色</th>
                <th>加入时间</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {members.map((m) => {
                const name = m.displayName || m.username || m.email || m.userId;
                const initial = (name || "?").trim().charAt(0).toUpperCase();
                const isAdmin = m.roleName === "tenant-admin";
                return (
                  <tr key={m.userId}>
                    <td>
                      <div className="row" style={{ gap: 10 }}>
                        <div className="avatar" style={{ width: 28, height: 28, fontSize: 12 }}>
                          {initial}
                        </div>
                        {name}
                      </div>
                    </td>
                    <td>
                      <select
                        className="input"
                        value={isAdmin ? "tenant-admin" : "user"}
                        disabled={updateRole.isPending}
                        aria-label="角色"
                        onChange={(e) =>
                          updateRole.mutate({
                            userId: m.userId,
                            roleName: e.target.value as "tenant-admin" | "user",
                          })
                        }
                      >
                        <option value="user">{roleLabel("user")}</option>
                        <option value="tenant-admin">{roleLabel("tenant-admin")}</option>
                      </select>
                    </td>
                    <td className="muted">{m.addedAt}</td>
                    <td>
                      <div className="row-actions">
                        <button
                          className="act act-danger"
                          title="移除成员"
                          aria-label="移除成员"
                          onClick={() =>
                            confirm({
                              title: `移除成员 ${name}？`,
                              desc: "移除后该成员将失去对此租户的访问权限。",
                              okLabel: "确认移除",
                              danger: true,
                              onConfirm: () => removeMember.mutate(m.userId),
                            })
                          }
                        >
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                            <circle cx="9" cy="7" r="4" />
                            <path d="M16 11h6" />
                          </svg>
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
              {membersQ.isLoading && (
                <tr>
                  <td colSpan={4} className="muted">
                    加载中…
                  </td>
                </tr>
              )}
              {membersQ.isError && (
                <tr>
                  <td colSpan={4} className="muted">
                    {errorText(membersQ.error)}
                  </td>
                </tr>
              )}
              {!membersQ.isLoading && !membersQ.isError && members.length === 0 && (
                <tr>
                  <td colSpan={4} className="muted">
                    暂无成员
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </Drawer>
  );
}

// ── Add-member drawer (controlled form → addTenantMember) ─────────────────────
function MemberDrawer({ ident, onClose }: { ident: string; onClose: () => void }) {
  const [account, setAccount] = useState("");
  const [roleName, setRoleName] = useState<"tenant-admin" | "user">("user");

  const add = useApiMutation(
    (body: sdk.MemberCreateRequest) => sdk.addTenantMember({ path: { name: ident }, body }),
    { invalidate: [["tenant-members", ident]], success: "成员已添加" },
  );

  const submit = () =>
    add.mutate({ account: account.trim(), roleName }, { onSuccess: onClose });

  return (
    <Drawer
      open
      onClose={onClose}
      title="添加成员"
      sub="为所选租户添加成员并分配角色"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            disabled={!account.trim() || add.isPending}
            onClick={submit}
          >
            {add.isPending ? "添加中…" : "添加成员"}
          </button>
        </>
      }
    >
      <div className="form-grid">
        <div className="field full">
          <label>
            成员账号 <span className="req">*</span>
          </label>
          <input
            className="input"
            placeholder="name@corp.com"
            value={account}
            onChange={(e) => setAccount(e.target.value)}
          />
          <span className="help">输入企业邮箱或账号，支持邀请平台已有用户</span>
        </div>
        <div className="field full">
          <label>
            角色 <span className="req">*</span>
          </label>
          <select
            className="input"
            value={roleName}
            onChange={(e) => setRoleName(e.target.value as "tenant-admin" | "user")}
          >
            <option value="user">普通用户</option>
            <option value="tenant-admin">租户管理员</option>
          </select>
        </div>
      </div>
    </Drawer>
  );
}
