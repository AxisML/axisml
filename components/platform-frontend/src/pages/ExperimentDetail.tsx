import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import {
  Table,
  Card,
  Tabs,
  Descriptions,
  Tag,
  Button,
  Space,
  Tooltip,
  Input,
  Select,
  Spin,
  Result,
  Breadcrumb,
  Divider,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  ArrowLeftOutlined,
  CaretRightOutlined,
  DeleteOutlined,
  SearchOutlined,
  FundOutlined,
} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useApiMutation } from "@/api/mutations";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";

// Experiments are specialized training Jobs (Job→Run model); this detail page
// mirrors JobDetail — an "experiment info" pane plus a runs table. Both the
// definition and its Runs come from the live API, scoped to the active tenant.
const ACTIVE_RUN_PHASES: sdk.RunPhase[] = ["Creating", "Pending", "Running", "Canceling"];

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

  const breadcrumb = (
    <Breadcrumb
      className="mb-3"
      items={[
        { title: t("nav.trainingCenter") },
        { title: <Link to="/experiments">{t("nav.experiments")}</Link> },
        { title: name },
      ]}
    />
  );

  if (expQ.isError) {
    return (
      <div className="mx-auto max-w-[1200px] p-6">
        {breadcrumb}
        <Result status="error" title={t("common.loadFailed")} extra={<Link to="/experiments"><Button>{t("experiments.backToList")}</Button></Link>} />
      </div>
    );
  }

  if (expQ.isLoading || !expQ.data) {
    return (
      <div className="mx-auto max-w-[1200px] p-6">
        {breadcrumb}
        <div className="grid place-items-center py-24">
          <Spin size="large" />
        </div>
      </div>
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
      breadcrumb={[t("nav.trainingCenter"), t("nav.experiments")]}
      title={
        <Space size="middle" align="center" className="min-w-0">
          <span className="font-mono">{name}</span>
        </Space>
      }
      subtitle={
        <span>
          {exp.description || exp.displayName || "—"}
          <span className="ml-2 text-muted">· {t("experiments.headRunsSummary", { count: runCount, owner: exp.owner ?? "—" })}</span>
        </span>
      }
      extra={
        <Space>
          <Link to="/experiments">
            <Button icon={<ArrowLeftOutlined />}>{t("experiments.backToList")}</Button>
          </Link>
          <Tooltip title={t("experiments.tensorboard")}>
            <Button icon={<FundOutlined style={{ color: "#FF6F00" }} />} />
          </Tooltip>
          <Button type="primary" icon={<CaretRightOutlined />} onClick={onRun}>
            {t("experiments.runAction")}
          </Button>
          <Button danger icon={<DeleteOutlined />} onClick={onDelete}>
            {t("common.delete")}
          </Button>
        </Space>
      }
    >
      <Tabs
        items={[
          { key: "info", label: t("experiments.tabInfo"), children: <InfoPane exp={exp} /> },
          {
            key: "runs",
            label: (
              <span>
                {t("experiments.tabRuns")}
                <Tag className="!ml-2 !mr-0" bordered={false}>
                  {runCount}
                </Tag>
              </span>
            ),
            children: <RunsPane name={name} q={runsQ} />,
          },
        ]}
      />
    </PageContainer>
  );
}

function chip(text?: string | null) {
  if (!text) return <span className="text-muted">—</span>;
  return <span className="font-mono rounded bg-surface-warm px-1.5 py-0.5 text-sm">{text}</span>;
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
    <Card title={t("experiments.infoTitle")}>
      <Descriptions column={1} size="middle" styles={{ label: { width: 120 } }}>
        <Descriptions.Item label={t("experiments.diName")}>{chip(exp.name)}</Descriptions.Item>
        <Descriptions.Item label={t("experiments.diDesc")}>{exp.description || "—"}</Descriptions.Item>
        <Descriptions.Item label={t("experiments.diImage")}>{chip(tpl?.image)}</Descriptions.Item>
        <Descriptions.Item label={t("experiments.diPool")}>{chip(exp.spec.poolName)}</Descriptions.Item>
        <Descriptions.Item label={t("experiments.diUnit")}>{chip(exp.spec.unitName)}</Descriptions.Item>
        <Descriptions.Item label={t("experiments.diReplicas")}>
          <span className="font-mono">{role?.replicas ?? "—"}</span>
        </Descriptions.Item>
        <Descriptions.Item label={t("experiments.diRunPolicy")}>
          {t("experiments.runPolicyValue", { timeout, retries })}
        </Descriptions.Item>
        <Descriptions.Item label={t("experiments.diCreator")}>
          {exp.owner}
          <span className="ml-2 font-mono text-muted">{exp.createdAt ? dayjs(exp.createdAt).format("YYYY-MM-DD") : ""}</span>
        </Descriptions.Item>
      </Descriptions>

      <div className="mt-6 border-t border-border-soft pt-5">
        <div className="mb-1.5 text-xs text-muted">{t("experiments.diCommand")}</div>
        <pre className="m-0 mb-4 overflow-auto rounded-md bg-bg p-3 font-mono text-sm text-fg-2">
          {command.length ? command.join(" ") : "—"}
        </pre>
        <div className="mb-2 text-xs text-muted">{t("experiments.diEnv")}</div>
        <Space size={[8, 8]} wrap>
          {env.length ? (
            env.map((e) => (
              <span key={e.name} className="font-mono rounded bg-surface-warm px-1.5 py-0.5 text-sm">
                {e.name}
                {e.value != null && e.value !== "" ? `=${e.value}` : ""}
              </span>
            ))
          ) : (
            <span className="text-muted">—</span>
          )}
        </Space>
      </div>
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

  const columns: ColumnsType<RunRow> = [
    {
      title: t("experiments.colRun"),
      dataIndex: "name",
      render: (_, r) => (
        <Link to={`/experiments/${name}/runs/${r.name}`} className="font-mono font-medium">
          {r.name}
        </Link>
      ),
    },
    {
      title: t("experiments.colRunStatus"),
      dataIndex: "phase",
      width: 130,
      render: (p: sdk.RunPhase | undefined) => <PhaseTag phase={p} />,
    },
    { title: t("experiments.colRunUnit"), dataIndex: "unit", width: 180, render: (v: string) => <span className="font-mono">{v}</span> },
    { title: t("experiments.colRunReplicas"), dataIndex: "replicas", width: 90, align: "right" },
    { title: t("experiments.colRunTrigger"), dataIndex: "owner", width: 130 },
    { title: t("experiments.colRunDuration"), dataIndex: "duration", width: 120, align: "right", render: (v: string) => <span className="font-mono">{v}</span> },
    {
      title: t("common.actions"),
      key: "actions",
      width: 170,
      align: "right",
      render: (_, r) => {
        const active = r.phase ? ACTIVE_RUN_PHASES.includes(r.phase) : false;
        return (
          <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
            <Link to={`/experiments/${name}/runs/${r.name}`}>
              <Button type="link" size="small" className="!px-1">
                {t("common.detail")}
              </Button>
            </Link>
            <Button type="link" size="small" className="!px-1">
              {t("experiments.actLog")}
            </Button>
            <Button type="link" size="small" className="!px-1">
              {t("experiments.actMonitor")}
            </Button>
            {active ? (
              <Button type="link" size="small" className="!px-1" onClick={() => onCancelRun(r)}>
                {t("experiments.actCancel")}
              </Button>
            ) : (
              <Button type="link" size="small" danger className="!px-1" onClick={() => onDeleteRun(r)}>
                {t("common.delete")}
              </Button>
            )}
          </Space>
        );
      },
    },
  ];

  return (
    <>
      <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 border-b border-border-soft p-4">
          <Input
            allowClear
            prefix={<SearchOutlined className="text-muted" />}
            placeholder={t("experiments.runsSearchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
          <Select
            value={phase || undefined}
            onChange={(v) => setPhase(v ?? "")}
            placeholder={t("experiments.runStatusAll")}
            allowClear
            className="min-w-40"
            options={phaseOptions.map((p) => ({ label: <PhaseTag phase={p} />, value: p }))}
          />
          <Select
            value={trigger || undefined}
            onChange={(v) => setTrigger(v ?? "")}
            placeholder={t("experiments.runTriggerAll")}
            allowClear
            className="min-w-40"
            options={triggerOptions.map((o) => ({ label: o, value: o }))}
          />
          <Button
            onClick={() => {
              setSearch("");
              setPhase("");
              setTrigger("");
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <Table<RunRow>
          rowKey="name"
          columns={columns}
          dataSource={rows}
          loading={q.isLoading}
          pagination={{ pageSize: 20, showTotal: (n) => t("experiments.runTotal", { count: n }), hideOnSinglePage: false }}
          locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
        />
      </Card>
    </>
  );
}
