import { useParams } from "react-router-dom";
import { Ban, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { useApiMutation } from "@/api/mutations";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { PodLogPane } from "@/components/pod-log-pane";
import { usePodLogs } from "@/lib/use-pod-logs";
import { CodeBlock } from "@/components/code-block";
import { BackLink } from "@/components/back-link";
import { MonoChip } from "@/components/mono-chip";
import { Descriptions, Desc } from "@/components/descriptions";
import { PageLoading, DetailError } from "@/components/page-state";
import { DataTable, type Column } from "@/components/data-table";
import { Timeline, type TimelineItem } from "@/components/timeline";
import * as sdk from "@/api/generated";
import { PolicyText, primaryRole } from "./job-detail";
import { fmtDateTime } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
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
      subtitle={<BackLink to={base}>{t("runDetail.backTo", { name })}</BackLink>}
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
        <PageLoading />
      ) : runQ.isError || !r ? (
        <DetailError message={t("common.loadFailed")} />
      ) : (
        <div className="flex flex-col gap-4">
          <Lifecycle run={r} />
          <Tabs defaultValue="info">
            <TabsList>
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
        <Descriptions columns="double">
          <Desc label={t("runDetail.fName")}>
            <span className="font-mono">{run.name}</span>
          </Desc>
          <Desc label={t("runDetail.fDesc")}>{run.description || "—"}</Desc>
          <Desc label={t("runDetail.fImage")}>
            {tpl?.image ? <MonoChip>{tpl.image}</MonoChip> : "—"}
          </Desc>
          <Desc label={t("runDetail.fPool")}>
            {run.poolName ? <span className="font-mono">{run.poolName}</span> : "—"}
          </Desc>
          <Desc label={t("runDetail.fUnit")}>
            {run.unitName ? <span className="font-mono">{run.unitName}</span> : "—"}
          </Desc>
          <Desc label={t("runDetail.fReplicas")}>
            <span className="font-mono">{role?.replicas ?? "—"}</span>
          </Desc>
          <Desc label={t("runDetail.fRunPolicy")} span>
            <PolicyText policy={run.runPolicy ?? run.spec?.runPolicy} />
          </Desc>
          <Desc label={t("runDetail.fStarted")}>
            <span className="font-mono text-muted-foreground">{fmtDateTime(run.startedAt)}</span>
          </Desc>
          <Desc label={t("runDetail.fFinished")}>
            <span className="font-mono text-muted-foreground">{fmtDateTime(run.finishedAt)}</span>
          </Desc>
        </Descriptions>

        <Separator className="my-5" />
        <div className="mb-1.5 text-xs text-muted-foreground">{t("runDetail.command")}</div>
        {command.length ? (
          <CodeBlock className="mb-5">{command.join(" ")}</CodeBlock>
        ) : (
          <div className="mb-5 text-sm text-muted-foreground">{t("runDetail.noCommand")}</div>
        )}
        <div className="mb-1.5 text-xs text-muted-foreground">{t("runDetail.env")}</div>
        {env.length ? (
          <div className="flex flex-wrap gap-1.5">
            {env.map((e) => (
              <MonoChip key={e.name}>
                {e.name}={e.value ?? ""}
              </MonoChip>
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
      key: "node",
      title: t("runDetail.colNode"),
      width: 150,
      render: (p) => <span className="font-mono text-muted-foreground">{p.nodeName || "—"}</span>,
    },
    {
      key: "restarts",
      title: t("runDetail.colRestarts"),
      width: 80,
      align: "right",
      render: (p) => <span className="font-mono">{p.restartCount ?? 0}</span>,
    },
    {
      key: "exitCode",
      title: t("runDetail.colExitCode"),
      width: 90,
      align: "right",
      render: (p) =>
        p.exitCode == null ? (
          <span className="text-muted-foreground">—</span>
        ) : (
          <span className="font-mono">{p.exitCode}</span>
        ),
    },
    {
      key: "startedAt",
      title: t("runDetail.fStarted"),
      width: 170,
      render: (p) => <span className="font-mono text-muted-foreground">{fmtDateTime(p.startedAt)}</span>,
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
  const logs = usePodLogs({
    queryKey: ["runDetail", kind, tenant, name, run],
    enabled: tenant !== "" && name !== "" && run !== "",
    listPods: async () => {
      const { data, error } = isExp
        ? await sdk.listExperimentRunPods({ path: { name, run } })
        : await sdk.listRunPods({ path: { name, run } });
      if (error) throw error;
      return data;
    },
    getLogs: async (pod) => {
      const { data, error } = isExp
        ? await sdk.getExperimentRunPodLogs({ path: { name, run, pod } })
        : await sdk.getRunPodLogs({ path: { name, run, pod } });
      if (error) throw error;
      return data as unknown as string;
    },
  });
  return <PodLogPane logs={logs} emptyText={t("runDetail.noLog")} />;
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

  const events = eventsQ.data?.items ?? [];
  // Oldest → newest so the rail reads as a chronological feed (the prototype's
  // `.timeline`). A Warning event tints its dot + tag amber; the involved Pod, if
  // any, is appended to the description.
  const items: TimelineItem[] = [...events]
    .sort((a, b) => (a.lastTimestamp < b.lastTimestamp ? -1 : 1))
    .map((e, i) => {
      const warn = e.type === "Warning";
      const pod = e.involvedObject?.name;
      const count = e.count ?? 1;
      return {
        id: `${e.reason}-${e.lastTimestamp}-${i}`,
        name: e.reason,
        tag: warn ? t("runDetail.eventWarning") : t("runDetail.eventNormal"),
        tagTone: warn ? "warn" : "normal",
        time: fmtDateTime(e.lastTimestamp),
        tone: warn ? "warn" : "info",
        desc: (
          <span>
            {e.message}
            {pod && <span className="ml-1.5 font-mono text-muted-foreground/80">· {pod}</span>}
            {count > 1 && <span className="ml-1.5 text-muted-foreground/80">×{count}</span>}
          </span>
        ),
      };
    });

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>{t("runDetail.eventsTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {eventsQ.isLoading ? (
          <PageLoading className="py-12" />
        ) : eventsQ.isError ? (
          <DetailError message={t("common.loadFailed")} />
        ) : items.length === 0 ? (
          <DetailError message={t("runDetail.noEvents")} />
        ) : (
          <Timeline items={items} />
        )}
      </CardContent>
    </Card>
  );
}
