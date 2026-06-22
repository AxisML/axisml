import { useState, type ComponentType, type ReactNode } from "react";
import { Link } from "react-router-dom";
import {
  RotateCw,
  Monitor,
  FlaskConical,
  Zap,
  Server,
  Database,
  ArrowRight,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import {
  useWorkspaces,
  useExperiments,
  useJobs,
  useServices,
  useModels,
  useImages,
  useResourcePools,
} from "@/api/hooks";
import { USE_MOCK } from "@/api/mock";
import { clusterUsage, type ClusterUsage, type UsageMetric } from "@/api/mock/data";
import * as sdk from "@/api/generated";
import { Card, CardContent, CardHeader, CardTitle, CardAction } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Empty, EmptyDescription, EmptyHeader } from "@/components/ui/empty";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";

// 首页 / Dashboard. KPI counts and recent activity come from the live list
// endpoints (scoped to the active tenant); active-run roll-ups come from
// getTenant. Cluster utilisation has no metrics source in the platform contract:
// in the real app it renders an honest zero / "指标接入中" state (frontend.md §6);
// under VITE_USE_MOCK_API it reads the demo fixtures so the page matches the
// product prototype.
export default function Dashboard() {
  const { tenant } = useApp();
  const { toast } = useUI();
  const { t } = useTranslation();
  const [range, setRange] = useState("24h");

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
          <ToggleGroup type="single" value={range} onValueChange={(v) => v && setRange(v)}>
            <ToggleGroupItem value="1h">1h</ToggleGroupItem>
            <ToggleGroupItem value="24h">24h</ToggleGroupItem>
            <ToggleGroupItem value="7d">7d</ToggleGroupItem>
          </ToggleGroup>
          <Button variant="outline" onClick={refresh}>
            <RotateCw data-icon="inline-start" />
            {t("dashboard.refresh")}
          </Button>
        </div>
      }
    >
      {/* KPI row */}
      <div className="mb-5 grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-5">
        <Kpi to="/workspaces" icon={Monitor} label={t("dashboard.kpiWorkspace")} value={countText(ws)} foot={t("dashboard.running", { count: wsRunning })} />
        <Kpi to="/experiments" icon={FlaskConical} label={t("dashboard.kpiExperiment")} value={countText(exp)} foot={t("dashboard.running", { count: runText(statsQ.data?.activeExperimentRuns, statsQ) })} />
        <Kpi to="/jobs" icon={Zap} label={t("dashboard.kpiJob")} value={countText(jobs)} foot={t("dashboard.running", { count: runText(statsQ.data?.activeJobRuns, statsQ) })} />
        <Kpi to="/services" icon={Server} label={t("dashboard.kpiService")} value={countText(svc)} foot={t("dashboard.readyDegraded", { ready: svcReady, degraded: svcDegraded })} />
        <Kpi to="/models" icon={Database} label={t("dashboard.kpiAsset")} value={assetErr ? "—" : String(assetTotal)} foot={t("dashboard.modelsImages", { models: models.data?.count ?? 0, images: images.data?.count ?? 0 })} />
      </div>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <Card className="gap-0 xl:col-span-2">
          <CardHeader className="border-b">
            <CardTitle>{t("dashboard.clusterUsage")}</CardTitle>
            <CardAction>
              <span className="text-xs text-muted-foreground">
                {t(USE_MOCK ? "dashboard.metricsHintMock" : "dashboard.metricsHint")}
              </span>
            </CardAction>
          </CardHeader>
          <CardContent className="pt-4">
            <Tabs defaultValue="all">
              <TabsList className="mb-4">
                <TabsTrigger value="all">{t("dashboard.all")}</TabsTrigger>
                {poolNames.map((name) => (
                  <TabsTrigger key={name} value={name}>
                    {name}
                  </TabsTrigger>
                ))}
              </TabsList>
              <TabsContent value="all">
                <ClusterPane pool="all" />
              </TabsContent>
              {poolNames.map((name) => (
                <TabsContent key={name} value={name}>
                  <ClusterPane pool={name} />
                </TabsContent>
              ))}
            </Tabs>
          </CardContent>
        </Card>

        <Card className="gap-0">
          <CardHeader className="border-b">
            <CardTitle>{t("dashboard.recentActivity")}</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <RecentActivity ws={ws} svc={svc} />
          </CardContent>
        </Card>
      </div>
    </PageContainer>
  );
}

// Focal KPI card — the prototype's `.kpi.focal`: a hover lift that reveals a
// forward arrow. Geist skin: ink value, hairline card, no chromatic fill.
function Kpi({
  to,
  icon: Icon,
  label,
  value,
  foot,
}: {
  to: string;
  icon: ComponentType<{ className?: string }>;
  label: string;
  value: ReactNode;
  foot: ReactNode;
}) {
  return (
    <Link
      to={to}
      className="group relative block h-full overflow-hidden rounded-lg border bg-card p-5 shadow-xs transition-all duration-150 hover:-translate-y-0.5 hover:border-foreground/20 hover:shadow-md"
    >
      <div className="flex items-center justify-between">
        <span className="text-[13px] text-muted-foreground">{label}</span>
        <span className="grid size-[30px] place-items-center rounded-md bg-muted text-muted-foreground">
          <Icon className="size-4" />
        </span>
      </div>
      <div className="mt-3 font-mono text-3xl leading-none font-semibold tracking-tight">{value}</div>
      <div className="mt-3 text-xs text-muted-foreground">{foot}</div>
      <ArrowRight className="absolute right-4 bottom-4 size-4 -translate-x-1 text-muted-foreground opacity-0 transition-all duration-150 group-hover:translate-x-0 group-hover:opacity-100" />
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

// Cluster usage — demo fixtures under mock, honest zero otherwise.
function ClusterPane({ pool }: { pool: string }) {
  const { t } = useTranslation();
  const usage: ClusterUsage | null = USE_MOCK ? clusterUsage(pool) : null;
  return (
    <div>
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
        <MeterStat label="GPU" icon={<GpuIcon />} m={usage?.gpu} />
        <MeterStat label="CPU" icon={<CpuIcon />} m={usage?.cpu} />
        <MeterStat label={t("dashboard.memLabel")} icon={<MemIcon />} m={usage?.mem} />
      </div>
      <Separator className="my-4" />
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm font-semibold">{t("dashboard.gpuTrend")}</span>
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <i className="inline-block size-2 rounded-full bg-info" />
            {t("dashboard.gpuUtil")}
          </span>
          <span className="flex items-center gap-1.5">
            <i className="inline-block size-2 rounded-full bg-muted-foreground" />
            {t("dashboard.gpuQuota")}
          </span>
        </div>
      </div>
      <TrendChart trend={usage?.trend} />
      {!usage && <div className="mt-1 text-center text-xs text-muted-foreground">{t("dashboard.metricsSyncing")}</div>}
    </div>
  );
}

function MeterStat({ label, icon, m }: { label: string; icon: ReactNode; m?: UsageMetric }) {
  const { t } = useTranslation();
  const fill = m?.state === "hot" ? "bg-destructive" : m?.state === "warn" ? "bg-warning" : "bg-success";
  const na = m?.state === "na";
  return (
    <div className={na ? "opacity-50" : undefined}>
      <div className="mb-2.5 flex items-center gap-2 text-[13px] text-muted-foreground">
        {icon}
        {label}
      </div>
      <div className="font-mono text-[22px] font-semibold">
        {m ? m.display : "0"}{" "}
        <span className="text-sm font-normal text-muted-foreground">
          / {m ? `${m.total} ${m.unit}` : `— ${label === "GPU" ? "卡" : ""}`}
        </span>
      </div>
      <div className="mt-2.5 h-[7px] overflow-hidden rounded-full bg-muted">
        <div className={cn("h-full rounded-full", fill)} style={{ width: `${m?.pct ?? 0}%` }} />
      </div>
      <div className="mt-1.5 font-mono text-xs text-muted-foreground">
        {na ? t("dashboard.noGpu") : m ? `${m.pct}% ${t("dashboard.utilization")}` : t("dashboard.metricsSyncing")}
      </div>
    </div>
  );
}

// Hand-drawn SVG trend (area + util line + dashed concurrency line), matching the
// prototype's inline chart. viewBox stretches to fill width.
function TrendChart({ trend }: { trend?: ClusterUsage["trend"] }) {
  const W = 720;
  const H = 200;
  const data = trend ?? Array.from({ length: 13 }, (_, i) => ({ t: `${i * 2}:00`, util: 0, tasks: 0 }));
  const n = data.length;
  const x = (i: number) => (i / (n - 1)) * W;
  const yUtil = (v: number) => H - (Math.max(0, Math.min(100, v)) / 100) * (H - 16) - 6;
  const maxTasks = Math.max(...data.map((d) => d.tasks), 1);
  const yTask = (v: number) => H - (v / maxTasks) * (H * 0.32) - 16;
  const utilPts = data.map((d, i) => `${x(i)} ${yUtil(d.util)}`);
  const linePath = "M" + utilPts.join(" L");
  const areaPath = `M0 ${H} L${utilPts.join(" L")} L${W} ${H} Z`;
  const taskPath = "M" + data.map((d, i) => `${x(i)} ${yTask(d.tasks)}`).join(" L");
  return (
    <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="h-[180px] w-full">
      <defs>
        <linearGradient id="gtrend" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0" stopColor="var(--info)" stopOpacity="0.16" />
          <stop offset="1" stopColor="var(--info)" stopOpacity="0" />
        </linearGradient>
      </defs>
      {[50, 100, 150].map((y) => (
        <line key={y} x1="0" y1={y} x2={W} y2={y} stroke="var(--border)" strokeWidth="1" />
      ))}
      <path d={areaPath} fill="url(#gtrend)" />
      <path d={linePath} fill="none" stroke="var(--info)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      <path d={taskPath} fill="none" stroke="var(--muted-foreground)" strokeWidth="1.6" strokeDasharray="4 4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

const ICO = "size-4";
function GpuIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" className={ICO}>
      <rect x="2" y="6" width="20" height="12" rx="2" />
      <circle cx="8" cy="12" r="2.5" />
      <path d="M14 10h5M14 14h5" />
    </svg>
  );
}
function CpuIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" className={ICO}>
      <rect x="6" y="6" width="12" height="12" rx="1.5" />
      <path d="M9 1v3M15 1v3M9 20v3M15 20v3M1 9h3M1 15h3M20 9h3M20 15h3" />
    </svg>
  );
}
function MemIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" className={ICO}>
      <rect x="3" y="7" width="18" height="10" rx="1.5" />
      <path d="M7 7V5M11 7V5M15 7V5M19 7V5" />
    </svg>
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
    return (
      <Empty className="py-10">
        <EmptyHeader>
          <EmptyDescription>{t("dashboard.noActivity")}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }
  return (
    <ul className="divide-y">
      {items.map((a, i) => (
        <li key={i} className="flex items-center justify-between gap-3 px-5 py-3">
          <div className="min-w-0">
            <div className="truncate font-mono text-sm font-medium">{a.name}</div>
            <div className="text-xs text-muted-foreground">{`${a.type} · ${a.at ? dayjs(a.at).fromNow() : "—"}`}</div>
          </div>
          <PhaseTag phase={a.phase} />
        </li>
      ))}
    </ul>
  );
}
