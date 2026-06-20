import { useState } from "react";
import { Link } from "react-router-dom";
import { useTrafficPolicies } from "@/api/hooks";
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

// One-off "禁用" (disable) glyph — circle with diagonal slash.
function DisableIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <circle cx="12" cy="12" r="9" />
      <path d="M5.6 5.6l12.8 12.8" />
    </svg>
  );
}

// One-off "启用" (enable) play glyph.
function PlayGlyph() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M6 4l14 8-14 8z" />
    </svg>
  );
}

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
  mode: string;
  status: StatusKind;
  statusLabel: string;
  split?: SplitBackend[];
  splitText?: string;
  endpoint?: string;
  action: "disable" | "enable";
}

// Faithful demo rows from prototype/traffic.html — rendered when the backend
// (contract-only shell) returns no items.
const FALLBACK: TrafficRow[] = [
  {
    name: "rt-chat",
    desc: "对话服务灰度发布",
    mode: "灰度",
    status: "pending",
    statusLabel: "灰度中",
    split: [
      { nm: "svc-chat-v1", width: 90, w: 90 },
      { nm: "svc-chat-v2", width: 10, w: 10, alt: true },
    ],
    endpoint: "/services/llm-lab/chat/",
    action: "disable",
  },
  {
    name: "rt-embed",
    desc: "向量服务加权切分",
    mode: "加权",
    status: "success",
    statusLabel: "生效中",
    split: [
      { nm: "svc-embed-a", width: 50, w: 50 },
      { nm: "svc-embed-b", width: 50, w: 50, alt: true },
    ],
    endpoint: "/services/llm-lab/embed/",
    action: "disable",
  },
  {
    name: "rt-rerank",
    desc: "重排序灰度",
    mode: "灰度",
    status: "stopped",
    statusLabel: "未就绪",
    splitText: "svc-rerank-v2 0 · 稳定后端缺失",
    action: "enable",
  },
];

export default function Traffic() {
  const { data } = useTrafficPolicies();
  const [drawer, setDrawer] = useState(false);

  const rows: TrafficRow[] = data?.items?.map((p) => {
    const ready = p.phase === "Ready";
    const status: StatusKind = ready ? "success" : p.phase === "Failed" || p.phase === "Deleting" ? "stopped" : "pending";
    const statusLabel = ready ? "生效中" : status === "stopped" ? "未就绪" : "灰度中";
    return {
      name: p.name,
      desc: p.description ?? p.displayName ?? "",
      mode: p.mode === "weighted" ? "加权" : "灰度",
      status,
      statusLabel,
      split: p.backends?.map((b, i) => ({
        nm: b.serviceName,
        width: b.actualPct ?? b.weight,
        w: b.weight,
        alt: i > 0,
      })),
      endpoint: p.accessUrl,
      // A Ready policy can be disabled; anything not yet serving offers enable.
      action: ready ? ("disable" as const) : ("enable" as const),
    };
  }) ?? FALLBACK;

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
          <button className="btn btn-primary" onClick={() => setDrawer(true)}>
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
                    <span className="badge badge-neutral">{r.mode}</span>
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
                      {r.action === "disable" ? (
                        <button className="act" title="禁用" aria-label="禁用">
                          <DisableIcon />
                        </button>
                      ) : (
                        <button className="act" title="启用" aria-label="启用">
                          <PlayGlyph />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
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

      {drawer && <TrafficDrawer onClose={() => setDrawer(false)} />}
    </main>
  );
}

function TrafficDrawer({ onClose }: { onClose: () => void }) {
  const { toast } = useUI();
  const [mode, setMode] = useState<"canary" | "weighted">("canary");
  const [weightRows, setWeightRows] = useState<number[]>([0, 1]);

  const addRow = () => setWeightRows((r) => [...r, r.length ? Math.max(...r) + 1 : 0]);
  const removeRow = (id: number) => setWeightRows((r) => r.filter((x) => x !== id));

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="新建流量策略"
      sub="绑定稳定对外入口，把流量分发到当前租户的在线服务后端"
      footer={
        <>
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("流量策略已创建");
              onClose();
            }}
          >
            创建策略
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
          <input className="input mono" placeholder="rt-chat" />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" placeholder="用途说明（可选）" />
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
          <input className="input mono" placeholder="留空自动生成 /services/llm-lab/rt-chat/" />
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
              <select className="input">
                <option>svc-chat-v1（Ready）</option>
                <option>svc-rerank-v1（Ready）</option>
              </select>
            </div>
            <div className="field">
              <label>
                灰度后端 <span className="req">*</span>
              </label>
              <select className="input">
                <option>svc-chat-v2（Ready）</option>
                <option>svc-rerank-v2（Ready）</option>
              </select>
            </div>
            <div className="field full">
              <label>初始灰度百分比</label>
              <input className="input num" defaultValue="5" />
              <span className="help">
                1 个稳定后端 + 1 个灰度后端，按百分比逐步放量。后端下拉只列当前租户 Ready 且未被其它活跃策略占用的服务。
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
                {weightRows.map((id) => (
                  <div className="vol-row" key={id}>
                    <select className="input" aria-label="后端服务">
                      <option>svc-embed-a（Ready）</option>
                      <option>svc-embed-b（Ready）</option>
                      <option>svc-chat-v1（Ready）</option>
                    </select>
                    <input
                      className="input num"
                      type="number"
                      min="0"
                      max="100"
                      defaultValue="50"
                      placeholder="权重 0–100"
                      aria-label="权重"
                    />
                    <button type="button" className="icon-btn" title="移除" onClick={() => removeRow(id)}>
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
                N 个后端按权重切分，权重之和需为 100。后端下拉只列当前租户 Ready 且未被其它活跃策略占用的服务。
              </span>
            </div>
          </div>
        </div>
      )}
    </Drawer>
  );
}
