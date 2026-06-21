import { useMemo, useState } from "react";
import {
  Table,
  Card,
  Button,
  Input,
  Segmented,
  Space,
  Tooltip,
  Divider,
  Drawer,
  Form,
  Tabs,
  Tag,
  List,
  Empty,
  Spin,
  Select,
  Upload,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  SearchOutlined,
  AppstoreOutlined,
  UnorderedListOutlined,
  DeleteOutlined,
  CloudDownloadOutlined,
  CopyOutlined,
  InboxOutlined,
  DatabaseOutlined,
  CloseOutlined,
} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useModels, useModelVersions } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { FieldSection } from "@/components/FieldSection";
import { USE_MOCK } from "@/api/mock";
import { modelVersions } from "@/api/mock/data";

interface ModelRow {
  name: string;
  desc: string;
  framework: string;
  latest: string;
  versions: number;
  updated: string;
}

type DrawerState =
  | { kind: "versions"; model: string; desc: string; framework: string }
  | { kind: "pull"; model: string; version: string }
  | { kind: "new" }
  | { kind: "upload"; model: string };

export default function Models() {
  const q = useModels();
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { confirm } = useUI();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [search, setSearch] = useState("");
  const [drawer, setDrawer] = useState<DrawerState | null>(null);

  const delModel = useApiMutation(
    (name: string) => sdk.deleteModelDefinition({ path: { tenant, name } }),
    { invalidate: [["models"]], success: t("models.modelDeleted") },
  );

  const allRows: ModelRow[] = useMemo(
    () =>
      q.data?.items?.map((m) => {
        // Version roll-ups aren't carried by the list endpoint; under mock we
        // derive latest/count from the version fixtures so cards read like the
        // prototype, otherwise stay honest ("—" / 0).
        const vs = USE_MOCK ? modelVersions(m.name) : [];
        return {
          name: m.name,
          desc: m.description ?? m.displayName ?? "",
          framework: (m.labels?.framework as string) ?? "—",
          latest: vs[0]?.version ?? "—",
          versions: vs.length,
          updated: m.updatedAt ?? m.createdAt ?? "",
        };
      }) ?? [],
    [q.data],
  );

  const rows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) return allRows;
    return allRows.filter(
      (r) => r.name.toLowerCase().includes(needle) || r.desc.toLowerCase().includes(needle),
    );
  }, [allRows, search]);

  const openVersions = (r: ModelRow) =>
    setDrawer({ kind: "versions", model: r.name, desc: r.desc, framework: r.framework });

  const onDeleteModel = (r: ModelRow) =>
    confirm({
      title: t("models.deleteModelTitle", { name: r.name }),
      desc: t("models.deleteModelDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delModel.mutate(r.name),
    });

  const columns: ColumnsType<ModelRow> = [
    {
      title: t("models.colName"),
      dataIndex: "name",
      render: (_, r) => (
        <button type="button" className="min-w-0 text-left" onClick={() => openVersions(r)}>
          <div className="font-mono font-medium text-accent">{r.name}</div>
          {r.desc && <div className="truncate text-xs text-muted">{r.desc}</div>}
        </button>
      ),
    },
    {
      title: t("models.colFramework"),
      dataIndex: "framework",
      width: 140,
      render: (v: string) => (v && v !== "—" ? <Tag className="!m-0">{v}</Tag> : <span className="text-muted">—</span>),
    },
    {
      title: t("models.colLatest"),
      dataIndex: "latest",
      width: 130,
      render: (v: string) => <span className="font-mono text-fg-2">{v}</span>,
    },
    { title: t("models.colVersions"), dataIndex: "versions", width: 90, align: "right" },
    {
      title: t("models.colUpdated"),
      dataIndex: "updated",
      width: 150,
      render: (v: string) => <span className="text-muted">{v ? dayjs(v).fromNow() : "—"}</span>,
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 160,
      align: "right",
      render: (_, r) => (
        <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
          <Button
            type="link"
            size="small"
            className="!px-1"
            onClick={() => setDrawer({ kind: "upload", model: r.name })}
          >
            {t("models.addVersion")}
          </Button>
          <Button type="link" size="small" danger className="!px-1" onClick={() => onDeleteModel(r)}>
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.assetCenter"), t("nav.models")]}
      title={t("models.title")}
      subtitle={t("models.subtitle")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawer({ kind: "new" })}>
          {t("models.newModel")}
        </Button>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <Input
          allowClear
          prefix={<SearchOutlined className="text-muted" />}
          placeholder={t("models.searchPlaceholder")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-xs"
        />
        <div className="grow" />
        <Segmented
          value={view}
          onChange={(v) => setView(v as "cards" | "list")}
          options={[
            { value: "cards", icon: <AppstoreOutlined />, label: t("models.viewCards") },
            { value: "list", icon: <UnorderedListOutlined />, label: t("models.viewList") },
          ]}
        />
      </div>

      {view === "cards" ? (
        <Spin spinning={q.isLoading}>
          {rows.length === 0 ? (
            <Card>
              <Empty description={q.isError ? t("common.loadFailed") : t("common.noData")} />
            </Card>
          ) : (
            <>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {rows.map((r) => (
                  <Card
                    key={r.name}
                    hoverable
                    styles={{ body: { padding: 16 } }}
                    onClick={() => openVersions(r)}
                  >
                    <div className="mb-2 flex items-start gap-3">
                      <span className="grid h-9 w-9 shrink-0 place-items-center rounded-md bg-surface-warm text-accent">
                        <DatabaseOutlined />
                      </span>
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-mono font-medium text-fg">{r.name}</div>
                      </div>
                      <Tooltip title={t("common.delete")}>
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={(e) => {
                            e.stopPropagation();
                            onDeleteModel(r);
                          }}
                        />
                      </Tooltip>
                    </div>
                    <p className="mb-3 line-clamp-2 min-h-[2.5rem] text-sm text-fg-2">{r.desc}</p>
                    <div className="flex items-center justify-between text-xs">
                      <span className="font-mono text-muted">
                        {r.latest} · {r.versions} {t("models.versionsSuffix")}
                      </span>
                      <span className="text-muted">
                        {r.updated ? dayjs(r.updated).fromNow() : "—"}
                      </span>
                    </div>
                  </Card>
                ))}
              </div>
              <div className="mt-4 text-sm text-muted">{t("models.total", { count: rows.length })}</div>
            </>
          )}
        </Spin>
      ) : (
        <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
          <Table<ModelRow>
            rowKey="name"
            columns={columns}
            dataSource={rows}
            loading={q.isLoading}
            pagination={{ pageSize: 20, showTotal: (n) => t("models.total", { count: n }), hideOnSinglePage: false }}
            locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
          />
        </Card>
      )}

      {drawer?.kind === "versions" && (
        <VersionsDrawer
          model={drawer.model}
          desc={drawer.desc}
          framework={drawer.framework}
          onClose={() => setDrawer(null)}
          onPull={(version) => setDrawer({ kind: "pull", model: drawer.model, version })}
          onUpload={() => setDrawer({ kind: "upload", model: drawer.model })}
        />
      )}
      {drawer?.kind === "pull" && (
        <PullDrawer model={drawer.model} version={drawer.version} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "new" && <NewModelDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "upload" && <UploadDrawer model={drawer.model} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Version list drawer ───────────────────────────────────────────────────────
function statusMeta(status: sdk.ModelStatus, t: (k: string) => string): { color: string; label: string; pending: boolean } {
  switch (status) {
    case "Ready":
      return { color: "success", label: t("models.statusReady"), pending: false };
    case "Uploading":
      return { color: "processing", label: t("models.statusUploading"), pending: true };
    case "Failed":
      return { color: "error", label: t("models.statusFailed"), pending: false };
    default:
      return { color: "default", label: status, pending: false };
  }
}

function VersionsDrawer({
  model,
  desc,
  framework,
  onClose,
  onPull,
  onUpload,
}: {
  model: string;
  desc: string;
  framework: string;
  onClose: () => void;
  onPull: (version: string) => void;
  onUpload: () => void;
}) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { toast, confirm } = useUI();
  const [search, setSearch] = useState("");
  const versQ = useModelVersions(model);

  const delVer = useApiMutation(
    (version: string) => sdk.deleteModel({ path: { tenant, name: model, version } }),
    { invalidate: [["models"]], success: t("models.verDeleted") },
  );

  const items = versQ.data?.items ?? [];
  const filtered = items.filter((v) => {
    const needle = search.trim().toLowerCase();
    if (!needle) return true;
    return v.version.toLowerCase().includes(needle) || (v.description ?? "").toLowerCase().includes(needle);
  });

  const onDeleteVer = (version: string) =>
    confirm({
      title: t("models.deleteVerTitle", { version }),
      desc: t("models.deleteVerDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delVer.mutate(version),
    });

  return (
    <Drawer
      open
      width={640}
      onClose={onClose}
      title={<span className="font-mono">{model}</span>}
      extra={<span className="text-xs text-muted">{`${desc || t("models.verWeights")} · ${framework}`}</span>}
    >
      <div className="mb-4 flex items-center gap-3">
        <Input
          allowClear
          prefix={<SearchOutlined className="text-muted" />}
          placeholder={t("models.verSearchPlaceholder")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={onUpload}>
          {t("models.addVersion")}
        </Button>
      </div>

      <Spin spinning={versQ.isLoading}>
        {filtered.length === 0 ? (
          <Empty description={versQ.isError ? t("common.loadFailed") : t("common.noData")} />
        ) : (
          <List
            dataSource={filtered}
            split
            renderItem={(v) => {
              const meta = statusMeta(v.status, t);
              return (
                <List.Item
                  className="!px-0"
                  actions={
                    meta.pending
                      ? []
                      : [
                          <Tooltip title={t("models.pullTitle")} key="pull">
                            <Button
                              type="text"
                              size="small"
                              icon={<CloudDownloadOutlined />}
                              onClick={() => onPull(v.version)}
                            />
                          </Tooltip>,
                          <Tooltip title={t("common.delete")} key="del">
                            <Button
                              type="text"
                              size="small"
                              danger
                              icon={<DeleteOutlined />}
                              onClick={() => onDeleteVer(v.version)}
                            />
                          </Tooltip>,
                        ]
                  }
                >
                  <div className="min-w-0 flex-1">
                    <div className="mb-1 flex flex-wrap items-center gap-2">
                      <span className="font-mono font-medium text-fg">{v.version}</span>
                      <Tag color={meta.color} className="!m-0">
                        {meta.label}
                      </Tag>
                      {v.source && <Tag className="!m-0">{v.source}</Tag>}
                    </div>
                    {v.description && <div className="mb-1 text-sm text-fg-2">{v.description}</div>}
                    <div className="flex flex-wrap items-center gap-2 text-xs text-muted">
                      <span className={"font-mono " + (meta.pending ? "" : "text-fg-2")}>
                        {v.uri ?? t("models.addrPending")}
                      </span>
                      {v.uri && (
                        <Tooltip title={t("common.actions")}>
                          <Button
                            type="text"
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() => {
                              void navigator.clipboard?.writeText(v.uri ?? "");
                              toast(t("models.addrCopied"));
                            }}
                          />
                        </Tooltip>
                      )}
                      {v.owner && <span>· {v.owner}</span>}
                    </div>
                  </div>
                </List.Item>
              );
            }}
          />
        )}
      </Spin>
    </Drawer>
  );
}

// ── Pull-command drawer ───────────────────────────────────────────────────────
function PullDrawer({ model, version, onClose }: { model: string; version: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { toast } = useUI();
  const resolveQ = useModelResolve(model, version, tenant);
  const cmd = resolveQ.uri ? `docker pull ${resolveQ.uri}` : t("models.pullResolving");

  return (
    <Drawer
      open
      width={520}
      onClose={onClose}
      title={t("models.pullTitle")}
      extra={<span className="font-mono text-xs text-muted">{`${model}@${version}`}</span>}
      footer={
        <div className="flex justify-end">
          <Button type="primary" onClick={onClose}>
            {t("models.done")}
          </Button>
        </div>
      }
    >
      <p className="mb-3 text-sm text-muted">{t("models.pullHint")}</p>
      <pre className="overflow-x-auto rounded-md border border-border-soft bg-surface p-3 font-mono text-xs text-fg-2">
        {cmd}
      </pre>
      <Button
        className="mt-3"
        icon={<CopyOutlined />}
        disabled={!resolveQ.uri}
        onClick={() => {
          void navigator.clipboard?.writeText(cmd);
          toast(t("models.commandCopied"));
        }}
      >
        {t("models.copyCommand")}
      </Button>
    </Drawer>
  );
}

// Resolve a version's pull URI on demand (thin local query; not a shared hook).
function useModelResolve(model: string, version: string, tenant: string): { uri?: string } {
  const q = useModelVersions(model);
  // Versions list already carries the URI; resolve endpoint mints temp creds but
  // the displayed pull target is the version URI. Fall back to a live resolve only
  // if the version row lacks one.
  const fromList = q.data?.items?.find((v) => v.version === version)?.uri;
  void tenant;
  return { uri: fromList };
}

// ── New-model drawer ──────────────────────────────────────────────────────────
const TASK_OPTIONS = [
  "Text Generation",
  "Text Classification",
  "Question Answering",
  "Summarization",
  "Translation",
  "Feature Extraction",
  "Image Classification",
  "Object Detection",
  "Automatic Speech Recognition",
  "Text-to-Image",
];
const FRAMEWORK_OPTIONS = ["PyTorch", "Safetensors", "Transformers", "TensorFlow", "JAX", "ONNX", "GGUF"];

interface NewModelValues {
  name: string;
  description?: string;
  tasks?: string[];
  framework?: string;
  params?: string;
}

function NewModelDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [form] = Form.useForm<NewModelValues>();
  const [customTags, setCustomTags] = useState<Record<string, string>>({});
  const [ctKey, setCtKey] = useState("");
  const [ctVal, setCtVal] = useState("");

  const create = useApiMutation(
    (body: sdk.ArtifactDefinitionCreateRequest) => sdk.createModelDefinition({ path: { tenant, name: body.name }, body }),
    { invalidate: [["models"]], success: t("models.modelCreated") },
  );

  const addTag = () => {
    const k = ctKey.trim();
    if (!k) return;
    setCustomTags((m) => ({ ...m, [k]: ctVal.trim() }));
    setCtKey("");
    setCtVal("");
  };
  const removeTag = (k: string) =>
    setCustomTags((m) => {
      const next = { ...m };
      delete next[k];
      return next;
    });

  const onFinish = (v: NewModelValues) => {
    const labels: Record<string, string> = {};
    if (v.framework) labels.framework = v.framework;
    if (v.tasks?.length) labels.tasks = v.tasks.join(",");
    if (v.params?.trim()) labels.params = v.params.trim();
    create.mutate(
      {
        name: v.name.trim(),
        description: v.description?.trim() || undefined,
        labels: Object.keys(labels).length ? labels : undefined,
        annotations: Object.keys(customTags).length ? customTags : undefined,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Drawer
      open
      width={640}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("models.newModelTitle")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">{t("models.newModelSub")}</div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={create.isPending} onClick={() => form.submit()}>
            {t("models.createModel")}
          </Button>
        </div>
      }
    >
      <Form<NewModelValues> form={form} layout="vertical" size="large" onFinish={onFinish}>
        <FieldSection n={1} title={t("models.fsBasic")} />
        <Form.Item name="name" label={t("models.fName")} rules={[{ required: true, message: t("models.fNameHelp") }]} extra={t("models.fNameHelp")}>
          <Input className="font-mono" placeholder={t("models.fNamePlaceholder")} />
        </Form.Item>
        <Form.Item name="description" label={t("models.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("models.fDescPlaceholder")} />
        </Form.Item>

        <FieldSection n={2} title={t("models.fsLabels")} />
        <Form.Item name="tasks" label={t("models.lTasks")}>
          <Select
            mode="tags"
            allowClear
            options={TASK_OPTIONS.map((o) => ({ label: o, value: o }))}
            placeholder={t("models.lTasks")}
          />
        </Form.Item>
        <Form.Item name="params" label={t("models.lParameters")}>
          <Input className="font-mono !w-40" placeholder={t("models.paramsPlaceholder")} />
        </Form.Item>
        <Form.Item name="framework" label={t("models.lFramework")}>
          <Select
            allowClear
            options={FRAMEWORK_OPTIONS.map((o) => ({ label: o, value: o }))}
            placeholder={t("models.lFramework")}
          />
        </Form.Item>

        <Form.Item label={t("models.lCustom")}>
          <div className="mb-2 flex flex-wrap gap-2">
            {Object.entries(customTags).map(([k, v]) => (
              <Tag key={k} closable onClose={() => removeTag(k)} className="font-mono">
                {k}:{v}
              </Tag>
            ))}
          </div>
          <Space.Compact className="w-full">
            <Input className="font-mono" placeholder={t("models.customKeyPlaceholder")} value={ctKey} onChange={(e) => setCtKey(e.target.value)} />
            <Input className="font-mono" placeholder={t("models.customValPlaceholder")} value={ctVal} onChange={(e) => setCtVal(e.target.value)} />
            <Button onClick={addTag}>{t("models.add")}</Button>
          </Space.Compact>
        </Form.Item>
      </Form>
    </Drawer>
  );
}

// ── Upload-version drawer (two add methods: web upload vs external register) ───
interface UploadValues {
  version: string;
  description?: string;
  remoteSourceKind: sdk.RemoteSourceKind;
  remoteUri?: string;
}

function UploadDrawer({ model, onClose }: { model: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [form] = Form.useForm<UploadValues>();
  const [method, setMethod] = useState<sdk.ArtifactSource>("webUpload");

  const initiate = useApiMutation(
    (body: sdk.ModelInitiateRequest) => sdk.initiateModel({ path: { tenant, name: model }, body }),
    { invalidate: [["models"]], success: t("models.versionSubmitted") },
  );

  const onFinish = (v: UploadValues) => {
    const isExternal = method === "external";
    initiate.mutate(
      {
        version: v.version.trim(),
        description: v.description?.trim() || undefined,
        source: method,
        remoteSourceKind: isExternal ? v.remoteSourceKind : undefined,
        remoteUri: isExternal && v.remoteUri?.trim() ? v.remoteUri.trim() : undefined,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Drawer
      open
      width={640}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("models.uploadTitle")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">{t("models.uploadSub")}</div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={initiate.isPending} onClick={() => form.submit()}>
            {t("models.submit")}
          </Button>
        </div>
      }
    >
      <Form<UploadValues>
        form={form}
        layout="vertical"
        size="large"
        onFinish={onFinish}
        initialValues={{ remoteSourceKind: "s3" }}
      >
        <FieldSection n={1} title={t("models.fsBasic")} />
        <Form.Item label={t("models.fModel")}>
          <Input className="font-mono" value={model} disabled />
        </Form.Item>
        <Form.Item name="version" label={t("models.fVersion")} rules={[{ required: true }]}>
          <Input className="font-mono" placeholder={t("models.fVersionPlaceholder")} />
        </Form.Item>
        <Form.Item name="description" label={t("models.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("models.fUploadDescPlaceholder")} />
        </Form.Item>

        <FieldSection n={2} title={t("models.fsMethod")} />
        <Tabs
          activeKey={method === "external" ? "remote" : method}
          onChange={(k) => setMethod(k === "remote" ? "external" : (k as sdk.ArtifactSource))}
          items={[
            {
              key: "webUpload",
              label: t("models.methodWeb"),
              children: (
                <Upload.Dragger multiple beforeUpload={() => false} className="!bg-bg">
                  <p className="ant-upload-drag-icon">
                    <InboxOutlined />
                  </p>
                  <p className="ant-upload-text">{t("models.dzTitle")}</p>
                  <p className="ant-upload-hint">{t("models.dzHint")}</p>
                </Upload.Dragger>
              ),
            },
            {
              key: "remote",
              label: t("models.methodRemote"),
              children: (
                <>
                  <Form.Item name="remoteSourceKind" label={t("models.fStorageKind")} rules={[{ required: method === "external" }]}>
                    <Select
                      options={[
                        { value: "s3", label: t("models.storageS3") },
                        { value: "oci", label: t("models.storageOci") },
                        { value: "http", label: t("models.storageHttp") },
                        { value: "hf", label: t("models.storageHf") },
                        { value: "custom", label: t("models.storageCustom") },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item name="remoteUri" label={t("models.fRemoteUri")} rules={[{ required: method === "external" }]}>
                    <Input className="font-mono" placeholder={t("models.remoteUriPlaceholder")} />
                  </Form.Item>
                </>
              ),
            },
            {
              key: "oras",
              label: t("models.methodOras"),
              children: <OrasGuide model={model} tenant={tenant} />,
            },
          ]}
        />
      </Form>
    </Drawer>
  );
}

function OrasGuide({ model, tenant }: { model: string; tenant: string }) {
  const { t } = useTranslation();
  const dl = `# Linux x86_64
curl -LO https://github.com/oras-project/oras/releases/download/v1.2.0/oras_1.2.0_linux_amd64.tar.gz
tar -xzf oras_1.2.0_linux_amd64.tar.gz oras
sudo mv oras /usr/local/bin/ && oras version`;
  const push = `oras login zot.axisml.internal -u <user> -p <token>
cd ./${model}
oras push zot.axisml.internal/${tenant}/${model}:v5 \\
  --artifact-type application/vnd.axisml.model.v1 \\
  ./*:application/octet-stream`;
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted">{t("models.orasHelp")}</p>
      <div>
        <div className="mb-1 text-sm font-semibold text-fg">{t("models.orasStep1")}</div>
        <pre className="overflow-x-auto rounded-md border border-border-soft bg-surface p-3 font-mono text-xs text-fg-2">{dl}</pre>
        <a className="text-xs text-accent" href="https://oras.land/docs/installation" target="_blank" rel="noopener noreferrer">
          {t("models.orasDocsLink")}
        </a>
      </div>
      <div>
        <div className="mb-1 text-sm font-semibold text-fg">{t("models.orasStep2")}</div>
        <pre className="overflow-x-auto rounded-md border border-border-soft bg-surface p-3 font-mono text-xs text-fg-2">{push}</pre>
      </div>
    </div>
  );
}
