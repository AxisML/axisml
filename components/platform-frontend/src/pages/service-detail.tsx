import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  ArrowLeft,
  Maximize2,
  Pencil,
  Pause,
  Play,
  Trash2,
  Copy,
  RotateCw,
  OctagonX,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useApp } from "@/app/store";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { LogViewer } from "@/components/log-viewer";
import { DataTable, type Column } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { fmtDateTime } from "./job-detail";

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
  const back = (
    <Button variant="ghost" size="sm" asChild>
      <Link to="/services">
        <ArrowLeft data-icon="inline-start" />
        {t("services.backToList")}
      </Link>
    </Button>
  );

  if (q.isLoading) {
    return (
      <PageContainer breadcrumb={breadcrumb} title={name}>
        {back}
        <div className="grid place-items-center py-24">
          <Spinner className="size-7 text-muted-foreground" />
        </div>
      </PageContainer>
    );
  }

  if (q.isError || !q.data) {
    return (
      <PageContainer breadcrumb={breadcrumb} title={name}>
        {back}
        <Alert variant="destructive" className="mt-4">
          <OctagonX />
          <AlertTitle>{t("services.notFound")}</AlertTitle>
          <AlertDescription>{t("services.loadFailedDesc")}</AlertDescription>
        </Alert>
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
      subtitle={svc.description ?? svc.displayName ?? undefined}
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
      <div className="mb-4">{back}</div>
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
function DescRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </>
  );
}

function InfoPane({ svc }: { svc: sdk.MlService }) {
  const { t } = useTranslation();
  const { toast } = useUI();

  const dash = <span className="text-muted-foreground">—</span>;
  const chip = (v?: string) =>
    v ? (
      <Badge variant="outline" className="font-mono">
        {v}
      </Badge>
    ) : (
      dash
    );

  return (
    <Card className="p-0">
      <CardHeader className="border-b py-4">
        <CardTitle>{t("services.configInfo")}</CardTitle>
      </CardHeader>
      <CardContent className="py-4">
        <dl className="grid grid-cols-[140px_1fr] items-center gap-x-4 gap-y-3 text-sm">
          <DescRow label={t("services.dName")}>{chip(svc.name)}</DescRow>
          <DescRow label={t("services.dDesc")}>{svc.description ?? dash}</DescRow>
          <DescRow label={t("services.dModelVersion")}>
            {svc.modelName
              ? chip(svc.modelVersion ? `${svc.modelName}@${svc.modelVersion}` : svc.modelName)
              : dash}
          </DescRow>
          <DescRow label={t("services.dImage")}>{chip(svc.image)}</DescRow>
          <DescRow label={t("services.dPool")}>{chip(svc.poolName)}</DescRow>
          <DescRow label={t("services.dUnit")}>{chip(svc.unitName)}</DescRow>
          <DescRow label={t("services.dReplicas")}>
            <span className="font-mono">
              {t("services.replicasReady", { ready: svc.readyReplicas ?? 0, total: svc.replicas ?? 0 })}
            </span>
          </DescRow>
          <DescRow label={t("services.dPorts")}>
            {svc.ports && svc.ports.length > 0 ? (
              <span className="flex flex-wrap gap-1.5">
                {svc.ports.map((p) => (
                  <Badge key={`${p.name}:${p.port}`} variant="outline" className="font-mono">
                    {p.name} : {p.port}
                  </Badge>
                ))}
              </span>
            ) : (
              dash
            )}
          </DescRow>
          <DescRow label={t("services.dAccess")}>
            {svc.accessUrl ? (
              <span className="flex items-center gap-2">
                <Badge variant="outline" className="font-mono">
                  {svc.accessUrl}
                </Badge>
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
          </DescRow>
          <DescRow label={t("services.dCreator")}>
            {svc.owner ? <span className="font-mono">{svc.owner}</span> : dash}
          </DescRow>
          <DescRow label={t("services.dCreatedAt")}>
            {svc.createdAt ? (
              <span className="font-mono text-muted-foreground">
                {dayjs(svc.createdAt).format("YYYY-MM-DD HH:mm:ss")}
              </span>
            ) : (
              dash
            )}
          </DescRow>
        </dl>
      </CardContent>
    </Card>
  );
}

// ── Monitoring: request-rate trend (mini SVG, mirrors the dashboard style) ──────
function MonitorPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const q = useQuery({
    queryKey: ["mlservices", tenant, name, "metrics"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getMlServiceMetrics({
        path: { name },
        query: { metric: "request_rate", range: "24h" },
      });
      if (error) throw error;
      return data;
    },
  });
  const series = (q.data?.series ?? []).map((p) => p.value ?? 0);
  return (
    <Card className="p-0">
      <CardHeader className="border-b py-4">
        <CardTitle>{t("services.tabMonitor")}</CardTitle>
      </CardHeader>
      <CardContent className="py-4">
        {q.isLoading ? (
          <div className="grid place-items-center py-16">
            <Spinner className="size-7 text-muted-foreground" />
          </div>
        ) : series.length ? (
          <MiniTrend values={series} />
        ) : (
          <Empty className="py-12">
            <EmptyHeader>
              <EmptyTitle>{t("services.monitorEmpty")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        )}
      </CardContent>
    </Card>
  );
}

function MiniTrend({ values }: { values: number[] }) {
  const W = 720;
  const H = 200;
  const n = values.length;
  const max = Math.max(...values, 1);
  const x = (i: number) => (i / Math.max(1, n - 1)) * W;
  const y = (v: number) => H - (v / max) * (H - 16) - 6;
  const pts = values.map((v, i) => `${x(i)} ${y(v)}`);
  return (
    <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="h-[200px] w-full">
      <defs>
        <linearGradient id="svc-trend" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0" stopColor="var(--info)" stopOpacity="0.16" />
          <stop offset="1" stopColor="var(--info)" stopOpacity="0" />
        </linearGradient>
      </defs>
      {[50, 100, 150].map((gy) => (
        <line key={gy} x1="0" y1={gy} x2={W} y2={gy} stroke="var(--border)" strokeWidth="1" />
      ))}
      <path d={`M0 ${H} L${pts.join(" L")} L${W} ${H} Z`} fill="url(#svc-trend)" />
      <path
        d={`M${pts.join(" L")}`}
        fill="none"
        stroke="var(--info)"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
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
  const [pod, setPod] = useState<string>("");
  const podsQ = useQuery({
    queryKey: ["mlservices", tenant, name, "pods"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.listMlServicePods({ path: { name } });
      if (error) throw error;
      return data;
    },
  });
  const pods = podsQ.data?.items ?? [];
  useEffect(() => {
    if (!pod && pods.length) setPod(pods[0].name);
  }, [pod, pods]);
  const logsQ = useQuery({
    queryKey: ["mlservices", tenant, name, "logs", pod],
    enabled: tenant !== "" && name !== "" && pod !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getMlServicePodLogs({ path: { name, pod } });
      if (error) throw error;
      return data as unknown as string;
    },
  });
  return (
    <Card className="p-0">
      <CardHeader className="flex flex-row items-center justify-between gap-2 border-b py-4">
        <CardTitle>{t("services.tabLog")}</CardTitle>
        <div className="flex items-center gap-2">
          <Select value={pod || undefined} onValueChange={setPod}>
            <SelectTrigger size="sm" className="min-w-52">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {pods.map((p) => (
                <SelectItem key={p.name} value={p.name}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button variant="outline" size="icon-sm" onClick={() => logsQ.refetch()}>
            <RotateCw />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="py-4">
        {podsQ.isLoading || logsQ.isLoading ? (
          <div className="grid place-items-center py-16">
            <Spinner className="size-7 text-muted-foreground" />
          </div>
        ) : !pods.length ? (
          <Empty className="py-12">
            <EmptyHeader>
              <EmptyTitle>{t("services.logEmpty")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          <LogViewer text={logsQ.data} empty={t("services.logEmpty")} />
        )}
      </CardContent>
    </Card>
  );
}

// ── Events ───────────────────────────────────────────────────────────────────────
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
  const columns: Column<sdk.Event>[] = [
    {
      key: "reason",
      title: t("services.colReason"),
      width: 180,
      render: (e) => <span className="font-mono">{e.reason}</span>,
    },
    {
      key: "type",
      title: t("services.colType"),
      width: 110,
      render: (e) => (
        <Badge variant={e.type === "Warning" ? "warning" : "outline"}>{e.type}</Badge>
      ),
    },
    { key: "message", title: t("services.colMessage"), dataIndex: "message" },
    {
      key: "time",
      title: t("services.colTime"),
      width: 170,
      render: (e) => <span className="text-muted-foreground">{fmtDateTime(e.lastTimestamp)}</span>,
    },
  ];
  return (
    <Card className="overflow-hidden p-0">
      <DataTable
        columns={columns}
        data={q.data?.items ?? []}
        rowKey={(e, i) => `${e.reason}-${i}`}
        loading={q.isLoading}
        error={q.isError}
        empty={t("services.eventsEmpty")}
      />
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
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[760px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("services.drawerEdit")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{svc.name}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
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
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={update.isPending} onClick={submit}>
            {update.isPending && <Spinner data-icon="inline-start" />}
            {t("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
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
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[420px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("services.drawerScale")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{svc.name}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
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
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!valid || scale.isPending} onClick={submit}>
            {scale.isPending && <Spinner data-icon="inline-start" />}
            {t("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
