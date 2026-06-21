import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  Button,
  Card,
  Tabs,
  Table,
  Select,
  Descriptions,
  Tag,
  Empty,
  Spin,
  Result,
  Space,
  Tooltip,
  Drawer,
  Form,
  InputNumber,
  Input,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  ArrowLeftOutlined,
  ExpandOutlined,
  EditOutlined,
  PauseOutlined,
  CaretRightOutlined,
  DeleteOutlined,
  CopyOutlined,
  ReloadOutlined,
  CloseOutlined,
} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useApp } from "@/app/store";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";
import { LogViewer } from "@/components/LogViewer";
import { fmtDateTime } from "./JobDetail";

const INVALIDATE = [["mlservices"]];
const RUNNING_PHASES = new Set(["Ready", "Degraded", "Creating", "Pending"]);

export default function ServiceDetail() {
  const { name = "" } = useParams<{ name: string }>();
  const { tenant } = useApp();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<"edit" | "scale" | null>(null);

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
    <Link to="/services">
      <Button type="text" size="small" icon={<ArrowLeftOutlined />}>
        {t("services.backToList")}
      </Button>
    </Link>
  );

  if (q.isLoading) {
    return (
      <PageContainer breadcrumb={breadcrumb} title={name}>
        {back}
        <div className="grid place-items-center py-24">
          <Spin />
        </div>
      </PageContainer>
    );
  }

  if (q.isError || !q.data) {
    return (
      <PageContainer breadcrumb={breadcrumb} title={name}>
        {back}
        <Result status="error" title={t("services.notFound")} subTitle={t("services.loadFailedDesc")} />
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
        <Space>
          <Button icon={<EditOutlined />} onClick={() => setDrawer("edit")}>
            {t("common.edit")}
          </Button>
          <Button icon={<ExpandOutlined />} onClick={() => setDrawer("scale")}>
            {t("services.scale")}
          </Button>
          {running ? (
            <Button icon={<PauseOutlined />} loading={stop.isPending} onClick={() => stop.mutate(undefined)}>
              {t("services.stop")}
            </Button>
          ) : (
            <Button icon={<CaretRightOutlined />} loading={start.isPending} onClick={() => start.mutate(undefined)}>
              {t("services.start")}
            </Button>
          )}
          <Button danger icon={<DeleteOutlined />} onClick={onDelete}>
            {t("common.delete")}
          </Button>
        </Space>
      }
    >
      <div className="mb-4">{back}</div>
      <Tabs
        items={[
          { key: "info", label: t("services.tabInfo"), children: <InfoPane svc={svc} /> },
          { key: "mon", label: t("services.tabMonitor"), children: <MonitorPane name={svc.name} /> },
          { key: "pods", label: t("services.tabPods"), children: <PodsPane name={svc.name} /> },
          { key: "log", label: t("services.tabLog"), children: <LogPane name={svc.name} /> },
          { key: "ev", label: t("services.tabEvents"), children: <EventsPane name={svc.name} /> },
        ]}
      />

      {drawer === "edit" && <EditSvcDrawer svc={svc} onClose={() => setDrawer(null)} />}
      {drawer === "scale" && <ScaleDrawer svc={svc} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Overview ──────────────────────────────────────────────────────────────────
function InfoPane({ svc }: { svc: sdk.MlService }) {
  const { t } = useTranslation();
  const { toast } = useUI();

  const dash = <span className="text-muted">—</span>;
  const chip = (v?: string) => (v ? <Tag className="!m-0 font-mono">{v}</Tag> : dash);

  return (
    <Card title={t("services.configInfo")}>
      <Descriptions column={1} bordered size="middle">
        <Descriptions.Item label={t("services.dName")}>{chip(svc.name)}</Descriptions.Item>
        <Descriptions.Item label={t("services.dDesc")}>{svc.description ?? dash}</Descriptions.Item>
        <Descriptions.Item label={t("services.dModelVersion")}>
          {svc.modelName ? chip(svc.modelVersion ? `${svc.modelName}@${svc.modelVersion}` : svc.modelName) : dash}
        </Descriptions.Item>
        <Descriptions.Item label={t("services.dImage")}>{chip(svc.image)}</Descriptions.Item>
        <Descriptions.Item label={t("services.dPool")}>{chip(svc.poolName)}</Descriptions.Item>
        <Descriptions.Item label={t("services.dUnit")}>{chip(svc.unitName)}</Descriptions.Item>
        <Descriptions.Item label={t("services.dReplicas")}>
          <span className="font-mono">
            {t("services.replicasReady", { ready: svc.readyReplicas ?? 0, total: svc.replicas ?? 0 })}
          </span>
        </Descriptions.Item>
        <Descriptions.Item label={t("services.dPorts")}>
          {svc.ports && svc.ports.length > 0 ? (
            <Space size={[6, 6]} wrap>
              {svc.ports.map((p) => (
                <Tag key={`${p.name}:${p.port}`} className="!m-0 font-mono">
                  {p.name} : {p.port}
                </Tag>
              ))}
            </Space>
          ) : (
            dash
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t("services.dAccess")}>
          {svc.accessUrl ? (
            <span className="flex items-center gap-2">
              <Tag className="!m-0 font-mono">{svc.accessUrl}</Tag>
              <Tooltip title={t("services.copyAccess")}>
                <Button
                  type="text"
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => {
                    void navigator.clipboard?.writeText(svc.accessUrl ?? "");
                    toast(t("services.accessCopied"));
                  }}
                />
              </Tooltip>
            </span>
          ) : (
            dash
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t("services.dCreator")}>
          {svc.owner ? <span className="font-mono">{svc.owner}</span> : dash}
        </Descriptions.Item>
        <Descriptions.Item label={t("services.dCreatedAt")}>
          {svc.createdAt ? <span className="font-mono text-muted">{dayjs(svc.createdAt).format("YYYY-MM-DD HH:mm:ss")}</span> : dash}
        </Descriptions.Item>
      </Descriptions>
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
      const { data, error } = await sdk.getMlServiceMetrics({ path: { name }, query: { metric: "request_rate", range: "24h" } });
      if (error) throw error;
      return data;
    },
  });
  const series = (q.data?.series ?? []).map((p) => p.value ?? 0);
  return (
    <Card title={t("services.tabMonitor")}>
      {q.isLoading ? (
        <div className="grid place-items-center py-16"><Spin /></div>
      ) : series.length ? (
        <MiniTrend values={series} />
      ) : (
        <div className="py-12"><Empty description={t("services.monitorEmpty")} /></div>
      )}
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
          <stop offset="0" stopColor="var(--accent)" stopOpacity="0.16" />
          <stop offset="1" stopColor="var(--accent)" stopOpacity="0" />
        </linearGradient>
      </defs>
      {[50, 100, 150].map((gy) => (
        <line key={gy} x1="0" y1={gy} x2={W} y2={gy} stroke="var(--border-soft)" strokeWidth="1" />
      ))}
      <path d={`M0 ${H} L${pts.join(" L")} L${W} ${H} Z`} fill="url(#svc-trend)" />
      <path d={`M${pts.join(" L")}`} fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
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
  const columns: ColumnsType<sdk.Pod> = [
    { title: t("services.colPod"), dataIndex: "name", render: (v: string) => <span className="font-mono">{v}</span> },
    { title: t("services.colPhase"), dataIndex: "phase", width: 120, render: (p: string) => <PhaseTag phase={p} /> },
    { title: t("services.colNode"), dataIndex: "nodeName", width: 160, render: (v?: string) => <span className="font-mono">{v || "—"}</span> },
    { title: t("services.colRestarts"), dataIndex: "restartCount", width: 90, align: "right", render: (v?: number) => <span className="font-mono">{v ?? 0}</span> },
    { title: t("services.colStarted"), dataIndex: "startedAt", width: 170, render: (v?: string) => <span className="text-muted">{fmtDateTime(v)}</span> },
  ];
  return (
    <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
      <Table<sdk.Pod>
        rowKey="name"
        columns={columns}
        dataSource={q.data?.items ?? []}
        loading={q.isLoading}
        pagination={{ pageSize: 20, hideOnSinglePage: true }}
        locale={{ emptyText: <Empty description={q.isError ? t("common.loadFailed") : t("services.podsEmpty")} /> }}
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
    <Card
      title={t("services.tabLog")}
      extra={
        <Space>
          <Select size="small" value={pod || undefined} onChange={setPod} className="min-w-52" options={pods.map((p) => ({ label: p.name, value: p.name }))} />
          <Button size="small" icon={<ReloadOutlined />} onClick={() => logsQ.refetch()} />
        </Space>
      }
    >
      {podsQ.isLoading || logsQ.isLoading ? (
        <div className="grid place-items-center py-16"><Spin /></div>
      ) : !pods.length ? (
        <div className="py-12"><Empty description={t("services.logEmpty")} /></div>
      ) : (
        <LogViewer text={logsQ.data} empty={t("services.logEmpty")} />
      )}
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
  const columns: ColumnsType<sdk.Event> = [
    { title: t("services.colReason"), dataIndex: "reason", width: 180, render: (v: string) => <span className="font-mono">{v}</span> },
    {
      title: t("services.colType"),
      dataIndex: "type",
      width: 110,
      render: (v: string) => (
        <Tag color={v === "Warning" ? "warning" : "default"} className="!m-0">
          {v}
        </Tag>
      ),
    },
    { title: t("services.colMessage"), dataIndex: "message" },
    { title: t("services.colTime"), dataIndex: "lastTimestamp", width: 170, render: (v?: string) => <span className="text-muted">{fmtDateTime(v)}</span> },
  ];
  return (
    <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
      <Table<sdk.Event>
        rowKey={(e, i) => `${e.reason}-${i}`}
        columns={columns}
        dataSource={q.data?.items ?? []}
        loading={q.isLoading}
        pagination={{ pageSize: 20, hideOnSinglePage: true }}
        locale={{ emptyText: <Empty description={q.isError ? t("common.loadFailed") : t("services.eventsEmpty")} /> }}
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
    <Drawer
      open
      width={640}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("services.drawerEdit")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            <span className="font-mono">{svc.name}</span>
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={update.isPending} onClick={submit}>
            {t("common.save")}
          </Button>
        </div>
      }
    >
      <Form layout="vertical" size="large">
        <p className="mb-4 text-sm text-muted">{t("services.editNote")}</p>
        <Form.Item label={t("services.fDisplayName")}>
          <Input
            placeholder={t("services.fDisplayNamePlaceholder")}
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </Form.Item>
        <Form.Item label={t("services.fDesc")}>
          <Input.TextArea
            rows={2}
            placeholder={t("services.fDescPlaceholder")}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </Form.Item>
      </Form>
    </Drawer>
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
    <Drawer
      open
      width={420}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("services.drawerScale")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            <span className="font-mono">{svc.name}</span>
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={scale.isPending} disabled={!valid} onClick={submit}>
            {t("common.save")}
          </Button>
        </div>
      }
    >
      <p className="mb-5 text-sm text-muted">{t("services.scaleNote")}</p>
      <Form layout="vertical" size="large">
        <Form.Item
          label={t("services.fTargetReplicas")}
          extra={t("services.scaleHint", {
            ready: `${svc.readyReplicas ?? 0} / ${svc.replicas ?? 0}`,
            unit,
          })}
        >
          <InputNumber min={0} value={replicas} onChange={(v) => setReplicas(v ?? 0)} className="!w-40" />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
