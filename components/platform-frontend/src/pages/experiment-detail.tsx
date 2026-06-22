import { useMemo, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { ArrowLeft, Play, Search, Trash2, LineChart } from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useApiMutation } from "@/api/mutations";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { DataTable, type Column } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// Experiments are specialized training Jobs (Job→Run model); this detail page
// mirrors JobDetail — an "experiment info" pane plus a runs table. Both the
// definition and its Runs come from the live API, scoped to the active tenant.
const ACTIVE_RUN_PHASES: sdk.RunPhase[] = ["Creating", "Pending", "Running", "Canceling"];
const ALL = "__all__";

function fmtDuration(start?: string | null, end?: string | null): string {
  if (!start) return "—";
  const from = dayjs(start);
  const to = end ? dayjs(end) : dayjs();
  const secs = Math.max(0, to.diff(from, "second"));
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  const s = secs % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(h)}:${pad(m)}:${pad(s)}`;
}

export default function ExperimentDetail() {
  const { name = "" } = useParams<{ name: string }>();
  const { tenant } = useApp();
  const { t } = useTranslation();
  const { confirm } = useUI();

  const expQ = useQuery({
    queryKey: ["experiments", tenant, name],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getExperiment({ path: { name } });
      if (error) throw error;
      return data;
    },
  });

  const runsQ = useQuery({
    queryKey: ["experiments", tenant, name, "runs"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.listExperimentRuns({ path: { name } });
      if (error) throw error;
      return data;
    },
  });

  const delExp = useApiMutation(() => sdk.deleteExperiment({ path: { name } }), {
    invalidate: [["experiments"]],
    success: t("experiments.deleted"),
  });
  const triggerExp = useApiMutation(() => sdk.triggerExperimentRun({ path: { name }, body: {} }), {
    invalidate: [["experiments"]],
    success: t("experiments.runTriggered"),
  });

  if (expQ.isError) {
    return (
      <PageContainer
        breadcrumb={[t("nav.trainingCenter"), t("nav.experiments"), name]}
        title={<span className="font-mono">{name}</span>}
      >
        <div className="grid place-items-center gap-4 py-24 text-center">
          <p className="text-destructive">{t("common.loadFailed")}</p>
          <Button variant="outline" asChild>
            <Link to="/experiments">{t("experiments.backToList")}</Link>
          </Button>
        </div>
      </PageContainer>
    );
  }

  if (expQ.isLoading || !expQ.data) {
    return (
      <PageContainer
        breadcrumb={[t("nav.trainingCenter"), t("nav.experiments"), name]}
        title={<span className="font-mono">{name}</span>}
      >
        <div className="grid place-items-center py-24">
          <Spinner className="size-7 text-muted-foreground" />
        </div>
      </PageContainer>
    );
  }

  const exp = expQ.data;
  const runCount = runsQ.data?.count ?? 0;
  const onRun = () =>
    confirm({
      title: t("experiments.runTitle", { name }),
      desc: t("experiments.runDesc"),
      okLabel: t("experiments.confirmRun"),
      danger: false,
      onConfirm: () => triggerExp.mutate(undefined),
    });
  const onDelete = () =>
    confirm({
      title: t("experiments.deleteTitle", { name }),
      desc: t("experiments.deleteDesc"),
      info: t("experiments.deleteInfo"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delExp.mutate(undefined),
    });

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.experiments"), name]}
      title={<span className="font-mono">{name}</span>}
      subtitle={
        <span>
          {exp.description || exp.displayName || "—"}
          <span className="ml-2 text-muted-foreground">
            · {t("experiments.headRunsSummary", { count: runCount, owner: exp.owner ?? "—" })}
          </span>
        </span>
      }
      extra={
        <div className="flex items-center gap-2">
          <Button variant="outline" asChild>
            <Link to="/experiments">
              <ArrowLeft data-icon="inline-start" />
              {t("experiments.backToList")}
            </Link>
          </Button>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="outline" size="icon">
                <LineChart className="text-warning" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("experiments.tensorboard")}</TooltipContent>
          </Tooltip>
          <Button onClick={onRun}>
            <Play data-icon="inline-start" />
            {t("experiments.runAction")}
          </Button>
          <Button variant="outline" className="text-destructive" onClick={onDelete}>
            <Trash2 data-icon="inline-start" />
            {t("common.delete")}
          </Button>
        </div>
      }
    >
      <Tabs defaultValue="info">
        <TabsList>
          <TabsTrigger value="info">{t("experiments.tabInfo")}</TabsTrigger>
          <TabsTrigger value="runs">
            {t("experiments.tabRuns")}
            <Badge variant="secondary" className="ml-2">
              {runCount}
            </Badge>
          </TabsTrigger>
        </TabsList>
        <TabsContent value="info" className="mt-4">
          <InfoPane exp={exp} />
        </TabsContent>
        <TabsContent value="runs" className="mt-4">
          <RunsPane name={name} q={runsQ} />
        </TabsContent>
      </Tabs>
    </PageContainer>
  );
}

function chip(text?: string | null) {
  if (!text) return <span className="text-muted-foreground">—</span>;
  return <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-sm">{text}</span>;
}

function Row({ label, children }: { label: ReactNode; children: ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </>
  );
}

function InfoPane({ exp }: { exp: sdk.Experiment }) {
  const { t } = useTranslation();
  const role = exp.spec.roles?.[0];
  const tpl = role?.template;
  const command = tpl?.command ?? [];
  const env = tpl?.env ?? [];
  const policy = exp.spec.runPolicy;
  const timeout = policy?.activeDeadlineSeconds != null ? `${policy.activeDeadlineSeconds}s` : "—";
  const retries = policy?.backoffLimit != null ? String(policy.backoffLimit) : "—";

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("experiments.infoTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2.5 text-sm">
          <Row label={t("experiments.diName")}>{chip(exp.name)}</Row>
          <Row label={t("experiments.diDesc")}>{exp.description || "—"}</Row>
          <Row label={t("experiments.diImage")}>{chip(tpl?.image)}</Row>
          <Row label={t("experiments.diPool")}>{chip(exp.spec.poolName)}</Row>
          <Row label={t("experiments.diUnit")}>{chip(exp.spec.unitName)}</Row>
          <Row label={t("experiments.diReplicas")}>
            <span className="font-mono">{role?.replicas ?? "—"}</span>
          </Row>
          <Row label={t("experiments.diRunPolicy")}>
            {t("experiments.runPolicyValue", { timeout, retries })}
          </Row>
          <Row label={t("experiments.diCreator")}>
            {exp.owner}
            <span className="ml-2 font-mono text-muted-foreground">
              {exp.createdAt ? dayjs(exp.createdAt).format("YYYY-MM-DD") : ""}
            </span>
          </Row>
        </dl>

        <div className="mt-6 border-t pt-5">
          <div className="mb-1.5 text-xs text-muted-foreground">{t("experiments.diCommand")}</div>
          <pre className="m-0 mb-4 overflow-auto rounded-md bg-foreground p-4 font-mono text-xs leading-relaxed text-background">
            {command.length ? command.join(" ") : "—"}
          </pre>
          <div className="mb-2 text-xs text-muted-foreground">{t("experiments.diEnv")}</div>
          <div className="flex flex-wrap gap-2">
            {env.length ? (
              env.map((e) => (
                <span key={e.name} className="rounded bg-muted px-1.5 py-0.5 font-mono text-sm">
                  {e.name}
                  {e.value != null && e.value !== "" ? `=${e.value}` : ""}
                </span>
              ))
            ) : (
              <span className="text-muted-foreground">—</span>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

interface RunRow {
  name: string;
  phase?: sdk.RunPhase;
  unit: string;
  replicas: number;
  owner: string;
  duration: string;
}

function RunsPane({ name, q }: { name: string; q: UseQueryResult<sdk.RunList> }) {
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [search, setSearch] = useState("");
  const [phase, setPhase] = useState<string>("");
  const [trigger, setTrigger] = useState<string>("");

  const delRun = useApiMutation(
    (run: string) => sdk.deleteExperimentRun({ path: { name, run } }),
    { invalidate: [["experiments"]] },
  );
  const cancelRun = useApiMutation(
    (run: string) => sdk.cancelExperimentRun({ path: { name, run } }),
    { invalidate: [["experiments"]], success: t("experiments.runCanceled") },
  );

  const allRows: RunRow[] = useMemo(
    () =>
      q.data?.items?.map((r) => ({
        name: r.name,
        phase: r.phase,
        unit: r.unitName ?? "—",
        replicas: r.roles?.[0]?.replicas ?? r.spec?.roles?.[0]?.replicas ?? 0,
        owner: r.owner ?? "—",
        duration: fmtDuration(r.startedAt, r.finishedAt),
      })) ?? [],
    [q.data],
  );

  const triggerOptions = useMemo(
    () => Array.from(new Set(allRows.map((r) => r.owner).filter((o) => o && o !== "—"))),
    [allRows],
  );

  const rows = allRows.filter(
    (r) =>
      (!search || r.name.includes(search)) &&
      (!phase || r.phase === phase) &&
      (!trigger || r.owner === trigger),
  );

  const phaseOptions: sdk.RunPhase[] = ["Running", "Succeeded", "Failed", "Pending", "Canceling", "Cancelled"];

  const onDeleteRun = (r: RunRow) =>
    confirm({
      title: t("experiments.runDeleteTitle", { name: r.name }),
      desc: t("experiments.runDeleteDesc"),
      okLabel: t("common.confirmDelete"),
      toast: t("experiments.runDeleted", { name: r.name }),
      onConfirm: () => delRun.mutate(r.name),
    });
  const onCancelRun = (r: RunRow) =>
    confirm({
      title: t("experiments.runCancelTitle", { name: r.name }),
      desc: t("experiments.runCancelDesc"),
      okLabel: t("experiments.confirmCancel"),
      danger: false,
      onConfirm: () => cancelRun.mutate(r.name),
    });

  const columns: Column<RunRow>[] = [
    {
      key: "name",
      title: t("experiments.colRun"),
      render: (r) => (
        <Link
          to={`/experiments/${name}/runs/${r.name}`}
          className="font-mono font-medium text-foreground hover:text-info hover:underline"
        >
          {r.name}
        </Link>
      ),
    },
    {
      key: "phase",
      title: t("experiments.colRunStatus"),
      width: 130,
      render: (r) => <PhaseTag phase={r.phase} />,
    },
    {
      key: "unit",
      title: t("experiments.colRunUnit"),
      width: 180,
      render: (r) => <span className="font-mono">{r.unit}</span>,
    },
    { key: "replicas", title: t("experiments.colRunReplicas"), width: 90, align: "right", dataIndex: "replicas" },
    { key: "owner", title: t("experiments.colRunTrigger"), width: 130, dataIndex: "owner" },
    {
      key: "duration",
      title: t("experiments.colRunDuration"),
      width: 120,
      align: "right",
      render: (r) => <span className="font-mono">{r.duration}</span>,
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 200,
      align: "right",
      render: (r) => {
        const active = r.phase ? ACTIVE_RUN_PHASES.includes(r.phase) : false;
        return (
          <div className="flex items-center justify-end gap-0.5">
            <Button variant="link" size="sm" asChild>
              <Link to={`/experiments/${name}/runs/${r.name}`}>{t("common.detail")}</Link>
            </Button>
            <Button variant="link" size="sm">
              {t("experiments.actLog")}
            </Button>
            <Button variant="link" size="sm">
              {t("experiments.actMonitor")}
            </Button>
            {active ? (
              <Button variant="link" size="sm" onClick={() => onCancelRun(r)}>
                {t("experiments.actCancel")}
              </Button>
            ) : (
              <Button variant="link" size="sm" className="text-destructive" onClick={() => onDeleteRun(r)}>
                {t("common.delete")}
              </Button>
            )}
          </div>
        );
      },
    },
  ];

  return (
    <Card className="overflow-hidden p-0">
      <div className="flex flex-wrap items-center gap-3 border-b p-4">
        <div className="relative max-w-xs flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-8"
            placeholder={t("experiments.runsSearchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <Select value={phase || ALL} onValueChange={(v) => setPhase(v === ALL ? "" : v)}>
          <SelectTrigger className="min-w-40">
            <SelectValue placeholder={t("experiments.runStatusAll")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("experiments.runStatusAll")}</SelectItem>
            {phaseOptions.map((p) => (
              <SelectItem key={p} value={p}>
                <PhaseTag phase={p} />
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={trigger || ALL} onValueChange={(v) => setTrigger(v === ALL ? "" : v)}>
          <SelectTrigger className="min-w-40">
            <SelectValue placeholder={t("experiments.runTriggerAll")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("experiments.runTriggerAll")}</SelectItem>
            {triggerOptions.map((o) => (
              <SelectItem key={o} value={o}>
                {o}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          onClick={() => {
            setSearch("");
            setPhase("");
            setTrigger("");
          }}
        >
          {t("common.reset")}
        </Button>
      </div>
      <DataTable
        columns={columns}
        data={rows}
        rowKey={(r) => r.name}
        loading={q.isLoading}
        error={q.isError}
      />
    </Card>
  );
}
