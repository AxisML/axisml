import { useState, type ReactNode } from "react";
import { useParams } from "react-router-dom";
import {
  Maximize2,
  Pencil,
  Pause,
  Play,
  Trash2,
  Copy,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery, useQueries } from "@tanstack/react-query";
import { Area, CartesianGrid, ComposedChart, Line, XAxis, YAxis } from "recharts";
import { useApp } from "@/app/store";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { PodLogPane } from "@/components/pod-log-pane";
import { usePodLogs } from "@/lib/use-pod-logs";
import { BackLink } from "@/components/back-link";
import { MonoChip } from "@/components/mono-chip";
import { Descriptions, Desc } from "@/components/descriptions";
import { PageLoading, DetailError } from "@/components/page-state";
import { DataTable, type Column } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { FormDrawer } from "@/components/form-drawer";
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { fmtDateTime, fmtDateTimeSec } from "@/lib/format";

const INVALIDATE = [["mlservices"]];
const RUNNING_PHASES = new Set(["Ready", "Degraded", "Creating", "Pending"]);

export default function ServiceDetail() {
  const { name = "" } = useParams<{ name: string }>();
  const { tenant } = useApp();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<"edit" | "scale" | null>(null);
  const [tab, setTab] = useState("info");

  const q = useQuery({
    queryKey: ["mlservices", tenant, name],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getMlService({ path: { name } });
      if (error) throw error;
      return data as sdk.MlService;
    },
  });

  const del = useApiMutation(() => sdk.deleteMlService({ path: { name } }), {
    invalidate: INVALIDATE,
    success: t("services.deleted"),
  });
  const start = useApiMutation(() => sdk.startMlService({ path: { name } }), {
    invalidate: INVALIDATE,
    success: t("services.starting"),
  });
  const stop = useApiMutation(() => sdk.stopMlService({ path: { name } }), {
    invalidate: INVALIDATE,
    success: t("services.stopping"),
  });

  const breadcrumb = [t("nav.serviceCenter"), t("nav.services"), name];
  const backLink = <BackLink to="/services">{t("services.backToList")}</BackLink>;

  if (q.isLoading) {
    return (
      <PageContainer breadcrumb={breadcrumb} title={name} subtitle={backLink}>
        <PageLoading />
      </PageContainer>
    );
  }

  if (q.isError || !q.data) {
    return (
      <PageContainer breadcrumb={breadcrumb} title={name} subtitle={backLink}>
        <DetailError message={t("services.notFound")} />
      </PageContainer>
    );
  }

  const svc = q.data;
  const running = RUNNING_PHASES.has(svc.phase ?? "");

  const onDelete = () =>
    confirm({
      title: t("services.deleteTitle", { name }),
      desc: t("services.deleteDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate(undefined),
    });

  return (
    <PageContainer
      breadcrumb={breadcrumb}
      title={
        <span className="flex items-center gap-3">
          <span className="font-mono">{name}</span>
          <PhaseTag phase={svc.phase} />
        </span>
      }
      subtitle={backLink}
      extra={
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setDrawer("edit")}>
            <Pencil data-icon="inline-start" />
            {t("common.edit")}
          </Button>
          <Button variant="outline" onClick={() => setDrawer("scale")}>
            <Maximize2 data-icon="inline-start" />
            {t("services.scale")}
          </Button>
          {running ? (
            <Button variant="outline" disabled={stop.isPending} onClick={() => stop.mutate(undefined)}>
              {stop.isPending ? <Spinner data-icon="inline-start" /> : <Pause data-icon="inline-start" />}
              {t("services.stop")}
            </Button>
          ) : (
            <Button variant="outline" disabled={start.isPending} onClick={() => start.mutate(undefined)}>
              {start.isPending ? <Spinner data-icon="inline-start" /> : <Play data-icon="inline-start" />}
              {t("services.start")}
            </Button>
          )}
          <Button variant="outline" className="text-destructive" onClick={onDelete}>
            <Trash2 data-icon="inline-start" />
            {t("common.delete")}
          </Button>
        </div>
      }
    >
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="info">{t("services.tabInfo")}</TabsTrigger>
          <TabsTrigger value="mon">{t("services.tabMonitor")}</TabsTrigger>
          <TabsTrigger value="pods">{t("services.tabPods")}</TabsTrigger>
          <TabsTrigger value="log">{t("services.tabLog")}</TabsTrigger>
          <TabsTrigger value="ev">{t("services.tabEvents")}</TabsTrigger>
        </TabsList>
        <TabsContent value="info" className="mt-4">
          <InfoPane svc={svc} />
        </TabsContent>
        <TabsContent value="mon" className="mt-4">
          <MonitorPane name={svc.name} />
        </TabsContent>
        <TabsContent value="pods" className="mt-4">
          <PodsPane name={svc.name} />
        </TabsContent>
        <TabsContent value="log" className="mt-4">
          <LogPane name={svc.name} />
        </TabsContent>
        <TabsContent value="ev" className="mt-4">
          <EventsPane name={svc.name} />
        </TabsContent>
      </Tabs>

      {drawer === "edit" && <EditSvcDrawer svc={svc} onClose={() => setDrawer(null)} />}
      {drawer === "scale" && <ScaleDrawer svc={svc} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Overview ──────────────────────────────────────────────────────────────────
function InfoPane({ svc }: { svc: sdk.MlService }) {
  const { t } = useTranslation();
  const { toast } = useUI();

  const dash = <span className="text-muted-foreground">—</span>;
  const chip = (v?: string) => (v ? <MonoChip>{v}</MonoChip> : dash);

  return (
    <Card className="p-0">
      <CardHeader className="border-b py-4">
        <CardTitle>{t("services.configInfo")}</CardTitle>
      </CardHeader>
      <CardContent className="py-4">
        <Descriptions columns="single">
          <Desc label={t("services.dName")}>{chip(svc.name)}</Desc>
          <Desc label={t("services.dDesc")}>{svc.description ?? dash}</Desc>
          <Desc label={t("services.dModelVersion")}>
            {svc.modelName
              ? chip(svc.modelVersion ? `${svc.modelName}@${svc.modelVersion}` : svc.modelName)
              : dash}
          </Desc>
          <Desc label={t("services.dImage")}>{chip(svc.image)}</Desc>
          <Desc label={t("services.dPool")}>{chip(svc.poolName)}</Desc>
          <Desc label={t("services.dUnit")}>{chip(svc.unitName)}</Desc>
          <Desc label={t("services.dReplicas")}>
            <span className="font-mono">
              {t("services.replicasReady", { ready: svc.readyReplicas ?? 0, total: svc.replicas ?? 0 })}
            </span>
          </Desc>
          <Desc label={t("services.dPorts")}>
            {svc.ports && svc.ports.length > 0 ? (
              <span className="flex flex-wrap gap-1.5">
                {svc.ports.map((p) => (
                  <MonoChip key={`${p.name}:${p.port}`}>
                    {p.name} : {p.port}
                  </MonoChip>
                ))}
              </span>
            ) : (
              dash
            )}
          </Desc>
          <Desc label={t("services.dAccess")}>
            {svc.accessUrl ? (
              <span className="flex items-center gap-2">
                <MonoChip>{svc.accessUrl}</MonoChip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => {
                        void navigator.clipboard?.writeText(svc.accessUrl ?? "");
                        toast(t("services.accessCopied"));
                      }}
                    >
                      <Copy />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("services.copyAccess")}</TooltipContent>
                </Tooltip>
              </span>
            ) : (
              dash
            )}
          </Desc>
          <Desc label={t("services.dCreator")}>
            {svc.owner ? <span className="font-mono">{svc.owner}</span> : dash}
          </Desc>
          <Desc label={t("services.dCreatedAt")}>
            {svc.createdAt ? (
              <span className="font-mono text-muted-foreground">{fmtDateTimeSec(svc.createdAt)}</span>
            ) : (
              dash
            )}
          </Desc>
        </Descriptions>
      </CardContent>
    </Card>
  );
}

// ── Monitoring: Prometheus metric grid (mirrors the prototype's 监控 panel) ──────
// A range segmented control over a 2-col grid of metric cards. Each card is a real
// getMlServiceMetrics read; when Prometheus has no feed the card renders an honest
// "指标接入中" placeholder rather than fabricated data.
const MONITOR_RANGES = ["5m", "1h", "24h"] as const;

function MonitorPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const [range, setRange] = useState<string>("1h");
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">{t("services.monitorHint")}</span>
        <ToggleGroup type="single" value={range} onValueChange={(v) => v && setRange(v)}>
          {MONITOR_RANGES.map((r) => (
            <ToggleGroupItem key={r} value={r}>
              {r}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <MetricCard name={name} range={range} metric="request_rate" title={t("services.mQps")} color="var(--info)" area />
        <LatencyCard name={name} range={range} />
        <MetricCard name={name} range={range} metric="error_rate" title={t("services.mError")} color="var(--destructive)" area />
        <MetricCard name={name} range={range} metric="cpu_util" title={t("services.mCpu")} color="var(--info)" area />
      </div>
    </div>
  );
}

function fmtMetric(v: number, unit?: string): string {
  const num = Number.isInteger(v) ? v : Math.round(v * 100) / 100;
  if (!unit) return String(num);
  if (unit === "%") return `${num}%`;
  return `${num} ${unit}`;
}

function MetricEmpty() {
  const { t } = useTranslation();
  return (
    <Empty className="py-10">
      <EmptyHeader>
        <EmptyTitle className="text-sm font-normal text-muted-foreground">
          {t("services.monitorEmpty")}
        </EmptyTitle>
      </EmptyHeader>
    </Empty>
  );
}

function MetricChartFrame({
  loading,
  empty,
  children,
}: {
  loading: boolean;
  empty: boolean;
  children: ReactNode;
}) {
  if (loading)
    return (
      <div className="grid h-[140px] place-items-center">
        <Spinner className="size-6 text-muted-foreground" />
      </div>
    );
  if (empty) return <MetricEmpty />;
  return <>{children}</>;
}

// Single-series card (QPS / error rate / CPU) — area chart with the latest value.
function MetricCard({
  name,
  range,
  metric,
  title,
  color,
  area,
}: {
  name: string;
  range: string;
  metric: sdk.MlServiceMetricName;
  title: string;
  color: string;
  area?: boolean;
}) {
  const { tenant } = useApp();
  const q = useQuery({
    queryKey: ["mlservices", tenant, name, "metrics", metric, range],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getMlServiceMetrics({ path: { name }, query: { metric, range } });
      if (error) throw error;
      return data;
    },
  });
  const series = q.data?.series ?? [];
  const data = series.map((p) => ({ t: p.timestamp, v: p.value }));
  const latest = series.at(-1)?.value;
  const fillId = `fill-${metric}`;
  const config = { v: { label: title, color } } satisfies ChartConfig;

  return (
    <Card className="gap-0">
      <CardHeader className="border-b py-3">
        <CardTitle className="text-sm">{title}</CardTitle>
        <CardAction>
          <span className="font-mono text-sm font-semibold">
            {latest != null ? fmtMetric(latest, q.data?.unit) : "—"}
          </span>
        </CardAction>
      </CardHeader>
      <CardContent className="pt-4">
        <MetricChartFrame loading={q.isLoading} empty={data.length === 0}>
          <ChartContainer config={config} className="h-[140px] w-full">
            <ComposedChart data={data} margin={{ left: 0, right: 0, top: 8, bottom: 0 }}>
              <defs>
                <linearGradient id={fillId} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={color} stopOpacity={0.18} />
                  <stop offset="95%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid vertical={false} />
              <XAxis dataKey="t" hide />
              <YAxis hide />
              <ChartTooltip content={<ChartTooltipContent />} />
              {area ? (
                <Area dataKey="v" type="monotone" fill={`url(#${fillId})`} stroke={color} strokeWidth={2} />
              ) : (
                <Line dataKey="v" type="monotone" stroke={color} strokeWidth={2} dot={false} />
              )}
            </ComposedChart>
          </ChartContainer>
        </MetricChartFrame>
      </CardContent>
    </Card>
  );
}

// Latency card — overlays p50 / p95 / p99 (three percentile reads) with a legend.
const LATENCY_PCTS = [
  { p: "p50", color: "var(--muted-foreground)" },
  { p: "p95", color: "var(--info)" },
  { p: "p99", color: "var(--destructive)" },
] as const;

function LatencyCard({ name, range }: { name: string; range: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const results = useQueries({
    queries: LATENCY_PCTS.map(({ p }) => ({
      queryKey: ["mlservices", tenant, name, "metrics", "latency", p, range],
      enabled: tenant !== "" && name !== "",
      queryFn: async () => {
        const { data, error } = await sdk.getMlServiceMetrics({
          path: { name },
          query: { metric: "latency" as const, range, percentile: p },
        });
        if (error) throw error;
        return data;
      },
    })),
  });

  const loading = results.some((r) => r.isLoading);
  const len = Math.max(...results.map((r) => r.data?.series?.length ?? 0), 0);
  const data = Array.from({ length: len }, (_, i) => {
    const row: Record<string, string | number> = { t: results[0]?.data?.series?.[i]?.timestamp ?? String(i) };
    LATENCY_PCTS.forEach(({ p }, qi) => {
      const v = results[qi]?.data?.series?.[i]?.value;
      if (v != null) row[p] = v;
    });
    return row;
  });
  const config = Object.fromEntries(
    LATENCY_PCTS.map(({ p, color }) => [p, { label: p, color }]),
  ) satisfies ChartConfig;

  return (
    <Card className="gap-0">
      <CardHeader className="border-b py-3">
        <CardTitle className="text-sm">{t("services.mLatency")}</CardTitle>
        <CardAction>
          <div className="flex items-center gap-3 text-xs text-muted-foreground">
            {LATENCY_PCTS.map(({ p, color }) => (
              <span key={p} className="flex items-center gap-1.5">
                <i className="inline-block size-2 rounded-full" style={{ background: color }} />
                {p}
              </span>
            ))}
          </div>
        </CardAction>
      </CardHeader>
      <CardContent className="pt-4">
        <MetricChartFrame loading={loading} empty={data.length === 0}>
          <ChartContainer config={config} className="h-[140px] w-full">
            <ComposedChart data={data} margin={{ left: 0, right: 0, top: 8, bottom: 0 }}>
              <CartesianGrid vertical={false} />
              <XAxis dataKey="t" hide />
              <YAxis hide />
              <ChartTooltip content={<ChartTooltipContent />} />
              {LATENCY_PCTS.map(({ p, color }) => (
                <Line key={p} dataKey={p} type="monotone" stroke={color} strokeWidth={1.8} dot={false} />
              ))}
            </ComposedChart>
          </ChartContainer>
        </MetricChartFrame>
      </CardContent>
    </Card>
  );
}

// ── Pods ───────────────────────────────────────────────────────────────────────
function PodsPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const q = useQuery({
    queryKey: ["mlservices", tenant, name, "pods"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.listMlServicePods({ path: { name } });
      if (error) throw error;
      return data;
    },
  });
  const columns: Column<sdk.Pod>[] = [
    {
      key: "name",
      title: t("services.colPod"),
      render: (p) => <span className="font-mono">{p.name}</span>,
    },
    {
      key: "phase",
      title: t("services.colPhase"),
      width: 120,
      render: (p) => <PhaseTag phase={p.phase} />,
    },
    {
      key: "node",
      title: t("services.colNode"),
      width: 160,
      render: (p) => <span className="font-mono">{p.nodeName || "—"}</span>,
    },
    {
      key: "restarts",
      title: t("services.colRestarts"),
      width: 90,
      align: "right",
      render: (p) => <span className="font-mono">{p.restartCount ?? 0}</span>,
    },
    {
      key: "started",
      title: t("services.colStarted"),
      width: 170,
      render: (p) => <span className="text-muted-foreground">{fmtDateTime(p.startedAt)}</span>,
    },
  ];
  return (
    <Card className="overflow-hidden p-0">
      <DataTable
        columns={columns}
        data={q.data?.items ?? []}
        rowKey={(p) => p.name}
        loading={q.isLoading}
        error={q.isError}
        empty={t("services.podsEmpty")}
      />
    </Card>
  );
}

// ── Logs (shared dark LogViewer) ─────────────────────────────────────────────────
function LogPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const logs = usePodLogs({
    queryKey: ["mlservices", tenant, name],
    enabled: tenant !== "" && name !== "",
    listPods: async () => {
      const { data, error } = await sdk.listMlServicePods({ path: { name } });
      if (error) throw error;
      return data;
    },
    getLogs: async (pod) => {
      const { data, error } = await sdk.getMlServicePodLogs({ path: { name, pod } });
      if (error) throw error;
      return data as unknown as string;
    },
    streamPath: (pod) => `/api/v1/mlservices/${name}/pods/${encodeURIComponent(pod)}/logs?follow=true`,
  });
  return <PodLogPane logs={logs} emptyText={t("services.logEmpty")} />;
}

// ── Events (timeline, mirrors the workspace events pane) ─────────────────────────
function EventsPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const q = useQuery({
    queryKey: ["mlservices", tenant, name, "events"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getMlServiceEvents({ path: { name } });
      if (error) throw error;
      return data;
    },
  });

  if (q.isLoading) {
    return (
      <Card>
        <CardContent>
          <div className="grid place-items-center py-10">
            <Spinner className="size-7 text-muted-foreground" />
          </div>
        </CardContent>
      </Card>
    );
  }
  const items = q.data?.items ?? [];
  if (q.isError || items.length === 0) {
    return (
      <Card>
        <Empty>
          <EmptyHeader>
            <EmptyTitle>{q.isError ? t("common.loadFailed") : t("services.eventsEmpty")}</EmptyTitle>
          </EmptyHeader>
        </Empty>
      </Card>
    );
  }
  return (
    <Card>
      <CardContent>
        <div className="flex flex-col gap-4">
          {items.map((e, i) => (
            <div key={`${e.reason}-${i}`} className="flex gap-3">
              <div className="flex flex-col items-center pt-1.5">
                <span
                  className={
                    "size-2 shrink-0 rounded-full " +
                    (e.type === "Warning" ? "bg-warning" : "bg-info")
                  }
                />
                {i < items.length - 1 && <span className="mt-1 w-px flex-1 bg-border" />}
              </div>
              <div className="min-w-0 pb-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium text-foreground">{e.reason}</span>
                  <Badge variant={e.type === "Warning" ? "warning" : "secondary"}>{e.type}</Badge>
                  <span className="font-mono text-xs text-muted-foreground">
                    {fmtDateTimeSec(e.lastTimestamp)}
                  </span>
                </div>
                <div className="mt-0.5 text-sm text-muted-foreground">{e.message}</div>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

// ── Edit drawer (display metadata only) ───────────────────────────────────────
function EditSvcDrawer({ svc, onClose }: { svc: sdk.MlService; onClose: () => void }) {
  const { t } = useTranslation();
  const [displayName, setDisplayName] = useState(svc.displayName ?? "");
  const [description, setDescription] = useState(svc.description ?? "");
  const update = useApiMutation(
    (body: sdk.MlServicePatchRequest) => sdk.updateMlService({ path: { name: svc.name }, body }),
    { invalidate: [["mlservices"], ["mlservices", svc.tenantName, svc.name]], success: t("services.saved") },
  );

  const submit = () =>
    update.mutate(
      {
        displayName: displayName.trim() || undefined,
        description: description.trim() || undefined,
      },
      { onSuccess: onClose },
    );

  return (
    <FormDrawer
      title={t("services.drawerEdit")}
      subtitle={<span className="font-mono">{svc.name}</span>}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("common.save")}
      submitting={update.isPending}
    >
      <p className="mb-4 text-sm text-muted-foreground">{t("services.editNote")}</p>
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="svc-display">{t("services.fDisplayName")}</FieldLabel>
          <Input
            id="svc-display"
            placeholder={t("services.fDisplayNamePlaceholder")}
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="svc-edit-desc">{t("services.fDesc")}</FieldLabel>
          <Textarea
            id="svc-edit-desc"
            rows={2}
            placeholder={t("services.fDescPlaceholder")}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </Field>
      </FieldGroup>
    </FormDrawer>
  );
}

// ── Scale drawer ──────────────────────────────────────────────────────────────
function ScaleDrawer({ svc, onClose }: { svc: sdk.MlService; onClose: () => void }) {
  const { t } = useTranslation();
  const [replicas, setReplicas] = useState<number>(svc.replicas ?? 0);
  const scale = useApiMutation(
    (body: sdk.MlServiceScaleRequest) => sdk.scaleMlService({ path: { name: svc.name }, body }),
    { invalidate: [["mlservices"], ["mlservices", svc.tenantName, svc.name]], success: t("services.scaleSubmitted") },
  );

  const valid = Number.isInteger(replicas) && replicas >= 0;
  const submit = () => scale.mutate({ replicas }, { onSuccess: onClose });
  const unit = `${svc.poolName ?? "—"}/${svc.unitName ?? "—"}`;

  return (
    <FormDrawer
      title={t("services.drawerScale")}
      subtitle={<span className="font-mono">{svc.name}</span>}
      size="compact"
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("common.save")}
      submitDisabled={!valid || scale.isPending}
      submitting={scale.isPending}
    >
      <p className="mb-5 text-sm text-muted-foreground">{t("services.scaleNote")}</p>
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="svc-scale-replicas">{t("services.fTargetReplicas")}</FieldLabel>
          <Input
            id="svc-scale-replicas"
            type="number"
            min={0}
            className="w-40"
            value={replicas}
            onChange={(e) => setReplicas(Number(e.target.value))}
          />
          <FieldDescription>
            {t("services.scaleHint", {
              ready: `${svc.readyReplicas ?? 0} / ${svc.replicas ?? 0}`,
              unit,
            })}
          </FieldDescription>
        </Field>
      </FieldGroup>
    </FormDrawer>
  );
}
