import { Link, useParams } from "react-router-dom";
import { Play, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { useApiMutation } from "@/api/mutations";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { CodeBlock } from "@/components/code-block";
import { BackLink } from "@/components/back-link";
import { Descriptions, Desc } from "@/components/descriptions";
import { PageLoading, DetailError } from "@/components/page-state";
import { fmtDateTime, fmtDuration } from "@/lib/format";
import { DataTable, type Column } from "@/components/data-table";
import * as sdk from "@/api/generated";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardAction, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

// ── Shared detail helpers (reused by RunDetail) ───────────────────────────────
// The first defined role carries the Job/Run's primary spec (image, command, …).
export function primaryRole(roles?: sdk.MlRunRole[]): sdk.MlRunRole | undefined {
  return roles?.[0];
}

// Replica count of a Run, summed across role statuses.
export function runReplicas(r: sdk.Run): number | undefined {
  if (!r.roles?.length) return undefined;
  const total = r.roles.reduce((n, role) => n + (role.replicas ?? 0), 0);
  return total > 0 ? total : undefined;
}

// 自定义任务详情 / Job detail. Metadata comes from getJob; the run list comes
// from listRuns. Both are tenant-scoped, real backend reads — no fabricated rows.
export default function JobDetail() {
  const { name = "" } = useParams();
  const { t } = useTranslation();
  const { tenant } = useApp();

  const jobQ = useQuery({
    queryKey: ["jobs", tenant, name],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getJob({ path: { name } });
      if (error) throw error;
      return data;
    },
  });

  const job = jobQ.data;
  const role = job ? primaryRole(job.spec.roles) : undefined;
  const latestPhase = job?.runSummary?.latestPhase ?? job?.runSummary?.recent?.at(-1);

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.jobs"), name]}
      title={
        <span className="flex items-center gap-3">
          <span className="font-mono">{name}</span>
          {latestPhase && <PhaseTag phase={latestPhase} />}
        </span>
      }
      subtitle={<BackLink to="/jobs">{t("jobDetail.backJobs")}</BackLink>}
      extra={job && <Actions name={name} />}
    >
      {jobQ.isLoading ? (
        <PageLoading />
      ) : jobQ.isError || !job ? (
        <DetailError message={t("common.loadFailed")} />
      ) : (
        <Tabs defaultValue="info">
          <TabsList>
            <TabsTrigger value="info">{t("jobDetail.tabInfo")}</TabsTrigger>
            <TabsTrigger value="runs">{t("jobDetail.tabRuns")}</TabsTrigger>
          </TabsList>
          <TabsContent value="info" className="mt-4">
            <InfoPane job={job} role={role} />
          </TabsContent>
          <TabsContent value="runs" className="mt-4">
            <RunsPane name={name} />
          </TabsContent>
        </Tabs>
      )}
    </PageContainer>
  );
}

function Actions({ name }: { name: string }) {
  const { t } = useTranslation();
  const { confirm } = useUI();

  const trigger = useApiMutation((body: sdk.RunTriggerRequest) => sdk.triggerRun({ path: { name }, body }), {
    invalidate: [["jobs"], ["runs"]],
    success: t("jobDetail.triggered"),
  });
  const del = useApiMutation(() => sdk.deleteJob({ path: { name } }), {
    invalidate: [["jobs"]],
    success: t("jobDetail.deleted"),
  });

  return (
    <div className="flex items-center gap-2">
      <Button disabled={trigger.isPending} onClick={() => trigger.mutate({})}>
        {trigger.isPending ? <Spinner data-icon="inline-start" /> : <Play data-icon="inline-start" />}
        {t("jobDetail.runNow")}
      </Button>
      <Button
        variant="outline"
        className="text-destructive"
        onClick={() =>
          confirm({
            title: t("jobDetail.deleteTitle", { name }),
            desc: t("jobDetail.deleteDesc"),
            info: t("jobDetail.deleteInfo"),
            okLabel: t("common.confirmDelete"),
            onConfirm: () => del.mutate(undefined),
          })
        }
      >
        <Trash2 data-icon="inline-start" />
        {t("common.delete")}
      </Button>
    </div>
  );
}

function InfoPane({ job, role }: { job: sdk.Job; role?: sdk.MlRunRole }) {
  const { t } = useTranslation();
  const tpl = role?.template;
  const policy = job.spec.runPolicy;

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>{t("jobDetail.sectionInfo")}</CardTitle>
        <CardAction>
          <span className="text-xs text-muted-foreground">{t("jobDetail.editHint")}</span>
        </CardAction>
      </CardHeader>
      <CardContent>
        <Descriptions columns="double">
          <Desc label={t("jobDetail.fName")}>
            <span className="font-mono">{job.name}</span>
          </Desc>
          <Desc label={t("jobDetail.fDesc")}>{job.description || "—"}</Desc>
          <Desc label={t("jobDetail.fImage")}>
            {tpl?.image ? (
              <Badge variant="secondary" className="font-mono">
                {tpl.image}
              </Badge>
            ) : (
              "—"
            )}
          </Desc>
          <Desc label={t("jobDetail.fPool")}>
            {job.spec.poolName ? <span className="font-mono">{job.spec.poolName}</span> : "—"}
          </Desc>
          <Desc label={t("jobDetail.fUnit")}>
            {job.spec.unitName ? <span className="font-mono">{job.spec.unitName}</span> : "—"}
          </Desc>
          <Desc label={t("jobDetail.fReplicas")}>
            <span className="font-mono">{role?.replicas ?? "—"}</span>
          </Desc>
          <Desc label={t("jobDetail.fArtifacts")} span>
            <ArtifactTags artifacts={job.spec.artifacts} />
          </Desc>
          <Desc label={t("jobDetail.fRunPolicy")} span>
            <PolicyText policy={policy} />
          </Desc>
          <Desc label={t("jobDetail.fCreator")} span>
            {job.owner} · <span className="font-mono">{fmtDateTime(job.createdAt)}</span>
          </Desc>
        </Descriptions>

        <Separator className="my-5" />
        <CommandBlock tpl={tpl} />
        <EnvBlock tpl={tpl} />
      </CardContent>
    </Card>
  );
}

export function ArtifactTags({ artifacts }: { artifacts?: sdk.ArtifactRef[] }) {
  if (!artifacts || artifacts.length === 0) return <span className="text-muted-foreground">—</span>;
  return (
    <div className="flex flex-wrap gap-1">
      {artifacts.map((a) => (
        <Badge key={`${a.kind}/${a.name}/${a.version}`} variant="secondary" className="font-mono">
          {a.name}@{a.version}
        </Badge>
      ))}
    </div>
  );
}

export function PolicyText({ policy }: { policy?: sdk.RunPolicy }) {
  const { t } = useTranslation();
  if (!policy || (policy.activeDeadlineSeconds == null && policy.backoffLimit == null))
    return <span className="text-muted-foreground">—</span>;
  const parts: string[] = [];
  if (policy.activeDeadlineSeconds != null)
    parts.push(`${t("jobDetail.policyTimeout")} ${fmtDuration(policy.activeDeadlineSeconds)}`);
  if (policy.backoffLimit != null) parts.push(`${t("jobDetail.policyRetries")} ${policy.backoffLimit}`);
  return <span className="font-mono">{parts.join(" · ")}</span>;
}

export function CommandBlock({ tpl }: { tpl?: sdk.RoleTemplate }) {
  const { t } = useTranslation();
  const cmd = [...(tpl?.command ?? []), ...(tpl?.args ?? [])];
  return (
    <div className="mb-5">
      <div className="mb-1.5 text-xs text-muted-foreground">{t("jobDetail.command")}</div>
      {cmd.length ? (
        <CodeBlock>{cmd.join(" ")}</CodeBlock>
      ) : (
        <div className="text-sm text-muted-foreground">{t("jobDetail.noCommand")}</div>
      )}
    </div>
  );
}

export function EnvBlock({ tpl }: { tpl?: sdk.RoleTemplate }) {
  const { t } = useTranslation();
  const env = tpl?.env ?? [];
  return (
    <div>
      <div className="mb-1.5 text-xs text-muted-foreground">{t("jobDetail.env")}</div>
      {env.length ? (
        <div className="flex flex-wrap gap-1.5">
          {env.map((e) => (
            <Badge key={e.name} variant="secondary" className="font-mono">
              {e.name}={e.value ?? ""}
            </Badge>
          ))}
        </div>
      ) : (
        <div className="text-sm text-muted-foreground">{t("jobDetail.noEnv")}</div>
      )}
    </div>
  );
}

function RunsPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { confirm } = useUI();

  const runsQ = useQuery({
    queryKey: ["runs", tenant, name],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.listRuns({ path: { name } });
      if (error) throw error;
      return data;
    },
  });

  const cancel = useApiMutation((run: string) => sdk.cancelRun({ path: { name, run } }), {
    invalidate: [["runs"]],
    success: t("jobDetail.cancelled"),
  });
  const del = useApiMutation((run: string) => sdk.deleteRun({ path: { name, run } }), {
    invalidate: [["runs"]],
    success: t("jobDetail.runDeleted"),
  });

  const active = (p?: string) => p === "Creating" || p === "Pending" || p === "Running" || p === "Canceling";

  const columns: Column<sdk.Run>[] = [
    {
      key: "name",
      title: t("jobDetail.colRun"),
      render: (r) => (
        <Link
          to={`/jobs/${name}/runs/${r.name}`}
          className="font-mono font-medium text-foreground hover:text-info hover:underline"
        >
          {r.name}
        </Link>
      ),
    },
    { key: "phase", title: t("jobDetail.colStatus"), width: 120, render: (r) => <PhaseTag phase={r.phase} /> },
    {
      key: "unitName",
      title: t("jobDetail.colUnit"),
      width: 160,
      render: (r) => <span className="font-mono">{r.unitName || "—"}</span>,
    },
    {
      key: "replicas",
      title: t("jobDetail.colReplicas"),
      width: 80,
      align: "right",
      render: (r) => runReplicas(r) ?? "—",
    },
    { key: "owner", title: t("jobDetail.colCreator"), width: 120, render: (r) => r.owner || "—" },
    {
      key: "startedAt",
      title: t("jobDetail.colStarted"),
      width: 170,
      render: (r) => <span className="text-muted-foreground">{fmtDateTime(r.startedAt)}</span>,
    },
    {
      key: "duration",
      title: t("jobDetail.colDuration"),
      width: 110,
      align: "right",
      render: (r) => <span className="font-mono">{durationOf(r)}</span>,
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 130,
      align: "right",
      render: (r) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" asChild>
            <Link to={`/jobs/${name}/runs/${r.name}`}>{t("common.detail")}</Link>
          </Button>
          {active(r.phase) ? (
            <Button
              variant="link"
              size="sm"
              onClick={() =>
                confirm({
                  title: t("jobDetail.cancelTitle", { run: r.name }),
                  desc: t("jobDetail.cancelDesc"),
                  okLabel: t("jobDetail.cancelOk"),
                  danger: false,
                  onConfirm: () => cancel.mutate(r.name),
                })
              }
            >
              {t("runDetail.cancelRun")}
            </Button>
          ) : (
            <Button
              variant="link"
              size="sm"
              className="text-destructive"
              onClick={() =>
                confirm({
                  title: t("jobDetail.runDeleteTitle", { run: r.name }),
                  desc: t("jobDetail.runDeleteDesc"),
                  okLabel: t("common.confirmDelete"),
                  onConfirm: () => del.mutate(r.name),
                })
              }
            >
              {t("common.delete")}
            </Button>
          )}
        </div>
      ),
    },
  ];

  return (
    <Card className="overflow-hidden p-0">
      <DataTable
        columns={columns}
        data={runsQ.data?.items ?? []}
        rowKey={(r) => r.name}
        loading={runsQ.isLoading}
        error={runsQ.isError}
        empty={t("jobDetail.noRuns")}
      />
    </Card>
  );
}

function durationOf(r: sdk.Run): string {
  if (!r.startedAt) return "—";
  const end = r.finishedAt ? dayjs(r.finishedAt) : dayjs();
  const secs = end.diff(dayjs(r.startedAt), "second");
  return secs >= 0 ? fmtDuration(secs) : "—";
}
