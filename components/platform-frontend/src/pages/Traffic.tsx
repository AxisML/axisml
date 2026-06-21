import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useTrafficPolicies, useServices } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { TableState } from "@/components/states";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { FieldsetTitle } from "@/components/forms";

// One-off "eye" detail glyph (not in the icon map).
function EyeIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

// 切流 / 调整权重 glyph — sliders.
function SplitIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3" />
      <path d="M1 14h6M9 8h6M17 16h6" />
    </svg>
  );
}

// 提升（promote）glyph — up arrow.
function PromoteIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M12 19V5M5 12l7-7 7 7" />
    </svg>
  );
}

// 回滚（rollback）glyph — counter-clockwise arrow.
function RollbackIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M3 12a9 9 0 1 0 3-6.7L3 8" />
      <path d="M3 3v5h5" />
    </svg>
  );
}

const TrashIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
    <path d="M3 6h18" />
    <path d="M8 6V4h8v2" />
    <path d="M19 6l-1 14H6L5 6" />
    <path d="M10 11v6M14 11v6" />
  </svg>
);

interface SplitBackend {
  nm: string;
  width: number;
  w: number;
  alt?: boolean;
}

type StatusKind = "pending" | "success" | "stopped";

interface TrafficRow {
  name: string;
  desc: string;
  mode: "weighted" | "canary";
  modeLabel: string;
  status: StatusKind;
  statusLabel: string;
  split?: SplitBackend[];
  splitText?: string;
  endpoint?: string;
}

export default function Traffic() {
  const q = useTrafficPolicies();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ kind: "create" } | { kind: "split"; row: TrafficRow } | null>(null);

  const del = useApiMutation(
    (name: string) => sdk.deleteTrafficPolicy({ path: { name } }),
    { invalidate: [["trafficpolicies"]], success: "流量策略已删除" },
  );
  const promote = useApiMutation(
    (name: string) => sdk.promoteTrafficPolicy({ path: { name } }),
    { invalidate: [["trafficpolicies"]], success: "已提升灰度后端为全量" },
  );
  const rollback = useApiMutation(
    (name: string) => sdk.rollbackTrafficPolicy({ path: { name } }),
    { invalidate: [["trafficpolicies"]], success: "已回滚至稳定后端" },
  );

  const rows: TrafficRow[] = q.data?.items?.map((p) => {
    const ready = p.phase === "Ready";
    const status: StatusKind = ready ? "success" : p.phase === "Failed" || p.phase === "Deleting" ? "stopped" : "pending";
    const statusLabel = ready ? "生效中" : status === "stopped" ? "未就绪" : "灰度中";
    return {
      name: p.name,
      desc: p.description ?? p.displayName ?? "",
      mode: p.mode,
      modeLabel: p.mode === "weighted" ? "加权" : "灰度",
      status,
      statusLabel,
      split: p.backends?.map((b, i) => ({
        nm: b.serviceName,
        width: b.actualPct ?? b.weight,
        w: b.weight,
        alt: i > 0,
      })),
      endpoint: p.accessUrl,
    };
  }) ?? [];

  const onDelete = (r: TrafficRow) => {
    confirm({
      title: `确定删除流量策略 ${r.name}？`,
      desc: "删除后对外入口将停止分发流量，且不可恢复。",
      okLabel: "确认删除",
      danger: true,
      onConfirm: () => del.mutate(r.name),
    });
  };

  const onPromote = (r: TrafficRow) => {
    confirm({
      title: `提升 ${r.name} 的灰度后端为全量？`,
      desc: "灰度后端将接管 100% 流量，稳定后端退出。",
      okLabel: "确认提升",
      danger: false,
      onConfirm: () => promote.mutate(r.name),
    });
  };

  const onRollback = (r: TrafficRow) => {
    confirm({
      title: `回滚 ${r.name} 至稳定后端？`,
      desc: "灰度后端将停止接收流量，稳定后端恢复全量。",
      okLabel: "确认回滚",
      danger: false,
      onConfirm: () => rollback.mutate(r.name),
    });
  };

  return (
    <main className="page">
      {/* Page-scoped styles ported verbatim from prototype/traffic.html <style>. */}
      <style>{`
        .mini-split { display:flex; flex-direction:column; gap:6px; min-width:200px; }
        .mini-split .ms { display:flex; align-items:center; gap:8px; font-size:12px; }
        .mini-split .ms .bar { flex:1; height:6px; border-radius:99px; background:var(--surface); overflow:hidden; }
        .mini-split .ms .bar > span { display:block; height:100%; background:var(--accent); }
        .mini-split .ms .bar.alt > span { background:var(--muted); }
        .mini-split .ms .w { font-family:var(--font-mono); width:30px; text-align:right; }
        .mini-split .ms .nm { font-family:var(--font-mono); width:96px; color:var(--fg-2); }
      `}</style>
      <div className="breadcrumb">
        <span>服务中心</span>
        <span className="sep">/</span>
        <span>流量配置</span>
      </div>
      <div className="page-head">
        <div>
          <h1>流量配置</h1>
          <p className="sub">
            为在线服务编排多版本流量：加权切分、灰度放量与蓝绿式全量切换。一个对外入口按权重分发到多个后端服务。
          </p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setDrawer({ kind: "create" })}>
            <Icon name="plus" />
            新建策略
          </button>
        </div>
      </div>

      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="名称搜索" />
        </div>
        <select className="select">
          <option>模式：全部</option>
          <option>加权</option>
          <option>灰度</option>
        </select>
        <select className="select">
          <option>状态：全部</option>
          <option>生效中</option>
          <option>灰度中</option>
          <option>未就绪</option>
        </select>
        <button className="btn btn-ghost">重置</button>
      </div>

      <div className="panel">
        <div className="table-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>名称</th>
                <th>模式</th>
                <th>状态</th>
                <th>后端（流量分布）</th>
                <th>访问地址</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.name}>
                  <td>
                    <Link className="t-name mono" to={`/traffic/${r.name}`}>
                      {r.name}
                    </Link>
                    <div className="t-sub">{r.desc}</div>
                  </td>
                  <td>
                    <span className="badge badge-neutral">{r.modeLabel}</span>
                  </td>
                  <td>
                    <span className={`status status-${r.status}`}>
                      <span className="dot" />
                      {r.statusLabel}
                    </span>
                  </td>
                  <td>
                    {r.split ? (
                      <div className="mini-split">
                        {r.split.map((b) => (
                          <div className="ms" key={b.nm}>
                            <span className="nm">{b.nm}</span>
                            <span className={"bar" + (b.alt ? " alt" : "")}>
                              <span style={{ width: `${b.width}%` }} />
                            </span>
                            <span className="w">{b.w}</span>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <span className="muted" style={{ fontSize: 12 }}>
                        {r.splitText}
                      </span>
                    )}
                  </td>
                  {r.endpoint ? (
                    <td>
                      <span className="mono muted" style={{ fontSize: 12 }}>
                        {r.endpoint}
                      </span>
                    </td>
                  ) : (
                    <td className="muted">—</td>
                  )}
                  <td>
                    <div className="row-actions">
                      <Link className="act" to={`/traffic/${r.name}`} title="详情" aria-label="详情">
                        <EyeIcon />
                      </Link>
                      <button
                        className="act"
                        title={r.mode === "canary" ? "调整灰度比例" : "调整权重"}
                        aria-label="切流"
                        onClick={() => setDrawer({ kind: "split", row: r })}
                      >
                        <SplitIcon />
                      </button>
                      {r.mode === "canary" && (
                        <>
                          <button className="act" title="提升为全量" aria-label="提升" onClick={() => onPromote(r)}>
                            <PromoteIcon />
                          </button>
                          <button className="act" title="回滚" aria-label="回滚" onClick={() => onRollback(r)}>
                            <RollbackIcon />
                          </button>
                        </>
                      )}
                      <button
                        className="act act-danger"
                        title="删除"
                        aria-label="删除"
                        onClick={() => onDelete(r)}
                      >
                        <TrashIcon />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              <TableState q={q} cols={6} isEmpty={rows.length === 0} />
            </tbody>
          </table>
        </div>
        <div className="pagination">
          <span>共 {rows.length} 条策略</span>
          <div className="pages">
            <span className="pg">‹</span>
            <span className="pg on">1</span>
            <span className="pg">›</span>
          </div>
          <span>每页 20 条</span>
        </div>
      </div>

      {drawer?.kind === "create" && <TrafficDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "split" && <SplitDrawer row={drawer.row} onClose={() => setDrawer(null)} />}
    </main>
  );
}

// Ready services for the current tenant, as backend dropdown options.
function useReadyServiceNames(): string[] {
  const sq = useServices();
  return useMemo(
    () =>
      sq.data?.items
        ?.filter((s) => s.phase === "Ready")
        .map((s) => s.name) ?? [],
    [sq.data],
  );
}

function TrafficDrawer({ onClose }: { onClose: () => void }) {
  const services = useReadyServiceNames();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [path, setPath] = useState("");
  const [mode, setMode] = useState<"canary" | "weighted">("canary");

  // canary state
  const [stable, setStable] = useState("");
  const [canary, setCanary] = useState("");
  const [canaryPercent, setCanaryPercent] = useState("5");

  // weighted state: rows of {serviceName, weight}
  const [weightRows, setWeightRows] = useState<{ id: number; service: string; weight: string }[]>([
    { id: 0, service: "", weight: "50" },
    { id: 1, service: "", weight: "50" },
  ]);
  const addRow = () =>
    setWeightRows((r) => [...r, { id: (r.length ? Math.max(...r.map((x) => x.id)) : 0) + 1, service: "", weight: "0" }]);
  const removeRow = (id: number) => setWeightRows((r) => (r.length <= 1 ? r : r.filter((x) => x.id !== id)));
  const updateRow = (id: number, patch: Partial<{ service: string; weight: string }>) =>
    setWeightRows((r) => r.map((x) => (x.id === id ? { ...x, ...patch } : x)));

  const create = useApiMutation(
    (body: sdk.TrafficPolicyCreateRequest) => sdk.createTrafficPolicy({ body }),
    { invalidate: [["trafficpolicies"]], success: "流量策略已创建" },
  );

  const buildBody = (): sdk.TrafficPolicyCreateRequest | null => {
    const nm = name.trim();
    if (!nm) return null;
    const endpoint = path.trim() ? { path: path.trim() } : undefined;
    if (mode === "canary") {
      if (!stable || !canary) return null;
      const pct = Number(canaryPercent);
      return {
        name: nm,
        mode: "canary",
        description: description.trim() || undefined,
        endpoint,
        canaryPercent: Number.isFinite(pct) ? pct : undefined,
        backends: [
          { serviceName: stable, role: "stable" },
          { serviceName: canary, role: "canary" },
        ],
      };
    }
    // weighted
    const backends = weightRows
      .filter((row) => row.service.trim())
      .map((row) => ({ serviceName: row.service.trim(), role: "member" as const, weight: Number(row.weight) || 0 }));
    if (backends.length === 0) return null;
    return {
      name: nm,
      mode: "weighted",
      description: description.trim() || undefined,
      endpoint,
      backends,
    };
  };

  const body = buildBody();

  const submit = () => {
    if (!body) return;
    create.mutate(body, { onSuccess: onClose });
  };

  const serviceOptions = (
    <>
      <option value="">选择后端服务…</option>
      {services.map((s) => (
        <option key={s} value={s}>
          {s}（Ready）
        </option>
      ))}
    </>
  );

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="新建流量策略"
      sub="绑定稳定对外入口，把流量分发到当前租户的在线服务后端"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" disabled={!body || create.isPending} onClick={submit}>
            {create.isPending ? "创建中…" : "创建策略"}
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息与模式</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>
            名称 <span className="req">*</span>
          </label>
          <input
            className="input mono"
            placeholder="rt-chat"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="用途说明（可选）"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <div className="field full">
          <label>
            模式 <span className="req">*</span>
          </label>
          <div className="pick-grid" style={{ gridTemplateColumns: "1fr 1fr" }}>
            <div className={"pick" + (mode === "canary" ? " on" : "")} onClick={() => setMode("canary")}>
              <div className="p-title">灰度（canary）</div>
              <div className="p-spec">1 稳定 + 1 灰度，按百分比逐步放量</div>
            </div>
            <div className={"pick" + (mode === "weighted" ? " on" : "")} onClick={() => setMode("weighted")}>
              <div className="p-title">加权（weighted）</div>
              <div className="p-spec">N 个后端按权重切分，Σ=100</div>
            </div>
          </div>
        </div>
      </div>

      <FieldsetTitle n={2}>对外入口</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>Path</label>
          <input
            className="input mono"
            placeholder="留空自动生成 /services/<tenant>/rt-chat/"
            value={path}
            onChange={(e) => setPath(e.target.value)}
          />
        </div>
      </div>

      {/* 后端服务：灰度模式 */}
      {mode === "canary" && (
        <div>
          <FieldsetTitle n={3}>后端服务（灰度）</FieldsetTitle>
          <div className="form-grid">
            <div className="field">
              <label>
                稳定后端 <span className="req">*</span>
              </label>
              <select className="input" value={stable} onChange={(e) => setStable(e.target.value)}>
                {serviceOptions}
              </select>
            </div>
            <div className="field">
              <label>
                灰度后端 <span className="req">*</span>
              </label>
              <select className="input" value={canary} onChange={(e) => setCanary(e.target.value)}>
                {serviceOptions}
              </select>
            </div>
            <div className="field full">
              <label>初始灰度百分比</label>
              <input
                className="input num"
                type="number"
                min="0"
                max="100"
                value={canaryPercent}
                onChange={(e) => setCanaryPercent(e.target.value)}
              />
              <span className="help">
                1 个稳定后端 + 1 个灰度后端，按百分比逐步放量。后端下拉只列当前租户 Ready 的服务。
              </span>
            </div>
          </div>
        </div>
      )}

      {/* 后端服务：加权模式 */}
      {mode === "weighted" && (
        <div>
          <FieldsetTitle n={3}>后端服务（加权）</FieldsetTitle>
          <div className="form-grid">
            <div className="field full">
              <label>
                后端与权重 <span className="req">*</span>
              </label>
              <div className="vol-list">
                {weightRows.map((row) => (
                  <div className="vol-row" key={row.id}>
                    <select
                      className="input"
                      aria-label="后端服务"
                      value={row.service}
                      onChange={(e) => updateRow(row.id, { service: e.target.value })}
                    >
                      {serviceOptions}
                    </select>
                    <input
                      className="input num"
                      type="number"
                      min="0"
                      max="100"
                      value={row.weight}
                      placeholder="权重 0–100"
                      aria-label="权重"
                      onChange={(e) => updateRow(row.id, { weight: e.target.value })}
                    />
                    <button type="button" className="icon-btn" title="移除" onClick={() => removeRow(row.id)}>
                      <Icon name="x" />
                    </button>
                  </div>
                ))}
              </div>
              <a className="link vol-add" role="button" tabIndex={0} onClick={addRow}>
                <Icon name="plus" />
                添加后端
              </a>
              <span className="help">
                N 个后端按权重切分，权重之和需为 100。后端下拉只列当前租户 Ready 的服务。
              </span>
            </div>
          </div>
        </div>
      )}
    </Drawer>
  );
}

// 切流 / 调整权重 — drives splitTrafficPolicy. For canary policies we adjust the
//灰度百分比; for weighted policies we re-submit per-backend weights.
function SplitDrawer({ row, onClose }: { row: TrafficRow; onClose: () => void }) {
  const [canaryPercent, setCanaryPercent] = useState(
    String(row.split?.[1]?.w ?? row.split?.[1]?.width ?? 5),
  );
  const [weights, setWeights] = useState<{ nm: string; weight: string }[]>(
    () => row.split?.map((b) => ({ nm: b.nm, weight: String(b.w) })) ?? [],
  );

  const split = useApiMutation(
    (body: sdk.TrafficPolicySplitRequest) => sdk.splitTrafficPolicy({ path: { name: row.name }, body }),
    { invalidate: [["trafficpolicies"]], success: "流量分布已更新" },
  );

  const buildBody = (): sdk.TrafficPolicySplitRequest => {
    if (row.mode === "canary") {
      const pct = Number(canaryPercent);
      return { canaryPercent: Number.isFinite(pct) ? pct : null };
    }
    return {
      backends: weights
        .filter((w) => w.nm.trim())
        .map((w) => ({ serviceName: w.nm, role: "member" as const, weight: Number(w.weight) || 0 })),
    };
  };

  const submit = () => split.mutate(buildBody(), { onSuccess: onClose });

  return (
    <Drawer
      open
      onClose={onClose}
      title={<span className="mono">{row.name}</span>}
      sub={row.mode === "canary" ? "调整灰度后端的放量百分比" : "调整各后端服务的流量权重（Σ=100）"}
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" disabled={split.isPending} onClick={submit}>
            {split.isPending ? "应用中…" : "应用"}
          </button>
        </>
      }
    >
      {row.mode === "canary" ? (
        <>
          <FieldsetTitle n={1}>灰度百分比</FieldsetTitle>
          <div className="form-grid">
            <div className="field full">
              <label>灰度后端放量百分比</label>
              <input
                className="input num"
                type="number"
                min="0"
                max="100"
                value={canaryPercent}
                onChange={(e) => setCanaryPercent(e.target.value)}
              />
              <span className="help">灰度后端接收的流量百分比，剩余流量由稳定后端承接。</span>
            </div>
          </div>
        </>
      ) : (
        <>
          <FieldsetTitle n={1}>后端权重</FieldsetTitle>
          <div className="form-grid">
            <div className="field full">
              <label>后端与权重</label>
              <div className="vol-list">
                {weights.map((w, i) => (
                  <div className="vol-row" key={w.nm}>
                    <input className="input mono" value={w.nm} readOnly aria-label="后端服务" />
                    <input
                      className="input num"
                      type="number"
                      min="0"
                      max="100"
                      value={w.weight}
                      placeholder="权重 0–100"
                      aria-label="权重"
                      onChange={(e) =>
                        setWeights((prev) => prev.map((x, j) => (j === i ? { ...x, weight: e.target.value } : x)))
                      }
                    />
                  </div>
                ))}
              </div>
              <span className="help">调整各后端服务的流量权重，权重之和需为 100。</span>
            </div>
          </div>
        </>
      )}
    </Drawer>
  );
}
