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
  ContainerOutlined,
  CloseOutlined,
} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useImages, useImageVersions } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { FieldSection } from "@/components/FieldSection";

interface ImageRow {
  name: string;
  desc: string;
  purpose: string;
  latest: string;
  versions: number;
  updated: string;
}

type DrawerState =
  | { kind: "versions"; image: string; desc: string }
  | { kind: "pull"; image: string; version: string; uri: string }
  | { kind: "new" }
  | { kind: "add"; image: string };

export default function Images() {
  const q = useImages();
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { confirm } = useUI();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [search, setSearch] = useState("");
  const [drawer, setDrawer] = useState<DrawerState | null>(null);

  const delImage = useApiMutation(
    (name: string) => sdk.deleteImageDefinition({ path: { tenant, name } }),
    { invalidate: [["images"]], success: t("images.imageDeleted") },
  );

  const allRows: ImageRow[] = useMemo(
    () =>
      q.data?.items?.map((m) => ({
        name: m.name,
        desc: m.description ?? m.displayName ?? "",
        purpose: (m.labels?.purpose as string) ?? "—",
        latest: "—",
        versions: 0,
        updated: m.updatedAt ?? m.createdAt ?? "",
      })) ?? [],
    [q.data],
  );

  const rows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) return allRows;
    return allRows.filter(
      (r) => r.name.toLowerCase().includes(needle) || r.desc.toLowerCase().includes(needle),
    );
  }, [allRows, search]);

  const openVersions = (r: ImageRow) => setDrawer({ kind: "versions", image: r.name, desc: r.desc });

  const onDeleteImage = (r: ImageRow) =>
    confirm({
      title: t("images.deleteImageTitle", { name: r.name }),
      desc: t("images.deleteImageDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delImage.mutate(r.name),
    });

  const columns: ColumnsType<ImageRow> = [
    {
      title: t("images.colName"),
      dataIndex: "name",
      render: (_, r) => (
        <button type="button" className="min-w-0 text-left" onClick={() => openVersions(r)}>
          <div className="font-mono font-medium text-accent">{r.name}</div>
          {r.desc && <div className="truncate text-xs text-muted">{r.desc}</div>}
        </button>
      ),
    },
    {
      title: t("images.colPurpose"),
      dataIndex: "purpose",
      width: 140,
      render: (v: string) => (v && v !== "—" ? <Tag className="!m-0">{v}</Tag> : <span className="text-muted">—</span>),
    },
    {
      title: t("images.colLatest"),
      dataIndex: "latest",
      width: 130,
      render: (v: string) => <span className="font-mono text-fg-2">{v}</span>,
    },
    { title: t("images.colVersions"), dataIndex: "versions", width: 90, align: "right" },
    {
      title: t("images.colUpdated"),
      dataIndex: "updated",
      width: 180,
      render: (v: string) => <span className="text-muted">{v ? dayjs(v).format("YYYY-MM-DD HH:mm") : "—"}</span>,
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
            onClick={() => setDrawer({ kind: "add", image: r.name })}
          >
            {t("images.addVersion")}
          </Button>
          <Button type="link" size="small" danger className="!px-1" onClick={() => onDeleteImage(r)}>
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.assetCenter"), t("nav.images")]}
      title={t("images.title")}
      subtitle={t("images.subtitle")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawer({ kind: "new" })}>
          {t("images.newImage")}
        </Button>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <Input
          allowClear
          prefix={<SearchOutlined className="text-muted" />}
          placeholder={t("images.searchPlaceholder")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-xs"
        />
        <div className="grow" />
        <Segmented
          value={view}
          onChange={(v) => setView(v as "cards" | "list")}
          options={[
            { value: "cards", icon: <AppstoreOutlined />, label: t("images.viewCards") },
            { value: "list", icon: <UnorderedListOutlined />, label: t("images.viewList") },
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
                        <ContainerOutlined />
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
                            onDeleteImage(r);
                          }}
                        />
                      </Tooltip>
                    </div>
                    <p className="mb-3 line-clamp-2 min-h-[2.5rem] text-sm text-fg-2">{r.desc}</p>
                    <div className="flex items-center justify-between text-xs">
                      <span className="font-mono text-muted">
                        {r.latest} · {r.versions} {t("images.versionsSuffix")}
                      </span>
                      <span className="text-muted">{r.updated ? dayjs(r.updated).fromNow() : "—"}</span>
                    </div>
                  </Card>
                ))}
              </div>
              <div className="mt-4 text-sm text-muted">{t("images.total", { count: rows.length })}</div>
            </>
          )}
        </Spin>
      ) : (
        <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
          <Table<ImageRow>
            rowKey="name"
            columns={columns}
            dataSource={rows}
            loading={q.isLoading}
            pagination={{ pageSize: 20, showTotal: (n) => t("images.total", { count: n }), hideOnSinglePage: false }}
            locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
          />
        </Card>
      )}

      {drawer?.kind === "versions" && (
        <VersionsDrawer
          image={drawer.image}
          desc={drawer.desc}
          onClose={() => setDrawer(null)}
          onPull={(version, uri) => setDrawer({ kind: "pull", image: drawer.image, version, uri })}
          onAdd={() => setDrawer({ kind: "add", image: drawer.image })}
        />
      )}
      {drawer?.kind === "pull" && (
        <PullDrawer image={drawer.image} version={drawer.version} uri={drawer.uri} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "new" && <NewImageDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "add" && <AddVersionDrawer image={drawer.image} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Version list drawer ───────────────────────────────────────────────────────
function statusMeta(status: sdk.ImageStatus, t: (k: string) => string): { color: string; label: string; pending: boolean } {
  switch (status) {
    case "Ready":
      return { color: "success", label: t("images.statusReady"), pending: false };
    case "Uploading":
      return { color: "processing", label: t("images.statusUploading"), pending: true };
    case "Failed":
      return { color: "error", label: t("images.statusFailed"), pending: false };
    default:
      return { color: "default", label: status, pending: false };
  }
}

function VersionsDrawer({
  image,
  desc,
  onClose,
  onPull,
  onAdd,
}: {
  image: string;
  desc: string;
  onClose: () => void;
  onPull: (version: string, uri: string) => void;
  onAdd: () => void;
}) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { toast, confirm } = useUI();
  const [search, setSearch] = useState("");
  const versQ = useImageVersions(image);

  const delVer = useApiMutation(
    (version: string) => sdk.deleteImage({ path: { tenant, name: image, version } }),
    { invalidate: [["images"]], success: t("images.verDeleted") },
  );

  const items = versQ.data?.items ?? [];
  const filtered = items.filter((v) => {
    const needle = search.trim().toLowerCase();
    if (!needle) return true;
    return v.version.toLowerCase().includes(needle) || (v.description ?? "").toLowerCase().includes(needle);
  });

  const onDeleteVer = (version: string) =>
    confirm({
      title: t("images.deleteVerTitle", { version }),
      desc: t("images.deleteVerDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delVer.mutate(version),
    });

  return (
    <Drawer
      open
      width={640}
      onClose={onClose}
      title={<span className="font-mono">{image}</span>}
      extra={<span className="text-xs text-muted">{desc || t("images.verImage")}</span>}
    >
      <div className="mb-4 flex items-center gap-3">
        <Input
          allowClear
          prefix={<SearchOutlined className="text-muted" />}
          placeholder={t("images.verSearchPlaceholder")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>
          {t("images.addVersion")}
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
                          <Tooltip title={t("images.pullTitle")} key="pull">
                            <Button
                              type="text"
                              size="small"
                              icon={<CloudDownloadOutlined />}
                              onClick={() => onPull(v.version, v.uri ?? "")}
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
                        {v.uri ?? t("images.addrPending")}
                      </span>
                      {v.uri && (
                        <Tooltip title={t("common.actions")}>
                          <Button
                            type="text"
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() => {
                              void navigator.clipboard?.writeText(v.uri ?? "");
                              toast(t("images.addrCopied"));
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
function PullDrawer({ image, version, uri, onClose }: { image: string; version: string; uri: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { toast } = useUI();
  const ref = uri || `zot.axisml.internal/<tenant>/${image}:${version}`;
  const cmd = `docker login zot.axisml.internal -u <user> -p <token>\ndocker pull ${ref}`;

  return (
    <Drawer
      open
      width={520}
      onClose={onClose}
      title={t("images.pullTitle")}
      extra={<span className="font-mono text-xs text-muted">{`${image}:${version}`}</span>}
      footer={
        <div className="flex justify-end">
          <Button type="primary" onClick={onClose}>
            {t("images.done")}
          </Button>
        </div>
      }
    >
      <p className="mb-3 text-sm text-muted">{t("images.pullHint")}</p>
      <pre className="overflow-x-auto rounded-md border border-border-soft bg-surface p-3 font-mono text-xs text-fg-2">
        {cmd}
      </pre>
      <Button
        className="mt-3"
        icon={<CopyOutlined />}
        onClick={() => {
          void navigator.clipboard?.writeText(cmd);
          toast(t("images.commandCopied"));
        }}
      >
        {t("images.copyCommand")}
      </Button>
    </Drawer>
  );
}

// ── New-image drawer ──────────────────────────────────────────────────────────
function NewImageDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [form] = Form.useForm<NewImageValues>();
  const [customTags, setCustomTags] = useState<Record<string, string>>({});
  const [ctKey, setCtKey] = useState("");
  const [ctVal, setCtVal] = useState("");

  const PURPOSE_OPTIONS = [
    { value: "training", label: t("images.purposeTraining") },
    { value: "inference", label: t("images.purposeInference") },
    { value: "workspace", label: t("images.purposeWorkspace") },
    { value: "custom", label: t("images.purposeCustom") },
  ];

  const create = useApiMutation(
    (body: sdk.ArtifactDefinitionCreateRequest) => sdk.createImageDefinition({ path: { tenant, name: body.name }, body }),
    { invalidate: [["images"]], success: t("images.imageCreated") },
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

  const onFinish = (v: NewImageValues) => {
    const labels: Record<string, string> = { ...customTags };
    if (v.purpose) labels.purpose = v.purpose;
    create.mutate(
      {
        name: v.name.trim(),
        description: v.description?.trim() || undefined,
        labels: Object.keys(labels).length ? labels : undefined,
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
          <div className="text-base font-semibold text-fg">{t("images.newImageTitle")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">{t("images.newImageSub")}</div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={create.isPending} onClick={() => form.submit()}>
            {t("images.createImage")}
          </Button>
        </div>
      }
    >
      <Form<NewImageValues> form={form} layout="vertical" size="large" onFinish={onFinish} initialValues={{ purpose: "training" }}>
        <FieldSection n={1} title={t("images.fsBasic")} />
        <Form.Item name="name" label={t("images.fName")} rules={[{ required: true, message: t("images.fNameHelp") }]} extra={t("images.fNameHelp")}>
          <Input className="font-mono" placeholder={t("images.fNamePlaceholder")} />
        </Form.Item>
        <Form.Item name="purpose" label={t("images.fPurpose")}>
          <Select options={PURPOSE_OPTIONS} />
        </Form.Item>
        <Form.Item name="description" label={t("images.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("images.fDescPlaceholder")} />
        </Form.Item>

        <FieldSection n={2} title={t("images.fsLabels")} />
        <Form.Item label={t("images.lCustom")}>
          <div className="mb-2 flex flex-wrap gap-2">
            {Object.entries(customTags).map(([k, v]) => (
              <Tag key={k} closable onClose={() => removeTag(k)} className="font-mono">
                {k}:{v}
              </Tag>
            ))}
          </div>
          <Space.Compact className="w-full">
            <Input className="font-mono" placeholder={t("images.customKeyPlaceholder")} value={ctKey} onChange={(e) => setCtKey(e.target.value)} />
            <Input className="font-mono" placeholder={t("images.customValPlaceholder")} value={ctVal} onChange={(e) => setCtVal(e.target.value)} />
            <Button onClick={addTag}>{t("images.add")}</Button>
          </Space.Compact>
        </Form.Item>
      </Form>
    </Drawer>
  );
}

interface NewImageValues {
  name: string;
  purpose?: string;
  description?: string;
}

// ── Add-version drawer (two methods: external register vs Docker push) ─────────
interface AddVersionValues {
  version: string;
  description?: string;
  sourceImageRef?: string;
}

function AddVersionDrawer({ image, onClose }: { image: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { toast } = useUI();
  const [form] = Form.useForm<AddVersionValues>();
  const [method, setMethod] = useState<"external" | "dockerPush">("external");

  const initiate = useApiMutation(
    (body: sdk.ImageInitiateRequest) => sdk.initiateImage({ path: { tenant, name: image }, body }),
    { invalidate: [["images"]], success: t("images.versionSubmitted") },
  );

  const onFinish = (v: AddVersionValues) => {
    const body: sdk.ImageInitiateRequest =
      method === "external"
        ? {
            version: v.version.trim(),
            spec: {},
            description: v.description?.trim() || undefined,
            source: "external",
            sourceImageRef: v.sourceImageRef?.trim() || undefined,
          }
        : {
            version: v.version.trim(),
            spec: {},
            description: v.description?.trim() || undefined,
            source: "dockerPush",
          };
    initiate.mutate(body, { onSuccess: onClose });
  };

  return (
    <Drawer
      open
      width={640}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("images.addVerTitle")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">{t("images.addVerSub")}</div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={initiate.isPending} onClick={() => form.submit()}>
            {t("images.submit")}
          </Button>
        </div>
      }
    >
      <Form<AddVersionValues> form={form} layout="vertical" size="large" onFinish={onFinish}>
        <FieldSection n={1} title={t("images.fsBasic")} />
        <Form.Item label={t("images.fImage")}>
          <Input className="font-mono" value={image} disabled />
        </Form.Item>
        <Form.Item name="version" label={t("images.fVersion")} rules={[{ required: true }]}>
          <Input className="font-mono" placeholder={t("images.fVersionPlaceholder")} />
        </Form.Item>
        <Form.Item name="description" label={t("images.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("images.fAddVerDescPlaceholder")} />
        </Form.Item>

        <FieldSection n={2} title={t("images.fsMethod")} />
        <Tabs
          activeKey={method}
          onChange={(k) => setMethod(k as "external" | "dockerPush")}
          items={[
            {
              key: "external",
              label: t("images.methodExternal"),
              children: (
                <>
                  <p className="mb-3 text-sm text-muted">{t("images.externalHelp")}</p>
                  <Form.Item name="sourceImageRef" label={t("images.fSourceRef")} rules={[{ required: method === "external" }]}>
                    <Input className="font-mono" placeholder={t("images.sourceRefPlaceholder")} />
                  </Form.Item>
                  <Form.Item label={t("images.fPullCred")}>
                    <Select
                      defaultValue="public"
                      options={[
                        { value: "public", label: t("images.credPublic") },
                        { value: "ngc", label: t("images.credNgc") },
                        { value: "harbor", label: t("images.credHarbor") },
                        { value: "new", label: t("images.credNew") },
                      ]}
                    />
                  </Form.Item>
                </>
              ),
            },
            {
              key: "dockerPush",
              label: t("images.methodDocker"),
              children: <DockerGuide image={image} tenant={tenant} onCopy={(text) => {
                void navigator.clipboard?.writeText(text);
                toast(t("images.commandCopied"));
              }} />,
            },
          ]}
        />
      </Form>
    </Drawer>
  );
}

function DockerGuide({ image, tenant, onCopy }: { image: string; tenant: string; onCopy: (text: string) => void }) {
  const { t } = useTranslation();
  const login = `# 临时凭证有效期 1h
docker login zot.axisml.internal -u <user> -p <token>`;
  const push = `# 1. 为本地镜像打上目标 tag
docker tag <local-image>:<tag> zot.axisml.internal/${tenant}/${image}:<tag>

# 2. 推送到镜像仓
docker push zot.axisml.internal/${tenant}/${image}:<tag>`;
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted">{t("images.dockerHelp")}</p>
      <div>
        <div className="mb-1 text-sm font-semibold text-fg">{t("images.dockerStep1")}</div>
        <pre className="overflow-x-auto rounded-md border border-border-soft bg-surface p-3 font-mono text-xs text-fg-2">{login}</pre>
        <Button className="mt-2" size="small" icon={<CopyOutlined />} onClick={() => onCopy(login)}>
          {t("images.copyCommand")}
        </Button>
      </div>
      <div>
        <div className="mb-1 text-sm font-semibold text-fg">{t("images.dockerStep2")}</div>
        <pre className="overflow-x-auto rounded-md border border-border-soft bg-surface p-3 font-mono text-xs text-fg-2">{push}</pre>
        <Button className="mt-2" size="small" icon={<CopyOutlined />} onClick={() => onCopy(push)}>
          {t("images.copyCommand")}
        </Button>
      </div>
    </div>
  );
}
