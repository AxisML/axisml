import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Copy, Trash2, CircleCheck, TriangleAlert, Undo2, ArrowUpToLine } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery, useQueries } from "@tanstack/react-query";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { BackLink } from "@/components/back-link";
import { Descriptions, Desc } from "@/components/descriptions";
import { PageLoading, DetailError } from "@/components/page-state";
import { DataTable, type Column } from "@/components/data-table";
import { fmtDateTimeSec } from "@/lib/format";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { CanaryRollout } from "@/components/canary-rollout";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Spinner } from "@/components/ui/spinner";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface BackendView {
  serviceName: string;
  role?: sdk.TrafficPolicyBackendRole;
  weight: number;
  actualPct: number;
  ready?: boolean;
}

export default function TrafficDetail() {
  const { name = "" } = useParams();
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { confirm } = useUI();

  const q = useQuery({
    queryKey: ["trafficpolicies", tenant, name],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getTrafficPolicy({ path: { name } });
      if (error) throw error;
      return data;
    },
  });

  const del = useApiMutation(() => sdk.deleteTrafficPolicy({ path: { name } }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.deleted"),
  });

  const backLink = <BackLink to="/traffic">{t("traffic.backToList")}</BackLink>;

  if (q.isLoading) {
    return (
      <PageContainer breadcrumb={[t("nav.serviceCenter"), t("nav.traffic"), name]} title={name} subtitle={backLink}>
        <PageLoading />
      </PageContainer>
    );
  }

  if (q.isError || !q.data) {
    return (
      <PageContainer breadcrumb={[t("nav.serviceCenter"), t("nav.traffic"), name]} title={t("traffic.loadFailedTitle")} subtitle={backLink}>
        <DetailError message={t("common.loadFailed")} />
      </PageContainer>
    );
  }

  const p = q.data;
  const backends: BackendView[] = (p.backends ?? []).map((b) => ({
    serviceName: b.serviceName,
    role: b.role,
    weight: b.weight,
    actualPct: b.actualPct ?? b.weight,
    ready: b.ready,
  }));
  const modeLabel = p.mode === "weighted" ? t("traffic.modeWeighted") : t("traffic.modeCanary");

  const onDelete = () =>
    confirm({
      title: t("traffic.deleteTitle", { name: p.name }),
      desc: t("traffic.detailDeleteDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate(undefined),
    });

  return (
    <PageContainer
      breadcrumb={[t("nav.serviceCenter"), t("nav.traffic"), p.name]}
      title={
        <span className="flex items-center gap-3">
          <span className="font-mono">{p.name}</span>
          <PhaseTag phase={p.phase} />
          <Badge variant="outline">{modeLabel}</Badge>
        </span>
      }
      subtitle={backLink}
      extra={
        <Button variant="outline" className="text-destructive" disabled={del.isPending} onClick={onDelete}>
          {del.isPending ? <Spinner data-icon="inline-start" /> : <Trash2 data-icon="inline-start" />}
          {t("traffic.delete")}
        </Button>
      }
    >
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">{t("traffic.tabOverview")}</TabsTrigger>
          <TabsTrigger value="dist">{t("traffic.tabDistribution")}</TabsTrigger>
          <TabsTrigger value="monitor">{t("traffic.tabMonitor")}</TabsTrigger>
          <TabsTrigger value="events">{t("traffic.tabEvents")}</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="mt-4">
          <Overview policy={p} backendCount={backends.length} />
        </TabsContent>
        <TabsContent value="dist" className="mt-4">
          {p.mode === "canary" ? (
            <CanaryDistribution name={p.name} initial={p.canaryPercent ?? 0} backends={backends} />
          ) : (
            <WeightedDistribution name={p.name} backends={backends} />
          )}
        </TabsContent>
        <TabsContent value="monitor" className="mt-4">
          <MonitorPane name={p.name} backends={backends} />
        </TabsContent>
        <TabsContent value="events" className="mt-4">
          <Events name={p.name} />
        </TabsContent>
      </Tabs>
    </PageContainer>
  );
}

// ── Overview ──────────────────────────────────────────────────────────────────
function Overview({ policy, backendCount }: { policy: sdk.TrafficPolicy; backendCount: number }) {
  const { t } = useTranslation();
  const { toast } = useUI();
  const endpoint = policy.accessUrl ?? policy.endpoint?.path;
  const modeLabel = policy.mode === "weighted" ? t("traffic.modeWeighted") : t("traffic.modeCanary");

  const copy = () => {
    if (!endpoint) return;
    void navigator.clipboard?.writeText(endpoint);
    toast(t("traffic.endpointCopied"));
  };

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>{t("traffic.policyInfo")}</CardTitle>
      </CardHeader>
      <CardContent>
        <Descriptions columns="single">
          <Desc label={t("traffic.fieldName")}>
            <span className="font-mono">{policy.name}</span>
          </Desc>
          <Desc label={t("traffic.fieldDesc")}>{policy.description ?? policy.displayName ?? "—"}</Desc>
          <Desc label={t("traffic.fieldMode")}>
            <Badge variant="outline">{modeLabel}</Badge>
          </Desc>
          <Desc label={t("traffic.fieldEndpoint")}>
            {endpoint ? (
              <span className="inline-flex items-center gap-1">
                <span className="font-mono">{endpoint}</span>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="ghost" size="icon-sm" onClick={copy}>
                      <Copy />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("traffic.copyEndpoint")}</TooltipContent>
                </Tooltip>
              </span>
            ) : (
              "—"
            )}
          </Desc>
          <Desc label={t("traffic.fieldBackendCount")}>
            <span className="font-mono">{backendCount}</span>
          </Desc>
          <Desc label={t("traffic.fieldOwner")}>{policy.owner || "—"}</Desc>
          <Desc label={t("traffic.fieldCreatedAt")}>
            <span className="font-mono">{fmtDateTimeSec(policy.createdAt)}</span>
          </Desc>
        </Descriptions>
      </CardContent>
    </Card>
  );
}

// ── Backend distribution table (shared) ───────────────────────────────────────
function roleLabel(t: (k: string) => string, role?: sdk.TrafficPolicyBackendRole): string {
  if (role === "stable") return t("traffic.roleStable");
  if (role === "canary") return t("traffic.roleCanary");
  return t("traffic.roleMember");
}

function backendColumns(
  t: (k: string) => string,
  weightCol: Column<BackendView>,
): Column<BackendView>[] {
  return [
    {
      key: "serviceName",
      title: t("traffic.colService"),
      render: (r) => (
        <Link to={`/services/${r.serviceName}`} className="font-mono text-foreground hover:text-info hover:underline">
          {r.serviceName}
        </Link>
      ),
    },
    {
      key: "role",
      title: t("traffic.colRole"),
      width: 90,
      render: (r) => <Badge variant="outline">{roleLabel(t, r.role)}</Badge>,
    },
    weightCol,
    {
      key: "actualPct",
      title: t("traffic.colActualPct"),
      width: 220,
      render: (r) => (
        <div className="flex items-center gap-2">
          <Progress value={r.actualPct} className="flex-1" />
          <span className="w-10 text-right font-mono text-xs">{r.actualPct}%</span>
        </div>
      ),
    },
    {
      key: "ready",
      title: t("traffic.colBackendStatus"),
      width: 110,
      render: (r) =>
        r.ready ? (
          <Badge variant="success">{t("traffic.backendReady")}</Badge>
        ) : (
          <Badge variant="outline">{t("traffic.backendNotReady")}</Badge>
        ),
    },
  ];
}

// ── Canary distribution ───────────────────────────────────────────────────────
function CanaryDistribution({ name, initial, backends }: { name: string; initial: number; backends: BackendView[] }) {
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [canary, setCanary] = useState(initial);
  const stable = 100 - canary;
  const dirty = canary !== initial;

  // Re-sync the slider to the live value after an apply / promote / rollback refetch.
  useEffect(() => setCanary(initial), [initial]);

  const split = useApiMutation(
    (body: sdk.TrafficPolicySplitRequest) => sdk.splitTrafficPolicy({ path: { name }, body }),
    { invalidate: [["trafficpolicies"]], success: t("traffic.splitApplied") },
  );
  const promote = useApiMutation(() => sdk.promoteTrafficPolicy({ path: { name } }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.promoted"),
  });
  const rollback = useApiMutation(() => sdk.rollbackTrafficPolicy({ path: { name } }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.rolledBack"),
  });

  const onPromote = () =>
    confirm({
      title: t("traffic.promoteTitle", { name }),
      desc: t("traffic.promoteDesc"),
      okLabel: t("traffic.promoteOk"),
      danger: false,
      onConfirm: () => promote.mutate(undefined),
    });
  const onRollback = () =>
    confirm({
      title: t("traffic.rollbackTitle", { name }),
      desc: t("traffic.rollbackDesc"),
      okLabel: t("traffic.rollbackOk"),
      danger: false,
      onConfirm: () => rollback.mutate(undefined),
    });

  // Reflect the live slider value into each backend's actual share for an immediate preview.
  const rows: BackendView[] = backends.map((b) =>
    b.role === "canary"
      ? { ...b, weight: canary, actualPct: canary }
      : b.role === "stable"
        ? { ...b, weight: stable, actualPct: stable }
        : b,
  );

  const busy = split.isPending || promote.isPending || rollback.isPending;

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>{t("traffic.canaryPercentTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        <CanaryRollout value={canary} onChange={setCanary} />

        {/* per-backend identity + health; the share tracks the live slider value */}
        <div className="mt-5 border-t pt-4">
          <div className="mb-1 text-xs font-medium text-muted-foreground">{t("traffic.backendDist")}</div>
          <div className="divide-y">
            {rows.map((r) => (
              <div key={r.serviceName} className="flex items-center gap-3 py-2.5 text-sm">
                <span
                  className={cn("size-2 shrink-0 rounded-full", r.role === "canary" ? "bg-warning" : "bg-info")}
                />
                <Badge variant="outline" className="shrink-0">
                  {roleLabel(t, r.role)}
                </Badge>
                <Link
                  to={`/services/${r.serviceName}`}
                  className="truncate font-mono text-foreground hover:text-info hover:underline"
                >
                  {r.serviceName}
                </Link>
                <span className="flex-1" />
                <Progress value={r.actualPct} className="w-28" />
                <span className="w-10 text-right font-mono text-xs tabular-nums">{r.actualPct}%</span>
                {r.ready ? (
                  <Badge variant="success" className="shrink-0">
                    {t("traffic.backendReady")}
                  </Badge>
                ) : (
                  <Badge variant="outline" className="shrink-0">
                    {t("traffic.backendNotReady")}
                  </Badge>
                )}
              </div>
            ))}
          </div>
        </div>

        <div className="mt-1 flex items-center gap-2 border-t pt-4">
          <Button disabled={!dirty || busy} onClick={() => split.mutate({ canaryPercent: canary })}>
            {split.isPending && <Spinner data-icon="inline-start" />}
            {t("traffic.applyCanary")}
          </Button>
          <span className="flex-1" />
          <Button variant="outline" disabled={busy} onClick={onRollback}>
            {rollback.isPending ? <Spinner data-icon="inline-start" /> : <Undo2 data-icon="inline-start" />}
            {t("traffic.rollback")}
          </Button>
          <Button variant="secondary" disabled={busy} onClick={onPromote}>
            {promote.isPending ? <Spinner data-icon="inline-start" /> : <ArrowUpToLine data-icon="inline-start" />}
            {t("traffic.promoteToStable")}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

// ── Weighted distribution ─────────────────────────────────────────────────────
function WeightedDistribution({ name, backends }: { name: string; backends: BackendView[] }) {
  const { t } = useTranslation();
  const [weights, setWeights] = useState<Record<string, number>>(() =>
    Object.fromEntries(backends.map((b) => [b.serviceName, b.weight])),
  );

  const split = useApiMutation(
    (body: sdk.TrafficPolicySplitRequest) => sdk.splitTrafficPolicy({ path: { name }, body }),
    { invalidate: [["trafficpolicies"]], success: t("traffic.weightsApplied") },
  );

  const sum = Object.values(weights).reduce((a, b) => a + (b || 0), 0);
  const ok = sum === 100;

  const rows: BackendView[] = backends.map((b) => {
    const w = weights[b.serviceName] ?? 0;
    return { ...b, weight: w, actualPct: sum ? Math.round((w / sum) * 100) : 0 };
  });

  const weightCol: Column<BackendView> = {
    key: "weight",
    title: t("traffic.colTargetWeight"),
    width: 130,
    align: "right",
    render: (r) => (
      <Input
        type="number"
        min={0}
        max={100}
        className="ml-auto w-24"
        value={weights[r.serviceName] ?? 0}
        onChange={(e) => setWeights((prev) => ({ ...prev, [r.serviceName]: Number(e.target.value) }))}
      />
    ),
  };

  const apply = () =>
    split.mutate({
      backends: backends.map((b) => ({
        serviceName: b.serviceName,
        role: "member" as const,
        weight: weights[b.serviceName] ?? 0,
      })),
    });

  return (
    <Card className="p-0">
      <CardHeader className="flex-row items-center justify-between gap-2 border-b py-4">
        <CardTitle>{t("traffic.backendDist")}</CardTitle>
        <span className="text-xs text-muted-foreground">{t("traffic.weightedHint")}</span>
      </CardHeader>
      <DataTable
        columns={backendColumns(t, weightCol)}
        data={rows}
        rowKey={(r) => r.serviceName}
        pageSize={rows.length || 1}
      />
      <div className="flex items-center gap-4 border-t p-4">
        <Alert variant={ok ? "success" : "warning"} className="w-auto">
          {ok ? <CircleCheck /> : <TriangleAlert />}
          <AlertDescription>
            {ok ? t("traffic.sumOk", { sum }) : t("traffic.sumBad", { sum })}
          </AlertDescription>
        </Alert>
        <span className="flex-1" />
        <Button disabled={!ok || split.isPending} onClick={apply}>
          {split.isPending && <Spinner data-icon="inline-start" />}
          {t("traffic.applyWeights")}
        </Button>
      </div>
    </Card>
  );
}

// ── Monitoring: per-backend QPS / latency / error-rate trends ──────────────────
const MONITOR_RANGES = ["5m", "1h", "24h"] as const;
// Per-series colours, assigned by backend order — mirrors the prototype legend
// (stable=info / canary=warn, or member-a / member-b for weighted).
const SERIES_COLORS = ["var(--info)", "var(--warning)", "var(--success)"];

interface MetricSpec {
  metric: sdk.MlServiceMetricName;
  title: string;
}

function MonitorPane({ name, backends }: { name: string; backends: BackendView[] }) {
  const { t } = useTranslation();
  const [range, setRange] = useState<string>("1h");

  if (!backends.length) {
    return (
      <Card className="p-0">
        <CardContent className="py-4">
          <Empty className="py-12">
            <EmptyHeader>
              <EmptyTitle>{t("traffic.monitorNoBackends")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    );
  }

  const series = backends.map((b, i) => ({
    serviceName: b.serviceName,
    color: SERIES_COLORS[i % SERIES_COLORS.length],
  }));
  const metrics: MetricSpec[] = [
    { metric: "request_rate", title: t("traffic.monQps") },
    { metric: "latency", title: t("traffic.monLatency") },
    { metric: "error_rate", title: t("traffic.monErrorRate") },
  ];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <span className="text-xs text-muted-foreground">{t("traffic.monGrouped")}</span>
        <span className="flex-1" />
        <ToggleGroup type="single" value={range} onValueChange={(v) => v && setRange(v)}>
          {MONITOR_RANGES.map((r) => (
            <ToggleGroupItem key={r} value={r}>
              {r}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        {metrics.map((m) => (
          <MetricCard key={m.metric} name={name} spec={m} series={series} range={range} />
        ))}
      </div>
    </div>
  );
}

function MetricCard({
  name,
  spec,
  series,
  range,
}: {
  name: string;
  spec: MetricSpec;
  series: { serviceName: string; color: string }[];
  range: string;
}) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const results = useQueries({
    queries: series.map((s) => ({
      queryKey: ["trafficpolicies", tenant, name, "metrics", spec.metric, range, s.serviceName],
      enabled: tenant !== "" && name !== "",
      queryFn: async () => {
        const { data, error } = await sdk.getTrafficPolicyMetrics({
          path: { name },
          query: { metric: spec.metric, range, backend: s.serviceName },
        });
        if (error) throw error;
        return data;
      },
    })),
  });

  const loading = results.some((r) => r.isLoading);
  const lines = series.map((s, i) => ({
    ...s,
    values: (results[i].data?.series ?? []).map((p) => p.value ?? 0),
  }));
  const hasData = lines.some((l) => l.values.length > 0);

  return (
    <Card className="p-0">
      <CardHeader className="flex-row items-center justify-between gap-2 border-b py-3">
        <CardTitle className="text-sm">{spec.title}</CardTitle>
        <div className="flex flex-wrap items-center justify-end gap-x-3 gap-y-1">
          {series.map((s) => (
            <span key={s.serviceName} className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <span className="size-2 rounded-full" style={{ background: s.color }} />
              <span className="font-mono">{s.serviceName}</span>
            </span>
          ))}
        </div>
      </CardHeader>
      <CardContent className="py-4">
        {loading ? (
          <div className="grid h-[132px] place-items-center">
            <Spinner className="size-6 text-muted-foreground" />
          </div>
        ) : hasData ? (
          <MultiLineChart lines={lines} />
        ) : (
          <div className="grid h-[132px] place-items-center text-sm text-muted-foreground">
            {t("traffic.monitorEmpty")}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// Overlaid multi-series line chart, matching the prototype's 600×132 SVG with two
// horizontal grid lines. Each backend gets its own coloured path.
function MultiLineChart({ lines }: { lines: { serviceName: string; color: string; values: number[] }[] }) {
  const W = 600;
  const H = 132;
  const max = Math.max(1, ...lines.flatMap((l) => l.values));
  const path = (values: number[]) => {
    const n = values.length;
    if (!n) return "";
    const x = (i: number) => (n === 1 ? W : (i / (n - 1)) * W);
    const y = (v: number) => H - (v / max) * (H - 24) - 12;
    return "M" + values.map((v, i) => `${x(i).toFixed(1)} ${y(v).toFixed(1)}`).join(" L");
  };
  return (
    <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="h-[132px] w-full">
      {[44, 88].map((gy) => (
        <line key={gy} x1="0" y1={gy} x2={W} y2={gy} stroke="var(--border)" strokeWidth="1" />
      ))}
      {lines.map((l) => (
        <path
          key={l.serviceName}
          d={path(l.values)}
          fill="none"
          stroke={l.color}
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      ))}
    </svg>
  );
}

// ── Events (timeline) ──────────────────────────────────────────────────────────
function Events({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const q = useQuery({
    queryKey: ["trafficpolicies", tenant, name, "events"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getTrafficPolicyEvents({ path: { name } });
      if (error) throw error;
      return data;
    },
  });

  const items = q.data?.items ?? [];

  return (
    <Card>
      <CardContent className="py-5">
        {q.isLoading ? (
          <div className="grid place-items-center py-12">
            <Spinner className="size-7 text-muted-foreground" />
          </div>
        ) : items.length === 0 ? (
          <div className="py-12 text-center text-sm text-muted-foreground">
            {q.isError ? t("common.loadFailed") : t("traffic.noEvents")}
          </div>
        ) : (
          <ol className="flex flex-col gap-5 border-l border-border pl-6">
            {items.map((e, i) => (
              <li key={i} className="relative">
                <span
                  className={cn(
                    "absolute top-1 -left-[29px] size-2.5 rounded-full border-2 border-background ring-1 ring-border",
                    e.type === "Warning" ? "bg-warning" : "bg-muted-foreground",
                  )}
                />
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{e.reason}</span>
                  <Badge variant={e.type === "Warning" ? "warning" : "outline"}>{e.type}</Badge>
                  <span className="font-mono text-xs text-muted-foreground">
                    {fmtDateTimeSec(e.lastTimestamp)}
                  </span>
                </div>
                {e.message && <div className="mt-1 text-sm text-muted-foreground">{e.message}</div>}
              </li>
            ))}
          </ol>
        )}
      </CardContent>
    </Card>
  );
}
