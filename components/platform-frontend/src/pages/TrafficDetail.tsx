import { useMemo, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import { useUI } from "@/app/ui";
import { Tabs } from "@/components/Tabs";
import { Segmented } from "@/components/Segmented";

type Mode = "灰度" | "加权";

// One-off back-arrow glyph (chevron-left, stroke-width 2).
function BackArrow() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M15 18l-6-6 6-6" />
    </svg>
  );
}

// One-off small copy glyph used by the entry-address copy button.
function CopyMini() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="11" height="11" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  );
}

export default function TrafficDetail() {
  const { name = "rt-chat" } = useParams();
  const [mode, setMode] = useState<Mode>("灰度");

  return (
    <main className="page traffic-detail">
      {/* Slider-thumb styling ported from prototype/traffic-detail.html <style>
          (pseudo-elements can't be set via inline style). The track fill is set
          inline on each <input type="range"> from the live canary value. */}
      <style>{`
        .traffic-detail input[type=range]::-webkit-slider-thumb { -webkit-appearance:none; width:20px; height:20px; border-radius:99px; background:var(--bg); border:2px solid var(--warn); box-shadow:var(--elev-card); cursor:pointer; }
        .traffic-detail input[type=range]::-moz-range-thumb { width:18px; height:18px; border-radius:99px; background:var(--bg); border:2px solid var(--warn); box-shadow:var(--elev-card); cursor:pointer; }
      `}</style>
      <Link className="back-link" to="/traffic">
        <BackArrow />
        返回流量策略列表
      </Link>

      <div className="toolbar" style={{ marginBottom: "var(--space-4)" }}>
        <span className="hint">演示：在灰度 / 加权两种模式间切换查看页面</span>
        <div className="grow" />
        <Segmented options={["灰度", "加权"]} defaultValue={mode} onChange={(v) => setMode(v as Mode)} />
      </div>

      {mode === "灰度" ? <CanaryDetail name={name} /> : <WeightedDetail name={name} />}
    </main>
  );
}

/* ───────────────────────────── 灰度 / canary ───────────────────────────── */

function CanaryDetail({ name }: { name: string }) {
  const { toast, confirm } = useUI();
  const [canary, setCanary] = useState(10);
  const stable = 100 - canary;

  return (
    <>
      <div className="page-head">
        <div>
          <h1 className="detail-title">
            {name}{" "}
            <span className="spill warn">
              <span className="dot" />
              灰度中
            </span>{" "}
            <span className="badge badge-neutral">灰度</span>
          </h1>
          <div className="detail-sub">对话服务灰度发布</div>
        </div>
        <div className="actions">
          <button
            className="btn btn-danger"
            onClick={() =>
              confirm({
                title: `删除流量策略 ${name}？`,
                desc: "将移除加权路由，流量回落到默认网关。该操作不可恢复。",
                okLabel: "确认删除",
                toast: `流量策略 ${name} 已删除`,
              })
            }
          >
            删除
          </button>
        </div>
      </div>

      <Tabs
        tabs={[
          {
            key: "info",
            label: "概览",
            content: (
              <div className="panel">
                <div className="panel-head">
                  <h3>策略信息</h3>
                  <button className="btn btn-sm" onClick={() => toast("进入编辑（仅后端权重可改）")}>
                    编辑
                  </button>
                </div>
                <div className="panel-body">
                  <dl className="kv kv-lg">
                    <dt>名称</dt>
                    <dd>
                      <span className="cchip">{name}</span>
                    </dd>
                    <dt>描述</dt>
                    <dd>将稳定版 chat-v1 的流量逐步切到 chat-v2，按灰度百分比放量</dd>
                    <dt>模式</dt>
                    <dd>
                      <span className="cchip">灰度</span>
                    </dd>
                    <dt>对外入口</dt>
                    <dd>
                      <span className="cchip">/services/team-a/chat/</span>
                      <button
                        className="icon-mini"
                        title="复制"
                        aria-label="复制入口地址"
                        onClick={() => toast("入口地址已复制")}
                      >
                        <CopyMini />
                      </button>
                    </dd>
                    <dt>后端数</dt>
                    <dd>
                      <span className="mono">2</span>
                    </dd>
                    <dt>创建人</dt>
                    <dd>张伟</dd>
                    <dt>创建时间</dt>
                    <dd className="mono">2026-06-10 15:42:08</dd>
                  </dl>
                </div>
              </div>
            ),
          },
          {
            key: "dist",
            label: "流量配置",
            content: (
              <>
                <div className="panel">
                  <div className="panel-body">
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "space-between",
                        marginBottom: 16,
                      }}
                    >
                      <label style={{ fontSize: 14, color: "var(--fg-2)" }}>灰度百分比</label>
                      <div
                        style={{
                          fontFamily: "var(--font-mono)",
                          fontWeight: 700,
                          fontSize: "var(--text-2xl)",
                          lineHeight: 1,
                        }}
                      >
                        <span>{canary}</span>
                        <span style={{ fontSize: 14, color: "var(--muted)", fontWeight: 500 }}>%</span>
                      </div>
                    </div>
                    <input
                      type="range"
                      min="0"
                      max="100"
                      value={canary}
                      onChange={(e) => setCanary(+e.target.value)}
                      style={{
                        WebkitAppearance: "none",
                        appearance: "none",
                        width: "100%",
                        height: 6,
                        borderRadius: 99,
                        background: `linear-gradient(90deg,var(--warn) ${canary}%,var(--surface) ${canary}%)`,
                        outline: "none",
                        cursor: "pointer",
                      }}
                    />
                    <div style={{ display: "flex", gap: 28, marginTop: 14, fontSize: 13, color: "var(--fg-2)" }}>
                      <span>
                        稳定 <b style={{ color: "var(--info)", fontFamily: "var(--font-mono)" }}>{stable}%</b>
                      </span>
                      <span>
                        灰度{" "}
                        <b style={{ color: "color-mix(in oklab,var(--warn),black 16%)", fontFamily: "var(--font-mono)" }}>
                          {canary}%
                        </b>
                      </span>
                    </div>
                    <hr className="divline" />
                    <div style={{ display: "flex", gap: 10 }}>
                      <button
                        className="btn btn-dark"
                        onClick={() => {
                          setCanary(100);
                          toast("已提升 svc-chat-v2 为新稳定基线");
                        }}
                      >
                        提升为稳定
                      </button>
                      <button
                        className="btn"
                        onClick={() => {
                          setCanary(0);
                          toast("灰度已回滚至稳定后端");
                        }}
                      >
                        回滚
                      </button>
                    </div>
                  </div>
                </div>

                <div className="panel" style={{ marginTop: "var(--space-5)" }}>
                  <div className="panel-head">
                    <h3>后端分布</h3>
                  </div>
                  <div className="table-wrap">
                    <table className="tbl">
                      <thead>
                        <tr>
                          <th>在线服务</th>
                          <th>角色</th>
                          <th className="num-col">目标权重</th>
                          <th>实际流量占比</th>
                          <th>后端状态</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr>
                          <td>
                            <Link className="t-name mono" to="/services/svc-chat-v1">
                              svc-chat-v1
                            </Link>
                          </td>
                          <td>
                            <span className="badge badge-neutral">稳定</span>
                          </td>
                          <td className="num-col">{stable}</td>
                          <td>
                            <div className="flowbar">
                              <span className="track">
                                <span style={{ width: `${stable}%` }} />
                              </span>
                              <span className="pct">{stable}%</span>
                            </div>
                          </td>
                          <td>
                            <span className="spill ok">
                              <span className="dot" />
                              就绪
                            </span>
                          </td>
                        </tr>
                        <tr>
                          <td>
                            <Link className="t-name mono" to="/services/svc-chat-v2">
                              svc-chat-v2
                            </Link>
                          </td>
                          <td>
                            <span className="badge badge-warn">灰度</span>
                          </td>
                          <td className="num-col">{canary}</td>
                          <td>
                            <div className="flowbar">
                              <span className="track">
                                <span style={{ width: `${canary}%` }} />
                              </span>
                              <span className="pct">{canary}%</span>
                            </div>
                          </td>
                          <td>
                            <span className="spill ok">
                              <span className="dot" />
                              就绪
                            </span>
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
              </>
            ),
          },
          {
            key: "mon",
            label: "监控",
            content: (
              <>
                <div className="toolbar">
                  <span className="hint">来自 compute 指标代理 · 按稳定 / 灰度分组</span>
                  <div className="grow" />
                  <div className="segmented">
                    <button>5m</button>
                    <button className="on">1h</button>
                    <button>24h</button>
                  </div>
                </div>
                <div className="grid cols-2">
                  <MCard title="QPS" legend={[["var(--info)", "svc-chat-v1"], ["var(--warn)", "svc-chat-v2"]]}>
                    <path d="M0 70 L60 50 L120 78 L180 44 L240 84 L300 40 L360 74 L420 48 L480 80 L540 46 L600 64" fill="none" stroke="var(--info)" strokeWidth="2" />
                    <path d="M0 96 L60 88 L120 100 L180 84 L240 104 L300 80 L360 98 L420 82 L480 102 L540 86 L600 94" fill="none" stroke="var(--warn)" strokeWidth="1.8" />
                  </MCard>
                  <MCard title="延迟 p95" legend={[["var(--info)", "svc-chat-v1"], ["var(--warn)", "svc-chat-v2"]]}>
                    <path d="M0 80 L60 66 L120 86 L180 60 L240 90 L300 56 L360 84 L420 58 L480 88 L540 64 L600 76" fill="none" stroke="var(--info)" strokeWidth="2" />
                    <path d="M0 74 L60 58 L120 82 L180 52 L240 86 L300 50 L360 80 L420 50 L480 84 L540 56 L600 70" fill="none" stroke="var(--warn)" strokeWidth="1.8" />
                  </MCard>
                  <MCard
                    title="错误率 (5xx)"
                    legend={[["var(--info)", "svc-chat-v1"], ["var(--warn)", "svc-chat-v2"]]}
                    note="灰度后端 v2 错误率略高（0.4% vs 0.1%），建议继续观察后再放量。"
                  >
                    <path d="M0 116 L120 114 L240 116 L360 114 L480 116 L600 114" fill="none" stroke="var(--info)" strokeWidth="1.8" />
                    <path d="M0 110 L60 104 L120 96 L180 86 L240 98 L300 84 L360 94 L420 80 L480 92 L540 100 L600 96" fill="none" stroke="var(--warn)" strokeWidth="1.8" />
                  </MCard>
                </div>
              </>
            ),
          },
          {
            key: "ev",
            label: "事件",
            content: (
              <div className="panel">
                <div className="panel-body">
                  <div className="timeline">
                    <TLItem name="WeightUpdated" time="2026-06-10 15:42:08" desc="灰度百分比调整为 10%" />
                    <TLItem name="BackendReady" time="2026-06-10 15:42:20" desc="后端 svc-chat-v2 已就绪" />
                    <TLItem name="RouteDerived" time="2026-06-10 15:42:08" desc="已派生加权路由（HTTPRoute 加权 backendRefs）" muted />
                    <TLItem name="PolicyCreated" time="2026-06-10 15:42:02" desc="创建灰度策略，对外入口 /services/team-a/chat/" muted />
                  </div>
                </div>
              </div>
            ),
          },
        ]}
      />
    </>
  );
}

/* ───────────────────────────── 加权 / weighted ───────────────────────────── */

function clamp(v: number) {
  if (Number.isNaN(v)) v = 0;
  return Math.max(0, Math.min(100, v));
}

function WeightedDetail({ name }: { name: string }) {
  const { toast, confirm } = useUI();
  const [a, setA] = useState(50);
  const [b, setB] = useState(50);

  const sum = a + b;
  const ok = sum === 100;
  const { pa, pb } = useMemo(() => {
    const _pa = sum ? Math.round((a / sum) * 100) : 0;
    return { pa: _pa, pb: sum ? 100 - _pa : 0 };
  }, [a, b, sum]);

  return (
    <>
      <div className="page-head">
        <div>
          <h1 className="detail-title">
            {name}{" "}
            <span className="spill ok">
              <span className="dot" />
              生效中
            </span>{" "}
            <span className="badge badge-neutral">加权</span>
          </h1>
          <div className="detail-sub">向量服务加权切分</div>
        </div>
        <div className="actions">
          <button
            className="btn btn-danger"
            onClick={() =>
              confirm({
                title: `删除流量策略 ${name}？`,
                desc: "将移除加权路由，流量回落到默认网关。该操作不可恢复。",
                okLabel: "确认删除",
                toast: `流量策略 ${name} 已删除`,
              })
            }
          >
            删除
          </button>
        </div>
      </div>

      <Tabs
        tabs={[
          {
            key: "info",
            label: "概览",
            content: (
              <div className="panel">
                <div className="panel-head">
                  <h3>策略信息</h3>
                  <button className="btn btn-sm" onClick={() => toast("进入编辑（仅后端权重可改）")}>
                    编辑
                  </button>
                </div>
                <div className="panel-body">
                  <dl className="kv kv-lg">
                    <dt>名称</dt>
                    <dd>
                      <span className="cchip">{name}</span>
                    </dd>
                    <dt>描述</dt>
                    <dd>把向量检索流量在 svc-embed-a / svc-embed-b 两个后端间按权重切分</dd>
                    <dt>模式</dt>
                    <dd>
                      <span className="cchip">加权</span>
                    </dd>
                    <dt>对外入口</dt>
                    <dd>
                      <span className="cchip">/services/team-a/embed/</span>
                      <button
                        className="icon-mini"
                        title="复制"
                        aria-label="复制入口地址"
                        onClick={() => toast("入口地址已复制")}
                      >
                        <CopyMini />
                      </button>
                    </dd>
                    <dt>后端数</dt>
                    <dd>
                      <span className="mono">2</span>
                    </dd>
                    <dt>创建人</dt>
                    <dd>李娜</dd>
                    <dt>创建时间</dt>
                    <dd className="mono">2026-06-09 10:20:31</dd>
                  </dl>
                </div>
              </div>
            ),
          },
          {
            key: "dist",
            label: "流量配置",
            content: (
              <div className="panel">
                <div className="panel-head">
                  <h3>后端分布</h3>
                  <span className="hint">直接编辑目标权重，实时 Σ=100 校验</span>
                </div>
                <div className="table-wrap">
                  <table className="tbl">
                    <thead>
                      <tr>
                        <th>在线服务</th>
                        <th>角色</th>
                        <th className="num-col">目标权重</th>
                        <th>实际流量占比</th>
                        <th>后端状态</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr>
                        <td>
                          <Link className="t-name mono" to="/services/svc-embed-a">
                            svc-embed-a
                          </Link>
                        </td>
                        <td>
                          <span className="badge badge-neutral">成员</span>
                        </td>
                        <td className="num-col">
                          <span className="wfield" style={{ justifyContent: "flex-end" }}>
                            <input
                              className="input wt"
                              type="number"
                              min="0"
                              max="100"
                              value={a}
                              onChange={(e) => setA(clamp(parseInt(e.target.value, 10)))}
                              aria-label="svc-embed-a 权重"
                            />
                            <span className="wpct">%</span>
                          </span>
                        </td>
                        <td>
                          <div className="flowbar">
                            <span className="track">
                              <span style={{ width: `${pa}%` }} />
                            </span>
                            <span className="pct">{pa}%</span>
                          </div>
                        </td>
                        <td>
                          <span className="spill ok">
                            <span className="dot" />
                            就绪
                          </span>
                        </td>
                      </tr>
                      <tr>
                        <td>
                          <Link className="t-name mono" to="/services/svc-embed-b">
                            svc-embed-b
                          </Link>
                        </td>
                        <td>
                          <span className="badge badge-neutral">成员</span>
                        </td>
                        <td className="num-col">
                          <span className="wfield" style={{ justifyContent: "flex-end" }}>
                            <input
                              className="input wt"
                              type="number"
                              min="0"
                              max="100"
                              value={b}
                              onChange={(e) => setB(clamp(parseInt(e.target.value, 10)))}
                              aria-label="svc-embed-b 权重"
                            />
                            <span className="wpct">%</span>
                          </span>
                        </td>
                        <td>
                          <div className="flowbar">
                            <span className="track">
                              <span style={{ width: `${pb}%` }} />
                            </span>
                            <span className="pct">{pb}%</span>
                          </div>
                        </td>
                        <td>
                          <span className="spill ok">
                            <span className="dot" />
                            就绪
                          </span>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div
                  className="panel-body"
                  style={{
                    borderTop: "1px solid var(--border-soft)",
                    display: "flex",
                    alignItems: "center",
                    gap: "var(--space-4)",
                  }}
                >
                  <span className={"wsum" + (ok ? "" : " bad")} style={{ padding: 0 }}>
                    Σ = {sum}% {ok ? "✓" : "✕"}
                  </span>
                  <span className="grow" />
                  <button
                    className="btn btn-dark"
                    disabled={!ok}
                    onClick={() => toast("权重已应用（Σ=100）")}
                  >
                    应用权重
                  </button>
                </div>
              </div>
            ),
          },
          {
            key: "mon",
            label: "监控",
            content: (
              <>
                <div className="toolbar">
                  <span className="hint">来自 compute 指标代理 · 按后端分组</span>
                  <div className="grow" />
                  <div className="segmented">
                    <button>5m</button>
                    <button className="on">1h</button>
                    <button>24h</button>
                  </div>
                </div>
                <div className="grid cols-2">
                  <MCard title="QPS" legend={[["var(--info)", "svc-embed-a"], ["var(--success)", "svc-embed-b"]]}>
                    <path d="M0 64 L60 52 L120 72 L180 48 L240 78 L300 46 L360 70 L420 50 L480 74 L540 48 L600 60" fill="none" stroke="var(--info)" strokeWidth="2" />
                    <path d="M0 70 L60 58 L120 76 L180 54 L240 82 L300 52 L360 74 L420 56 L480 78 L540 54 L600 66" fill="none" stroke="var(--success)" strokeWidth="1.8" />
                  </MCard>
                  <MCard title="延迟 p95" legend={[["var(--info)", "svc-embed-a"], ["var(--success)", "svc-embed-b"]]}>
                    <path d="M0 80 L60 66 L120 86 L180 60 L240 90 L300 56 L360 84 L420 58 L480 88 L540 64 L600 76" fill="none" stroke="var(--info)" strokeWidth="2" />
                    <path d="M0 76 L60 62 L120 82 L180 56 L240 86 L300 52 L360 80 L420 54 L480 84 L540 60 L600 72" fill="none" stroke="var(--success)" strokeWidth="1.8" />
                  </MCard>
                  <MCard
                    title="错误率 (5xx)"
                    legend={[["var(--info)", "svc-embed-a"], ["var(--success)", "svc-embed-b"]]}
                    note="两后端健康度相当，错误率均 < 0.1%，权重均分稳定。"
                  >
                    <path d="M0 116 L120 114 L240 116 L360 113 L480 116 L600 114" fill="none" stroke="var(--info)" strokeWidth="1.8" />
                    <path d="M0 115 L120 116 L240 113 L360 116 L480 114 L600 116" fill="none" stroke="var(--success)" strokeWidth="1.8" />
                  </MCard>
                </div>
              </>
            ),
          },
          {
            key: "ev",
            label: "事件",
            content: (
              <div className="panel">
                <div className="panel-body">
                  <div className="timeline">
                    <TLItem name="WeightUpdated" time="2026-06-09 14:08:55" desc="权重调整为 svc-embed-a 50 / svc-embed-b 50（Σ=100）" />
                    <TLItem name="BackendReady" time="2026-06-09 10:21:12" desc="后端 svc-embed-b 已就绪，纳入加权后端集" />
                    <TLItem name="RouteDerived" time="2026-06-09 10:20:38" desc="已派生加权路由（HTTPRoute 加权 backendRefs · 2 后端）" muted />
                    <TLItem name="PolicyCreated" time="2026-06-09 10:20:31" desc="创建加权策略，对外入口 /services/team-a/embed/" muted />
                  </div>
                </div>
              </div>
            ),
          },
        ]}
      />
    </>
  );
}

/* ───────────────────────────── shared bits ───────────────────────────── */

function MCard({
  title,
  legend,
  note,
  children,
}: {
  title: string;
  legend: [string, string][];
  note?: string;
  children: ReactNode;
}) {
  return (
    <div className="mcard">
      <div className="mc-head">
        <span className="mc-title">{title}</span>
        <div className="legend">
          {legend.map(([color, label]) => (
            <span key={label}>
              <i style={{ background: color }} />
              {label}
            </span>
          ))}
        </div>
      </div>
      <svg className="mc-chart chart" viewBox="0 0 600 132" preserveAspectRatio="none">
        <line className="grid-line" x1="0" y1="44" x2="600" y2="44" />
        <line className="grid-line" x1="0" y1="88" x2="600" y2="88" />
        {children}
      </svg>
      {note && (
        <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
          {note}
        </p>
      )}
    </div>
  );
}

function TLItem({ name, time, desc, muted }: { name: string; time: string; desc: string; muted?: boolean }) {
  return (
    <div className={"tl-item" + (muted ? " is-muted" : "")}>
      <span className="tl-dot" />
      <div className="tl-head">
        <span className="tl-name">{name}</span>
        <span className="tl-tag">NORMAL</span>
        <span className="tl-time">{time}</span>
      </div>
      <div className="tl-desc">{desc}</div>
    </div>
  );
}
