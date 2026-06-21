import { useMemo, useState, type Dispatch, type SetStateAction } from "react";
import { Link } from "react-router-dom";
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
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { FieldsetTitle } from "@/components/forms";
import { TableState } from "@/components/states";

type SvcStatus = "success" | "pending" | "stopped";

interface SvcRow {
  name: string;
  desc: string;
  status: SvcStatus;
  statusLabel: string;
  replicas: string;
  replicaCount: number;
  unit: string;
  url?: string;
  running: boolean; // true → 停止 action; false → 启动 action
  displayName?: string;
}

const PHASE_MAP: Record<string, { status: SvcStatus; label: string; running: boolean }> = {
  Ready: { status: "success", label: "就绪", running: true },
  Degraded: { status: "pending", label: "降级", running: true },
  Creating: { status: "pending", label: "部署中", running: true },
  Pending: { status: "pending", label: "部署中", running: true },
  Stopped: { status: "stopped", label: "已停止", running: false },
  Failed: { status: "stopped", label: "已停止", running: false },
  Deleted: { status: "stopped", label: "已停止", running: false },
  Deleting: { status: "stopped", label: "已停止", running: false },
};

type DrawerMode = "new" | "edit" | "scale";

const INVALIDATE = [["mlservices"]];

export default function Services() {
  const q = useServices();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; row?: SvcRow } | null>(null);

  const del = useApiMutation(
    (name: string) => sdk.deleteMlService({ path: { name } }),
    { invalidate: INVALIDATE, success: "服务已删除" },
  );
  const start = useApiMutation(
    (name: string) => sdk.startMlService({ path: { name } }),
    { invalidate: INVALIDATE, success: "服务启动中…" },
  );
  const stop = useApiMutation(
    (name: string) => sdk.stopMlService({ path: { name } }),
    { invalidate: INVALIDATE, success: "服务停止中…" },
  );

  const rows: SvcRow[] =
    q.data?.items?.map((s) => {
      const p = PHASE_MAP[s.phase ?? "Pending"] ?? PHASE_MAP.Pending;
      return {
        name: s.name,
        desc: s.description ?? s.displayName ?? "",
        status: p.status,
        statusLabel: p.label,
        replicas: `${s.readyReplicas ?? 0} / ${s.replicas ?? 0}`,
        replicaCount: s.replicas ?? 0,
        unit: `${s.poolName ?? "—"}/${s.unitName ?? "—"}`,
        url: s.accessUrl,
        running: p.running,
        displayName: s.displayName,
      };
    }) ?? [];

  const onDelete = (r: SvcRow) =>
    confirm({
      title: `删除服务 ${r.name}？`,
      desc: "将下线服务并回收副本，访问路由一并移除。该操作不可恢复。",
      okLabel: "确认删除",
      danger: true,
      onConfirm: () => del.mutate(r.name),
    });

  return (
    <main className="page">
      <div className="breadcrumb">
        <span>服务中心</span>
        <span className="sep">/</span>
        <span>在线服务</span>
      </div>
      <div className="page-head">
        <div>
          <h1>在线服务</h1>
          <p className="sub">常驻的在线推理服务，可对外提供访问并按需扩缩容。多版本灰度由「流量配置」承接。</p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setDrawer({ mode: "new" })}>
            <Icon name="plus" />
            新建服务
          </button>
        </div>
      </div>

      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="名称搜索" />
        </div>
        <select className="select">
          <option>状态：全部</option>
          <option>就绪</option>
          <option>降级</option>
          <option>已停止</option>
        </select>
        <select className="select">
          <option>资源池：全部</option>
          <option>gpu-h100</option>
          <option>gpu-l40s</option>
          <option>cpu-large</option>
        </select>
        <button className="btn btn-ghost">重置</button>
      </div>

      <div className="panel">
        <div className="table-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>名称</th>
                <th>状态</th>
                <th className="num-col">副本</th>
                <th>资源单元</th>
                <th>访问地址</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.name}>
                  <td>
                    <Link className="t-name mono" to={`/services/${r.name}`}>
                      {r.name}
                    </Link>
                    <div className="t-sub">{r.desc}</div>
                  </td>
                  <td>
                    <span className={`status status-${r.status}`}>
                      <span className="dot" />
                      {r.statusLabel}
                    </span>
                  </td>
                  <td className="num-col">{r.replicas}</td>
                  <td className="mono">{r.unit}</td>
                  <td>
                    {r.url ? (
                      <span className="mono muted" style={{ fontSize: 12 }}>
                        {r.url}
                      </span>
                    ) : (
                      <span className="muted">—</span>
                    )}
                  </td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="act"
                        title="编辑"
                        aria-label="编辑"
                        onClick={() => setDrawer({ mode: "edit", row: r })}
                      >
                        <EditIcon />
                      </button>
                      <button
                        className="act"
                        title="扩缩容"
                        aria-label="扩缩容"
                        onClick={() => setDrawer({ mode: "scale", row: r })}
                      >
                        <ScaleIcon />
                      </button>
                      {r.running ? (
                        <button
                          className="act"
                          title="停止"
                          aria-label="停止"
                          disabled={stop.isPending}
                          onClick={() => stop.mutate(r.name)}
                        >
                          <StopIcon />
                        </button>
                      ) : (
                        <button
                          className="act"
                          title="启动"
                          aria-label="启动"
                          disabled={start.isPending}
                          onClick={() => start.mutate(r.name)}
                        >
                          <PlayIcon />
                        </button>
                      )}
                      <button
                        className="act act-danger"
                        title="删除"
                        aria-label="删除"
                        onClick={() => onDelete(r)}
                      >
                        <TrashIcon />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              <TableState q={q} cols={6} isEmpty={rows.length === 0} />
            </tbody>
          </table>
        </div>
        <div className="pagination">
          <span>共 {rows.length} 个服务</span>
          <div className="pages">
            <span className="pg">‹</span>
            <span className="pg on">1</span>
            <span className="pg">›</span>
          </div>
          <span>每页 20 条</span>
        </div>
      </div>

      {drawer?.mode === "new" && <NewSvcDrawer onClose={() => setDrawer(null)} />}
      {drawer?.mode === "edit" && drawer.row && (
        <EditSvcDrawer row={drawer.row} onClose={() => setDrawer(null)} />
      )}
      {drawer?.mode === "scale" && drawer.row && (
        <ScaleDrawer row={drawer.row} onClose={() => setDrawer(null)} />
      )}
    </main>
  );
}

// ── one-off action glyphs not in the shared icon map ─────────────────────────
function EditIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M12 20h9" />
      <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4 12.5-12.5z" />
    </svg>
  );
}
function ScaleIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M15 3h6v6" />
      <path d="M9 21H3v-6" />
      <path d="M21 3l-7 7" />
      <path d="M3 21l7-7" />
    </svg>
  );
}
function StopIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <rect x="6" y="6" width="12" height="12" rx="1" />
    </svg>
  );
}
function PlayIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M6 4l14 8-14 8z" />
    </svg>
  );
}
function TrashIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M3 6h18" />
      <path d="M8 6V4h8v2" />
      <path d="M19 6l-1 14H6L5 6" />
      <path d="M10 11v6M14 11v6" />
    </svg>
  );
}

// ── Port rows (端口) — controlled repeatable widget feeding ServicePort[] ──────
interface PortRow {
  id: number;
  name: string;
  port: string;
}
let portSeq = 0;
function PortList({
  rows,
  setRows,
}: {
  rows: PortRow[];
  setRows: Dispatch<SetStateAction<PortRow[]>>;
}) {
  const add = () => setRows((r) => [...r, { id: ++portSeq, name: "", port: "" }]);
  const remove = (id: number) => setRows((r) => (r.length <= 1 ? r : r.filter((x) => x.id !== id)));
  const update = (id: number, patch: Partial<PortRow>) =>
    setRows((r) => r.map((x) => (x.id === id ? { ...x, ...patch } : x)));
  return (
    <>
      <div className="vol-list">
        {rows.map((row) => (
          <div className="vol-row" key={row.id}>
            <input
              className="input mono"
              value={row.name}
              onChange={(e) => update(row.id, { name: e.target.value })}
              placeholder="名称，如 http"
              aria-label="端口名"
              maxLength={15}
            />
            <input
              className="input mono"
              value={row.port}
              onChange={(e) => update(row.id, { port: e.target.value })}
              placeholder="端口号"
              aria-label="端口号"
              inputMode="numeric"
            />
            <button type="button" className="icon-btn" title="移除" onClick={() => remove(row.id)}>
              <Icon name="x" />
            </button>
          </div>
        ))}
      </div>
      <a className="link vol-add" role="button" tabIndex={0} onClick={add}>
        <Icon name="plus" />
        添加端口
      </a>
    </>
  );
}

// Build the ServicePort[] payload from the controlled port rows (drops blanks).
function toServicePorts(rows: PortRow[]): sdk.ServicePort[] {
  return rows
    .filter((r) => r.name.trim() && r.port.trim())
    .map((r) => ({ name: r.name.trim(), port: Number(r.port) }));
}

interface UnitPick {
  title: string;
  spec: string;
}

// Controlled radio-card grid (mirrors shared PickGrid styling) so the chosen
// resource unit name flows into the create payload.
function UnitPickGrid({
  options,
  value,
  onChange,
}: {
  options: UnitPick[];
  value: string;
  onChange: (v: string) => void;
}) {
  if (options.length === 0) {
    return <span className="muted">请先选择资源池</span>;
  }
  return (
    <div className="pick-grid">
      {options.map((o) => (
        <div
          key={o.title}
          className={"pick" + (o.title === value ? " on" : "")}
          onClick={() => onChange(o.title)}
        >
          <div className="p-title">{o.title}</div>
          <div className="p-spec">{o.spec}</div>
        </div>
      ))}
    </div>
  );
}

// Compact human spec from a ResourceUnit's requests map (best-effort display).
function unitSpec(u: sdk.ResourceUnit): string {
  const r = (u.requests ?? {}) as Record<string, string | undefined>;
  const parts: string[] = [];
  for (const [k, v] of Object.entries(r)) {
    if (v) parts.push(`${k}=${v}`);
  }
  return parts.join(" · ") || u.name;
}

function NewSvcDrawer({ onClose }: { onClose: () => void }) {
  const models = useModels();
  const images = useImages();
  const pools = useResourcePools();

  const create = useApiMutation(
    (body: sdk.MlServiceCreateRequest) => sdk.createMlService({ body }),
    { invalidate: INVALIDATE, success: "服务上线中…" },
  );

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  // Model is chosen by name; the version comes from that model's version list,
  // which the create LIST item (ArtifactDefinitionView) does not carry.
  const [modelName, setModelName] = useState("");
  const [modelVersion, setModelVersion] = useState("");
  // Image, like model, is chosen by name + version; the body's `image` string is
  // the selected version's pull URI.
  const [imageName, setImageName] = useState("");
  const [imageVersion, setImageVersion] = useState("");
  const [poolName, setPoolName] = useState("");
  const [unitName, setUnitName] = useState("");
  const [replicas, setReplicas] = useState("1");
  const [ports, setPorts] = useState<PortRow[]>(() => [{ id: ++portSeq, name: "http", port: "8000" }]);
  const [routeEnabled, setRouteEnabled] = useState(true);
  const [routePath, setRoutePath] = useState("");

  const modelVersions = useModelVersions(modelName);
  const modelOptions = useMemo(
    () =>
      (models.data?.items ?? []).map((m) => ({
        name: m.name,
        label: m.displayName || m.name,
      })),
    [models.data],
  );
  const versionOptions = useMemo(
    () => (modelVersions.data?.items ?? []).map((v) => v.version),
    [modelVersions.data],
  );

  // Reset version when the model changes so a stale version can't be submitted.
  const onPickModel = (v: string) => {
    setModelName(v);
    setModelVersion("");
  };
  const imageVersions = useImageVersions(imageName);
  const imageOptions = useMemo(
    () =>
      (images.data?.items ?? []).map((i) => ({
        name: i.name,
        label: i.displayName || i.name,
      })),
    [images.data],
  );
  // Map version string → pull URI so the submit body sends a usable image ref.
  const imageVersionOptions = useMemo(
    () =>
      (imageVersions.data?.items ?? []).map((v) => ({
        version: v.version,
        uri: v.uri || `${imageName}:${v.version}`,
      })),
    [imageVersions.data, imageName],
  );
  const selectedImage = imageVersionOptions.find((v) => v.version === imageVersion)?.uri ?? "";

  const onPickImage = (v: string) => {
    setImageName(v);
    setImageVersion("");
  };
  const unitOptions: UnitPick[] = useMemo(() => {
    const pool = (pools.data?.items ?? []).find((p) => p.name === poolName);
    return (pool?.units ?? []).map((u) => ({ title: u.name, spec: unitSpec(u) }));
  }, [pools.data, poolName]);

  const onPickPool = (v: string) => {
    setPoolName(v);
    setUnitName(""); // reset unit when pool changes
  };

  const validPorts = toServicePorts(ports);
  const canSubmit =
    !!name.trim() &&
    !!modelName &&
    !!modelVersion &&
    !!selectedImage &&
    !!poolName &&
    !!unitName &&
    Number(replicas) >= 0 &&
    validPorts.length > 0 &&
    !create.isPending;

  const submit = () => {
    const route: sdk.MlServiceRoute | undefined = routeEnabled
      ? { enabled: true, ...(routePath.trim() ? { path: routePath.trim() } : {}) }
      : { enabled: false };
    const body: sdk.MlServiceCreateRequest = {
      name: name.trim(),
      modelName,
      modelVersion,
      image: selectedImage,
      poolName,
      unitName,
      replicas: Number(replicas),
      ports: validPorts,
      ...(description.trim() ? { description: description.trim() } : {}),
      route,
    };
    create.mutate(body, { onSuccess: onClose });
  };

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="新建在线服务"
      sub="从已注册模型版本部署常驻推理服务"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" disabled={!canSubmit} onClick={submit}>
            {create.isPending ? "上线中…" : "上线服务"}
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>
            名称 <span className="req">*</span>
          </label>
          <input
            className="input mono"
            placeholder="svc-chat-api"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <span className="help">用于在列表与详情中展示</span>
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="用途说明（可选）"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
      </div>

      <FieldsetTitle n={2}>模型与镜像</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>
            模型 <span className="req">*</span>
          </label>
          <select className="input" value={modelName} onChange={(e) => onPickModel(e.target.value)}>
            <option value="">{models.isLoading ? "加载中…" : "请选择模型"}</option>
            {modelOptions.map((m) => (
              <option key={m.name} value={m.name}>
                {m.label}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label>
            模型版本 <span className="req">*</span>
          </label>
          <select
            className="input"
            value={modelVersion}
            disabled={!modelName}
            onChange={(e) => setModelVersion(e.target.value)}
          >
            <option value="">
              {!modelName
                ? "请先选择模型"
                : modelVersions.isLoading
                  ? "加载中…"
                  : versionOptions.length === 0
                    ? "无可用版本"
                    : "请选择版本"}
            </option>
            {versionOptions.map((v) => (
              <option key={v} value={v}>
                {v}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label>
            推理镜像 <span className="req">*</span>
          </label>
          <select className="input" value={imageName} onChange={(e) => onPickImage(e.target.value)}>
            <option value="">{images.isLoading ? "加载中…" : "请选择推理镜像"}</option>
            {imageOptions.map((i) => (
              <option key={i.name} value={i.name}>
                {i.label}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label>
            镜像版本 <span className="req">*</span>
          </label>
          <select
            className="input"
            value={imageVersion}
            disabled={!imageName}
            onChange={(e) => setImageVersion(e.target.value)}
          >
            <option value="">
              {!imageName
                ? "请先选择镜像"
                : imageVersions.isLoading
                  ? "加载中…"
                  : imageVersionOptions.length === 0
                    ? "无可用版本"
                    : "请选择版本"}
            </option>
            {imageVersionOptions.map((v) => (
              <option key={v.version} value={v.version}>
                {v.version}
              </option>
            ))}
          </select>
        </div>
      </div>

      <FieldsetTitle n={3}>资源选择</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            资源池 <span className="req">*</span>
          </label>
          <select className="input" value={poolName} onChange={(e) => onPickPool(e.target.value)}>
            <option value="">{pools.isLoading ? "加载中…" : "请选择资源池"}</option>
            {(pools.data?.items ?? []).map((p) => (
              <option key={p.name} value={p.name}>
                {p.name}
                {p.description ? ` · ${p.description}` : ""}
              </option>
            ))}
          </select>
        </div>
      </div>
      <div className="field" style={{ marginTop: "var(--space-4)" }}>
        <label>
          资源单元 <span className="req">*</span>
        </label>
        <UnitPickGrid options={unitOptions} value={unitName} onChange={setUnitName} />
      </div>
      <div className="form-grid" style={{ marginTop: "var(--space-4)" }}>
        <div className="field">
          <label>
            副本数 <span className="req">*</span>
          </label>
          <input
            className="input num mono"
            inputMode="numeric"
            value={replicas}
            onChange={(e) => setReplicas(e.target.value)}
          />
        </div>
      </div>

      <FieldsetTitle n={4}>端口与路由</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            端口 <span className="req">*</span>
          </label>
          <PortList rows={ports} setRows={setPorts} />
        </div>
      </div>
      <div className="form-grid">
        <div
          className="field full"
          style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}
        >
          <div>
            <label style={{ margin: 0 }}>启用对外路由</label>
            <span className="help">关闭后仅集群内可访问</span>
          </div>
          <RouteToggle on={routeEnabled} setOn={setRouteEnabled} />
        </div>
        <div className="field">
          <label>Path</label>
          <input
            className="input mono"
            placeholder="/services/llm-lab/chat/"
            value={routePath}
            disabled={!routeEnabled}
            onChange={(e) => setRoutePath(e.target.value)}
          />
        </div>
      </div>
    </Drawer>
  );
}

function RouteToggle({ on, setOn }: { on: boolean; setOn: (v: boolean) => void }) {
  return (
    <button
      className={"toggle" + (on ? " on" : "")}
      aria-label="启用对外路由"
      onClick={() => setOn(!on)}
    />
  );
}

// Edit only patches display metadata (the API exposes MlServicePatchRequest:
// description / displayName); replica + spec changes go through scale / recreate.
function EditSvcDrawer({ row, onClose }: { row: SvcRow; onClose: () => void }) {
  const [displayName, setDisplayName] = useState(row.displayName ?? "");
  const [description, setDescription] = useState(row.desc ?? "");
  const update = useApiMutation(
    (body: sdk.MlServicePatchRequest) => sdk.updateMlService({ path: { name: row.name }, body }),
    { invalidate: INVALIDATE, success: "服务配置已保存" },
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
      wide
      onClose={onClose}
      title="编辑服务"
      sub={<span className="mono">{row.name}</span>}
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" disabled={update.isPending} onClick={submit}>
            {update.isPending ? "保存中…" : "保存"}
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息</FieldsetTitle>
      <p className="muted" style={{ fontSize: 13, marginBottom: "var(--space-4)" }}>
        仅可修改展示信息；副本数请使用「扩缩容」，模型与资源规格不可在线变更。
      </p>
      <div className="form-grid">
        <div className="field">
          <label>显示名称</label>
          <input
            className="input"
            placeholder="（可选）"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="用途说明（可选）"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
      </div>
    </Drawer>
  );
}

function ScaleDrawer({ row, onClose }: { row: SvcRow; onClose: () => void }) {
  const [replicas, setReplicas] = useState(String(row.replicaCount));
  const scale = useApiMutation(
    (body: sdk.MlServiceScaleRequest) => sdk.scaleMlService({ path: { name: row.name }, body }),
    { invalidate: INVALIDATE, success: "扩缩容请求已提交" },
  );

  const n = Number(replicas);
  const valid = replicas.trim() !== "" && Number.isInteger(n) && n >= 0;

  const submit = () => scale.mutate({ replicas: n }, { onSuccess: onClose });

  return (
    <Drawer
      open
      onClose={onClose}
      title="扩缩容"
      sub={<span className="mono">{row.name}</span>}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            disabled={!valid || scale.isPending}
            onClick={submit}
          >
            {scale.isPending ? "应用中…" : "应用"}
          </button>
        </>
      }
    >
      <p className="muted" style={{ fontSize: 13, marginBottom: "var(--space-5)" }}>
        扩缩容仅修改副本数。
      </p>
      <div className="field">
        <label>目标副本数</label>
        <input
          className="input num"
          inputMode="numeric"
          value={replicas}
          onChange={(e) => setReplicas(e.target.value)}
        />
        <span className="help">当前 {row.replicas} 就绪 · 资源单元 {row.unit}</span>
      </div>
    </Drawer>
  );
}
