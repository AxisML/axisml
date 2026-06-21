import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  Button,
  Card,
  Tabs,
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
import {
  ArrowLeftOutlined,
  ExpandOutlined,
  EditOutlined,
  PauseOutlined,
  CaretRightOutlined,
  DeleteOutlined,
  CopyOutlined,
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
          { key: "mon", label: t("services.tabMonitor"), children: <EmptyPane msg={t("services.monitorEmpty")} /> },
          { key: "pods", label: t("services.tabPods"), children: <EmptyPane msg={t("services.podsEmpty")} /> },
          { key: "log", label: t("services.tabLog"), children: <EmptyPane msg={t("services.logEmpty")} /> },
          { key: "ev", label: t("services.tabEvents"), children: <EmptyPane msg={t("services.eventsEmpty")} /> },
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

// Monitoring / Pods / Logs / Events have no backend feed yet → honest empty pane.
function EmptyPane({ msg }: { msg: string }) {
  return (
    <Card>
      <div className="py-12">
        <Empty description={msg} />
      </div>
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
