import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useModels } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import type { RemoteSourceKind } from "@/api/generated";
import { TableState, BlockState } from "@/components/states";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { Tabs } from "@/components/Tabs";
import { FieldsetTitle } from "@/components/forms";

interface ModelRow {
  name: string;
  desc: string;
  icon: "model" | "shield" | "graph";
  framework: string;
  latest: string;
  versions: number;
  tags: string[];
  updated: string; // card "更新 …" label
  updatedShort: string; // list "更新时间" cell
  canUpload: boolean; // whether the row offers 上传新版本 (externally-sourced models do not)
}

// One-off glyphs from the prototype card icons (not in the shared icon map).
function CardIcon({ name }: { name: ModelRow["icon"] }) {
  if (name === "shield") {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z" />
      </svg>
    );
  }
  if (name === "graph") {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="6" cy="12" r="2.5" />
        <circle cx="18" cy="6" r="2.5" />
        <circle cx="18" cy="18" r="2.5" />
        <path d="M8.2 10.8 15.8 7.2M8.2 13.2l7.6 3.6" />
      </svg>
    );
  }
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
      <path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z" />
    </svg>
  );
}

function UploadGlyph() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
      <path d="M17 8l-5-5-5 5" />
      <path d="M12 3v12" />
    </svg>
  );
}

const TrashIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
    <path d="M3 6h18" />
    <path d="M8 6V4h8v2" />
    <path d="M19 6l-1 14H6L5 6" />
    <path d="M10 11v6M14 11v6" />
  </svg>
);

// Drawer state carries the active model name so version-scoped operations
// (list / upload / delete) address the right model definition.
type DrawerKind =
  | { kind: "ver"; model: string; desc: string; framework: string }
  | { kind: "pull"; model: string; version: string }
  | { kind: "newModel" }
  | { kind: "up"; model: string };

export default function Models() {
  const q = useModels();
  const { confirm } = useUI();
  const { tenant } = useApp();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [query, setQuery] = useState("");
  const [drawer, setDrawer] = useState<DrawerKind | null>(null);

  const delModel = useApiMutation(
    (name: string) => sdk.deleteModelDefinition({ path: { tenant, name } }),
    { invalidate: [["models"]], success: "模型已删除" },
  );

  const rows: ModelRow[] =
    q.data?.items?.map((m) => ({
      name: m.name,
      desc: m.description ?? m.displayName ?? "",
      icon: "model" as const,
      framework: (m.labels?.framework as string) ?? "—",
      latest: "—",
      versions: 0,
      tags: [],
      updated: "刚刚",
      updatedShort: "刚刚",
      canUpload: true,
    })) ?? [];

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return rows;
    return rows.filter((r) => r.name.toLowerCase().includes(needle) || r.desc.toLowerCase().includes(needle));
  }, [rows, query]);

  const openVer = (r: ModelRow) => setDrawer({ kind: "ver", model: r.name, desc: r.desc, framework: r.framework });

  const onDeleteModel = (r: ModelRow) => {
    confirm({
      title: `确定删除模型 ${r.name}？`,
      desc: "删除后该模型的所有版本权重将一并移除，且不可恢复。",
      okLabel: "确认删除",
      danger: true,
      onConfirm: () => delModel.mutate(r.name),
    });
  };

  return (
    <main className="page">
      <div className="breadcrumb">
        <span>资产中心</span>
        <span className="sep">/</span>
        <span>模型仓</span>
      </div>
      <div className="page-head">
        <div>
          <h1>模型仓</h1>
          <p className="sub">
            集中管理模型版本、状态、来源与使用记录，实现模型资产的规范化沉淀。支持模型快速检索、发布与复用，加速从研发到应用的转化。
          </p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setDrawer({ kind: "newModel" })}>
            <Icon name="plus" />
            新建模型
          </button>
        </div>
      </div>

      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="搜索名称 / 描述" value={query} onChange={(e) => setQuery(e.target.value)} />
        </div>
        <div className="grow" />
        <div className="segmented">
          <button className={view === "cards" ? "on" : ""} onClick={() => setView("cards")}>
            ▦ 卡片
          </button>
          <button className={view === "list" ? "on" : ""} onClick={() => setView("list")}>
            ☰ 列表
          </button>
        </div>
      </div>

      {view === "cards" && (
        <div>
          <div className="art-cards">
            {filtered.map((r) => (
              <div className="art-card" key={r.name} onClick={() => openVer(r)}>
                <div className="ac-top">
                  <div className="ac-ico">
                    <CardIcon name={r.icon} />
                  </div>
                  <div>
                    <div className="ac-name">{r.name}</div>
                  </div>
                  <div className="grow" />
                  <button
                    className="act act-danger"
                    title="删除"
                    aria-label="删除"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDeleteModel(r);
                    }}
                  >
                    <TrashIcon />
                  </button>
                </div>
                <p className="ac-desc">{r.desc}</p>
                <div className="ac-foot">
                  <span className="mono">
                    {r.latest} · {r.versions} 版本
                  </span>
                  <span>{r.updated}</span>
                </div>
              </div>
            ))}
            <BlockState q={q} isEmpty={filtered.length === 0} />
          </div>
          <div className="pagination" style={{ marginTop: "var(--space-4)" }}>
            <span>共 {filtered.length} 个</span>
            <div className="pages">
              <span className="pg">‹</span>
              <span className="pg on">1</span>
              <span className="pg">›</span>
            </div>
            <span>每页 20 条</span>
          </div>
        </div>
      )}

      {view === "list" && (
        <div>
          <div className="panel">
            <div className="table-wrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>名称</th>
                    <th>框架</th>
                    <th>最新版本</th>
                    <th className="num-col">版本数</th>
                    <th>标签</th>
                    <th>更新时间</th>
                    <th style={{ textAlign: "right" }}>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((r) => (
                    <tr key={r.name} onClick={() => openVer(r)}>
                      <td>
                        <span className="t-name mono">{r.name}</span>
                        <div className="t-sub">{r.desc}</div>
                      </td>
                      <td>
                        <span className="badge badge-neutral">{r.framework}</span>
                      </td>
                      <td className="mono">{r.latest}</td>
                      <td className="num-col">{r.versions}</td>
                      <td>
                        {r.tags.map((t) => (
                          <span className="tag" key={t}>
                            {t}
                          </span>
                        ))}
                      </td>
                      <td className="muted">{r.updatedShort}</td>
                      <td>
                        <div className="row-actions">
                          {r.canUpload ? (
                            <button
                              className="act"
                              title="上传新版本"
                              aria-label="上传新版本"
                              onClick={(e) => {
                                e.stopPropagation();
                                setDrawer({ kind: "up", model: r.name });
                              }}
                            >
                              <UploadGlyph />
                            </button>
                          ) : null}
                          <button
                            className="act act-danger"
                            title="删除"
                            aria-label="删除"
                            onClick={(e) => {
                              e.stopPropagation();
                              onDeleteModel(r);
                            }}
                          >
                            <TrashIcon />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  <TableState q={q} cols={7} isEmpty={filtered.length === 0} />
                </tbody>
              </table>
            </div>
            <div className="pagination">
              <span>共 {filtered.length} 个</span>
              <div className="pages">
                <span className="pg">‹</span>
                <span className="pg on">1</span>
                <span className="pg">›</span>
              </div>
              <span>每页 20 条</span>
            </div>
          </div>
        </div>
      )}

      {drawer?.kind === "ver" && (
        <VerDrawer
          model={drawer.model}
          desc={drawer.desc}
          framework={drawer.framework}
          onClose={() => setDrawer(null)}
          onPull={(version) => setDrawer({ kind: "pull", model: drawer.model, version })}
          onUpload={() => setDrawer({ kind: "up", model: drawer.model })}
        />
      )}
      {drawer?.kind === "pull" && (
        <PullDrawer model={drawer.model} version={drawer.version} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "newModel" && <NewModelDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "up" && <UploadDrawer model={drawer.model} onClose={() => setDrawer(null)} />}
    </main>
  );
}

function statusMeta(status: string): { cls: "success" | "pending"; label: string; pending: boolean } {
  switch (status) {
    case "Ready":
      return { cls: "success", label: "就绪", pending: false };
    case "Uploading":
      return { cls: "pending", label: "上传中", pending: true };
    case "Failed":
      return { cls: "pending", label: "失败", pending: false };
    default:
      return { cls: "pending", label: status, pending: status === "Uploading" };
  }
}

function VerDrawer({
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
  const { tenant } = useApp();
  const { toast, confirm } = useUI();
  const [verQuery, setVerQuery] = useState("");

  const versQ = useQuery({
    queryKey: ["modelVersions", model, tenant],
    enabled: tenant !== "",
    queryFn: async () => {
      const { data, error } = await sdk.listModelVersions({ path: { tenant, name: model } });
      if (error) throw error;
      return data;
    },
  });

  const delVer = useApiMutation(
    (version: string) => sdk.deleteModel({ path: { tenant, name: model, version } }),
    { invalidate: [["modelVersions", model], ["models"]], success: "版本已删除" },
  );

  const items = versQ.data?.items ?? [];
  const filtered = items.filter((v) => {
    const needle = verQuery.trim().toLowerCase();
    if (!needle) return true;
    return v.version.toLowerCase().includes(needle) || (v.description ?? "").toLowerCase().includes(needle);
  });

  const onDeleteVer = (version: string) => {
    confirm({
      title: `确定删除版本 ${version}？`,
      desc: "删除后该版本权重将不可恢复。",
      okLabel: "确认删除",
      danger: true,
      onConfirm: () => delVer.mutate(version),
    });
  };

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title={<span className="mono">{model}</span>}
      sub={`${desc || "模型权重"} · ${framework}`}
    >
      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input
            placeholder="搜索版本名称 / 描述"
            value={verQuery}
            onChange={(e) => setVerQuery(e.target.value)}
          />
        </div>
        <div className="grow" />
        <button className="btn btn-sm btn-primary" onClick={onUpload}>
          + 上传新版本
        </button>
      </div>
      <div className="ver-list">
        {filtered.map((v) => {
          const meta = statusMeta(v.status);
          return (
            <div className="ver-item" key={v.version}>
              <div className="ver-top">
                <span className="ver-name">{v.version}</span>
                <span className={"status status-" + meta.cls}>
                  <span className="dot" />
                  {meta.label}
                </span>
                {v.source && <span className="badge badge-neutral ver-src">{v.source}</span>}
                <div className="ver-actions">
                  {!meta.pending && (
                    <>
                      <button className="act" aria-label="拉取命令" onClick={() => onPull(v.version)} />
                      <button
                        className="act act-danger"
                        aria-label="删除"
                        onClick={() => onDeleteVer(v.version)}
                      />
                    </>
                  )}
                </div>
              </div>
              <div className="ver-desc">{v.description ?? ""}</div>
              <div className="ver-meta">
                <span className={"ver-addr" + (meta.pending ? " muted" : "")}>{v.uri ?? "地址生成中…"}</span>
                {v.uri && (
                  <button
                    className="act ver-copy"
                    aria-label="复制"
                    onClick={() => {
                      void navigator.clipboard?.writeText(v.uri ?? "");
                      toast("地址已复制");
                    }}
                  />
                )}
                <span className="ver-by">{v.owner ?? ""}</span>
              </div>
            </div>
          );
        })}
        <BlockState q={versQ} isEmpty={filtered.length === 0} />
      </div>
    </Drawer>
  );
}

function PullDrawer({ model, version, onClose }: { model: string; version: string; onClose: () => void }) {
  const { tenant } = useApp();
  const { toast } = useUI();
  const resolveQ = useQuery({
    queryKey: ["modelResolve", model, version, tenant],
    enabled: tenant !== "",
    queryFn: async () => {
      const { data, error } = await sdk.resolveModel({ path: { tenant, name: model, version } });
      if (error) throw error;
      return data;
    },
  });

  const cmd = resolveQ.data?.uri ? `docker pull ${resolveQ.data.uri}` : "解析拉取地址中…";

  return (
    <Drawer
      open
      onClose={onClose}
      title="拉取命令"
      sub={<span className="mono">{`${model}@${version}`}</span>}
      footer={
        <button className="btn btn-primary" onClick={onClose}>
          完成
        </button>
      }
    >
      <p className="muted" style={{ fontSize: 13, marginBottom: 12 }}>
        模型存储于 OCI（zot），用以下命令拉取。临时凭证有效期 1 小时。
      </p>
      <pre className="logbox" style={{ maxHeight: "none" }}>
        {cmd}
      </pre>
      <button
        className="btn"
        style={{ marginTop: 14 }}
        disabled={!resolveQ.data?.uri}
        onClick={() => {
          void navigator.clipboard?.writeText(cmd);
          toast("命令已复制到剪贴板");
        }}
      >
        复制命令
      </button>
    </Drawer>
  );
}

// Tag-chip multi-select: clicking a preset toggles it; custom values added via
// the inline input. Selected set is collected into the create payload's labels.
function ChipSelect({
  label,
  options,
  selected,
  onToggle,
  onAdd,
}: {
  label: string;
  options: string[];
  selected: Set<string>;
  onToggle: (v: string) => void;
  onAdd: (v: string) => void;
}) {
  const [draft, setDraft] = useState("");
  const all = Array.from(new Set([...options, ...Array.from(selected)]));
  return (
    <div className="tag-group">
      <span className="tg-label">{label}</span>
      <div className="chip-row">
        {all.map((o) => (
          <span
            key={o}
            className={"tag-opt" + (selected.has(o) ? " on" : "")}
            role="button"
            tabIndex={0}
            onClick={() => onToggle(o)}
          >
            {o}
          </span>
        ))}
        <input
          className="tag-add-input"
          placeholder="自定义，回车添加"
          aria-label={`添加自定义 ${label}`}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && draft.trim()) {
              e.preventDefault();
              onAdd(draft.trim());
              setDraft("");
            }
          }}
        />
      </div>
    </div>
  );
}

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

function NewModelDrawer({ onClose }: { onClose: () => void }) {
  const { tenant } = useApp();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [params, setParams] = useState("");
  const [tasks, setTasks] = useState<Set<string>>(new Set());
  const [frameworks, setFrameworks] = useState<Set<string>>(new Set());
  const [customTags, setCustomTags] = useState<Record<string, string>>({});
  const [ctKey, setCtKey] = useState("");
  const [ctVal, setCtVal] = useState("");

  const create = useApiMutation(
    (body: sdk.ArtifactDefinitionCreateInput) =>
      sdk.createModelDefinition({ path: { tenant, name: body.name }, body }),
    { invalidate: [["models"]], success: "模型已创建，可在版本列表上传权重" },
  );

  const toggle = (set: Set<string>, setter: (s: Set<string>) => void, v: string) => {
    const next = new Set(set);
    if (next.has(v)) next.delete(v);
    else next.add(v);
    setter(next);
  };

  const submit = () => {
    // labels carry the structured taxonomy (framework / tasks / params); free-form
    // key:value pairs go to annotations. Empty groups are omitted entirely.
    const labels: Record<string, string> = {};
    const framework = Array.from(frameworks)[0];
    if (framework) labels.framework = framework;
    if (tasks.size) labels.tasks = Array.from(tasks).join(",");
    if (params.trim()) labels.params = params.trim();

    create.mutate(
      {
        name: name.trim(),
        description: description.trim() || undefined,
        labels: Object.keys(labels).length ? labels : undefined,
        annotations: Object.keys(customTags).length ? customTags : undefined,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="新建模型"
      sub="先创建模型条目，再上传具体版本权重"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" disabled={!name.trim() || create.isPending} onClick={submit}>
            {create.isPending ? "创建中…" : "创建模型"}
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            模型名 <span className="req">*</span>
          </label>
          <input
            className="input mono"
            placeholder="my-llm-model（仅英文、数字与连字符）"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="简要说明模型用途、训练数据与适用场景"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
      </div>

      <FieldsetTitle n={2}>标签</FieldsetTitle>

      <ChipSelect
        label="Tasks"
        options={TASK_OPTIONS}
        selected={tasks}
        onToggle={(v) => toggle(tasks, setTasks, v)}
        onAdd={(v) => setTasks(new Set(tasks).add(v))}
      />

      <div className="tag-group">
        <span className="tg-label">Parameters</span>
        <div className="param-slider">
          <div className="ps-row">
            <input
              type="range"
              className="range"
              min="0"
              max="8"
              step="1"
              aria-label="参数量"
              onChange={(e) => {
                const ticks = ["<1B", "1B", "3B", "7B", "8B", "13B", "32B", "70B", ">100B"];
                setParams(ticks[Number(e.target.value)] ?? "");
              }}
            />
            <input
              className="input mono ps-input"
              aria-label="参数量"
              placeholder="7B"
              value={params}
              onChange={(e) => setParams(e.target.value)}
            />
          </div>
          <div className="ps-ticks">
            <span>&lt;1B</span>
            <span>1B</span>
            <span>3B</span>
            <span>7B</span>
            <span>8B</span>
            <span>13B</span>
            <span>32B</span>
            <span>70B</span>
            <span>&gt;100B</span>
          </div>
        </div>
      </div>

      <ChipSelect
        label="Framework"
        options={FRAMEWORK_OPTIONS}
        selected={frameworks}
        onToggle={(v) => toggle(frameworks, setFrameworks, v)}
        onAdd={(v) => setFrameworks(new Set(frameworks).add(v))}
      />

      <div className="tag-group">
        <span className="tg-label">自定义标签</span>
        <div className="custom-tags">
          <div className="chip-row">
            {Object.entries(customTags).map(([k, v]) => (
              <span
                key={k}
                className="tag mono"
                role="button"
                tabIndex={0}
                onClick={() => {
                  const next = { ...customTags };
                  delete next[k];
                  setCustomTags(next);
                }}
              >
                {k}:{v} ✕
              </span>
            ))}
          </div>
          <div className="cta-input">
            <input
              className="input mono"
              placeholder="键，如 license"
              value={ctKey}
              onChange={(e) => setCtKey(e.target.value)}
            />
            <span className="cta-sep mono">:</span>
            <input
              className="input mono"
              placeholder="值，如 apache-2.0"
              value={ctVal}
              onChange={(e) => setCtVal(e.target.value)}
            />
            <button
              className="btn btn-sm"
              type="button"
              onClick={() => {
                if (ctKey.trim()) {
                  setCustomTags({ ...customTags, [ctKey.trim()]: ctVal.trim() });
                  setCtKey("");
                  setCtVal("");
                }
              }}
            >
              添加
            </button>
          </div>
        </div>
      </div>
    </Drawer>
  );
}

function UploadDrawer({ model, onClose }: { model: string; onClose: () => void }) {
  const { tenant } = useApp();
  const [version, setVersion] = useState("");
  const [description, setDescription] = useState("");
  const [remoteKind, setRemoteKind] = useState<RemoteSourceKind>("s3");
  const [remoteUri, setRemoteUri] = useState("");
  // "web" → webUpload (client cannot push bytes here — see report); "remote" →
  // external source registration via remoteUri/remoteSourceKind.
  const [method, setMethod] = useState<"web" | "remote" | "oras">("web");

  const initiate = useApiMutation(
    (body: sdk.ModelInitiateRequest) => sdk.initiateModel({ path: { tenant, name: model }, body }),
    { invalidate: [["modelVersions", model], ["models"]], success: "已提交，版本正在上传 / 拉取" },
  );

  const submit = () => {
    const isExternal = method === "remote";
    initiate.mutate(
      {
        version: version.trim(),
        description: description.trim() || undefined,
        source: method === "oras" ? "oras" : isExternal ? "external" : "webUpload",
        remoteSourceKind: isExternal ? remoteKind : undefined,
        remoteUri: isExternal && remoteUri.trim() ? remoteUri.trim() : undefined,
      },
      { onSuccess: onClose },
    );
  };

  const disabled =
    !version.trim() || initiate.isPending || (method === "remote" && !remoteUri.trim());

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="上传新版本"
      sub="向已有模型推送新版本权重"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" disabled={disabled} onClick={submit}>
            {initiate.isPending ? "提交中…" : "提交"}
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>模型</label>
          <input className="input mono" value={model} disabled />
        </div>
        <div className="field">
          <label>
            版本号 <span className="req">*</span>
          </label>
          <input
            className="input mono"
            placeholder="v5 / 1.5.0 / 2026-06"
            value={version}
            onChange={(e) => setVersion(e.target.value)}
          />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="本次更新内容，如：扩充中文 SFT 数据、修复输出截断"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
      </div>

      <FieldsetTitle n={2}>上传方式</FieldsetTitle>
      <Tabs
        defaultKey="web"
        tabs={[
          {
            key: "web",
            label: "通过 Web 上传",
            content: (
              <>
                <WebMethodSetter onShow={() => setMethod("web")} />
                <label className="dropzone">
                  <input type="file" multiple hidden />
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <path d="M17 8l-5-5-5 5" />
                    <path d="M12 3v12" />
                  </svg>
                  <div className="dz-title">
                    拖拽文件到此处，或 <span className="dz-link">点击选择</span>
                  </div>
                  <div className="dz-hint">支持权重与配置文件（.safetensors / .bin / .json / .model 等），单文件最大 50GB</div>
                </label>
                <div className="dz-files" hidden />
              </>
            ),
          },
          {
            key: "remote",
            label: "添加外部模型",
            content: (
              <div className="form-grid">
                <WebMethodSetter onShow={() => setMethod("remote")} />
                <div className="field full">
                  <label>
                    存储类型 <span className="req">*</span>
                  </label>
                  <select
                    className="input"
                    value={remoteKind}
                    onChange={(e) => setRemoteKind(e.target.value as RemoteSourceKind)}
                  >
                    <option value="s3">S3 / S3 兼容（MinIO）</option>
                    <option value="oci">OCI Registry</option>
                    <option value="http">HTTP(S) URL</option>
                    <option value="hf">HuggingFace Hub</option>
                    <option value="custom">自定义</option>
                  </select>
                </div>
                <div className="field full">
                  <label>
                    地址 <span className="req">*</span>
                  </label>
                  <input
                    className="input mono"
                    placeholder="s3://bucket/prefix"
                    value={remoteUri}
                    onChange={(e) => setRemoteUri(e.target.value)}
                  />
                </div>
              </div>
            ),
          },
          {
            key: "oras",
            label: "使用 Oras 推送",
            content: (
              <>
                <WebMethodSetter onShow={() => setMethod("oras")} />
                <p className="help" style={{ marginBottom: 14 }}>
                  使用 <b>ORAS</b> 将本地模型目录作为 OCI 制品直接推送到模型仓，适合大体积权重与 CI 流水线。
                </p>

                <div className="oras-step">
                  <div className="os-head">1 · 下载 ORAS 工具</div>
                  <pre className="logbox" style={{ maxHeight: "none" }}>{`# Linux x86_64（其他平台见官方文档）
curl -LO https://github.com/oras-project/oras/releases/download/v1.2.0/oras_1.2.0_linux_amd64.tar.gz
tar -xzf oras_1.2.0_linux_amd64.tar.gz oras
sudo mv oras /usr/local/bin/ && oras version`}</pre>
                  <div className="chip-row" style={{ marginTop: 10, alignItems: "center" }}>
                    <a className="link" href="https://oras.land/docs/installation" target="_blank" rel="noopener">
                      其他平台安装文档 ↗
                    </a>
                  </div>
                </div>

                <div className="oras-step">
                  <div className="os-head">2 · 登录并推送模型</div>
                  <pre className="logbox" style={{ maxHeight: "none" }}>{`# 1. 登录模型仓（临时凭证有效期 1h）
oras login zot.axisml.internal -u <用户名> -p <token>

# 2. 进入本地模型目录，推送为指定版本
cd ./${model}
oras push zot.axisml.internal/${tenant}/${model}:${version || "v5"} \\
  --artifact-type application/vnd.axisml.model.v1 \\
  ./*:application/octet-stream`}</pre>
                </div>
              </>
            ),
          },
        ]}
      />
    </Drawer>
  );
}

// Reports which upload tab is active to the parent so the submit payload reflects
// the chosen source. Renders nothing; the Tabs component only mounts the active
// pane's content, so this fires whenever its tab becomes visible.
function WebMethodSetter({ onShow }: { onShow: () => void }) {
  useEffect(() => {
    onShow();
  }, []);
  return null;
}
