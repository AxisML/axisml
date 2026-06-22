import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Copy, Trash2, CircleCheck, TriangleAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { DataTable, type Column } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Slider } from "@/components/ui/slider";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Spinner } from "@/components/ui/spinner";
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

  const back = (
    <Link
      to="/traffic"
      className="mb-3 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-info"
    >
      <ArrowLeft className="size-4" />
      {t("traffic.backToList")}
    </Link>
  );

  if (q.isLoading) {
    return (
      <PageContainer breadcrumb={[t("nav.serviceCenter"), t("nav.traffic"), name]} title={name}>
        {back}
        <div className="grid place-items-center py-24">
          <Spinner className="size-7 text-muted-foreground" />
        </div>
      </PageContainer>
    );
  }

  if (q.isError || !q.data) {
    return (
      <PageContainer breadcrumb={[t("nav.serviceCenter"), t("nav.traffic"), name]} title={t("traffic.loadFailedTitle")}>
        {back}
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertDescription>{t("common.loadFailed")}</AlertDescription>
        </Alert>
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
      subtitle={p.description ?? p.displayName ?? undefined}
      extra={
        <Button variant="outline" className="text-destructive" disabled={del.isPending} onClick={onDelete}>
          {del.isPending ? <Spinner data-icon="inline-start" /> : <Trash2 data-icon="inline-start" />}
          {t("traffic.delete")}
        </Button>
      }
    >
      {back}
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">{t("traffic.tabOverview")}</TabsTrigger>
          <TabsTrigger value="dist">{t("traffic.tabDistribution")}</TabsTrigger>
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
        <TabsContent value="events">
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
        <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-3 text-sm">
          <dt className="text-muted-foreground">{t("traffic.fieldName")}</dt>
          <dd className="font-mono">{policy.name}</dd>

          <dt className="text-muted-foreground">{t("traffic.fieldDesc")}</dt>
          <dd>{policy.description ?? policy.displayName ?? "—"}</dd>

          <dt className="text-muted-foreground">{t("traffic.fieldMode")}</dt>
          <dd>
            <Badge variant="outline">{modeLabel}</Badge>
          </dd>

          <dt className="text-muted-foreground">{t("traffic.fieldEndpoint")}</dt>
          <dd>
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
          </dd>

          <dt className="text-muted-foreground">{t("traffic.fieldBackendCount")}</dt>
          <dd className="font-mono">{backendCount}</dd>

          <dt className="text-muted-foreground">{t("traffic.fieldOwner")}</dt>
          <dd>{policy.owner || "—"}</dd>

          <dt className="text-muted-foreground">{t("traffic.fieldCreatedAt")}</dt>
          <dd className="font-mono">
            {policy.createdAt ? dayjs(policy.createdAt).format("YYYY-MM-DD HH:mm:ss") : "—"}
          </dd>
        </dl>
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
  const [canary, setCanary] = useState(initial);
  const stable = 100 - canary;

  const split = useApiMutation(
    (body: sdk.TrafficPolicySplitRequest) => sdk.splitTrafficPolicy({ path: { name }, body }),
    { invalidate: [["trafficpolicies"]], success: t("traffic.splitApplied") },
  );
  const promote = useApiMutation(() => sdk.promoteTrafficPolicy({ path: { name } }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.promoted"),
  });

  // Reflect the live slider value into the table's actual share for an immediate preview.
  const rows: BackendView[] = backends.map((b) =>
    b.role === "canary"
      ? { ...b, weight: canary, actualPct: canary }
      : b.role === "stable"
        ? { ...b, weight: stable, actualPct: stable }
        : b,
  );

  const weightCol: Column<BackendView> = {
    key: "weight",
    title: t("traffic.colTargetWeight"),
    width: 110,
    align: "right",
    render: (r) => <span className="font-mono">{r.weight}</span>,
  };

  return (
    <div className="flex flex-col gap-5">
      <Card>
        <CardHeader className="border-b">
          <CardTitle>{t("traffic.canaryPercentTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="mb-3 flex items-center justify-between">
            <span className="text-sm text-muted-foreground">{t("traffic.canaryShare")}</span>
            <span className="font-mono text-2xl font-semibold">
              {canary}
              <span className="text-sm text-muted-foreground">%</span>
            </span>
          </div>
          <Slider min={0} max={100} value={[canary]} onValueChange={(vals) => setCanary(vals[0])} />
          <div className="mt-2 flex gap-7 text-sm text-muted-foreground">
            <span>
              {t("traffic.stableShare")} <b className="font-mono text-info">{stable}%</b>
            </span>
            <span>
              {t("traffic.canaryShare")} <b className="font-mono text-warning">{canary}%</b>
            </span>
          </div>
          <div className="mt-4 flex gap-2">
            <Button disabled={split.isPending} onClick={() => split.mutate({ canaryPercent: canary })}>
              {split.isPending && <Spinner data-icon="inline-start" />}
              {t("traffic.applyCanary")}
            </Button>
            <Button variant="outline" disabled={promote.isPending} onClick={() => promote.mutate(undefined)}>
              {promote.isPending && <Spinner data-icon="inline-start" />}
              {t("traffic.promoteToStable")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card className="p-0">
        <CardHeader className="border-b py-4">
          <CardTitle>{t("traffic.backendDist")}</CardTitle>
        </CardHeader>
        <DataTable
          columns={backendColumns(t, weightCol)}
          data={rows}
          rowKey={(r) => r.serviceName}
          pageSize={rows.length || 1}
        />
      </Card>
    </div>
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
        <Alert variant={ok ? "default" : "warning"} className="w-auto">
          {ok ? <CircleCheck className="text-success" /> : <TriangleAlert />}
          <AlertDescription className={ok ? "text-success" : undefined}>
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

// ── Events ────────────────────────────────────────────────────────────────────
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
    <Card className="p-0">
      {q.isLoading ? (
        <div className="grid place-items-center py-16">
          <Spinner className="size-7 text-muted-foreground" />
        </div>
      ) : items.length === 0 ? (
        <div className="py-16 text-center text-sm text-muted-foreground">
          {q.isError ? t("common.loadFailed") : t("traffic.noEvents")}
        </div>
      ) : (
        <ul className="flex flex-col">
          {items.map((e, i) => (
            <li
              key={i}
              className="flex items-start justify-between gap-4 border-b px-4 py-3 last:border-b-0"
            >
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{e.reason}</span>
                  <Badge variant={e.type === "Warning" ? "warning" : "outline"}>{e.type}</Badge>
                </div>
                <div className="mt-0.5 text-sm text-muted-foreground">{e.message}</div>
              </div>
              <span className="shrink-0 font-mono text-xs text-muted-foreground">
                {e.lastTimestamp ? dayjs(e.lastTimestamp).format("YYYY-MM-DD HH:mm:ss") : "—"}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
