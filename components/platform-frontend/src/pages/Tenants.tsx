import { useState } from "react";
import { useTenants } from "@/api/hooks";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { FieldsetTitle } from "@/components/forms";

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
`;

interface QuotaChip {
  text: string;
}
interface TenantRow {
  ident: string;
  display: string;
  active: boolean;
  chips: QuotaChip[];
  members: number | string; // "—" when the count isn't in the list payload (live data)
  created: string;
}

// Faithful demo rows from prototype/tenants.html — rendered when the backend
// (contract-only shell) returns no items.
const FALLBACK: TenantRow[] = [
  {
    ident: "llm-lab",
    display: "大模型研究院",
    active: true,
    chips: [{ text: "gpu-h100 14/16" }, { text: "cpu-large +1" }],
    members: 12,
    created: "2026-03-08",
  },
  {
    ident: "rec-algo",
    display: "推荐算法团队",
    active: true,
    chips: [{ text: "gpu-a100 6/12" }],
    members: 8,
    created: "2026-04-02",
  },
  {
    ident: "av-perception",
    display: "智能驾驶感知",
    active: true,
    chips: [{ text: "gpu-l40s 11/20" }, { text: "gpu-h100 +1" }],
    members: 15,
    created: "2026-04-19",
  },
  {
    ident: "risk-ai",
    display: "风控 AI",
    active: false,
    chips: [{ text: "cpu-large 0/8" }],
    members: 6,
    created: "2026-05-30",
  },
];

// 资源配额数据 — one entry per resource pool, each listing its resource units.
interface QuotaUnit {
  name: string;
  spec: string;
  qty: number;
}
interface QuotaPool {
  pool: string;
  meta: string;
  units: QuotaUnit[];
}

const CREATE_QUOTA: QuotaPool[] = [
  {
    pool: "gpu-h100",
    meta: "H100 训练/推理池",
    units: [
      { name: "h100-4x-xlarge", spec: "4×H100 · 48 vCPU · 384 GiB", qty: 2 },
      { name: "h100-8x-xlarge-ib", spec: "8×H100 · 96 vCPU · 768 GiB", qty: 1 },
      { name: "h100-1x-large", spec: "1×H100 · 12 vCPU · 96 GiB", qty: 0 },
    ],
  },
  {
    pool: "gpu-a100",
    meta: "A100 训练池",
    units: [
      { name: "a100-1x-large", spec: "1×A100 · 8 vCPU · 64 GiB", qty: 4 },
      { name: "a100-4x-xlarge", spec: "4×A100 · 32 vCPU · 256 GiB", qty: 2 },
      { name: "a100-8x-xlarge-ib", spec: "8×A100 · 64 vCPU · 512 GiB", qty: 0 },
    ],
  },
  {
    pool: "cpu-large",
    meta: "大内存 CPU 池",
    units: [
      { name: "cpu-large-1", spec: "64 vCPU · 512 GiB", qty: 4 },
      { name: "cpu-large-2", spec: "32 vCPU · 256 GiB", qty: 0 },
    ],
  },
];

const EDIT_QUOTA: QuotaPool[] = [
  {
    pool: "gpu-h100",
    meta: "H100 训练/推理池",
    units: [
      { name: "h100-8x-xlarge-ib", spec: "8×H100 · 96 vCPU · 768 GiB", qty: 2 },
      { name: "h100-4x-xlarge", spec: "4×H100 · 48 vCPU · 384 GiB", qty: 1 },
      { name: "h100-1x-large", spec: "1×H100 · 12 vCPU · 96 GiB", qty: 0 },
    ],
  },
  {
    pool: "gpu-a100",
    meta: "A100 训练池",
    units: [
      { name: "a100-1x-large", spec: "1×A100 · 8 vCPU · 64 GiB", qty: 4 },
      { name: "a100-4x-xlarge", spec: "4×A100 · 32 vCPU · 256 GiB", qty: 2 },
      { name: "a100-8x-xlarge-ib", spec: "8×A100 · 64 vCPU · 512 GiB", qty: 1 },
    ],
  },
  {
    pool: "cpu-large",
    meta: "大内存 CPU 池",
    units: [
      { name: "cpu-large-1", spec: "64 vCPU · 512 GiB", qty: 2 },
      { name: "cpu-large-2", spec: "32 vCPU · 256 GiB", qty: 0 },
    ],
  },
];

type DrawerKind =
  | { kind: "tenant" }
  | { kind: "quota"; ident: string; display: string }
  | { kind: "members"; ident: string; display: string }
  | { kind: "member" };

export default function Tenants() {
  const { data } = useTenants();
  const { confirm, toast } = useUI();
  const [drawer, setDrawer] = useState<DrawerKind | null>(null);

  const rows: TenantRow[] =
    data?.items?.map((t) => ({
      ident: t.identifier,
      display: t.displayName,
      active: !t.suspended,
      // List payload carries allocated quota (pool + unit quantities) but no
      // live usage and no member count — show the allocated total, and "—" for
      // members (resolved lazily via the members drawer / its own endpoint).
      chips: (t.quotas ?? []).map((q) => {
        const total = (q.units ?? []).reduce((sum, u) => sum + (u.quantity ?? 0), 0);
        return { text: total ? `${q.pool} ×${total}` : q.pool };
      }),
      members: "—",
      created: t.createdAt,
    })) ?? FALLBACK;

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
                <th>资源配额</th>
                <th className="num-col">成员</th>
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
                    <div className="chip-row">
                      {r.chips.map((c) => (
                        <span className="tag mono" key={c.text}>
                          {c.text}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="num-col">{r.members}</td>
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
                              toast: `租户 ${r.ident} 已禁用`,
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
                            onClick={() => toast(`租户 ${r.ident} 已启用`)}
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
                                toast: `租户 ${r.ident} 已删除`,
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
          onAddMember={() => setDrawer({ kind: "member" })}
          onClose={() => setDrawer(null)}
        />
      )}
      {drawer?.kind === "member" && <MemberDrawer onClose={() => setDrawer(null)} />}
    </main>
  );
}

// ── Multi-tab quota editor (reused by create + quota drawers) ─────────────────
function PoolTabs({ pools }: { pools: QuotaPool[] }) {
  const [active, setActive] = useState(pools[0]?.pool);
  return (
    <div className="pool-tabs">
      <div className="ptab-nav">
        {pools.map((p) => (
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
        {pools.map((p) => (
          <div key={p.pool} className={"ptab-pane" + (p.pool === active ? " on" : "")}>
            <div className="ptab-meta">{p.meta}</div>
            <div className="qp-units">
              {p.units.map((u) => (
                <QtyRow key={u.name} unit={u} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function QtyRow({ unit }: { unit: QuotaUnit }) {
  const [qty, setQty] = useState(unit.qty);
  return (
    <div className={"q-row" + (qty === 0 ? " is-zero" : "")}>
      <div className="q-card">
        <div className="uc-name">{unit.name}</div>
        <div className="uc-spec">{unit.spec}</div>
      </div>
      <label className="qu-qty">
        <input
          className="step-val"
          type="number"
          min="0"
          step="1"
          inputMode="numeric"
          value={qty}
          aria-label="配额数量"
          onChange={(e) => setQty(Math.max(0, Number(e.target.value) || 0))}
        />
      </label>
    </div>
  );
}

function TenantDrawer({ onClose }: { onClose: () => void }) {
  const { toast } = useUI();
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
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("租户创建中，正在初始化基础资源…");
              onClose();
            }}
          >
            创建租户
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
          <input className="input" placeholder="大模型研究院" />
          <span className="help">用于在列表与详情中展示</span>
        </div>
        <div className="field">
          <label>
            租户标识 <span className="req">*</span>
          </label>
          <input className="input mono" placeholder="llm-lab" />
          <span className="help">小写字母、数字、连字符</span>
        </div>
        <div className="field full">
          <label>
            初始管理员 <span className="req">*</span>
          </label>
          <input className="input" placeholder="zhangwei@corp.com" />
        </div>
      </div>

      <FieldsetTitle n={2}>初始配额</FieldsetTitle>
      <div className="field">
        <label>
          资源配额 <span className="req">*</span>
        </label>
        <span className="help" style={{ marginBottom: 10 }}>
          按资源池分配可用的资源单元数量，切换上方 Tab 维护不同资源池。
        </span>
        <PoolTabs pools={CREATE_QUOTA} />
      </div>
    </Drawer>
  );
}

function QuotaDrawer({ ident, display, onClose }: { ident: string; display: string; onClose: () => void }) {
  const { toast } = useUI();
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
            取消
          </button>
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("配额已保存");
              onClose();
            }}
          >
            保存
          </button>
        </>
      }
    >
      <PoolTabs pools={EDIT_QUOTA} />
    </Drawer>
  );
}

interface MemberRow {
  initial: string;
  name: string;
  role: string;
  badge: "accent" | "neutral";
  joined: string;
  bg?: string;
}
const MEMBERS: MemberRow[] = [
  { initial: "张", name: "张伟", role: "租户管理员", badge: "accent", joined: "2026-03-08" },
  { initial: "李", name: "李娜", role: "普通用户", badge: "neutral", joined: "2026-03-12", bg: "var(--fg-2)" },
  { initial: "陈", name: "陈曦", role: "普通用户", badge: "neutral", joined: "2026-04-01", bg: "var(--fg-2)" },
];

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
              {MEMBERS.map((m) => (
                <tr key={m.name}>
                  <td>
                    <div className="row" style={{ gap: 10 }}>
                      <div
                        className="avatar"
                        style={{ width: 28, height: 28, fontSize: 12, ...(m.bg ? { background: m.bg } : {}) }}
                      >
                        {m.initial}
                      </div>
                      {m.name}
                    </div>
                  </td>
                  <td>
                    <span className={"badge badge-" + m.badge}>{m.role}</span>
                  </td>
                  <td className="muted">{m.joined}</td>
                  <td>
                    <div className="row-actions">
                      <button className="act act-danger" title="移除成员" aria-label="移除成员">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                          <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                          <circle cx="9" cy="7" r="4" />
                          <path d="M16 11h6" />
                        </svg>
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </Drawer>
  );
}

function MemberDrawer({ onClose }: { onClose: () => void }) {
  const { toast } = useUI();
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
            onClick={() => {
              toast("成员邀请已发送");
              onClose();
            }}
          >
            添加成员
          </button>
        </>
      }
    >
      <div className="form-grid">
        <div className="field full">
          <label>
            成员账号 <span className="req">*</span>
          </label>
          <input className="input" placeholder="name@corp.com" />
          <span className="help">输入企业邮箱或账号，支持邀请平台已有用户</span>
        </div>
        <div className="field full">
          <label>
            角色 <span className="req">*</span>
          </label>
          <select className="input">
            <option>普通用户</option>
            <option>租户管理员</option>
          </select>
        </div>
      </div>
    </Drawer>
  );
}
