import { Link } from "react-router-dom";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Segmented } from "@/components/Segmented";

// 首页 / Dashboard — faithful port of prototype/index.html. Content is the
// scope-overview demo data; the active-tenant scope (store) drives the subtitle.
export default function Dashboard() {
  const { tenant } = useApp();
  const { toast } = useUI();

  return (
    <main className="page">
      <div className="breadcrumb">
        <span>AxisML</span>
        <span className="sep">/</span>
        <span>首页</span>
      </div>

      <div className="page-head">
        <div>
          <h1>首页</h1>
          <p className="sub">
            {tenant === "all"
              ? "大模型研究院 · 全部租户的运行概览 —— 看一眼负载、容量与资源用量。"
              : "当前租户的运行概览 —— 看一眼负载、配额与资源用量。"}
          </p>
        </div>
        <div className="actions">
          <Segmented options={["1h", "24h", "7d"]} defaultValue="24h" />
          <button className="btn" onClick={() => toast("已刷新概览数据")}>
            <Icon name="refresh" />
            刷新
          </button>
        </div>
      </div>

      {/* KPI row */}
      <div className="grid cols-4" style={{ marginBottom: "var(--space-5)" }}>
        <Link className="kpi focal" to="/jobs">
          <div className="k-top">
            <span className="k-label">活跃任务</span>
            <span className="k-ico">
              <Icon name="job" />
            </span>
          </div>
          <div className="k-val num">23</div>
          <div className="k-foot">
            <span className="trend up">
              <Icon name="arrowUp" size={12} />4
            </span>{" "}
            运行 + 排队
          </div>
        </Link>
        <Link className="kpi" to="/services">
          <div className="k-top">
            <span className="k-label">在线服务</span>
            <span className="k-ico">
              <Icon name="service" />
            </span>
          </div>
          <div className="k-val num">
            9 <small>就绪 / 降级</small>
          </div>
          <div className="k-foot">7 就绪 · 2 降级</div>
        </Link>
        <Link className="kpi" to="/workspaces">
          <div className="k-top">
            <span className="k-label">工作区</span>
            <span className="k-ico">
              <Icon name="workspace" />
            </span>
          </div>
          <div className="k-val num">
            17 <small>运行中</small>
          </div>
          <div className="k-foot">活跃开发实例</div>
        </Link>
        <Link className="kpi" to="/models">
          <div className="k-top">
            <span className="k-label">模型</span>
            <span className="k-ico">
              <Icon name="model" />
            </span>
          </div>
          <div className="k-val num">
            128 <small>含公共</small>
          </div>
          <div className="k-foot">私有 + 公共仓库</div>
        </Link>
      </div>

      {/* cluster usage + trend */}
      <div className="grid cols-3" style={{ marginBottom: "var(--space-5)" }}>
        <div className="panel span-2">
          <div className="panel-head">
            <h3>集群资源用量</h3>
            <span className="hint">全集群可分配容量 · 30s 轮询</span>
          </div>
          <div className="panel-body">
            <div className="grid cols-3">
              <MeterStat icon="gpu" label="GPU" value="38" total="/ 56 卡" pct={68} cls="hot" />
              <MeterStat icon="cpu" label="CPU" value="410" total="/ 720 核" pct={57} cls="warn" />
              <MeterStat icon="mem" label="内存" value="2.1" total="/ 4.0 TiB" pct={53} cls="ok" />
            </div>
            <hr className="divline" />
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 14 }}>
              <h3 style={{ fontSize: 14 }}>集群 GPU 利用率趋势</h3>
              <div className="legend">
                <span>
                  <i style={{ background: "var(--accent)" }} />
                  GPU 利用率
                </span>
                <span>
                  <i style={{ background: "var(--muted)" }} />
                  任务并发
                </span>
              </div>
            </div>
            <svg className="chart" viewBox="0 0 720 200" preserveAspectRatio="none" style={{ height: 200 }}>
              <line className="grid-line" x1="0" y1="50" x2="720" y2="50" />
              <line className="grid-line" x1="0" y1="100" x2="720" y2="100" />
              <line className="grid-line" x1="0" y1="150" x2="720" y2="150" />
              <defs>
                <linearGradient id="g1" x1="0" x2="0" y1="0" y2="1">
                  <stop offset="0" stopColor="var(--accent)" stopOpacity="0.16" />
                  <stop offset="1" stopColor="var(--accent)" stopOpacity="0" />
                </linearGradient>
              </defs>
              <path
                d="M0 150 L60 138 L120 120 L180 128 L240 96 L300 104 L360 70 L420 88 L480 58 L540 74 L600 50 L660 66 L720 56 L720 200 L0 200 Z"
                fill="url(#g1)"
              />
              <path
                d="M0 150 L60 138 L120 120 L180 128 L240 96 L300 104 L360 70 L420 88 L480 58 L540 74 L600 50 L660 66 L720 56"
                fill="none"
                stroke="var(--accent)"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
              <path
                d="M0 170 L60 168 L120 160 L180 165 L240 150 L300 158 L360 140 L420 150 L480 134 L540 148 L600 130 L660 142 L720 132"
                fill="none"
                stroke="var(--muted)"
                strokeWidth="1.6"
                strokeDasharray="4 4"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </div>
        </div>

        {/* quick actions + recent activity */}
        <div style={{ display: "flex", flexDirection: "column", gap: "var(--space-5)" }}>
          <div className="panel">
            <div className="panel-head">
              <h3>快捷入口</h3>
            </div>
            <div className="panel-body" style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              <Link className="btn btn-primary btn-lg" to="/jobs">
                <Icon name="plus" />
                新建自定义任务
              </Link>
              <Link className="btn btn-lg" to="/services">
                <Icon name="plus" />
                部署在线服务
              </Link>
              <Link className="btn btn-lg" to="/workspaces">
                <Icon name="plus" />
                启动工作区
              </Link>
            </div>
          </div>
          <div className="panel" style={{ flex: 1 }}>
            <div className="panel-head">
              <h3>最近活动</h3>
              <Link className="link" to="/jobs" style={{ fontSize: 12 }}>
                全部
              </Link>
            </div>
            <div className="panel-body flush">
              <div className="table-wrap">
                <table className="tbl">
                  <tbody>
                    <RecentRow name="train-llm-7b-12" sub="自定义任务 · 2 分钟前" status="running" label="运行中" />
                    <RecentRow name="svc-chat-api" sub="在线服务 · 18 分钟前" status="success" label="就绪" />
                    <RecentRow name="llama3-sft-lr-sweep" sub="实验 · 1 小时前" status="running" label="运行中" />
                    <RecentRow name="rec-recall-train-07" sub="自定义任务 · 3 小时前" status="success" label="成功" />
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* per-tenant quota water level */}
      <div className="panel">
        <div className="panel-head">
          <h3>租户配额水位</h3>
          <Link className="link" to="/tenants" style={{ fontSize: 12 }}>
            租户管理 →
          </Link>
        </div>
        <div className="panel-body flush">
          <div className="table-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>租户</th>
                  <th>资源池</th>
                  <th className="num-col">GPU 用量 / max</th>
                  <th style={{ width: 240 }}>水位</th>
                  <th className="num-col">活跃任务</th>
                  <th className="num-col">在线服务</th>
                </tr>
              </thead>
              <tbody>
                <QuotaRow t="大模型研究院" pool="gpu-h100" usage="14 / 16" pct={88} min={50} jobs={9} svcs={3} />
                <QuotaRow t="推荐算法团队" pool="gpu-a100" usage="6 / 12" pct={50} min={33} jobs={5} svcs={2} />
                <QuotaRow t="智能驾驶感知" pool="gpu-l40s" usage="11 / 20" pct={55} min={40} jobs={7} svcs={3} />
                <QuotaRow t="风控 AI" pool="cpu-large" usage="0 / 0" pct={62} jobs={2} svcs={1} success />
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </main>
  );
}

function MeterStat({
  icon,
  label,
  value,
  total,
  pct,
  cls,
}: {
  icon: string;
  label: string;
  value: string;
  total: string;
  pct: number;
  cls: string;
}) {
  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", gap: 8, color: "var(--muted)", fontSize: 13, marginBottom: 10 }}>
        <Icon name={icon} size={16} />
        {label}
      </div>
      <div className="num" style={{ fontSize: 22, fontWeight: 600 }}>
        {value} <span className="muted" style={{ fontSize: 14 }}>{total}</span>
      </div>
      <div className={"meter " + cls} style={{ marginTop: 10 }}>
        <span style={{ width: pct + "%" }} />
      </div>
      <div className="num muted" style={{ fontSize: 12, marginTop: 6 }}>
        {pct}% 使用率
      </div>
    </div>
  );
}

function RecentRow({ name, sub, status, label }: { name: string; sub: string; status: string; label: string }) {
  return (
    <tr>
      <td>
        <div className="t-name mono">{name}</div>
        <div className="t-sub">{sub}</div>
      </td>
      <td style={{ textAlign: "right" }}>
        <span className={"status status-" + status}>
          <span className="dot" />
          {label}
        </span>
      </td>
    </tr>
  );
}

function QuotaRow({
  t,
  pool,
  usage,
  pct,
  min,
  jobs,
  svcs,
  success,
}: {
  t: string;
  pool: string;
  usage: string;
  pct: number;
  min?: number;
  jobs: number;
  svcs: number;
  success?: boolean;
}) {
  return (
    <tr>
      <td className="t-name">{t}</td>
      <td>
        <span className="tag mono">{pool}</span>
      </td>
      <td className="num-col">{usage}</td>
      <td>
        <div className="qbar">
          <span className="used" style={{ width: pct + "%", ...(success ? { background: "var(--success)" } : {}) }} />
          {min != null && <span className="min-mark" style={{ left: min + "%" }} />}
        </div>
      </td>
      <td className="num-col">{jobs}</td>
      <td className="num-col">{svcs}</td>
    </tr>
  );
}
