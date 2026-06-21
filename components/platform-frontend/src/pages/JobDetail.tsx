import { Link, useParams } from "react-router-dom";
import {
  Card,
  Tabs,
  Table,
  Descriptions,
  Button,
  Space,
  Divider,
  Tag,
  Empty,
  Spin,
  Result,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  CaretRightOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { useApiMutation } from "@/api/mutations";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";
import { USE_MOCK } from "@/api/mock";
import { runSummary } from "@/api/mock/data";
import * as sdk from "@/api/generated";

// ── Shared detail helpers (reused by RunDetail) ───────────────────────────────
export function fmtDateTime(v?: string | null): string {
  return v ? dayjs(v).format("YYYY-MM-DD HH:mm") : "—";
}

// Humanize a second count into a compact h/m/s string.
export function fmtDuration(secs: number): string {
  if (!Number.isFinite(secs) || secs < 0) return "—";
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  const s = Math.floor(secs % 60);
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

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
  const latestPhase = USE_MOCK ? runSummary(name).recent.at(-1) : undefined;

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.jobs"), name]}
      title={
        <span className="flex items-center gap-3">
          <span className="font-mono">{name}</span>
          {latestPhase && <PhaseTag phase={latestPhase} />}
        </span>
      }
      subtitle={
        <Link to="/jobs" className="inline-flex items-center gap-1 text-sm">
          <ArrowLeftOutlined /> {t("jobDetail.backJobs")}
        </Link>
      }
      extra={job && <Actions name={name} />}
    >
      {jobQ.isLoading ? (
        <div className="grid place-items-center py-24">
          <Spin />
        </div>
      ) : jobQ.isError || !job ? (
        <Result status="error" title={t("common.loadFailed")} />
      ) : (
        <Tabs
          items={[
            { key: "info", label: t("jobDetail.tabInfo"), children: <InfoPane job={job} role={role} /> },
            { key: "runs", label: t("jobDetail.tabRuns"), children: <RunsPane name={name} /> },
          ]}
        />
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
    <Space>
      <Button type="primary" icon={<CaretRightOutlined />} loading={trigger.isPending} onClick={() => trigger.mutate({})}>
        {t("jobDetail.runNow")}
      </Button>
      <Button
        danger
        icon={<DeleteOutlined />}
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
        {t("common.delete")}
      </Button>
    </Space>
  );
}

function InfoPane({ job, role }: { job: sdk.Job; role?: sdk.MlRunRole }) {
  const { t } = useTranslation();
  const tpl = role?.template;
  const policy = job.spec.runPolicy;

  return (
    <Card title={t("jobDetail.sectionInfo")} extra={<span className="text-xs text-muted">{t("jobDetail.editHint")}</span>}>
      <Descriptions column={{ xs: 1, md: 2 }} bordered size="middle">
        <Descriptions.Item label={t("jobDetail.fName")}>
          <span className="font-mono">{job.name}</span>
        </Descriptions.Item>
        <Descriptions.Item label={t("jobDetail.fDesc")}>{job.description || "—"}</Descriptions.Item>
        <Descriptions.Item label={t("jobDetail.fImage")}>
          {tpl?.image ? <Tag className="!m-0 font-mono">{tpl.image}</Tag> : "—"}
        </Descriptions.Item>
        <Descriptions.Item label={t("jobDetail.fPool")}>
          {job.spec.poolName ? <span className="font-mono">{job.spec.poolName}</span> : "—"}
        </Descriptions.Item>
        <Descriptions.Item label={t("jobDetail.fUnit")}>
          {job.spec.unitName ? <span className="font-mono">{job.spec.unitName}</span> : "—"}
        </Descriptions.Item>
        <Descriptions.Item label={t("jobDetail.fReplicas")}>
          <span className="font-mono">{role?.replicas ?? "—"}</span>
        </Descriptions.Item>
        <Descriptions.Item label={t("jobDetail.fArtifacts")} span={2}>
          <ArtifactTags artifacts={job.spec.artifacts} />
        </Descriptions.Item>
        <Descriptions.Item label={t("jobDetail.fRunPolicy")} span={2}>
          <PolicyText policy={policy} />
        </Descriptions.Item>
        <Descriptions.Item label={t("jobDetail.fCreator")} span={2}>
          {job.owner} · <span className="font-mono">{fmtDateTime(job.createdAt)}</span>
        </Descriptions.Item>
      </Descriptions>

      <div className="mt-6 border-t border-border-soft pt-5">
        <CommandBlock tpl={tpl} />
        <EnvBlock tpl={tpl} />
      </div>
    </Card>
  );
}

export function ArtifactTags({ artifacts }: { artifacts?: sdk.ArtifactRef[] }) {
  if (!artifacts || artifacts.length === 0) return <span className="text-muted">—</span>;
  return (
    <Space size={[4, 4]} wrap>
      {artifacts.map((a) => (
        <Tag key={`${a.kind}/${a.name}/${a.version}`} className="!m-0 font-mono">
          {a.name}@{a.version}
        </Tag>
      ))}
    </Space>
  );
}

export function PolicyText({ policy }: { policy?: sdk.RunPolicy }) {
  const { t } = useTranslation();
  if (!policy || (policy.activeDeadlineSeconds == null && policy.backoffLimit == null))
    return <span className="text-muted">—</span>;
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
      <div className="mb-1.5 text-xs text-muted">{t("jobDetail.command")}</div>
      {cmd.length ? (
        <pre
          className="m-0 overflow-auto rounded-md p-4 font-mono text-xs leading-relaxed"
          style={{ background: "#16181d", color: "#e6e6e6" }}
        >
          {cmd.join(" ")}
        </pre>
      ) : (
        <div className="text-sm text-muted">{t("jobDetail.noCommand")}</div>
      )}
    </div>
  );
}

export function EnvBlock({ tpl }: { tpl?: sdk.RoleTemplate }) {
  const { t } = useTranslation();
  const env = tpl?.env ?? [];
  return (
    <div>
      <div className="mb-1.5 text-xs text-muted">{t("jobDetail.env")}</div>
      {env.length ? (
        <Space size={[6, 6]} wrap>
          {env.map((e) => (
            <Tag key={e.name} className="!m-0 font-mono">
              {e.name}={e.value ?? ""}
            </Tag>
          ))}
        </Space>
      ) : (
        <div className="text-sm text-muted">{t("jobDetail.noEnv")}</div>
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

  const columns: ColumnsType<sdk.Run> = [
    {
      title: t("jobDetail.colRun"),
      dataIndex: "name",
      render: (_, r) => (
        <Link to={`/jobs/${name}/runs/${r.name}`} className="font-mono font-medium">
          {r.name}
        </Link>
      ),
    },
    { title: t("jobDetail.colStatus"), key: "phase", width: 120, render: (_, r) => <PhaseTag phase={r.phase} /> },
    {
      title: t("jobDetail.colUnit"),
      dataIndex: "unitName",
      width: 160,
      render: (v: string) => <span className="font-mono">{v || "—"}</span>,
    },
    {
      title: t("jobDetail.colReplicas"),
      key: "replicas",
      width: 80,
      align: "right",
      render: (_, r) => runReplicas(r) ?? "—",
    },
    { title: t("jobDetail.colCreator"), dataIndex: "owner", width: 120, render: (v: string) => v || "—" },
    {
      title: t("jobDetail.colStarted"),
      dataIndex: "startedAt",
      width: 170,
      render: (v: string) => <span className="text-muted">{fmtDateTime(v)}</span>,
    },
    {
      title: t("jobDetail.colDuration"),
      key: "duration",
      width: 110,
      align: "right",
      render: (_, r) => <span className="font-mono">{durationOf(r)}</span>,
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 130,
      align: "right",
      render: (_, r) => (
        <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
          <Link to={`/jobs/${name}/runs/${r.name}`}>
            <Button type="link" size="small" className="!px-1">
              {t("common.detail")}
            </Button>
          </Link>
          {active(r.phase) ? (
            <Button
              type="link"
              size="small"
              className="!px-1"
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
              type="link"
              size="small"
              danger
              className="!px-1"
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
        </Space>
      ),
    },
  ];

  return (
    <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
      <Table<sdk.Run>
        rowKey="name"
        columns={columns}
        dataSource={runsQ.data?.items ?? []}
        loading={runsQ.isLoading}
        pagination={{ pageSize: 20, hideOnSinglePage: true }}
        locale={{
          emptyText: runsQ.isError ? (
            <Empty description={t("common.loadFailed")} />
          ) : (
            <Empty description={t("jobDetail.noRuns")} />
          ),
        }}
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
