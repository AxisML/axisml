import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  Card,
  Tabs,
  Table,
  Descriptions,
  Tag,
  Button,
  Space,
  Select,
  Steps,
  Empty,
  Spin,
  Result,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { ArrowLeftOutlined, StopOutlined, ReloadOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { useApiMutation } from "@/api/mutations";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";
import { LogViewer } from "@/components/LogViewer";
import * as sdk from "@/api/generated";
import { PolicyText, fmtDateTime, primaryRole } from "./JobDetail";

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
        <Space size="middle" align="center" className="min-w-0">
          <span className="font-mono">{run}</span>
          {r && <PhaseTag phase={r.phase} />}
        </Space>
      }
      subtitle={
        <span>
          <Link to={base} className="inline-flex items-center gap-1 text-sm">
            <ArrowLeftOutlined /> {t("runDetail.backTo", { name })}
          </Link>
          {r && (
            <span className="ml-3 text-muted">
              {t("runDetail.subtitle", { num: r.runNumber ?? "—", owner: r.owner || "—" })}
            </span>
          )}
        </span>
      }
      extra={
        r && (
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => runQ.refetch()}>
              {t("runDetail.refresh")}
            </Button>
            {active && (
              <Button
                danger
                icon={<StopOutlined />}
                loading={cancel.isPending}
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
                {t("runDetail.cancelRun")}
              </Button>
            )}
          </Space>
        )
      }
    >
      {runQ.isLoading ? (
        <div className="grid place-items-center py-24">
          <Spin size="large" />
        </div>
      ) : runQ.isError || !r ? (
        <Result
          status="error"
          title={t("common.loadFailed")}
          extra={
            <Link to={base}>
              <Button>{t("runDetail.backTo", { name })}</Button>
            </Link>
          }
        />
      ) : (
        <>
          <Lifecycle run={r} />
          <Tabs
            items={[
              { key: "info", label: t("runDetail.tabInfo"), children: <InfoPane run={r} /> },
              { key: "pods", label: t("runDetail.tabPods"), children: <PodsPane kind={kind} name={name} run={run} /> },
              { key: "log", label: t("runDetail.tabLog"), children: <LogPane kind={kind} name={name} run={run} /> },
              { key: "ev", label: t("runDetail.tabEvents"), children: <EventsPane kind={kind} name={name} run={run} /> },
            ]}
          />
        </>
      )}
    </PageContainer>
  );
}

function Lifecycle({ run }: { run: sdk.Run }) {
  const { t } = useTranslation();
  const { current, error } = lifecycleState(run.phase);
  return (
    <Card className="mb-4">
      <Steps
        size="small"
        current={current}
        status={error ? "error" : "process"}
        items={[
          { title: t("runDetail.stepCreated"), description: fmtDateTime(run.createdAt) },
          { title: t("runDetail.stepScheduled") },
          { title: t("runDetail.stepRunning"), description: fmtDateTime(run.startedAt) },
          { title: t("runDetail.stepFinished"), description: fmtDateTime(run.finishedAt) },
        ]}
      />
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
    <Card title={t("runDetail.sectionConfig")}>
      <Descriptions column={{ xs: 1, md: 2 }} bordered size="middle">
        <Descriptions.Item label={t("runDetail.fName")}>
          <span className="font-mono">{run.name}</span>
        </Descriptions.Item>
        <Descriptions.Item label={t("runDetail.fDesc")}>{run.description || "—"}</Descriptions.Item>
        <Descriptions.Item label={t("runDetail.fImage")}>
          {tpl?.image ? <Tag className="!m-0 font-mono">{tpl.image}</Tag> : "—"}
        </Descriptions.Item>
        <Descriptions.Item label={t("runDetail.fPool")}>
          {run.poolName ? <span className="font-mono">{run.poolName}</span> : "—"}
        </Descriptions.Item>
        <Descriptions.Item label={t("runDetail.fUnit")}>
          {run.unitName ? <span className="font-mono">{run.unitName}</span> : "—"}
        </Descriptions.Item>
        <Descriptions.Item label={t("runDetail.fReplicas")}>
          <span className="font-mono">{role?.replicas ?? "—"}</span>
        </Descriptions.Item>
        <Descriptions.Item label={t("runDetail.fRunPolicy")} span={2}>
          <PolicyText policy={run.runPolicy ?? run.spec?.runPolicy} />
        </Descriptions.Item>
        <Descriptions.Item label={t("runDetail.fStarted")}>
          <span className="font-mono text-muted">{fmtDateTime(run.startedAt)}</span>
        </Descriptions.Item>
        <Descriptions.Item label={t("runDetail.fFinished")}>
          <span className="font-mono text-muted">{fmtDateTime(run.finishedAt)}</span>
        </Descriptions.Item>
      </Descriptions>

      <div className="mt-6 border-t border-border-soft pt-5">
        <div className="mb-1.5 text-xs text-muted">{t("runDetail.command")}</div>
        {command.length ? (
          <pre className="m-0 mb-5 overflow-auto rounded-md bg-surface-warm p-3 font-mono text-xs text-fg-2">
            {command.join(" ")}
          </pre>
        ) : (
          <div className="mb-5 text-sm text-muted">{t("runDetail.noCommand")}</div>
        )}
        <div className="mb-1.5 text-xs text-muted">{t("runDetail.env")}</div>
        {env.length ? (
          <Space size={[6, 6]} wrap>
            {env.map((e) => (
              <Tag key={e.name} className="!m-0 font-mono">
                {e.name}={e.value ?? ""}
              </Tag>
            ))}
          </Space>
        ) : (
          <div className="text-sm text-muted">{t("runDetail.noEnv")}</div>
        )}
      </div>
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

  const columns: ColumnsType<sdk.Pod> = [
    {
      title: t("runDetail.colPod"),
      dataIndex: "name",
      render: (v: string) => <span className="font-mono">{v}</span>,
    },
    {
      title: t("runDetail.colPhase"),
      dataIndex: "phase",
      width: 120,
      render: (p: string) => <PhaseTag phase={p} />,
    },
    {
      title: t("runDetail.colRole"),
      dataIndex: "role",
      width: 140,
      render: (v?: string) => <span className="font-mono">{v || "—"}</span>,
    },
    {
      title: t("runDetail.colReady"),
      key: "restarts",
      width: 90,
      align: "right",
      render: (_, p) => <span className="font-mono">{p.restartCount ?? 0}</span>,
    },
    {
      title: t("runDetail.fStarted"),
      dataIndex: "startedAt",
      width: 170,
      render: (v?: string | null) => <span className="text-muted">{fmtDateTime(v)}</span>,
    },
  ];

  return (
    <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
      <Table<sdk.Pod>
        rowKey="name"
        columns={columns}
        dataSource={podsQ.data?.items ?? []}
        loading={podsQ.isLoading}
        pagination={{ pageSize: 20, hideOnSinglePage: true }}
        locale={{
          emptyText: podsQ.isError ? (
            <Empty description={t("common.loadFailed")} />
          ) : (
            <Empty description={t("runDetail.noPods")} />
          ),
        }}
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
    <Card
      title={t("runDetail.logTitle")}
      extra={
        <Space>
          <Select
            size="small"
            value={pod || undefined}
            placeholder={t("runDetail.colPod")}
            onChange={setPod}
            className="min-w-52"
            options={pods.map((p) => ({ label: p.name, value: p.name }))}
          />
          <Button size="small" icon={<ReloadOutlined />} onClick={() => logsQ.refetch()} />
        </Space>
      }
    >
      {podsQ.isLoading || logsQ.isLoading ? (
        <div className="grid place-items-center py-16">
          <Spin />
        </div>
      ) : !pods.length ? (
        <div className="rounded-md bg-surface-warm p-6">
          <Empty description={t("runDetail.noLog")} />
        </div>
      ) : (
        <LogViewer text={logsQ.data} empty={t("runDetail.noLog")} />
      )}
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

  const columns: ColumnsType<sdk.Event> = [
    {
      title: t("runDetail.colReason"),
      dataIndex: "reason",
      width: 180,
      render: (v: string) => <span className="font-mono">{v}</span>,
    },
    {
      title: t("runDetail.colType"),
      dataIndex: "type",
      width: 120,
      render: (v: string) => (
        <Tag color={v === "Warning" ? "warning" : "default"} className="!m-0">
          {v === "Warning" ? t("runDetail.eventWarning") : t("runDetail.eventNormal")}
        </Tag>
      ),
    },
    { title: t("runDetail.colMessage"), dataIndex: "message", render: (v: string) => <span className="text-fg-2">{v}</span> },
    {
      title: t("runDetail.colCount"),
      dataIndex: "count",
      width: 80,
      align: "right",
      render: (v?: number) => <span className="font-mono">{v ?? 1}</span>,
    },
    {
      title: t("runDetail.colTime"),
      dataIndex: "lastTimestamp",
      width: 170,
      render: (v: string) => <span className="text-muted">{fmtDateTime(v)}</span>,
    },
  ];

  return (
    <Card title={t("runDetail.eventsTitle")} styles={{ body: { padding: 0 } }} className="overflow-hidden">
      <Table<sdk.Event>
        rowKey={(e) => `${e.reason}-${e.lastTimestamp}-${e.message}`}
        columns={columns}
        dataSource={eventsQ.data?.items ?? []}
        loading={eventsQ.isLoading}
        pagination={{ pageSize: 20, hideOnSinglePage: true }}
        locale={{
          emptyText: eventsQ.isError ? (
            <Empty description={t("common.loadFailed")} />
          ) : (
            <Empty description={t("runDetail.noEvents")} />
          ),
        }}
      />
    </Card>
  );
}
