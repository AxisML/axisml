import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useImages } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { Tabs } from "@/components/Tabs";
import { FieldsetTitle } from "@/components/forms";
import { TableState, BlockState } from "@/components/states";

interface ImageRow {
  name: string;
  desc: string;
  icon: "box" | "bolt" | "code" | "chart";
  purpose: string;
  latest: string;
  versions: number;
  updated: string; // card "更新 …" label
  updatedShort: string; // list "更新时间" cell
}

// One-off card glyphs from the prototype (not in the shared icon map).
function CardIcon({ name }: { name: ImageRow["icon"] }) {
  if (name === "bolt") {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <path d="M13 2 4 14h7l-1 8 9-12h-7l1-8z" />
      </svg>
    );
  }
  if (name === "code") {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <path d="m8 18-6-6 6-6M16 6l6 6-6 6" />
      </svg>
    );
  }
  if (name === "chart") {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <path d="M3 3v18h18" />
        <path d="M8 17v-5M13 17V8M18 17v-8" />
      </svg>
    );
  }
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
      <rect x="5" y="5" width="14" height="14" rx="2" />
      <rect x="9" y="9" width="6" height="6" rx="1" />
      <path d="M9 2v3M15 2v3M9 19v3M15 19v3M2 9h3M2 15h3M19 9h3M19 15h3" />
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

const PURPOSE_OPTIONS = [
  { value: "training", label: "训练镜像" },
  { value: "inference", label: "推理镜像" },
  { value: "workspace", label: "评估镜像" },
  { value: "custom", label: "自定义" },
] as const;

type DrawerKind =
  | { kind: "ver"; image: string; desc: string }
  | { kind: "pull"; image: string; version: string; uri: string }
  | { kind: "newImg" }
  | { kind: "addVer"; image: string };

export default function Images() {
  const q = useImages();
  const { tenant } = useApp();
  const { confirm } = useUI();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [query, setQuery] = useState("");
  const [drawer, setDrawer] = useState<DrawerKind | null>(null);

  const delDef = useApiMutation(
    (name: string) => sdk.deleteImageDefinition({ path: { tenant, name } }),
    { invalidate: [["images"]], success: "镜像已删除" },
  );

  const rows: ImageRow[] =
    q.data?.items?.map((m) => ({
      name: m.name,
      desc: m.description ?? m.displayName ?? "",
      icon: "box" as const,
      purpose: (m.labels?.purpose as string) ?? "—",
      latest: "—",
      versions: 0,
      updated: "刚刚",
      updatedShort: "刚刚",
    })) ?? [];

  const filtered = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) return rows;
    return rows.filter((r) => r.name.toLowerCase().includes(term) || r.desc.toLowerCase().includes(term));
  }, [rows, query]);

  const onDelete = (r: ImageRow) => {
    confirm({
      title: `确定删除镜像 ${r.name}？`,
      desc: "删除后该镜像下所有版本将一并移除，且不可恢复。",
      okLabel: "确认删除",
      danger: true,
      onConfirm: () => delDef.mutate(r.name),
    });
  };

  return (
    <main className="page">
      <div className="breadcrumb">
        <span>资产中心</span>
        <span className="sep">/</span>
        <span>镜像仓</span>
      </div>
      <div className="page-head">
        <div>
          <h1>镜像仓</h1>
          <p className="sub">
            统一维护训练、推理等运行环境镜像，保障任务环境一致性与可复现性。支持镜像版本管理与快速分发，降低环境配置和迁移成本。
          </p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setDrawer({ kind: "newImg" })}>
            <Icon name="plus" />
            新建镜像
          </button>
        </div>
      </div>

      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="搜索名称 / 描述" value={query} onChange={(e) => setQuery(e.target.value)} />
        </div>
        <select className="select">
          <option>用途：全部</option>
          <option>training</option>
          <option>inference</option>
          <option>dev</option>
        </select>
        <button className="btn btn-ghost">重置</button>
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
              <div className="art-card" key={r.name} onClick={() => setDrawer({ kind: "ver", image: r.name, desc: r.desc })}>
                <div className="ac-top">
                  <div className="ac-ico">
                    <CardIcon name={r.icon} />
                  </div>
                  <div>
                    <div className="ac-name">{r.name}</div>
                  </div>
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
                    <th>用途</th>
                    <th>最新版本</th>
                    <th className="num-col">版本数</th>
                    <th>更新时间</th>
                    <th style={{ textAlign: "right" }}>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((r) => (
                    <tr key={r.name} onClick={() => setDrawer({ kind: "ver", image: r.name, desc: r.desc })}>
                      <td>
                        <span className="t-name mono">{r.name}</span>
                        <div className="t-sub">{r.desc}</div>
                      </td>
                      <td>
                        <span className="badge badge-neutral">{r.purpose}</span>
                      </td>
                      <td className="mono">{r.latest}</td>
                      <td className="num-col">{r.versions}</td>
                      <td className="muted">{r.updatedShort}</td>
                      <td>
                        <div className="row-actions">
                          <button
                            className="act"
                            aria-label="添加版本"
                            onClick={(e) => {
                              e.stopPropagation();
                              setDrawer({ kind: "addVer", image: r.name });
                            }}
                          >
                            <UploadGlyph />
                          </button>
                          <button
                            className="act act-danger"
                            aria-label="删除"
                            onClick={(e) => {
                              e.stopPropagation();
                              onDelete(r);
                            }}
                          />
                        </div>
                      </td>
                    </tr>
                  ))}
                  <TableState q={q} cols={6} isEmpty={filtered.length === 0} />
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
          image={drawer.image}
          desc={drawer.desc}
          onClose={() => setDrawer(null)}
          onPull={(version, uri) => setDrawer({ kind: "pull", image: drawer.image, version, uri })}
          onAdd={() => setDrawer({ kind: "addVer", image: drawer.image })}
        />
      )}
      {drawer?.kind === "pull" && (
        <PullDrawer image={drawer.image} version={drawer.version} uri={drawer.uri} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "newImg" && <NewImgDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "addVer" && <AddVerDrawer image={drawer.image} onClose={() => setDrawer(null)} />}
    </main>
  );
}

interface VerView {
  version: string;
  status: string; // raw ImageStatus
  statusLabel: string;
  statusClass: "success" | "pending";
  src: string;
  desc: string;
  addr: string;
  by: string;
  pending: boolean;
}

const SOURCE_LABEL: Record<string, string> = {
  dockerPush: "Docker 推送",
  oras: "ORAS 推送",
  webUpload: "页面上传",
  external: "外部镜像",
};

function VerDrawer({
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
  const { tenant } = useApp();
  const { toast, confirm } = useUI();
  const [search, setSearch] = useState("");

  const versionsQ = useQuery({
    queryKey: ["images", "versions", tenant, image],
    enabled: !!image,
    queryFn: async () => {
      const { data, error } = await sdk.listImageVersions({ path: { tenant, name: image } });
      if (error) throw error;
      return data;
    },
  });

  const delVer = useApiMutation(
    (version: string) => sdk.deleteImage({ path: { tenant, name: image, version } }),
    { invalidate: [["images"], ["images", "versions", tenant, image]], success: "版本已删除" },
  );

  const items: VerView[] =
    versionsQ.data?.items?.map((v) => {
      const pending = v.status !== "Ready";
      return {
        version: v.version,
        status: v.status,
        statusLabel: pending ? "推送中" : "就绪",
        statusClass: pending ? "pending" : "success",
        src: SOURCE_LABEL[v.source ?? ""] ?? "—",
        desc: v.description ?? v.displayName ?? "",
        addr: v.uri || (pending ? "推送完成后生成…" : "—"),
        by: v.owner ?? "—",
        pending,
      };
    }) ?? [];

  const filtered = items.filter((v) => {
    const term = search.trim().toLowerCase();
    if (!term) return true;
    return v.version.toLowerCase().includes(term) || v.desc.toLowerCase().includes(term);
  });

  const onDeleteVer = (v: VerView) => {
    confirm({
      title: `确定删除版本 ${v.version}？`,
      desc: "删除后该版本镜像不可恢复。",
      okLabel: "确认删除",
      danger: true,
      onConfirm: () => delVer.mutate(v.version),
    });
  };

  return (
    <Drawer open wide onClose={onClose} title={<span className="mono">{image}</span>} sub={desc || "镜像版本"}>
      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="搜索版本名称 / 描述" value={search} onChange={(e) => setSearch(e.target.value)} />
        </div>
        <div className="grow" />
        <button className="btn btn-sm btn-primary" onClick={onAdd}>
          + 添加版本
        </button>
      </div>
      <div className="ver-list">
        {filtered.map((v) => (
          <div className="ver-item" key={v.version}>
            <div className="ver-top">
              <span className="ver-name">{v.version}</span>
              <span className={"status status-" + v.statusClass}>
                <span className="dot" />
                {v.statusLabel}
              </span>
              <span className="badge badge-neutral ver-src">{v.src}</span>
              <div className="ver-actions">
                {v.pending ? (
                  <button className="act act-run" aria-label="查看进度" />
                ) : (
                  <>
                    <button className="act" aria-label="拉取命令" onClick={() => onPull(v.version, v.addr)} />
                    <button
                      className="act act-danger"
                      aria-label="删除"
                      onClick={() => onDeleteVer(v)}
                    />
                  </>
                )}
              </div>
            </div>
            <div className="ver-desc">{v.desc}</div>
            <div className="ver-meta">
              <span className={"ver-addr" + (v.pending ? " muted" : "")}>{v.addr}</span>
              {!v.pending && (
                <button
                  className="act ver-copy"
                  aria-label="复制"
                  onClick={() => {
                    void navigator.clipboard?.writeText(v.addr);
                    toast("地址已复制");
                  }}
                />
              )}
              <span className="ver-by">{v.by}</span>
            </div>
          </div>
        ))}
        <BlockState q={versionsQ} isEmpty={filtered.length === 0} />
      </div>
    </Drawer>
  );
}

function PullDrawer({ image, version, uri, onClose }: { image: string; version: string; uri: string; onClose: () => void }) {
  const { toast } = useUI();
  const ref = uri || `zot.axisml.internal/<tenant>/${image}:${version}`;
  const cmd = `docker login zot.axisml.internal -u <用户名> -p <token>\ndocker pull ${ref}`;
  return (
    <Drawer
      open
      onClose={onClose}
      title="拉取命令"
      sub={<span className="mono">{image}:{version}</span>}
      footer={
        <button className="btn btn-primary" onClick={onClose}>
          完成
        </button>
      }
    >
      <p className="muted" style={{ fontSize: 13, marginBottom: 12 }}>
        镜像存储于 OCI（zot），使用以下命令登录并拉取。临时凭证有效期 1 小时。
      </p>
      <pre className="logbox" style={{ maxHeight: "none" }}>{cmd}</pre>
      <button
        className="btn"
        style={{ marginTop: 14 }}
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

function NewImgDrawer({ onClose }: { onClose: () => void }) {
  const { tenant } = useApp();
  const [name, setName] = useState("");
  const [purpose, setPurpose] = useState<string>(PURPOSE_OPTIONS[0].value);
  const [description, setDescription] = useState("");
  const [tagKey, setTagKey] = useState("");
  const [tagVal, setTagVal] = useState("");
  const [tags, setTags] = useState<Record<string, string>>({});

  const create = useApiMutation(
    (body: sdk.ArtifactDefinitionCreateRequest) =>
      sdk.createImageDefinition({ path: { tenant, name: body.name }, body }),
    { invalidate: [["images"]], success: "镜像已创建，可在版本列表添加版本" },
  );

  const addTag = () => {
    const k = tagKey.trim();
    if (!k) return;
    setTags((t) => ({ ...t, [k]: tagVal.trim() }));
    setTagKey("");
    setTagVal("");
  };
  const removeTag = (k: string) =>
    setTags((t) => {
      const next = { ...t };
      delete next[k];
      return next;
    });

  const submit = () => {
    const labels: Record<string, string> = { ...tags, purpose };
    create.mutate(
      {
        name: name.trim(),
        description: description.trim() || undefined,
        labels,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="新建镜像"
      sub="先创建镜像条目，再添加具体版本"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" disabled={!name.trim() || create.isPending} onClick={submit}>
            {create.isPending ? "创建中…" : "创建镜像"}
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>
            镜像名 <span className="req">*</span>
          </label>
          <input
            className="input mono"
            placeholder="my-image"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="field">
          <label>用途</label>
          <select className="input" value={purpose} onChange={(e) => setPurpose(e.target.value)}>
            {PURPOSE_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="简要说明镜像用途、基础环境与适用场景"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
      </div>

      <FieldsetTitle n={2}>自定义标签</FieldsetTitle>
      <div className="tag-group">
        <div className="custom-tags">
          <div className="chip-row">
            {Object.entries(tags).map(([k, v]) => (
              <span className="tag mono" key={k}>
                {k}:{v}{" "}
                <button type="button" className="chip-x" aria-label="移除" onClick={() => removeTag(k)}>
                  ✕
                </button>
              </span>
            ))}
          </div>
          <div className="cta-input">
            <input
              className="input mono"
              placeholder="键，如 cuda"
              value={tagKey}
              onChange={(e) => setTagKey(e.target.value)}
            />
            <span className="cta-sep mono">:</span>
            <input
              className="input mono"
              placeholder="值，如 12.1"
              value={tagVal}
              onChange={(e) => setTagVal(e.target.value)}
            />
            <button className="btn btn-sm" type="button" onClick={addTag}>
              添加
            </button>
          </div>
        </div>
      </div>
    </Drawer>
  );
}

function AddVerDrawer({ image, onClose }: { image: string; onClose: () => void }) {
  const { tenant } = useApp();
  const { toast } = useUI();
  const [version, setVersion] = useState("");
  const [description, setDescription] = useState("");
  const [mode, setMode] = useState<"remote" | "docker">("remote");
  const [sourceRef, setSourceRef] = useState("");

  const initiate = useApiMutation(
    (body: sdk.ImageInitiateRequest) =>
      sdk.initiateImage({ path: { tenant, name: image }, body }),
    { invalidate: [["images"], ["images", "versions", tenant, image]] },
  );

  const submit = () => {
    const v = version.trim();
    if (!v) return;
    // "remote" registers an external image (no client upload); "docker" creates
    // the version record so the user can push against the returned URI. Both go
    // through initiate; the resulting URI / status is read back in the version
    // list. completeImage requires a content digest the form does not capture,
    // so the push path is left for the server-side sync / push to finalize.
    const body: sdk.ImageInitiateRequest =
      mode === "remote"
        ? {
            version: v,
            spec: {},
            description: description.trim() || undefined,
            source: "external",
            sourceImageRef: sourceRef.trim() || undefined,
          }
        : {
            version: v,
            spec: {},
            description: description.trim() || undefined,
            source: "dockerPush",
          };
    initiate.mutate(body, {
      onSuccess: () => {
        toast("已提交，版本将在推送 / 同步完成后就绪");
        onClose();
      },
    });
  };

  const canSubmit = !!version.trim() && (mode !== "remote" || !!sourceRef.trim()) && !initiate.isPending;

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="添加版本"
      sub="向已有镜像添加新版本"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button className="btn btn-primary" disabled={!canSubmit} onClick={submit}>
            {initiate.isPending ? "提交中…" : "提交"}
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>镜像</label>
          <input className="input mono" value={image} disabled />
        </div>
        <div className="field">
          <label>
            版本 / tag <span className="req">*</span>
          </label>
          <input
            className="input mono"
            placeholder="2.4-cu124"
            value={version}
            onChange={(e) => setVersion(e.target.value)}
          />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="本次更新内容，如：升级 CUDA、集成 FlashAttention-2"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
      </div>

      <FieldsetTitle n={2}>添加方式</FieldsetTitle>
      <Tabs
        defaultKey="remote"
        tabs={[
          {
            key: "remote",
            label: "添加外部镜像",
            content: (
              <>
                <MethodSetter onShow={() => setMode("remote")} />
                <p className="help" style={{ marginBottom: 14 }}>
                  引用外部仓库中的已有镜像，平台将其同步到镜像仓，适合从 Docker Hub、NGC、Harbor 等公开或私有仓库导入。
                </p>
                <div className="form-grid">
                  <div className="field full">
                    <label>
                      源镜像地址（地址 + tag/digest）<span className="req">*</span>
                    </label>
                    <input
                      className="input mono"
                      placeholder="docker.io/pytorch/pytorch:2.4.0-cuda12.4-cudnn9-runtime"
                      value={sourceRef}
                      onChange={(e) => setSourceRef(e.target.value)}
                    />
                  </div>
                  <div className="field full">
                    <label>拉取凭证</label>
                    <select className="input">
                      <option>公开镜像，无需认证</option>
                      <option>ngc-cred（已保存）</option>
                      <option>harbor-cred（已保存）</option>
                      <option>+ 新建凭证…</option>
                    </select>
                  </div>
                </div>
              </>
            ),
          },
          {
            key: "docker",
            label: "通过 Docker 推送",
            content: (
              <>
                <MethodSetter onShow={() => setMode("docker")} />
                <p className="help" style={{ marginBottom: 14 }}>
                  使用 <b>Docker</b> 将本地构建好的镜像直接推送到镜像仓，适合自建镜像与 CI 流水线。
                </p>

                <div className="oras-step">
                  <div className="os-head">1 · 登录镜像仓</div>
                  <pre className="logbox" style={{ maxHeight: "none" }}>{`# 临时凭证有效期 1h
docker login zot.axisml.internal -u <用户名> -p <token>`}</pre>
                  <button className="btn btn-sm" style={{ marginTop: 10 }} onClick={() => toast("登录命令已复制")}>
                    复制命令
                  </button>
                </div>

                <div className="oras-step">
                  <div className="os-head">2 · 打标签并推送</div>
                  <pre className="logbox" style={{ maxHeight: "none" }}>{`# 1. 为本地镜像打上目标 tag
docker tag <本地镜像>:<tag> zot.axisml.internal/<tenant>/${image}:${version || "<tag>"}

# 2. 推送到镜像仓
docker push zot.axisml.internal/<tenant>/${image}:${version || "<tag>"}`}</pre>
                  <button className="btn btn-sm" style={{ marginTop: 10 }} onClick={() => toast("推送命令已复制")}>
                    复制命令
                  </button>
                </div>
              </>
            ),
          },
        ]}
      />
    </Drawer>
  );
}

// Tabs renders only the active pane, so mounting a pane signals the active add
// method back to the form (Tabs has no onChange) — mirrors Models' WebMethodSetter.
function MethodSetter({ onShow }: { onShow: () => void }) {
  useEffect(() => {
    onShow();
  }, []);
  return null;
}
