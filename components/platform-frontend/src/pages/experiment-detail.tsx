import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { Play, Trash2, LineChart } from "lucide-react";
import { useTranslation } from "react-i18next";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useApiMutation } from "@/api/mutations";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { BackLink } from "@/components/back-link";
import { DataTable, type Column } from "@/components/data-table";
import { SearchInput } from "@/components/search-input";
import { FilterSelect } from "@/components/filter-select";
import { MonoChip } from "@/components/mono-chip";
import { ExpDrawer } from "@/components/exp-drawer";
import { CodeBlock } from "@/components/code-block";
import { Descriptions, Desc } from "@/components/descriptions";
import { PageLoading, DetailError } from "@/components/page-state";
import { fmtRange, fmtDateTime } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
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

// Experiments are specialized training Jobs (Job→Run model); this detail page
// mirrors JobDetail — an "experiment info" pane plus a runs table. Both the
// definition and its Runs come from the live API, scoped to the active tenant.
const ACTIVE_RUN_PHASES: sdk.RunPhase[] = ["Creating", "Pending", "Running", "Canceling"];

export default function ExperimentDetail() {
  const { name = "" } = useParams<{ name: string }>();
  const { tenant } = useApp();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [editing, setEditing] = useState(false);

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

  const backLink = <BackLink to="/experiments">{t("experiments.backToList")}</BackLink>;

  if (expQ.isError) {
    return (
      <PageContainer
        breadcrumb={[t("nav.trainingCenter"), t("nav.experiments"), name]}
        title={<span className="font-mono">{name}</span>}
        subtitle={backLink}
      >
        <DetailError message={t("common.loadFailed")} />
      </PageContainer>
    );
  }

  if (expQ.isLoading || !expQ.data) {
    return (
      <PageContainer
        breadcrumb={[t("nav.trainingCenter"), t("nav.experiments"), name]}
        title={<span className="font-mono">{name}</span>}
        subtitle={backLink}
      >
        <PageLoading />
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
      subtitle={backLink}
      extra={
        <div className="flex items-center gap-2">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="outline" size="icon">
                <LineChart />
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
          <InfoPane exp={exp} onEdit={() => setEditing(true)} />
        </TabsContent>
        <TabsContent value="runs" className="mt-4">
          <RunsPane name={name} q={runsQ} />
        </TabsContent>
      </Tabs>

      {editing && <ExpDrawer mode="edit" name={name} onClose={() => setEditing(false)} />}
    </PageContainer>
  );
}

function chip(text?: string | null) {
  if (!text) return <span className="text-muted-foreground">—</span>;
  return <MonoChip>{text}</MonoChip>;
}

interface VolumeMount {
  name?: string;
  mountPath?: string;
}

function InfoPane({ exp, onEdit }: { exp: sdk.Experiment; onEdit: () => void }) {
  const { t } = useTranslation();
  const role = exp.spec.roles?.[0];
  const tpl = role?.template;
  const command = tpl?.command ?? [];
  const env = tpl?.env ?? [];
  const mounts = (tpl?.volumeMounts ?? []) as VolumeMount[];
  const policy = exp.spec.runPolicy;
  const timeout = policy?.activeDeadlineSeconds != null ? `${policy.activeDeadlineSeconds}s` : "—";
  const retries = policy?.backoffLimit != null ? String(policy.backoffLimit) : "—";

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>{t("experiments.infoTitle")}</CardTitle>
        <CardAction>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="outline" size="sm" onClick={onEdit}>
                {t("experiments.edit")}
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("experiments.editHint")}</TooltipContent>
          </Tooltip>
        </CardAction>
      </CardHeader>
      <CardContent>
        <Descriptions columns="single">
          <Desc label={t("experiments.diName")}>{chip(exp.name)}</Desc>
          <Desc label={t("experiments.diDesc")}>{exp.description || "—"}</Desc>
          <Desc label={t("experiments.diImage")}>{chip(tpl?.image)}</Desc>
          <Desc label={t("experiments.diPool")}>{chip(exp.spec.poolName)}</Desc>
          <Desc label={t("experiments.diUnit")}>{chip(exp.spec.unitName)}</Desc>
          <Desc label={t("experiments.diReplicas")}>
            <span className="font-mono">{role?.replicas ?? "—"}</span>
          </Desc>
          <Desc label={t("experiments.diVolume")}>
            {mounts.length ? (
              <div className="flex flex-wrap gap-2">
                {mounts.map((m, i) => (
                  <MonoChip key={i}>
                    {m.name}
                    {m.mountPath ? ` → ${m.mountPath}` : ""}
                  </MonoChip>
                ))}
              </div>
            ) : (
              <span className="text-muted-foreground">—</span>
            )}
          </Desc>
          <Desc label={t("experiments.diRunPolicy")}>
            {t("experiments.runPolicyValue", { timeout, retries })}
          </Desc>
          <Desc label={t("experiments.diCreator")}>
            {exp.owner}
            <span className="ml-2 font-mono text-muted-foreground">{fmtDateTime(exp.createdAt)}</span>
          </Desc>
        </Descriptions>

        <div className="mt-6 border-t pt-5">
          <div className="mb-1.5 text-xs text-muted-foreground">{t("experiments.diCommand")}</div>
          <CodeBlock className="mb-4">{command.length ? command.join(" ") : "—"}</CodeBlock>
          <div className="mb-2 text-xs text-muted-foreground">{t("experiments.diEnv")}</div>
          <div className="flex flex-wrap gap-2">
            {env.length ? (
              env.map((e) => (
                <MonoChip key={e.name}>
                  {e.name}
                  {e.value != null && e.value !== "" ? `=${e.value}` : ""}
                </MonoChip>
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
        duration: fmtRange(r.startedAt, r.finishedAt),
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
        <SearchInput
          className="max-w-xs flex-1"
          placeholder={t("experiments.runsSearchPlaceholder")}
          value={search}
          onChange={setSearch}
        />
        <FilterSelect
          value={phase}
          onChange={setPhase}
          options={phaseOptions.map((p) => ({ value: p, label: t(`phase.${p}`, { defaultValue: p }) }))}
          allLabel={t("experiments.runStatusAll")}
        />
        <FilterSelect
          value={trigger}
          onChange={setTrigger}
          options={triggerOptions}
          allLabel={t("experiments.runTriggerAll")}
        />
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
        <span className="ml-auto text-xs text-muted-foreground">
          {t("experiments.runsHint", { count: rows.length })}
        </span>
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
