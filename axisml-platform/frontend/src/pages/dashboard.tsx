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
import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
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
  useClusterUsage,
  useClusterMetric,
  useActivity,
} from "@/api/hooks";
import { USE_MOCK } from "@/api/mock";
import * as sdk from "@/api/generated";
import type { ClusterMeter, MetricPoint } from "@/api/generated";
import { Card, CardContent, CardHeader, CardTitle, CardAction } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Empty, EmptyDescription, EmptyHeader } from "@/components/ui/empty";
import { Separator } from "@/components/ui/separator";
import { Progress } from "@/components/ui/progress";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Area, CartesianGrid, ComposedChart, Line, XAxis, YAxis } from "recharts";
import { cn } from "@/lib/utils";

// 首页 / Dashboard. KPI counts come from the live list endpoints (scoped to the
// active tenant); active-run roll-ups from getTenant. Cluster usage/metrics and
// the recent-activity feed come from the dashboard aggregate endpoints
// (/dashboard/cluster-usage · /cluster-metrics · /activity). Those are declared
// in the contract but not yet implemented by the backend, so against a real
// backend they 501 and the page renders an honest empty / "指标接入中" state
// (see api/hooks useAux); under VITE_USE_MOCK_API the mock router answers them.
export default function Dashboard() {
  const { tenant } = useApp();
  const { toast } = useUI();
  const { t } = useTranslation();
  const qc = useQueryClient();
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
    // The cluster / activity queries live inside child components; invalidate by
    // key prefix so they refetch too.
    ["cluster-usage", "cluster-metric", "activity"].forEach((k) =>
      qc.invalidateQueries({ queryKey: [k] }),
    );
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
              <TabsList>
                <TabsTrigger value="all">{t("dashboard.all")}</TabsTrigger>
                {poolNames.map((name) => (
                  <TabsTrigger key={name} value={name}>
                    {name}
                  </TabsTrigger>
                ))}
              </TabsList>
              <TabsContent value="all" className="mt-4">
                <ClusterPane pool="all" range={range} />
              </TabsContent>
              {poolNames.map((name) => (
                <TabsContent key={name} value={name} className="mt-4">
                  <ClusterPane pool={name} range={range} />
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
            <RecentActivity />
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

// Cluster usage meters from /dashboard/cluster-usage; the GPU util/quota trend
// from /dashboard/cluster-metrics (two series, scoped to the selected range).
// Both degrade to an honest "指标接入中" empty state when the backend 501s.
function ClusterPane({ pool, range }: { pool: string; range: string }) {
  const { t } = useTranslation();
  const usageQ = useClusterUsage(pool);
  const utilQ = useClusterMetric("gpu_util", pool, range);
  const quotaQ = useClusterMetric("gpu_quota", pool, range);

  const meters = usageQ.data?.pools?.flatMap((p) => p.meters) ?? [];
  const gpu = meters.find((m) => m.resource === "gpu");
  const cpu = meters.find((m) => m.resource === "cpu");
  const mem = meters.find((m) => m.resource === "memory");
  const hasUsage = meters.length > 0;
  const trend = zipTrend(utilQ.data?.series, quotaQ.data?.series);

  return (
    <div>
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
        <MeterStat label="GPU" icon={<GpuIcon />} m={gpu} />
        <MeterStat label="CPU" icon={<CpuIcon />} m={cpu} />
        <MeterStat label={t("dashboard.memLabel")} icon={<MemIcon />} m={mem} />
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
      <TrendChart trend={trend} />
      {!hasUsage && <div className="mt-1 text-center text-xs text-muted-foreground">{t("dashboard.metricsSyncing")}</div>}
    </div>
  );
}

// Present a raw ClusterMeter (resource / used / total / unit) — percentage and
// alert state are derived here (presentation), not carried by the API.
function MeterStat({ label, icon, m }: { label: string; icon: ReactNode; m?: ClusterMeter }) {
  const { t } = useTranslation();
  const total = m?.total ?? 0;
  const used = m?.used ?? 0;
  const pct = total === 0 ? 0 : Math.round((used / total) * 100);
  const na = !m || total === 0;
  const fill = na
    ? "[&_[data-slot=progress-indicator]]:bg-success"
    : pct >= 80
      ? "[&_[data-slot=progress-indicator]]:bg-destructive"
      : pct >= 60
        ? "[&_[data-slot=progress-indicator]]:bg-warning"
        : "[&_[data-slot=progress-indicator]]:bg-success";
  const display = m ? (Number.isInteger(used) ? String(used) : used.toFixed(1)) : "0";
  return (
    <div className={na ? "opacity-50" : undefined}>
      <div className="mb-2.5 flex items-center gap-2 text-[13px] text-muted-foreground">
        {icon}
        {label}
      </div>
      <div className="font-mono text-[22px] font-semibold">
        {display}{" "}
        <span className="text-sm font-normal text-muted-foreground">
          / {m ? `${m.total} ${m.unit ?? ""}` : `— ${label === "GPU" ? "卡" : ""}`}
        </span>
      </div>
      <Progress value={pct} className={cn("mt-2.5 h-1.5", fill)} />
      <div className="mt-1.5 font-mono text-xs text-muted-foreground">
        {na ? t("dashboard.noGpu") : `${pct}% ${t("dashboard.utilization")}`}
      </div>
    </div>
  );
}

type TrendPoint = { t: string; util: number; quota: number };

// Zip the gpu_util + gpu_quota metric series into the composed-chart shape.
function zipTrend(util?: MetricPoint[], quota?: MetricPoint[]): TrendPoint[] | undefined {
  if (!util || !util.length) return undefined;
  return util.map((p, i) => ({
    t: dayjs(p.timestamp).format("HH:mm"),
    util: Math.round(p.value),
    quota: Math.round(quota?.[i]?.value ?? 0),
  }));
}

// GPU utilisation trend (filled util area + dashed quota line). recharts
// ComposedChart via the shadcn Chart wrapper.
function TrendChart({ trend }: { trend?: TrendPoint[] }) {
  const { t } = useTranslation();
  const data = trend ?? Array.from({ length: 13 }, (_, i) => ({ t: `${i * 2}:00`, util: 0, quota: 0 }));
  const chartConfig = {
    util: { label: t("dashboard.gpuUtil"), color: "var(--info)" },
    quota: { label: t("dashboard.gpuQuota"), color: "var(--muted-foreground)" },
  } satisfies ChartConfig;
  return (
    <ChartContainer config={chartConfig} className="h-[180px] w-full">
      <ComposedChart data={data} margin={{ left: 0, right: 0, top: 8, bottom: 0 }}>
        <defs>
          <linearGradient id="fillUtil" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="var(--color-util)" stopOpacity={0.16} />
            <stop offset="95%" stopColor="var(--color-util)" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid vertical={false} />
        <XAxis dataKey="t" tickLine={false} axisLine={false} tickMargin={8} minTickGap={28} />
        <YAxis hide domain={[0, 100]} unit="%" />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Area
          dataKey="util"
          type="monotone"
          fill="url(#fillUtil)"
          stroke="var(--color-util)"
          strokeWidth={2}
        />
        <Line
          dataKey="quota"
          type="monotone"
          stroke="var(--color-quota)"
          strokeWidth={1.6}
          strokeDasharray="4 4"
          dot={false}
        />
      </ComposedChart>
    </ChartContainer>
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

// Recent-activity feed from /dashboard/activity (tenant-scoped). Honest empty
// state when the backend 501s or there is no activity.
function RecentActivity() {
  const { t } = useTranslation();
  const q = useActivity();
  const items = (q.data?.items ?? []).slice(0, 6);

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
        <li key={a.id ?? i} className="flex items-center justify-between gap-3 px-5 py-3">
          <div className="min-w-0">
            <div className="truncate font-mono text-sm font-medium">{a.name}</div>
            <div className="text-xs text-muted-foreground">
              {`${a.kind}${a.action ? ` · ${a.action}` : ""} · ${a.timestamp ? dayjs(a.timestamp).fromNow() : "—"}`}
            </div>
          </div>
          <PhaseTag phase={a.phase} />
        </li>
      ))}
    </ul>
  );
}
