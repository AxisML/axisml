import { useEffect, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import {
  Card,
  Tabs,
  Descriptions,
  Button,
  Space,
  Tag,
  Timeline,
  Empty,
  Spin,
  Result,
  Select,
  Switch,
  Drawer,
  Form,
  Input,
  Alert,
  Tooltip,
} from "antd";
import {
  ArrowLeftOutlined,
  CaretRightOutlined,
  PoweroffOutlined,
  DeleteOutlined,
  CopyOutlined,
  CodeOutlined,
  InfoCircleOutlined,
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

const isRunning = (phase?: string) =>
  phase === "Running" || phase === "Degraded" || phase === "Starting" || phase === "Creating" || phase === "Pending";

function useWorkspace(name: string) {
  const { tenant } = useApp();
  return useQuery<sdk.Workspace>({
    queryKey: ["workspaces", tenant, name],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getWorkspace({ path: { name } });
      if (error) throw error;
      return data as sdk.Workspace;
    },
  });
}

export default function WorkspaceDetail() {
  const { name = "" } = useParams();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const q = useWorkspace(name);
  const [edit, setEdit] = useState(false);

  const start = useApiMutation((n: string) => sdk.startWorkspace({ path: { name: n } }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.starting"),
  });
  const stop = useApiMutation((n: string) => sdk.stopWorkspace({ path: { name: n } }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.stopped"),
  });
  const del = useApiMutation(
    (vars: { name: string; deletePvc: boolean }) =>
      sdk.deleteWorkspace({ path: { name: vars.name }, body: { deletePvc: vars.deletePvc } }),
    { invalidate: [["workspaces"]], success: t("workspaces.deleted") },
  );

  const back = (
    <Link to="/workspaces" className="mb-3 inline-flex items-center gap-1.5 text-sm text-fg-2 hover:text-accent">
      <ArrowLeftOutlined />
      {t("workspaces.backToList")}
    </Link>
  );

  if (q.isLoading) {
    return (
      <PageContainer breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]} title={name}>
        {back}
        <div className="grid place-items-center py-20">
          <Spin />
        </div>
      </PageContainer>
    );
  }
  if (q.isError || !q.data) {
    return (
      <PageContainer breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]} title={name}>
        {back}
        <Card>
          <Result status="404" title={t("workspaces.notFound")} />
        </Card>
      </PageContainer>
    );
  }

  const w = q.data;
  const running = isRunning(w.phase);
  const pvc = w.volumes?.find((v) => v.size)?.size;

  const onDelete = () => {
    let deletePvc = pvc != null;
    confirm({
      title: t("workspaces.deleteTitle", { name: w.name }),
      desc: running ? t("workspaces.deleteDescRunning") : t("workspaces.deleteDescStopped"),
      info:
        pvc != null ? (
          <label className="flex items-center gap-2">
            <input type="checkbox" defaultChecked onChange={(e) => (deletePvc = e.target.checked)} />
            {t("workspaces.deletePvc", { size: pvc })}
          </label>
        ) : undefined,
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate({ name: w.name, deletePvc }),
    });
  };

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]}
      title={
        <span className="inline-flex items-center gap-3">
          <span className="font-mono">{w.name}</span>
          <PhaseTag phase={w.phase} />
        </span>
      }
      subtitle={`${w.description ?? w.displayName ?? ""}${w.owner ? ` · ${t("common.creator")} ${w.owner}` : ""}`}
      extra={
        <Space>
          {running ? (
            <Button icon={<PoweroffOutlined />} onClick={() => stop.mutate(w.name)} loading={stop.isPending}>
              {t("phase.Stopped")}
            </Button>
          ) : (
            <Button type="primary" icon={<CaretRightOutlined />} onClick={() => start.mutate(w.name)} loading={start.isPending}>
              {t("workspaces.start")}
            </Button>
          )}
          <Button danger icon={<DeleteOutlined />} onClick={onDelete}>
            {t("common.delete")}
          </Button>
        </Space>
      }
    >
      <div className="-mt-2">{back}</div>
      <Tabs
        items={[
          { key: "info", label: t("workspaces.tabInfo"), children: <InfoPane w={w} running={running} onEdit={() => setEdit(true)} /> },
          { key: "log", label: t("workspaces.tabLog"), children: <LogPane name={w.name} /> },
          { key: "ev", label: t("workspaces.tabEvents"), children: <EventsPane name={w.name} /> },
        ]}
      />
      {edit && <EditDrawer w={w} onClose={() => setEdit(false)} />}
    </PageContainer>
  );
}

function Chip({ children }: { children: ReactNode }) {
  return <span className="rounded-md border border-border-soft bg-surface px-2 py-0.5 font-mono text-sm text-fg-2">{children}</span>;
}

function InfoPane({ w, running, onEdit }: { w: sdk.Workspace; running: boolean; onEdit: () => void }) {
  const { t } = useTranslation();
  const { toast } = useUI();
  const accessUrl = w.endpoint?.accessUrl;
  const vol = w.volumes?.find((v) => v.size) ?? w.volumes?.[0];

  return (
    <Card
      title={t("workspaces.configTitle")}
      extra={
        running ? (
          <Button size="small" onClick={onEdit}>
            {t("common.edit")}
          </Button>
        ) : undefined
      }
    >
      <Descriptions column={1} size="middle" colon={false} labelStyle={{ width: 120 }}>
        <Descriptions.Item label={t("common.name")}>
          <Chip>{w.name}</Chip>
        </Descriptions.Item>
        {(w.description || w.displayName) && (
          <Descriptions.Item label={t("common.description")}>{w.description ?? w.displayName}</Descriptions.Item>
        )}
        <Descriptions.Item label={t("workspaces.fPool")}>
          {w.poolName ? <Chip>{w.poolName}</Chip> : <span className="text-muted">—</span>}
        </Descriptions.Item>
        <Descriptions.Item label={t("workspaces.fUnit")}>
          {w.unitName ? <Chip>{w.unitName}</Chip> : <span className="text-muted">—</span>}
        </Descriptions.Item>
        <Descriptions.Item label={t("workspaces.fImage")}>
          <Chip>{w.image}</Chip>
        </Descriptions.Item>
        <Descriptions.Item label={t("workspaces.fPort")}>
          <span className="font-mono">{w.containerPort}</span>
        </Descriptions.Item>
        {accessUrl && (
          <Descriptions.Item label={t("workspaces.fAccessUrl")}>
            <span className="inline-flex items-center gap-1.5">
              <Chip>{accessUrl}</Chip>
              <Tooltip title={t("workspaces.copyAddr")}>
                <Button
                  type="text"
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => {
                    void navigator.clipboard?.writeText(accessUrl);
                    toast(t("workspaces.addrCopied"));
                  }}
                />
              </Tooltip>
            </span>
          </Descriptions.Item>
        )}
        {w.endpoint?.internalDns && (
          <Descriptions.Item label={t("workspaces.fInternalDns")}>
            <Chip>{w.endpoint.internalDns}</Chip>
          </Descriptions.Item>
        )}
        <Descriptions.Item label={t("workspaces.fVolume")}>
          {vol ? (
            <span className="inline-flex items-center gap-2">
              <Chip>{vol.name ?? vol.mountPath}</Chip>
              <span className="text-sm text-muted">
                {[vol.size, vol.storageClass].filter(Boolean).join(" · ") || "—"}
              </span>
            </span>
          ) : (
            <span className="text-muted">{t("workspaces.noVolume")}</span>
          )}
        </Descriptions.Item>
        {vol?.mountPath && (
          <Descriptions.Item label={t("workspaces.fMountPath")}>
            <Chip>{vol.mountPath}</Chip>
          </Descriptions.Item>
        )}
        <Descriptions.Item label={t("workspaces.fEnv")}>
          {w.env?.length ? (
            <div className="flex flex-wrap gap-1.5">
              {w.env.map((e) => (
                <Chip key={e.name}>
                  {e.name}={e.value}
                </Chip>
              ))}
            </div>
          ) : (
            <span className="text-muted">{t("workspaces.noEnv")}</span>
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t("common.creator")}>
          {w.owner} · <span className="font-mono text-muted">{dayjs(w.createdAt).format("YYYY-MM-DD HH:mm")}</span>
        </Descriptions.Item>
      </Descriptions>
    </Card>
  );
}

function LogPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [follow, setFollow] = useState(true);
  const [pod, setPod] = useState<string>("");
  const podsQ = useQuery<sdk.PodList>({
    queryKey: ["workspaces", tenant, name, "pods"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.listWorkspacePods({ path: { name } });
      if (error) throw error;
      return data as sdk.PodList;
    },
  });

  const pods = podsQ.data?.items ?? [];
  useEffect(() => {
    if (!pod && pods.length) setPod(pods[0].name);
  }, [pod, pods]);

  const logsQ = useQuery({
    queryKey: ["workspaces", tenant, name, "logs", pod],
    enabled: tenant !== "" && name !== "" && pod !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getWorkspacePodLogs({ path: { name, pod } });
      if (error) throw error;
      return data as unknown as string;
    },
  });

  return (
    <Card>
      <div className="mb-3 flex items-center gap-3">
        <Select
          className="min-w-56"
          value={pod || undefined}
          onChange={setPod}
          placeholder={t("workspaces.noPods")}
          prefix={<Tag className="!m-0">{t("workspaces.podLabel")}</Tag>}
          options={pods.map((p) => ({ label: p.name, value: p.name }))}
          notFoundContent={podsQ.isError ? t("common.loadFailed") : t("workspaces.noPods")}
        />
        <div className="grow" />
        <span className="flex items-center gap-2 text-sm text-fg-2">
          {t("workspaces.follow")}
          <Switch checked={follow} onChange={setFollow} size="small" />
        </span>
      </div>
      {!pods.length ? (
        <Alert type="info" showIcon icon={<InfoCircleOutlined />} message={t("workspaces.logHint")} />
      ) : (
        <LogViewer text={logsQ.data} empty={t("workspaces.logHint")} />
      )}
    </Card>
  );
}

function EventsPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const q = useQuery<sdk.EventList>({
    queryKey: ["workspaces", tenant, name, "events"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getWorkspaceEvents({ path: { name } });
      if (error) throw error;
      return data as sdk.EventList;
    },
  });

  if (q.isLoading) {
    return (
      <Card>
        <div className="grid place-items-center py-10">
          <Spin />
        </div>
      </Card>
    );
  }
  const items = q.data?.items ?? [];
  if (q.isError || items.length === 0) {
    return (
      <Card>
        <Empty description={q.isError ? t("common.loadFailed") : t("workspaces.noEvents")} />
      </Card>
    );
  }
  return (
    <Card>
      <Timeline
        items={items.map((e) => ({
          color: e.type === "Warning" ? "orange" : "blue",
          children: (
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium text-fg">{e.reason}</span>
                <Tag color={e.type === "Warning" ? "warning" : "default"} className="!m-0">
                  {e.type}
                </Tag>
                <span className="font-mono text-xs text-muted">{dayjs(e.lastTimestamp).format("YYYY-MM-DD HH:mm:ss")}</span>
              </div>
              <div className="mt-0.5 text-sm text-fg-2">{e.message}</div>
            </div>
          ),
        }))}
      />
    </Card>
  );
}

interface EditFormValues {
  displayName?: string;
  description?: string;
}

function EditDrawer({ w, onClose }: { w: sdk.Workspace; onClose: () => void }) {
  const { t } = useTranslation();
  const [form] = Form.useForm<EditFormValues>();

  const update = useApiMutation(
    (vars: { name: string; body: sdk.WorkspacePatchRequest }) =>
      sdk.updateWorkspace({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["workspaces"]], success: t("workspaces.updated") },
  );

  const onFinish = (v: EditFormValues) => {
    update.mutate(
      {
        name: w.name,
        body: { displayName: v.displayName?.trim() || undefined, description: v.description?.trim() || undefined },
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Drawer
      open
      width={560}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("workspaces.drawerEdit")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            <span className="font-mono">{w.name}</span>
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={update.isPending} onClick={() => form.submit()}>
            {t("workspaces.saveChanges")}
          </Button>
        </div>
      }
    >
      <Alert
        type="info"
        showIcon
        icon={<InfoCircleOutlined />}
        className="mb-4"
        message={t("workspaces.editNotice")}
      />
      <Form<EditFormValues>
        form={form}
        layout="vertical"
        size="large"
        onFinish={onFinish}
        initialValues={{ displayName: w.displayName ?? w.name, description: w.description }}
      >
        <Form.Item name="displayName" label={t("workspaces.fName")} extra={t("workspaces.fNameHelp")}>
          <Input />
        </Form.Item>
        <Form.Item name="description" label={t("workspaces.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("workspaces.fDescPlaceholder")} />
        </Form.Item>
        <div className="rounded-lg border border-border-soft bg-surface-warm p-3 text-sm text-muted">
          <CodeOutlined className="mr-2 text-accent" />
          {t("workspaces.fImage")}: <span className="font-mono text-fg-2">{w.image}</span>
          {w.unitName && (
            <>
              {" · "}
              {t("workspaces.fUnit")}: <span className="font-mono text-fg-2">{w.unitName}</span>
            </>
          )}
        </div>
      </Form>
    </Drawer>
  );
}
