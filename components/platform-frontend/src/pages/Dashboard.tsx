import { useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { Card, Segmented, Button, Tabs, Progress, List, Empty } from "antd";
import {
  ReloadOutlined,
  DesktopOutlined,
  ExperimentOutlined,
  ThunderboltOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
} from "@ant-design/icons";
import { Area } from "@ant-design/charts";
import { useTranslation } from "react-i18next";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";
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
// endpoints (scoped to the active tenant); active-run roll-ups come from
// getTenant. Cluster utilisation has no metrics source yet → it renders an
// honest zero / "指标接入中" state rather than fabricated numbers (frontend.md §6).
export default function Dashboard() {
  const { tenant } = useApp();
  const { toast } = useUI();
  const { t } = useTranslation();
  const [, setRange] = useState("24h");

  const ws = useWorkspaces();
  const exp = useExperiments();
  const jobs = useJobs();
  const svc = useServices();
  const models = useModels();
  const images = useImages();
  const pools = useResourcePools();

  const statsQ = useQuery({
    queryKey: ["tenant", tenant, "stats"],
    enabled: tenant !== "" && tenant !== "all",
    queryFn: async () => {
      const { data, error } = await sdk.getTenant({ path: { name: tenant } });
      if (error) throw error;
      return data;
    },
  });

  const wsRunning = (ws.data?.items ?? []).filter((w) => w.phase === "Running").length;
  const svcReady = (svc.data?.items ?? []).filter((s) => s.phase === "Ready").length;
  const svcDegraded = (svc.data?.items ?? []).filter((s) => s.phase === "Degraded").length;
  const assetErr = models.isLoading || models.isError || images.isLoading || images.isError;
  const assetTotal = (models.data?.count ?? 0) + (images.data?.count ?? 0);

  const refresh = () => {
    [ws, exp, jobs, svc, models, images, statsQ].forEach((q) => void q.refetch());
    toast(t("dashboard.refreshed"));
  };

  const poolNames = (pools.data?.items ?? []).map((p) => p.name);

  return (
    <PageContainer
      breadcrumb={["AxisML", t("dashboard.home")]}
      title={t("dashboard.home")}
      subtitle={t("dashboard.subtitle")}
      extra={
        <div className="flex items-center gap-3">
          <Segmented options={["1h", "24h", "7d"]} defaultValue="24h" onChange={(v) => setRange(String(v))} />
          <Button icon={<ReloadOutlined />} onClick={refresh}>
            {t("dashboard.refresh")}
          </Button>
        </div>
      }
    >
      {/* KPI row */}
      <div className="mb-5 grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-5">
        <Kpi to="/workspaces" icon={<DesktopOutlined />} label={t("dashboard.kpiWorkspace")} value={countText(ws)} foot={t("dashboard.running", { count: wsRunning })} />
        <Kpi to="/experiments" icon={<ExperimentOutlined />} label={t("dashboard.kpiExperiment")} value={countText(exp)} foot={t("dashboard.running", { count: runText(statsQ.data?.activeExperimentRuns, statsQ) })} />
        <Kpi to="/jobs" icon={<ThunderboltOutlined />} label={t("dashboard.kpiJob")} value={countText(jobs)} foot={t("dashboard.running", { count: runText(statsQ.data?.activeJobRuns, statsQ) })} />
        <Kpi to="/services" icon={<CloudServerOutlined />} label={t("dashboard.kpiService")} value={countText(svc)} foot={t("dashboard.readyDegraded", { ready: svcReady, degraded: svcDegraded })} />
        <Kpi to="/models" icon={<DatabaseOutlined />} label={t("dashboard.kpiAsset")} value={assetErr ? "—" : String(assetTotal)} foot={t("dashboard.modelsImages", { models: models.data?.count ?? 0, images: images.data?.count ?? 0 })} />
      </div>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <Card
          className="xl:col-span-2"
          title={t("dashboard.clusterUsage")}
          extra={<span className="text-xs text-muted">{t("dashboard.metricsHint")}</span>}
        >
          <Tabs
            items={[
              { key: "all", label: t("dashboard.all"), children: <ClusterPane /> },
              ...poolNames.map((name) => ({ key: name, label: name, children: <ClusterPane /> })),
            ]}
          />
        </Card>

        <Card title={t("dashboard.recentActivity")} styles={{ body: { padding: 0 } }}>
          <RecentActivity ws={ws} svc={svc} />
        </Card>
      </div>
    </PageContainer>
  );
}

function Kpi({ to, icon, label, value, foot }: { to: string; icon: ReactNode; label: string; value: ReactNode; foot: ReactNode }) {
  return (
    <Link to={to}>
      <Card hoverable styles={{ body: { padding: 16 } }} className="h-full bg-surface-warm">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm text-fg-2">{label}</span>
          <span className="grid h-7 w-7 place-items-center rounded-md bg-bg text-accent">{icon}</span>
        </div>
        <div className="font-mono text-2xl font-semibold text-fg">{value}</div>
        <div className="mt-1 text-xs text-muted">{foot}</div>
      </Card>
    </Link>
  );
}

function countText(q: UseQueryResult<{ count?: number }>): ReactNode {
  if (q.isLoading) return "…";
  if (q.isError) return "—";
  return String(q.data?.count ?? 0);
}

function runText(n: number | undefined, q: UseQueryResult<unknown>): number | string {
  if (q.isLoading) return "…";
  if (q.isError || n == null) return "—";
  return n;
}

// Cluster usage — honest zero state until a metrics source is wired.
function ClusterPane() {
  const { t } = useTranslation();
  const trend = Array.from({ length: 24 }, (_, i) => ({ t: `${i}:00`, v: 0 }));
  return (
    <div>
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
        <MeterStat label="GPU" unit={t("dashboard.gpuUnit")} />
        <MeterStat label="CPU" unit={t("dashboard.cpuUnit")} />
        <MeterStat label={t("dashboard.memLabel")} unit="TiB" />
      </div>
      <div className="my-4 border-t border-border-soft" />
      <div className="mb-2 flex items-center justify-between">
        <span className="text-sm font-semibold text-fg">{t("dashboard.gpuTrend")}</span>
        <div className="flex items-center gap-4 text-xs text-muted">
          <span className="flex items-center gap-1.5">
            <i className="inline-block h-2 w-2 rounded-full bg-accent" />
            {t("dashboard.gpuUtil")}
          </span>
          <span className="flex items-center gap-1.5">
            <i className="inline-block h-2 w-2 rounded-full bg-muted" />
            {t("dashboard.gpuQuota")}
          </span>
        </div>
      </div>
      <Area
        data={trend}
        xField="t"
        yField="v"
        height={180}
        shapeField="smooth"
        scale={{ y: { domainMin: 0, domainMax: 100 } }}
        axis={{ y: { labelFormatter: (v: number) => `${v}%` } }}
        style={{ fill: "var(--accent)", fillOpacity: 0.08 }}
      />
      <div className="mt-1 text-center text-xs text-muted">{t("dashboard.metricsSyncing")}</div>
    </div>
  );
}

function MeterStat({ label, unit }: { label: string; unit: string }) {
  return (
    <div>
      <div className="mb-2 text-sm text-muted">{label}</div>
      <div className="font-mono text-xl font-semibold text-fg">
        0 <span className="text-sm text-muted">/ — {unit}</span>
      </div>
      <Progress percent={0} showInfo={false} strokeColor="var(--accent)" className="mt-2" />
    </div>
  );
}

interface Activity {
  name: string;
  type: string;
  at?: string;
  phase?: string;
}

function RecentActivity({
  ws,
  svc,
}: {
  ws: UseQueryResult<sdk.ListWorkspacesResponse>;
  svc: UseQueryResult<sdk.ListMlServicesResponse>;
}) {
  const { t } = useTranslation();
  const items: Activity[] = [
    ...(ws.data?.items ?? []).map((w) => ({ name: w.name, type: t("dashboard.typeWorkspace"), at: w.updatedAt, phase: w.phase })),
    ...(svc.data?.items ?? []).map((s) => ({ name: s.name, type: t("dashboard.typeService"), at: s.updatedAt, phase: s.phase })),
  ]
    .sort((a, b) => (b.at || "").localeCompare(a.at || ""))
    .slice(0, 6);

  if (!items.length) {
    return <div className="py-10"><Empty description={t("dashboard.noActivity")} /></div>;
  }
  return (
    <List
      dataSource={items}
      renderItem={(a) => (
        <List.Item className="!px-4">
          <List.Item.Meta
            title={<span className="font-mono text-sm">{a.name}</span>}
            description={`${a.type} · ${a.at ? dayjs(a.at).fromNow() : "—"}`}
          />
          <PhaseTag phase={a.phase} />
        </List.Item>
      )}
    />
  );
}
