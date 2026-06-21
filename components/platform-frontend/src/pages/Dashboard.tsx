import { type ReactNode } from "react";
import { Link } from "react-router-dom";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Segmented } from "@/components/Segmented";
import { Tabs, type TabDef } from "@/components/Tabs";
import { BlockState, type QueryLike } from "@/components/states";
import {
  useWorkspaces,
  useExperiments,
  useJobs,
  useServices,
  useModels,
  useImages,
  useResourcePools,
} from "@/api/hooks";
import * as sdk from "@/api/generated";

// 首页 / Dashboard. KPI counts and recent activity come from the live list
// endpoints (scoped to the active tenant); the active-tenant run roll-ups
// (实验/任务 "运行") come from getTenant?stats. Cluster utilisation has no
// metrics source yet — it renders a structured zero state (GPU 利用率 / GPU
// 使用额度 are a separate, pending topic) rather than fabricated numbers.
export default function Dashboard() {
  const { tenant } = useApp();
  const { toast } = useUI();

  const ws = useWorkspaces();
  const exp = useExperiments();
  const jobs = useJobs();
  const svc = useServices();
  const models = useModels();
  const images = useImages();
  const pools = useResourcePools();

  // Active-tenant workload roll-ups (active job/experiment runs). getTenant is
  // enriched server-side; skip the all-tenants pseudo-scope where it 404s.
  const statsQ = useQuery({
    queryKey: ["tenant", tenant, "stats"],
    enabled: tenant !== "" && tenant !== "all",
    queryFn: async () => {
      // getTenant is always enriched server-side (member / active-run counts).
      const { data, error } = await sdk.getTenant({ path: { name: tenant } });
      if (error) throw error;
      return data;
    },
  });

  const wsRunning = (ws.data?.items ?? []).filter((w) => w.phase === "Running").length;
  const svcReady = (svc.data?.items ?? []).filter((s) => s.phase === "Ready").length;
  const svcDegraded = (svc.data?.items ?? []).filter((s) => s.phase === "Degraded").length;
  const assetTotal = (models.data?.count ?? 0) + (images.data?.count ?? 0);

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
          <p className="sub">当前租户的运行概览 —— 看一眼负载、容量与资源用量。</p>
        </div>
        <div className="actions">
          <Segmented options={["1h", "24h", "7d"]} defaultValue="24h" />
          <button className="btn" onClick={() => toast("已刷新概览数据")}>
            <Icon name="refresh" />
            刷新
          </button>
        </div>
      </div>

      {/* KPI row: 工作区 · 实验 · 自定义任务 · 在线服务 · 资产 */}
      <div className="grid cols-5" style={{ marginBottom: "var(--space-5)" }}>
        <Kpi to="/workspaces" icon="workspace" label="工作区" value={countText(ws)} foot={<>{wsRunning} 运行</>} />
        <Kpi
          to="/experiments"
          icon="experiment"
          label="实验"
          value={countText(exp)}
          foot={<>{runText(statsQ, statsQ.data?.activeExperimentRuns)} 运行</>}
        />
        <Kpi
          to="/jobs"
          icon="job"
          label="自定义任务"
          value={countText(jobs)}
          foot={<>{runText(statsQ, statsQ.data?.activeJobRuns)} 运行</>}
        />
        <Kpi
          to="/services"
          icon="service"
          label="在线服务"
          value={countText(svc)}
          foot={
            <>
              {svcReady} 就绪 · {svcDegraded} 降级
            </>
          }
        />
        <Kpi
          to="/models"
          icon="layers"
          label="资产"
          value={anyLoadErr([models, images]) ? "—" : String(assetTotal)}
          foot={
            <>
              {models.data?.count ?? 0} 模型 · {images.data?.count ?? 0} 镜像
            </>
          }
        />
      </div>

      {/* cluster usage (tabs: 全部 + per pool) + recent activity */}
      <div className="grid cols-3" style={{ marginBottom: "var(--space-5)" }}>
        <div className="panel span-2">
          <div className="panel-head">
            <h3>集群资源用量</h3>
            <span className="hint">指标接入中 · GPU 利用率 / 额度待对接</span>
          </div>
          <div className="panel-body">
            <Tabs tabs={clusterTabs(pools)} />
          </div>
        </div>

        {/* recent activity (real: workspaces + services by updatedAt) */}
        <div className="panel">
          <div className="panel-head">
            <h3>最近活动</h3>
          </div>
          <div className="panel-body flush">
            <RecentActivity ws={ws} svc={svc} />
          </div>
        </div>
      </div>
    </main>
  );
}

// ── KPI card ──────────────────────────────────────────────────────────────────
function Kpi({
  to,
  icon,
  label,
  value,
  foot,
}: {
  to: string;
  icon: string;
  label: string;
  value: ReactNode;
  foot: ReactNode;
}) {
  return (
    <Link className="kpi focal" to={to}>
      <div className="k-top">
        <span className="k-label">{label}</span>
        <span className="k-ico">
          <Icon name={icon} />
        </span>
      </div>
      <div className="k-val num">{value}</div>
      <div className="k-foot">{foot}</div>
    </Link>
  );
}

// countText renders the list total, or a loading/error placeholder so a KPI
// never shows a fabricated number.
function countText(q: UseQueryResult<{ count?: number }>): ReactNode {
  if (q.isLoading) return "…";
  if (q.isError) return "—";
  return String(q.data?.count ?? 0);
}

// runText renders an active-run roll-up from the tenant stats query.
function runText(q: UseQueryResult<unknown>, n: number | undefined): ReactNode {
  if (q.isLoading) return "…";
  if (q.isError || n == null) return "—";
  return String(n);
}

function anyLoadErr(qs: UseQueryResult<unknown>[]): boolean {
  return qs.some((q) => q.isLoading || q.isError);
}

// ── Cluster usage (zero state until a metrics source exists) ──────────────────
function clusterTabs(pools: UseQueryResult<{ items?: Array<{ name: string }> }>): TabDef[] {
  const poolNames = (pools.data?.items ?? []).map((p) => p.name);
  return [
    { key: "all", label: "全部", content: <ClusterPane /> },
    ...poolNames.map((name) => ({ key: name, label: name, content: <ClusterPane /> })),
  ];
}

function ClusterPane() {
  return (
    <>
      <div className="grid cols-3">
        <MeterStat icon="gpu" label="GPU" unit="卡" />
        <MeterStat icon="cpu" label="CPU" unit="核" />
        <MeterStat icon="mem" label="内存" unit="TiB" />
      </div>
      <hr className="divline" />
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 14 }}>
        <h3 style={{ fontSize: 14 }}>GPU 趋势</h3>
        <div className="legend">
          <span>
            <i style={{ background: "var(--accent)" }} />
            GPU 利用率
          </span>
          <span>
            <i style={{ background: "var(--muted)" }} />
            GPU 使用额度
          </span>
        </div>
      </div>
      <svg className="chart" viewBox="0 0 720 200" preserveAspectRatio="none" style={{ height: 200 }}>
        <line className="grid-line" x1="0" y1="50" x2="720" y2="50" />
        <line className="grid-line" x1="0" y1="100" x2="720" y2="100" />
        <line className="grid-line" x1="0" y1="150" x2="720" y2="150" />
        {/* No metrics yet → both series flat at 0 (baseline). */}
        <path d="M0 196 L720 196" fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round" />
        <path
          d="M0 198 L720 198"
          fill="none"
          stroke="var(--muted)"
          strokeWidth="1.6"
          strokeDasharray="4 4"
          strokeLinecap="round"
        />
      </svg>
    </>
  );
}

function MeterStat({ icon, label, unit }: { icon: string; label: string; unit: string }) {
  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", gap: 8, color: "var(--muted)", fontSize: 13, marginBottom: 10 }}>
        <Icon name={icon} size={16} />
        {label}
      </div>
      <div className="num" style={{ fontSize: 22, fontWeight: 600 }}>
        0 <span className="muted" style={{ fontSize: 14 }}>/ — {unit}</span>
      </div>
      <div className="meter" style={{ marginTop: 10 }}>
        <span style={{ width: "0%" }} />
      </div>
      <div className="num muted" style={{ fontSize: 12, marginTop: 6 }}>
        指标接入中
      </div>
    </div>
  );
}

// ── Recent activity (workspaces + services, newest first) ─────────────────────
interface Activity {
  name: string;
  type: string;
  at: string;
  phase?: string;
}

function RecentActivity({
  ws,
  svc,
}: {
  ws: UseQueryResult<sdk.ListWorkspacesResponse>;
  svc: UseQueryResult<sdk.ListMlServicesResponse>;
}) {
  const items: Activity[] = [
    ...(ws.data?.items ?? []).map((w) => ({ name: w.name, type: "工作区", at: w.updatedAt, phase: w.phase })),
    ...(svc.data?.items ?? []).map((s) => ({ name: s.name, type: "在线服务", at: s.updatedAt, phase: s.phase })),
  ]
    .sort((a, b) => (b.at || "").localeCompare(a.at || ""))
    .slice(0, 6);

  const stateQ: QueryLike = {
    isLoading: ws.isLoading || svc.isLoading,
    isError: ws.isError || svc.isError,
    error: ws.error || svc.error,
    refetch: ws.refetch,
  };

  return (
    <div className="table-wrap">
      <table className="tbl">
        <tbody>
          {items.map((a) => {
            const st = phaseStatus(a.phase);
            return (
              <tr key={a.type + a.name}>
                <td>
                  <div className="t-name mono">{a.name}</div>
                  <div className="t-sub">
                    {a.type} · {timeAgo(a.at)}
                  </div>
                </td>
                <td style={{ textAlign: "right" }}>
                  <span className={"status status-" + st.cls}>
                    <span className="dot" />
                    {st.label}
                  </span>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      <BlockState q={stateQ} isEmpty={items.length === 0} empty="暂无活动" />
    </div>
  );
}

function phaseStatus(phase?: string): { cls: string; label: string } {
  switch (phase) {
    case "Running":
      return { cls: "running", label: "运行中" };
    case "Ready":
      return { cls: "success", label: "就绪" };
    case "Succeeded":
      return { cls: "success", label: "成功" };
    case "Degraded":
      return { cls: "pending", label: "降级" };
    case "Creating":
    case "Starting":
    case "Pending":
      return { cls: "pending", label: "准备中" };
    default: // Stopped / Failed / Deleting / Deleted / undefined
      return { cls: "stopped", label: phase || "—" };
  }
}

function timeAgo(iso?: string): string {
  if (!iso) return "—";
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "—";
  const sec = Math.max(0, Math.floor((Date.now() - then) / 1000));
  if (sec < 60) return "刚刚";
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min} 分钟前`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr} 小时前`;
  return `${Math.floor(hr / 24)} 天前`;
}
