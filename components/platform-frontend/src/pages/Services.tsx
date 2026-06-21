import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  Table,
  Button,
  Input,
  Select,
  Space,
  Card,
  Divider,
  Drawer,
  Form,
  InputNumber,
  Switch,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  SearchOutlined,
  CloseOutlined,
} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import {
  useServices,
  useModels,
  useImages,
  useResourcePools,
  useModelVersions,
  useImageVersions,
} from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";
import { FieldSection } from "@/components/FieldSection";
import { CardRadio } from "@/components/CardRadio";

interface SvcRow {
  name: string;
  desc: string;
  phase?: string;
  replicas: string;
  replicaCount: number;
  poolName?: string;
  unitName?: string;
  unit: string;
  url?: string;
  running: boolean; // true → can stop; false → can start
  displayName?: string;
}

// Phases that mean the service is up enough to offer a "stop" action; the rest
// offer "start". The display label itself comes from the shared PhaseTag catalog.
const RUNNING_PHASES = new Set(["Ready", "Degraded", "Creating", "Pending"]);

type DrawerMode = "new" | "edit" | "scale";
const INVALIDATE = [["mlservices"]];

export default function Services() {
  const q = useServices();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; row?: SvcRow } | null>(null);
  const [search, setSearch] = useState("");
  const [phase, setPhase] = useState<string>("");
  const [pool, setPool] = useState<string>("");

  const del = useApiMutation((name: string) => sdk.deleteMlService({ path: { name } }), {
    invalidate: INVALIDATE,
    success: t("services.deleted"),
  });
  const start = useApiMutation((name: string) => sdk.startMlService({ path: { name } }), {
    invalidate: INVALIDATE,
    success: t("services.starting"),
  });
  const stop = useApiMutation((name: string) => sdk.stopMlService({ path: { name } }), {
    invalidate: INVALIDATE,
    success: t("services.stopping"),
  });

  const allRows: SvcRow[] = useMemo(
    () =>
      q.data?.items?.map((s) => ({
        name: s.name,
        desc: s.description ?? s.displayName ?? "",
        phase: s.phase,
        replicas: `${s.readyReplicas ?? 0} / ${s.replicas ?? 0}`,
        replicaCount: s.replicas ?? 0,
        poolName: s.poolName,
        unitName: s.unitName,
        unit: `${s.poolName ?? "—"}/${s.unitName ?? "—"}`,
        url: s.accessUrl,
        running: RUNNING_PHASES.has(s.phase ?? ""),
        displayName: s.displayName,
      })) ?? [],
    [q.data],
  );

  const poolOptions = useMemo(
    () => Array.from(new Set(allRows.map((r) => r.poolName).filter((p): p is string => !!p))),
    [allRows],
  );
  const phaseOptions = useMemo(
    () => Array.from(new Set(allRows.map((r) => r.phase).filter((p): p is string => !!p))),
    [allRows],
  );

  const rows = allRows.filter(
    (r) =>
      (!search || r.name.includes(search)) &&
      (!phase || r.phase === phase) &&
      (!pool || r.poolName === pool),
  );

  const onDelete = (r: SvcRow) =>
    confirm({
      title: t("services.deleteTitle", { name: r.name }),
      desc: t("services.deleteDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate(r.name),
    });

  const columns: ColumnsType<SvcRow> = [
    {
      title: t("services.colName"),
      dataIndex: "name",
      render: (_, r) => (
        <div className="min-w-0">
          <Link to={`/services/${r.name}`} className="font-mono font-medium">
            {r.name}
          </Link>
          {r.desc && <div className="truncate text-xs text-muted">{r.desc}</div>}
        </div>
      ),
    },
    {
      title: t("services.colStatus"),
      dataIndex: "phase",
      width: 120,
      render: (v: string) => <PhaseTag phase={v} />,
    },
    {
      title: t("services.colReplicas"),
      dataIndex: "replicas",
      width: 100,
      align: "right",
      render: (v: string) => <span className="font-mono">{v}</span>,
    },
    {
      title: t("services.colUnit"),
      dataIndex: "unit",
      width: 180,
      render: (v: string) => <span className="font-mono text-muted">{v}</span>,
    },
    {
      title: t("services.colAccess"),
      dataIndex: "url",
      render: (v?: string) =>
        v ? <span className="font-mono text-xs text-muted">{v}</span> : <span className="text-muted">—</span>,
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 240,
      align: "right",
      render: (_, r) => (
        <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
          <Link to={`/services/${r.name}`}>
            <Button type="link" size="small" className="!px-1">
              {t("common.detail")}
            </Button>
          </Link>
          <Button type="link" size="small" className="!px-1" onClick={() => setDrawer({ mode: "edit", row: r })}>
            {t("common.edit")}
          </Button>
          <Button type="link" size="small" className="!px-1" onClick={() => setDrawer({ mode: "scale", row: r })}>
            {t("services.scale")}
          </Button>
          {r.running ? (
            <Button type="link" size="small" className="!px-1" loading={stop.isPending} onClick={() => stop.mutate(r.name)}>
              {t("services.stop")}
            </Button>
          ) : (
            <Button type="link" size="small" className="!px-1" loading={start.isPending} onClick={() => start.mutate(r.name)}>
              {t("services.start")}
            </Button>
          )}
          <Button type="link" size="small" danger className="!px-1" onClick={() => onDelete(r)}>
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.serviceCenter"), t("nav.services")]}
      title={t("services.title")}
      subtitle={t("services.subtitle")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawer({ mode: "new" })}>
          {t("services.newService")}
        </Button>
      }
    >
      <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 border-b border-border-soft p-4">
          <Input
            allowClear
            prefix={<SearchOutlined className="text-muted" />}
            placeholder={t("services.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
          <Select
            value={phase || undefined}
            onChange={(v) => setPhase(v ?? "")}
            placeholder={t("services.statusAll")}
            allowClear
            className="min-w-40"
            options={phaseOptions.map((p) => ({ label: t(`phase.${p}`, { defaultValue: p }), value: p }))}
          />
          <Select
            value={pool || undefined}
            onChange={(v) => setPool(v ?? "")}
            placeholder={t("services.poolAll")}
            allowClear
            className="min-w-40"
            options={poolOptions.map((p) => ({ label: p, value: p }))}
          />
          <Button
            onClick={() => {
              setSearch("");
              setPhase("");
              setPool("");
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <Table<SvcRow>
          rowKey="name"
          columns={columns}
          dataSource={rows}
          loading={q.isLoading}
          pagination={{ pageSize: 20, showTotal: (n) => t("services.total", { count: n }), hideOnSinglePage: false }}
          locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
        />
      </Card>

      {drawer?.mode === "new" && <NewSvcDrawer onClose={() => setDrawer(null)} />}
      {drawer?.mode === "edit" && drawer.row && <EditSvcDrawer row={drawer.row} onClose={() => setDrawer(null)} />}
      {drawer?.mode === "scale" && drawer.row && <ScaleDrawer row={drawer.row} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Resource-unit spec helper (best-effort human label from requests map) ──────
function unitSpec(u: sdk.ResourceUnit): string {
  const r = (u.requests ?? {}) as Record<string, string | undefined>;
  const parts = Object.entries(r)
    .filter(([, v]) => !!v)
    .map(([k, v]) => `${k}=${v}`);
  return parts.join(" · ") || u.name;
}

// ── Controlled port rows feeding ServicePort[] ────────────────────────────────
interface PortRow {
  id: number;
  name: string;
  port: string;
}
let portSeq = 0;

function PortList({ rows, setRows }: { rows: PortRow[]; setRows: (fn: (r: PortRow[]) => PortRow[]) => void }) {
  const { t } = useTranslation();
  const add = () => setRows((r) => [...r, { id: ++portSeq, name: "", port: "" }]);
  const remove = (id: number) => setRows((r) => (r.length <= 1 ? r : r.filter((x) => x.id !== id)));
  const update = (id: number, patch: Partial<PortRow>) =>
    setRows((r) => r.map((x) => (x.id === id ? { ...x, ...patch } : x)));
  return (
    <div className="space-y-2">
      {rows.map((row) => (
        <div className="flex items-center gap-2" key={row.id}>
          <Input
            className="font-mono"
            value={row.name}
            onChange={(e) => update(row.id, { name: e.target.value })}
            placeholder={t("services.fPortName")}
            maxLength={15}
          />
          <Input
            className="font-mono"
            value={row.port}
            onChange={(e) => update(row.id, { port: e.target.value })}
            placeholder={t("services.fPortNumber")}
            inputMode="numeric"
          />
          <Button type="text" icon={<CloseOutlined />} onClick={() => remove(row.id)} />
        </div>
      ))}
      <Button type="link" size="small" icon={<PlusOutlined />} className="!px-0" onClick={add}>
        {t("services.addPort")}
      </Button>
    </div>
  );
}

function toServicePorts(rows: PortRow[]): sdk.ServicePort[] {
  return rows
    .filter((r) => r.name.trim() && r.port.trim())
    .map((r) => ({ name: r.name.trim(), port: Number(r.port) }));
}

// ── Create drawer ─────────────────────────────────────────────────────────────
function NewSvcDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const models = useModels();
  const images = useImages();
  const pools = useResourcePools();

  const create = useApiMutation((body: sdk.MlServiceCreateRequest) => sdk.createMlService({ body }), {
    invalidate: INVALIDATE,
    success: t("services.onlineProgress"),
  });

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [modelName, setModelName] = useState("");
  const [modelVersion, setModelVersion] = useState("");
  const [imageName, setImageName] = useState("");
  const [imageVersion, setImageVersion] = useState("");
  const [poolName, setPoolName] = useState("");
  const [unitName, setUnitName] = useState("");
  const [replicas, setReplicas] = useState<number>(1);
  const [ports, setPorts] = useState<PortRow[]>(() => [{ id: ++portSeq, name: "http", port: "8000" }]);
  const [routeEnabled, setRouteEnabled] = useState(true);
  const [routePath, setRoutePath] = useState("");

  const modelVersions = useModelVersions(modelName);
  const imageVersions = useImageVersions(imageName);

  const modelOptions = useMemo(
    () => (models.data?.items ?? []).map((m) => ({ label: m.displayName || m.name, value: m.name })),
    [models.data],
  );
  const modelVersionOptions = useMemo(
    () => (modelVersions.data?.items ?? []).map((v) => ({ label: v.version, value: v.version })),
    [modelVersions.data],
  );
  const imageOptions = useMemo(
    () => (images.data?.items ?? []).map((i) => ({ label: i.displayName || i.name, value: i.name })),
    [images.data],
  );
  const imageVersionOptions = useMemo(
    () =>
      (imageVersions.data?.items ?? []).map((v) => ({
        label: v.version,
        value: v.version,
        uri: v.uri || `${imageName}:${v.version}`,
      })),
    [imageVersions.data, imageName],
  );
  const selectedImage = imageVersionOptions.find((v) => v.value === imageVersion)?.uri ?? "";

  const poolOptions = useMemo(
    () =>
      (pools.data?.items ?? []).map((p) => ({
        label: p.description ? `${p.name} · ${p.description}` : p.name,
        value: p.name,
      })),
    [pools.data],
  );
  const unitOptions = useMemo(() => {
    const pool = (pools.data?.items ?? []).find((p) => p.name === poolName);
    return (pool?.units ?? []).map((u) => ({ value: u.name, title: u.name, desc: unitSpec(u) }));
  }, [pools.data, poolName]);

  const onPickModel = (v: string) => {
    setModelName(v);
    setModelVersion("");
  };
  const onPickImage = (v: string) => {
    setImageName(v);
    setImageVersion("");
  };
  const onPickPool = (v: string) => {
    setPoolName(v);
    setUnitName("");
  };

  const validPorts = toServicePorts(ports);
  const canSubmit =
    !!name.trim() &&
    !!modelName &&
    !!modelVersion &&
    !!selectedImage &&
    !!poolName &&
    !!unitName &&
    replicas >= 0 &&
    validPorts.length > 0 &&
    !create.isPending;

  const submit = () => {
    const route: sdk.MlServiceRoute = routeEnabled
      ? { enabled: true, ...(routePath.trim() ? { path: routePath.trim() } : {}) }
      : { enabled: false };
    const body: sdk.MlServiceCreateRequest = {
      name: name.trim(),
      modelName,
      modelVersion,
      image: selectedImage,
      poolName,
      unitName,
      replicas,
      ports: validPorts,
      ...(description.trim() ? { description: description.trim() } : {}),
      route,
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
          <div className="text-base font-semibold text-fg">{t("services.drawerNew")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">{t("services.drawerNewSub")}</div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={create.isPending} disabled={!canSubmit} onClick={submit}>
            {t("services.online")}
          </Button>
        </div>
      }
    >
      <Form layout="vertical" size="large">
        <FieldSection n={1} title={t("services.fsBasic")} />
        <Form.Item label={t("services.fName")} required extra={t("services.fNameHelp")}>
          <Input
            className="font-mono"
            placeholder={t("services.fNamePlaceholder")}
            value={name}
            onChange={(e) => setName(e.target.value)}
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

        <FieldSection n={2} title={t("services.fsModelImage")} />
        <div className="grid grid-cols-2 gap-x-4">
          <Form.Item label={t("services.fModel")} required>
            <Select
              value={modelName || undefined}
              onChange={onPickModel}
              loading={models.isLoading}
              placeholder={t("services.selectModel")}
              options={modelOptions}
            />
          </Form.Item>
          <Form.Item label={t("services.fModelVersion")} required>
            <Select
              value={modelVersion || undefined}
              onChange={setModelVersion}
              disabled={!modelName}
              loading={!!modelName && modelVersions.isLoading}
              placeholder={
                !modelName ? t("services.selectModelFirst") : modelVersionOptions.length === 0 ? t("services.noVersion") : t("services.selectVersion")
              }
              options={modelVersionOptions}
            />
          </Form.Item>
          <Form.Item label={t("services.fImage")} required>
            <Select
              value={imageName || undefined}
              onChange={onPickImage}
              loading={images.isLoading}
              placeholder={t("services.selectImage")}
              options={imageOptions}
            />
          </Form.Item>
          <Form.Item label={t("services.fImageVersion")} required>
            <Select
              value={imageVersion || undefined}
              onChange={setImageVersion}
              disabled={!imageName}
              loading={!!imageName && imageVersions.isLoading}
              placeholder={
                !imageName ? t("services.selectImageFirst") : imageVersionOptions.length === 0 ? t("services.noVersion") : t("services.selectVersion")
              }
              options={imageVersionOptions.map((v) => ({ label: v.label, value: v.value }))}
            />
          </Form.Item>
        </div>

        <FieldSection n={3} title={t("services.fsResource")} />
        <Form.Item label={t("services.fPool")} required>
          <Select
            value={poolName || undefined}
            onChange={onPickPool}
            loading={pools.isLoading}
            placeholder={t("services.selectPool")}
            options={poolOptions}
          />
        </Form.Item>
        <Form.Item label={t("services.fUnit")} required>
          {unitOptions.length === 0 ? (
            <span className="text-muted">{t("services.pickPoolFirst")}</span>
          ) : (
            <CardRadio options={unitOptions} value={unitName} onChange={setUnitName} />
          )}
        </Form.Item>
        <Form.Item label={t("services.fReplicas")} required>
          <InputNumber min={0} value={replicas} onChange={(v) => setReplicas(v ?? 0)} className="!w-40" />
        </Form.Item>

        <FieldSection n={4} title={t("services.fsPortRoute")} />
        <Form.Item label={t("services.fPorts")} required>
          <PortList rows={ports} setRows={setPorts} />
        </Form.Item>
        <div className="mb-4 flex items-center justify-between">
          <div>
            <div className="text-sm text-fg">{t("services.fRouteEnabled")}</div>
            <div className="text-xs text-muted">{t("services.fRouteHelp")}</div>
          </div>
          <Switch checked={routeEnabled} onChange={setRouteEnabled} />
        </div>
        <Form.Item label={t("services.fPath")}>
          <Input
            className="font-mono"
            placeholder={t("services.fPathPlaceholder")}
            value={routePath}
            disabled={!routeEnabled}
            onChange={(e) => setRoutePath(e.target.value)}
          />
        </Form.Item>
      </Form>
    </Drawer>
  );
}

// ── Edit drawer (display metadata only) ───────────────────────────────────────
function EditSvcDrawer({ row, onClose }: { row: SvcRow; onClose: () => void }) {
  const { t } = useTranslation();
  const [displayName, setDisplayName] = useState(row.displayName ?? "");
  const [description, setDescription] = useState(row.desc ?? "");
  const update = useApiMutation(
    (body: sdk.MlServicePatchRequest) => sdk.updateMlService({ path: { name: row.name }, body }),
    { invalidate: INVALIDATE, success: t("services.saved") },
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
            <span className="font-mono">{row.name}</span>
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
        <FieldSection n={1} title={t("services.fsBasic")} />
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
function ScaleDrawer({ row, onClose }: { row: SvcRow; onClose: () => void }) {
  const { t } = useTranslation();
  const [replicas, setReplicas] = useState<number>(row.replicaCount);
  const scale = useApiMutation(
    (body: sdk.MlServiceScaleRequest) => sdk.scaleMlService({ path: { name: row.name }, body }),
    { invalidate: INVALIDATE, success: t("services.scaleSubmitted") },
  );

  const valid = Number.isInteger(replicas) && replicas >= 0;
  const submit = () => scale.mutate({ replicas }, { onSuccess: onClose });

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
            <span className="font-mono">{row.name}</span>
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
          extra={t("services.scaleHint", { ready: row.replicas, unit: row.unit })}
        >
          <InputNumber min={0} value={replicas} onChange={(v) => setReplicas(v ?? 0)} className="!w-40" />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
