import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Ban, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { useApiMutation } from "@/api/mutations";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";
import { LogViewer } from "@/components/LogViewer";
import { DataTable, type Column } from "@/components/DataTable";
import * as sdk from "@/api/generated";
import { PolicyText, fmtDateTime, primaryRole } from "./JobDetail";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardAction, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

// Run detail for both Job-runs (/jobs/:name/runs/:run) and Experiment-runs
// (/experiments/:name/runs/:run). The `kind` prop (preserved from the router)
// selects the back link, breadcrumb and which tenant-scoped SDK family to call —
// the Run/Pod/Event shapes are identical across both. No fabricated data: logs
// surface an Empty placeholder until backend log streaming lands.

const ACTIVE_PHASES: sdk.RunPhase[] = ["Creating", "Pending", "Running", "Canceling"];

// Phase → lifecycle Steps mapping (current step index + status).
function lifecycleState(phase?: sdk.RunPhase): { current: number; error: boolean } {
  switch (phase) {
    case "Creating":
      return { current: 0, error: false };
    case "Pending":
      return { current: 1, error: false };
    case "Running":
    case "Canceling":
      return { current: 2, error: false };
    case "Succeeded":
      return { current: 3, error: false };
    case "Failed":
      return { current: 3, error: true };
    case "Cancelled":
    case "Deleting":
    case "Deleted":
      return { current: 3, error: false };
    default:
      return { current: 0, error: false };
  }
}

// Key/value detail grid — the Descriptions replacement (two columns on md+).
function DescGrid({ children }: { children: React.ReactNode }) {
  return (
    <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-3 text-sm md:grid-cols-[120px_1fr_120px_1fr]">
      {children}
    </dl>
  );
}

function DescItem({ label, span, children }: { label: string; span?: boolean; children: React.ReactNode }) {
  return (
    <div className={span ? "contents md:col-span-4 md:grid md:grid-cols-[120px_1fr]" : "contents"}>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </div>
  );
}

export default function RunDetail({ kind }: { kind: "experiment" | "job" }) {
  const { name = "", run = "" } = useParams();
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { confirm } = useUI();

  const isExp = kind === "experiment";
  const base = isExp ? `/experiments/${name}` : `/jobs/${name}`;
  const navParent = isExp ? t("nav.experiments") : t("nav.jobs");

  const runQ = useQuery({
    queryKey: ["runDetail", kind, tenant, name, run],
    enabled: tenant !== "" && name !== "" && run !== "",
    queryFn: async () => {
      const { data, error } = isExp
        ? await sdk.getExperimentRun({ path: { name, run } })
        : await sdk.getRun({ path: { name, run } });
      if (error) throw error;
      return data;
    },
  });

  const cancel = useApiMutation(
    () =>
      isExp
        ? sdk.cancelExperimentRun({ path: { name, run } })
        : sdk.cancelRun({ path: { name, run } }),
    { invalidate: [["runDetail"], ["runs"], ["experiments"]], success: t("runDetail.cancelled") },
  );

  const r = runQ.data;
  const active = r?.phase ? ACTIVE_PHASES.includes(r.phase) : false;

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), navParent, name, run]}
      title={
        <span className="flex min-w-0 items-center gap-3">
          <span className="font-mono">{run}</span>
          {r && <PhaseTag phase={r.phase} />}
        </span>
      }
      subtitle={
        <span>
          <Link to={base} className="inline-flex items-center gap-1 text-sm text-info hover:underline">
            <ArrowLeft className="size-3.5" /> {t("runDetail.backTo", { name })}
          </Link>
          {r && (
            <span className="ml-3 text-muted-foreground">
              {t("runDetail.subtitle", { num: r.runNumber ?? "—", owner: r.owner || "—" })}
            </span>
          )}
        </span>
      }
      extra={
        r && (
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => runQ.refetch()}>
              <RefreshCw data-icon="inline-start" />
              {t("runDetail.refresh")}
            </Button>
            {active && (
              <Button
                variant="outline"
                className="text-destructive"
                disabled={cancel.isPending}
                onClick={() =>
                  confirm({
                    title: t("runDetail.cancelTitle", { run }),
                    desc: t("runDetail.cancelDesc"),
                    okLabel: t("runDetail.cancelOk"),
                    danger: false,
                    onConfirm: () => cancel.mutate(undefined),
                  })
                }
              >
                {cancel.isPending ? <Spinner data-icon="inline-start" /> : <Ban data-icon="inline-start" />}
                {t("runDetail.cancelRun")}
              </Button>
            )}
          </div>
        )
      }
    >
      {runQ.isLoading ? (
        <div className="grid place-items-center py-24">
          <Spinner className="size-7 text-muted-foreground" />
        </div>
      ) : runQ.isError || !r ? (
        <Card>
          <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
            <span className="text-sm text-muted-foreground">{t("common.loadFailed")}</span>
            <Button variant="outline" asChild>
              <Link to={base}>{t("runDetail.backTo", { name })}</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Lifecycle run={r} />
          <Tabs defaultValue="info">
            <TabsList variant="line">
              <TabsTrigger value="info">{t("runDetail.tabInfo")}</TabsTrigger>
              <TabsTrigger value="pods">{t("runDetail.tabPods")}</TabsTrigger>
              <TabsTrigger value="log">{t("runDetail.tabLog")}</TabsTrigger>
              <TabsTrigger value="ev">{t("runDetail.tabEvents")}</TabsTrigger>
            </TabsList>
            <TabsContent value="info" className="mt-4">
              <InfoPane run={r} />
            </TabsContent>
            <TabsContent value="pods" className="mt-4">
              <PodsPane kind={kind} name={name} run={run} />
            </TabsContent>
            <TabsContent value="log" className="mt-4">
              <LogPane kind={kind} name={name} run={run} />
            </TabsContent>
            <TabsContent value="ev" className="mt-4">
              <EventsPane kind={kind} name={name} run={run} />
            </TabsContent>
          </Tabs>
        </div>
      )}
    </PageContainer>
  );
}

function Lifecycle({ run }: { run: sdk.Run }) {
  const { t } = useTranslation();
  const { current, error } = lifecycleState(run.phase);
  const steps = [
    { title: t("runDetail.stepCreated"), description: fmtDateTime(run.createdAt) },
    { title: t("runDetail.stepScheduled"), description: undefined as string | undefined },
    { title: t("runDetail.stepRunning"), description: fmtDateTime(run.startedAt) },
    { title: t("runDetail.stepFinished"), description: fmtDateTime(run.finishedAt) },
  ];

  return (
    <Card>
      <CardContent>
        <ol className="flex items-start">
          {steps.map((step, i) => {
            const done = i < current;
            const isCurrent = i === current;
            const isErr = error && isCurrent;
            const dot = isErr
              ? "border-destructive bg-destructive text-destructive-foreground"
              : done || isCurrent
                ? "border-primary bg-primary text-primary-foreground"
                : "border-border bg-card text-muted-foreground";
            const lineDone = i < current;
            return (
              <li key={i} className={cn("flex min-w-0 flex-1 items-start", i === steps.length - 1 && "flex-none")}>
                <div className="flex flex-col items-center">
                  <span
                    className={cn(
                      "grid size-6 shrink-0 place-items-center rounded-full border text-xs font-medium",
                      dot,
                    )}
                  >
                    {i + 1}
                  </span>
                </div>
                <div className="mt-0.5 ml-2 min-w-0">
                  <div className={cn("text-sm font-medium", isErr ? "text-destructive" : "text-foreground")}>
                    {step.title}
                  </div>
                  {step.description && (
                    <div className="text-xs text-muted-foreground">{step.description}</div>
                  )}
                </div>
                {i < steps.length - 1 && (
                  <div
                    className={cn(
                      "mx-3 mt-3 h-px flex-1",
                      lineDone ? "bg-primary" : "bg-border",
                    )}
                  />
                )}
              </li>
            );
          })}
        </ol>
      </CardContent>
    </Card>
  );
}

function InfoPane({ run }: { run: sdk.Run }) {
  const { t } = useTranslation();
  const role = primaryRole(run.spec?.roles);
  const tpl = role?.template;
  const command = [...(tpl?.command ?? []), ...(tpl?.args ?? [])];
  const env = tpl?.env ?? [];

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>{t("runDetail.sectionConfig")}</CardTitle>
      </CardHeader>
      <CardContent>
        <DescGrid>
          <DescItem label={t("runDetail.fName")}>
            <span className="font-mono">{run.name}</span>
          </DescItem>
          <DescItem label={t("runDetail.fDesc")}>{run.description || "—"}</DescItem>
          <DescItem label={t("runDetail.fImage")}>
            {tpl?.image ? (
              <Badge variant="secondary" className="font-mono">
                {tpl.image}
              </Badge>
            ) : (
              "—"
            )}
          </DescItem>
          <DescItem label={t("runDetail.fPool")}>
            {run.poolName ? <span className="font-mono">{run.poolName}</span> : "—"}
          </DescItem>
          <DescItem label={t("runDetail.fUnit")}>
            {run.unitName ? <span className="font-mono">{run.unitName}</span> : "—"}
          </DescItem>
          <DescItem label={t("runDetail.fReplicas")}>
            <span className="font-mono">{role?.replicas ?? "—"}</span>
          </DescItem>
          <DescItem label={t("runDetail.fRunPolicy")} span>
            <PolicyText policy={run.runPolicy ?? run.spec?.runPolicy} />
          </DescItem>
          <DescItem label={t("runDetail.fStarted")}>
            <span className="font-mono text-muted-foreground">{fmtDateTime(run.startedAt)}</span>
          </DescItem>
          <DescItem label={t("runDetail.fFinished")}>
            <span className="font-mono text-muted-foreground">{fmtDateTime(run.finishedAt)}</span>
          </DescItem>
        </DescGrid>

        <Separator className="my-5" />
        <div className="mb-1.5 text-xs text-muted-foreground">{t("runDetail.command")}</div>
        {command.length ? (
          <pre
            className="m-0 mb-5 overflow-auto rounded-md p-4 font-mono text-xs leading-relaxed"
            style={{ background: "#16181d", color: "#e6e6e6" }}
          >
            {command.join(" ")}
          </pre>
        ) : (
          <div className="mb-5 text-sm text-muted-foreground">{t("runDetail.noCommand")}</div>
        )}
        <div className="mb-1.5 text-xs text-muted-foreground">{t("runDetail.env")}</div>
        {env.length ? (
          <div className="flex flex-wrap gap-1.5">
            {env.map((e) => (
              <Badge key={e.name} variant="secondary" className="font-mono">
                {e.name}={e.value ?? ""}
              </Badge>
            ))}
          </div>
        ) : (
          <div className="text-sm text-muted-foreground">{t("runDetail.noEnv")}</div>
        )}
      </CardContent>
    </Card>
  );
}

function PodsPane({ kind, name, run }: { kind: "experiment" | "job"; name: string; run: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const isExp = kind === "experiment";

  const podsQ = useQuery({
    queryKey: ["runDetail", kind, tenant, name, run, "pods"],
    enabled: tenant !== "" && name !== "" && run !== "",
    queryFn: async () => {
      const { data, error } = isExp
        ? await sdk.listExperimentRunPods({ path: { name, run } })
        : await sdk.listRunPods({ path: { name, run } });
      if (error) throw error;
      return data;
    },
  });

  const columns: Column<sdk.Pod>[] = [
    {
      key: "name",
      title: t("runDetail.colPod"),
      render: (p) => <span className="font-mono">{p.name}</span>,
    },
    {
      key: "phase",
      title: t("runDetail.colPhase"),
      width: 120,
      render: (p) => <PhaseTag phase={p.phase} />,
    },
    {
      key: "role",
      title: t("runDetail.colRole"),
      width: 140,
      render: (p) => <span className="font-mono">{p.role || "—"}</span>,
    },
    {
      key: "restarts",
      title: t("runDetail.colReady"),
      width: 90,
      align: "right",
      render: (p) => <span className="font-mono">{p.restartCount ?? 0}</span>,
    },
    {
      key: "startedAt",
      title: t("runDetail.fStarted"),
      width: 170,
      render: (p) => <span className="text-muted-foreground">{fmtDateTime(p.startedAt)}</span>,
    },
  ];

  return (
    <Card className="overflow-hidden p-0">
      <DataTable
        columns={columns}
        data={podsQ.data?.items ?? []}
        rowKey={(p) => p.name}
        loading={podsQ.isLoading}
        error={podsQ.isError}
        empty={t("runDetail.noPods")}
      />
    </Card>
  );
}

// Pod log viewer — the prototype's dark `.logbox` (shared <LogViewer/>). Pods come
// from the run's pod list; the selected pod's logs are fetched on demand (the
// snapshot endpoint; SSE follow is a future backend feature).
function LogPane({ kind, name, run }: { kind: "experiment" | "job"; name: string; run: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const isExp = kind === "experiment";
  const [pod, setPod] = useState<string>("");

  const podsQ = useQuery({
    queryKey: ["runDetail", kind, tenant, name, run, "pods"],
    enabled: tenant !== "" && name !== "" && run !== "",
    queryFn: async () => {
      const { data, error } = isExp
        ? await sdk.listExperimentRunPods({ path: { name, run } })
        : await sdk.listRunPods({ path: { name, run } });
      if (error) throw error;
      return data;
    },
  });

  const pods = podsQ.data?.items ?? [];
  useEffect(() => {
    if (!pod && pods.length) setPod(pods[0].name);
  }, [pod, pods]);

  const logsQ = useQuery({
    queryKey: ["runDetail", kind, tenant, name, run, "logs", pod],
    enabled: tenant !== "" && name !== "" && run !== "" && pod !== "",
    queryFn: async () => {
      const { data, error } = isExp
        ? await sdk.getExperimentRunPodLogs({ path: { name, run, pod } })
        : await sdk.getRunPodLogs({ path: { name, run, pod } });
      if (error) throw error;
      return data as unknown as string;
    },
  });

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>{t("runDetail.logTitle")}</CardTitle>
        <CardAction>
          <div className="flex items-center gap-2">
            <Select value={pod || undefined} onValueChange={setPod} disabled={!pods.length}>
              <SelectTrigger size="sm" className="min-w-52">
                <SelectValue placeholder={t("runDetail.colPod")} />
              </SelectTrigger>
              <SelectContent>
                {pods.map((p) => (
                  <SelectItem key={p.name} value={p.name}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button variant="outline" size="icon-sm" onClick={() => logsQ.refetch()} aria-label={t("runDetail.refresh")}>
              <RefreshCw />
            </Button>
          </div>
        </CardAction>
      </CardHeader>
      <CardContent>
        {podsQ.isLoading || logsQ.isLoading ? (
          <div className="grid place-items-center py-16">
            <Spinner className="size-7 text-muted-foreground" />
          </div>
        ) : !pods.length ? (
          <Empty className="border bg-muted">
            <EmptyHeader>
              <EmptyTitle>{t("runDetail.noLog")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          <LogViewer text={logsQ.data} empty={t("runDetail.noLog")} />
        )}
      </CardContent>
    </Card>
  );
}

function EventsPane({ kind, name, run }: { kind: "experiment" | "job"; name: string; run: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const isExp = kind === "experiment";

  const eventsQ = useQuery({
    queryKey: ["runDetail", kind, tenant, name, run, "events"],
    enabled: tenant !== "" && name !== "" && run !== "",
    queryFn: async () => {
      const { data, error } = isExp
        ? await sdk.listExperimentRunEvents({ path: { name, run } })
        : await sdk.listRunEvents({ path: { name, run } });
      if (error) throw error;
      return data;
    },
  });

  const columns: Column<sdk.Event>[] = [
    {
      key: "reason",
      title: t("runDetail.colReason"),
      width: 180,
      render: (e) => <span className="font-mono">{e.reason}</span>,
    },
    {
      key: "type",
      title: t("runDetail.colType"),
      width: 120,
      render: (e) => (
        <Badge variant={e.type === "Warning" ? "warning" : "secondary"}>
          {e.type === "Warning" ? t("runDetail.eventWarning") : t("runDetail.eventNormal")}
        </Badge>
      ),
    },
    {
      key: "message",
      title: t("runDetail.colMessage"),
      render: (e) => <span className="text-muted-foreground">{e.message}</span>,
    },
    {
      key: "count",
      title: t("runDetail.colCount"),
      width: 80,
      align: "right",
      render: (e) => <span className="font-mono">{e.count ?? 1}</span>,
    },
    {
      key: "lastTimestamp",
      title: t("runDetail.colTime"),
      width: 170,
      render: (e) => <span className="text-muted-foreground">{fmtDateTime(e.lastTimestamp)}</span>,
    },
  ];

  return (
    <Card className="overflow-hidden p-0">
      <CardHeader className="border-b p-4">
        <CardTitle>{t("runDetail.eventsTitle")}</CardTitle>
      </CardHeader>
      <DataTable
        columns={columns}
        data={eventsQ.data?.items ?? []}
        rowKey={(e) => `${e.reason}-${e.lastTimestamp}-${e.message}`}
        loading={eventsQ.isLoading}
        error={eventsQ.isError}
        empty={t("runDetail.noEvents")}
      />
    </Card>
  );
}
