import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  Table,
  Button,
  Input,
  Select,
  Space,
  Card,
  Tooltip,
  Divider,
  Drawer,
  Form,
  InputNumber,
  Segmented,
  Empty,
  Spin,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  SearchOutlined,
  DeleteOutlined,
  CaretRightOutlined,
  PoweroffOutlined,
  AppstoreOutlined,
  UnorderedListOutlined,
  CodeOutlined,
  CloseOutlined,
} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useWorkspaces } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";
import { FieldSection } from "@/components/FieldSection";
import { CardRadio } from "@/components/CardRadio";

interface WsRow {
  name: string;
  desc: string;
  phase?: string;
  unit: string;
  image: string;
  owner: string;
  pvc?: string;
}

const isRunning = (phase?: string) =>
  phase === "Running" || phase === "Degraded" || phase === "Starting" || phase === "Creating" || phase === "Pending";
const isStopped = (phase?: string) => !isRunning(phase);

export default function Workspaces() {
  const q = useWorkspaces();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [drawer, setDrawer] = useState(false);
  const [search, setSearch] = useState("");
  const [phase, setPhase] = useState<string>("");
  const [pool, setPool] = useState<string>("");
  const [creator, setCreator] = useState<string>("");

  const start = useApiMutation((name: string) => sdk.startWorkspace({ path: { name } }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.starting"),
  });
  const stop = useApiMutation((name: string) => sdk.stopWorkspace({ path: { name } }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.stopped"),
  });
  const del = useApiMutation(
    (vars: { name: string; deletePvc: boolean }) =>
      sdk.deleteWorkspace({ path: { name: vars.name }, body: { deletePvc: vars.deletePvc } }),
    { invalidate: [["workspaces"]], success: t("workspaces.deleted") },
  );

  const allRows: WsRow[] = useMemo(
    () =>
      q.data?.items?.map((w) => ({
        name: w.name,
        desc: w.description ?? w.displayName ?? "",
        phase: w.phase,
        unit: w.unitName ?? "—",
        image: w.image ?? "—",
        owner: w.owner ?? "—",
        pvc: w.volumes?.find((v) => v.size)?.size,
      })) ?? [],
    [q.data],
  );

  const poolOptions = useMemo(
    () => Array.from(new Set((q.data?.items ?? []).map((w) => w.poolName).filter(Boolean) as string[])),
    [q.data],
  );
  const creatorOptions = useMemo(
    () => Array.from(new Set(allRows.map((r) => r.owner).filter((o) => o && o !== "—"))),
    [allRows],
  );

  const rows = allRows.filter(
    (r) =>
      (!search || r.name.includes(search)) &&
      (!phase || r.phase === phase) &&
      (!creator || r.owner === creator) &&
      (!pool ||
        (q.data?.items ?? []).some((w) => w.name === r.name && w.poolName === pool)),
  );

  const onDelete = (r: WsRow, descKey: "running" | "stopped" | "default") => {
    let deletePvc = r.pvc != null;
    confirm({
      title: t("workspaces.deleteTitle", { name: r.name }),
      desc:
        descKey === "running"
          ? t("workspaces.deleteDescRunning")
          : descKey === "stopped"
            ? t("workspaces.deleteDescStopped")
            : t("workspaces.deleteDescDefault"),
      info:
        r.pvc != null ? (
          <label className="flex items-center gap-2">
            <input type="checkbox" defaultChecked onChange={(e) => (deletePvc = e.target.checked)} />
            {t("workspaces.deletePvc", { size: r.pvc })}
          </label>
        ) : undefined,
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate({ name: r.name, deletePvc }),
    });
  };

  const columns: ColumnsType<WsRow> = [
    {
      title: t("workspaces.colName"),
      dataIndex: "name",
      render: (_, r) => (
        <div className="min-w-0">
          <Link to={`/workspaces/${r.name}`} className="font-mono font-medium">
            {r.name}
          </Link>
          {r.desc && <div className="truncate text-xs text-muted">{r.desc}</div>}
        </div>
      ),
    },
    { title: t("workspaces.colStatus"), dataIndex: "phase", width: 120, render: (v?: string) => <PhaseTag phase={v} /> },
    { title: t("workspaces.colUnit"), dataIndex: "unit", width: 160, render: (v: string) => <span className="font-mono text-sm">{v}</span> },
    { title: t("workspaces.colImage"), dataIndex: "image", width: 200, render: (v: string) => <span className="font-mono text-sm text-muted">{v}</span> },
    { title: t("workspaces.colCreator"), dataIndex: "owner", width: 140 },
    {
      title: t("common.actions"),
      key: "actions",
      width: 160,
      align: "right",
      render: (_, r) => (
        <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
          <Link to={`/workspaces/${r.name}`}>
            <Button type="link" size="small" className="!px-1">
              {t("common.detail")}
            </Button>
          </Link>
          {isStopped(r.phase) ? (
            <Button type="link" size="small" className="!px-1" onClick={() => start.mutate(r.name)}>
              {t("workspaces.start")}
            </Button>
          ) : (
            <Button type="link" size="small" className="!px-1" onClick={() => stop.mutate(r.name)}>
              {t("workspaces.stop")}
            </Button>
          )}
          <Button
            type="link"
            size="small"
            danger
            className="!px-1"
            onClick={() => onDelete(r, isStopped(r.phase) ? "stopped" : "running")}
          >
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]}
      title={t("workspaces.title")}
      subtitle={t("workspaces.subtitle")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawer(true)}>
          {t("workspaces.newWorkspace")}
        </Button>
      }
    >
      <Card styles={{ body: { padding: 16 } }} className="mb-4">
        <div className="flex flex-wrap items-center gap-3">
          <Input
            allowClear
            prefix={<SearchOutlined className="text-muted" />}
            placeholder={t("workspaces.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
          <Select
            value={phase || undefined}
            onChange={(v) => setPhase(v ?? "")}
            placeholder={t("workspaces.statusAll")}
            allowClear
            className="min-w-36"
            options={["Running", "Starting", "Stopped"].map((p) => ({ label: t(`phase.${p}`), value: p }))}
          />
          <Select
            value={pool || undefined}
            onChange={(v) => setPool(v ?? "")}
            placeholder={t("workspaces.poolAll")}
            allowClear
            className="min-w-36"
            options={poolOptions.map((o) => ({ label: o, value: o }))}
          />
          <Select
            value={creator || undefined}
            onChange={(v) => setCreator(v ?? "")}
            placeholder={t("workspaces.creatorAll")}
            allowClear
            className="min-w-36"
            options={creatorOptions.map((o) => ({ label: o, value: o }))}
          />
          <Button
            onClick={() => {
              setSearch("");
              setPhase("");
              setPool("");
              setCreator("");
            }}
          >
            {t("common.reset")}
          </Button>
          <div className="grow" />
          <Segmented
            value={view}
            onChange={(v) => setView(v as "cards" | "list")}
            options={[
              { value: "cards", icon: <AppstoreOutlined />, label: t("workspaces.viewCards") },
              { value: "list", icon: <UnorderedListOutlined />, label: t("workspaces.viewList") },
            ]}
          />
        </div>
      </Card>

      {view === "cards" ? (
        <CardsView
          q={q}
          rows={rows}
          onStart={(name) => start.mutate(name)}
          onStop={(name) => stop.mutate(name)}
          onDelete={onDelete}
        />
      ) : (
        <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
          <Table<WsRow>
            rowKey="name"
            columns={columns}
            dataSource={rows}
            loading={q.isLoading}
            pagination={{ pageSize: 20, showTotal: (n) => t("workspaces.total", { count: n }), hideOnSinglePage: false }}
            locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
          />
        </Card>
      )}

      {drawer && <WsDrawer onClose={() => setDrawer(false)} />}
    </PageContainer>
  );
}

function CardsView({
  q,
  rows,
  onStart,
  onStop,
  onDelete,
}: {
  q: ReturnType<typeof useWorkspaces>;
  rows: WsRow[];
  onStart: (name: string) => void;
  onStop: (name: string) => void;
  onDelete: (r: WsRow, descKey: "running" | "stopped" | "default") => void;
}) {
  const { t } = useTranslation();
  if (q.isLoading) {
    return (
      <div className="grid place-items-center py-20">
        <Spin />
      </div>
    );
  }
  if (q.isError) {
    return (
      <Card>
        <Empty description={t("common.loadFailed")} />
      </Card>
    );
  }
  if (rows.length === 0) {
    return (
      <Card>
        <Empty description={t("common.noData")} />
      </Card>
    );
  }
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      {rows.map((r) => (
        <WsCard key={r.name} row={r} onStart={onStart} onStop={onStop} onDelete={onDelete} />
      ))}
    </div>
  );
}

function WsCard({
  row,
  onStart,
  onStop,
  onDelete,
}: {
  row: WsRow;
  onStart: (name: string) => void;
  onStop: (name: string) => void;
  onDelete: (r: WsRow, descKey: "running" | "stopped" | "default") => void;
}) {
  const { t } = useTranslation();
  const running = row.phase === "Running" || row.phase === "Degraded";
  const stopped = isStopped(row.phase);

  return (
    <Card hoverable styles={{ body: { padding: 16 } }} className="h-full">
      <div className="mb-3 flex items-start gap-3">
        <div className="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-surface text-accent">
          <CodeOutlined />
        </div>
        <div className="min-w-0 flex-1">
          <Link to={`/workspaces/${row.name}`} className="font-mono text-sm font-semibold text-fg">
            {row.name}
          </Link>
          {row.desc && <div className="truncate text-xs text-muted">{row.desc}</div>}
        </div>
        <PhaseTag phase={row.phase} />
      </div>
      <div className="flex items-center gap-2 text-xs text-fg-2">
        <span className="rounded-full border border-border-soft bg-surface px-2 py-0.5 font-mono">{row.unit}</span>
        <span className="ml-auto inline-flex items-center gap-1.5">
          <span className="grid h-5 w-5 place-items-center rounded-full bg-surface-warm text-[10px] font-semibold text-accent">
            {row.owner.slice(0, 1)}
          </span>
          {row.owner}
        </span>
      </div>
      <div className="my-3 border-t border-border-soft" />
      <div className="flex items-center gap-1">
        {stopped ? (
          <>
            <Tooltip title={t("workspaces.start")}>
              <Button type="text" size="small" icon={<CaretRightOutlined />} className="text-accent" onClick={() => onStart(row.name)} />
            </Tooltip>
            <div className="grow" />
            <Tooltip title={t("workspaces.remove")}>
              <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => onDelete(row, "stopped")} />
            </Tooltip>
          </>
        ) : (
          <>
            <Tooltip title={running ? t("workspaces.openJupyter") : t("workspaces.availableAfterStart")}>
              <Link to={`/workspaces/${row.name}`}>
                <Button type="text" size="small" disabled={!running} icon={<CodeOutlined />} />
              </Link>
            </Tooltip>
            <div className="grow" />
            <Tooltip title={t("workspaces.stop")}>
              <Button type="text" size="small" icon={<PoweroffOutlined />} onClick={() => onStop(row.name)} />
            </Tooltip>
            {running && (
              <Tooltip title={t("workspaces.remove")}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => onDelete(row, "running")} />
              </Tooltip>
            )}
          </>
        )}
      </div>
    </Card>
  );
}

// ── Create drawer ─────────────────────────────────────────────────────────────
const WS_IMAGES: { value: string; title: string; desc: string }[] = [
  { value: "jupyter-ds:2024.3", title: "jupyter-ds:2024.3", desc: "Jupyter 开发环境 · 公共" },
  { value: "pytorch:2.3-cu121", title: "pytorch:2.3-cu121", desc: "PyTorch 训练镜像" },
  { value: "vscode-server:1.90", title: "vscode-server:1.90", desc: "VS Code 开发环境 · 公共" },
];
const WS_UNITS: { value: string; pool: string; title: string; desc: string }[] = [
  { value: "cpu-medium", pool: "cpu-medium", title: "cpu-medium", desc: "8 vCPU · 32 GiB" },
  { value: "cpu-large", pool: "cpu-medium", title: "cpu-large", desc: "16 vCPU · 64 GiB" },
  { value: "a100-1x", pool: "gpu-a100", title: "a100-1x", desc: "1×A100 · 8 vCPU · 64 GiB" },
];

interface WsFormValues {
  name: string;
  description?: string;
  image: string;
  unitName: string;
  containerPort: number;
  env?: string;
  volSize?: string;
  mountPath?: string;
}

function parseEnv(text: string): sdk.EnvVar[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)
    .map((l) => {
      const eq = l.indexOf("=");
      const name = (eq === -1 ? l : l.slice(0, eq)).trim();
      return { name, value: eq === -1 ? "" : l.slice(eq + 1).trim() };
    })
    .filter((e) => e.name);
}

function WsDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const [form] = Form.useForm<WsFormValues>();

  const create = useApiMutation((body: sdk.WorkspaceCreateRequest) => sdk.createWorkspace({ body }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.created"),
  });

  const onFinish = (v: WsFormValues) => {
    const unit = WS_UNITS.find((u) => u.value === v.unitName) ?? WS_UNITS[0];
    const envVars = parseEnv(v.env || "");
    const mountPath = v.mountPath?.trim();
    const body: sdk.WorkspaceCreateRequest = {
      name: v.name.trim(),
      image: v.image,
      poolName: unit.pool,
      unitName: unit.value,
      description: v.description?.trim() || undefined,
      containerPort: v.containerPort && v.containerPort > 0 ? v.containerPort : undefined,
      env: envVars.length ? envVars : undefined,
      volumes: mountPath ? [{ mountPath, size: v.volSize?.trim() || undefined }] : undefined,
    };
    create.mutate(body, { onSuccess: onClose });
  };

  return (
    <Drawer
      open
      width={640}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("workspaces.drawerNew")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">{t("workspaces.drawerNewSub")}</div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={create.isPending} onClick={() => form.submit()}>
            {t("workspaces.createWorkspace")}
          </Button>
        </div>
      }
    >
      <Form<WsFormValues>
        form={form}
        layout="vertical"
        size="large"
        onFinish={onFinish}
        initialValues={{
          name: "",
          image: WS_IMAGES[0].value,
          unitName: WS_UNITS[0].value,
          containerPort: 8888,
          env: "",
          volSize: "50Gi",
          mountPath: "/workspace",
        }}
      >
        <FieldSection n={1} title={t("workspaces.fsBasic")} />
        <Form.Item name="name" label={t("workspaces.fName")} rules={[{ required: true }]} extra={t("workspaces.fNameHelp")}>
          <Input className="font-mono" placeholder={t("workspaces.fNamePlaceholder")} />
        </Form.Item>
        <Form.Item name="description" label={t("workspaces.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("workspaces.fDescPlaceholder")} />
        </Form.Item>

        <FieldSection n={2} title={t("workspaces.fsImage")} />
        <Form.Item name="image" label={t("workspaces.fImage")} rules={[{ required: true }]}>
          <CardRadio options={WS_IMAGES} />
        </Form.Item>

        <FieldSection n={3} title={t("workspaces.fsResource")} />
        <Form.Item name="unitName" label={t("workspaces.fUnit")} rules={[{ required: true }]}>
          <CardRadio options={WS_UNITS} />
        </Form.Item>
        <Form.Item name="containerPort" label={t("workspaces.fContainerPort")} extra={t("workspaces.fPortHelp")}>
          <InputNumber min={1} max={65535} className="!w-40 font-mono" />
        </Form.Item>

        <FieldSection n={4} title={t("workspaces.fsEnv")} />
        <Form.Item name="env" label={t("workspaces.fEnv")} extra={t("workspaces.fEnvHelp")}>
          <Input.TextArea rows={2} className="font-mono" placeholder={"HF_HOME=/data/hf\nCUDA_VISIBLE_DEVICES=0"} />
        </Form.Item>

        <FieldSection n={5} title={t("workspaces.fsVolume")} />
        <div className="flex gap-3">
          <Form.Item name="volSize" label={t("workspaces.fVolSize")} className="w-40">
            <Input className="font-mono" placeholder="50Gi" />
          </Form.Item>
          <Form.Item name="mountPath" label={t("workspaces.fMountPath")} className="flex-1" extra={t("workspaces.fVolSizeHelp")}>
            <Input className="font-mono" placeholder="/workspace" />
          </Form.Item>
        </div>
      </Form>
    </Drawer>
  );
}
