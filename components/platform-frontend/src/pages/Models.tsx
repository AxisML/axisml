import { useMemo, useState } from "react";
import { useModels } from "@/api/hooks";
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

// Faithful demo rows from prototype/models.html — rendered when the backend
// (contract-only shell) returns no items. Each row carries its own update label
// and upload eligibility so search/filter never reassigns another model's values.
const FALLBACK: ModelRow[] = [
  { name: "llama3-8b-sft", desc: "LLaMA3-8B 监督微调权重", icon: "model", framework: "pytorch", latest: "v4", versions: 4, tags: ["task=chat", "lang=zh"], updated: "更新 2 天前", updatedShort: "2 天前", canUpload: true },
  { name: "bge-embed", desc: "BGE 文本向量模型", icon: "shield", framework: "safetensors", latest: "1.5.0", versions: 5, tags: ["embed"], updated: "更新 1 周前", updatedShort: "1 周前", canUpload: false },
  { name: "resnet-cls", desc: "ResNet 图像分类", icon: "graph", framework: "onnx", latest: "2024-06", versions: 2, tags: ["vision"], updated: "更新 3 天前", updatedShort: "3 天前", canUpload: true },
  { name: "qwen2-vl-ft", desc: "Qwen2-VL 视觉指令微调", icon: "model", framework: "pytorch", latest: "v2", versions: 2, tags: ["vision", "chat"], updated: "更新 5 小时前", updatedShort: "5 小时前", canUpload: true },
];

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

type DrawerKind = "ver" | "pull" | "newModel" | "up";

export default function Models() {
  const { data } = useModels();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [query, setQuery] = useState("");
  const [drawer, setDrawer] = useState<DrawerKind | null>(null);

  const rows: ModelRow[] =
    data?.items?.map((m) => ({
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
    })) ?? FALLBACK;

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter((r) => r.name.toLowerCase().includes(q) || r.desc.toLowerCase().includes(q));
  }, [rows, query]);

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
          <button className="btn btn-primary" onClick={() => setDrawer("newModel")}>
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
              <div className="art-card" key={r.name} onClick={() => setDrawer("ver")}>
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
                    <tr key={r.name} onClick={() => setDrawer("ver")}>
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
                        {r.canUpload ? (
                          <div className="row-actions">
                            <button
                              className="act"
                              title="上传新版本"
                              aria-label="上传新版本"
                              onClick={(e) => {
                                e.stopPropagation();
                                setDrawer("up");
                              }}
                            >
                              <UploadGlyph />
                            </button>
                          </div>
                        ) : null}
                      </td>
                    </tr>
                  ))}
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

      <VerDrawer open={drawer === "ver"} onClose={() => setDrawer(null)} onPull={() => setDrawer("pull")} onUpload={() => setDrawer("up")} />
      <PullDrawer open={drawer === "pull"} onClose={() => setDrawer(null)} />
      <NewModelDrawer open={drawer === "newModel"} onClose={() => setDrawer(null)} />
      <UploadDrawer open={drawer === "up"} onClose={() => setDrawer(null)} />
    </main>
  );
}

interface VerItem {
  name: string;
  status: "success" | "pending";
  statusLabel: string;
  src: string;
  desc: string;
  addr: string;
  by: string;
  pending?: boolean;
}

const VERSIONS: VerItem[] = [
  { name: "v4", status: "success", statusLabel: "就绪", src: "Oras 推送", desc: "扩充中文 SFT 数据，修复长文本截断", addr: "zot.axisml.internal/llm-lab/llama3-8b-sft:v4", by: "张伟 · 2 天前" },
  { name: "v3", status: "success", statusLabel: "就绪", src: "Web 上传", desc: "对齐 RLHF 偏好数据", addr: "zot.axisml.internal/llm-lab/llama3-8b-sft:v3", by: "张伟 · 5 天前" },
  { name: "v2", status: "success", statusLabel: "就绪", src: "S3 地址", desc: "基线 SFT 权重", addr: "zot.axisml.internal/llm-lab/llama3-8b-sft:v2", by: "李娜 · 1 周前" },
  { name: "v1", status: "pending", statusLabel: "上传中", src: "Web 上传", desc: "首次导入权重", addr: "地址生成中…", by: "李娜 · 刚刚", pending: true },
];

function VerDrawer({ open, onClose, onPull, onUpload }: { open: boolean; onClose: () => void; onPull: () => void; onUpload: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer open={open} wide onClose={onClose} title={<span className="mono">llama3-8b-sft</span>} sub="LLaMA3-8B 监督微调权重 · pytorch · 创建人 张伟">
      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="搜索版本名称 / 描述" />
        </div>
        <div className="grow" />
        <button className="btn btn-sm btn-primary" onClick={onUpload}>
          + 上传新版本
        </button>
      </div>
      <div className="ver-list">
        {VERSIONS.map((v) => (
          <div className="ver-item" key={v.name}>
            <div className="ver-top">
              <span className="ver-name">{v.name}</span>
              <span className={"status status-" + v.status}>
                <span className="dot" />
                {v.statusLabel}
              </span>
              <span className="badge badge-neutral ver-src">{v.src}</span>
              <div className="ver-actions">
                {v.pending ? (
                  <button className="act act-run" aria-label="完成上传" />
                ) : (
                  <>
                    <button className="act" aria-label="拉取命令" onClick={onPull} />
                    <button className="act act-danger" aria-label="删除" />
                  </>
                )}
              </div>
            </div>
            <div className="ver-desc">{v.desc}</div>
            <div className="ver-meta">
              <span className={"ver-addr" + (v.pending ? " muted" : "")}>{v.addr}</span>
              {!v.pending && <button className="act ver-copy" aria-label="复制" onClick={() => toast("地址已复制")} />}
              <span className="ver-by">{v.by}</span>
            </div>
          </div>
        ))}
      </div>
    </Drawer>
  );
}

function PullDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="拉取命令"
      sub={<span className="mono">llama3-8b-sft@v4</span>}
      footer={
        <button className="btn btn-primary" onClick={onClose}>
          完成
        </button>
      }
    >
      <p className="muted" style={{ fontSize: 13, marginBottom: 12 }}>
        模型存储于 OCI（zot），用以下命令拉取。临时凭证有效期 1 小时。
      </p>
      <pre className="logbox" style={{ maxHeight: "none" }}>{`docker pull zot.axisml.internal/llm-lab/\\
  llama3-8b-sft@sha256:a1b2c3d4e5f6...`}</pre>
      <button className="btn" style={{ marginTop: 14 }} onClick={() => toast("命令已复制到剪贴板")}>
        复制命令
      </button>
    </Drawer>
  );
}

function NewModelDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer
      open={open}
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
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("模型已创建，可在版本列表上传权重");
              onClose();
            }}
          >
            创建模型
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
          <input className="input mono" placeholder="my-llm-model（仅英文、数字与连字符）" />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" placeholder="简要说明模型用途、训练数据与适用场景" />
        </div>
      </div>

      <FieldsetTitle n={2}>标签</FieldsetTitle>

      <div className="tag-group">
        <span className="tg-label">Tasks</span>
        <div className="chip-row">
          <span className="tag-opt">Text Generation</span>
          <span className="tag-opt">Text Classification</span>
          <span className="tag-opt">Question Answering</span>
          <span className="tag-opt">Summarization</span>
          <span className="tag-opt">Translation</span>
          <span className="tag-opt">Feature Extraction</span>
          <span className="tag-opt">Image Classification</span>
          <span className="tag-opt">Object Detection</span>
          <span className="tag-opt">Automatic Speech Recognition</span>
          <span className="tag-opt">Text-to-Image</span>
          <input className="tag-add-input" placeholder="自定义，回车添加" aria-label="添加自定义 Task" />
        </div>
      </div>

      <div className="tag-group">
        <span className="tg-label">Parameters</span>
        <div className="param-slider">
          <div className="ps-row">
            <input type="range" className="range" min="0" max="8" step="1" defaultValue="3" aria-label="参数量" />
            <input className="input mono ps-input" aria-label="参数量" placeholder="7B" />
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

      <div className="tag-group">
        <span className="tg-label">Framework</span>
        <div className="chip-row">
          <span className="tag-opt">PyTorch</span>
          <span className="tag-opt">Safetensors</span>
          <span className="tag-opt">Transformers</span>
          <span className="tag-opt">TensorFlow</span>
          <span className="tag-opt">JAX</span>
          <span className="tag-opt">ONNX</span>
          <span className="tag-opt">GGUF</span>
          <input className="tag-add-input" placeholder="自定义，回车添加" aria-label="添加自定义 Framework" />
        </div>
      </div>

      <div className="tag-group">
        <span className="tg-label">自定义标签</span>
        <div className="custom-tags">
          <div className="chip-row" />
          <div className="cta-input">
            <input className="input mono" placeholder="键，如 license" />
            <span className="cta-sep mono">:</span>
            <input className="input mono" placeholder="值，如 apache-2.0" />
            <button className="btn btn-sm" type="button">
              添加
            </button>
          </div>
        </div>
      </div>
    </Drawer>
  );
}

function UploadDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer
      open={open}
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
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("已提交，版本正在上传 / 拉取");
              onClose();
            }}
          >
            提交
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>模型</label>
          <input className="input mono" value="llama3-8b-sft" disabled />
        </div>
        <div className="field">
          <label>
            版本号 <span className="req">*</span>
          </label>
          <input className="input mono" placeholder="v5 / 1.5.0 / 2026-06" />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" placeholder="本次更新内容，如：扩充中文 SFT 数据、修复输出截断" />
        </div>
      </div>

      <FieldsetTitle n={2}>上传方式</FieldsetTitle>
      <Tabs
        tabs={[
          {
            key: "web",
            label: "通过 Web 上传",
            content: (
              <>
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
                <div className="field full">
                  <label>
                    存储类型 <span className="req">*</span>
                  </label>
                  <select className="input">
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
                  <input className="input mono" placeholder="s3://bucket/prefix" />
                </div>
              </div>
            ),
          },
          {
            key: "oras",
            label: "使用 Oras 推送",
            content: (
              <>
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
                    <button className="btn btn-sm" onClick={() => toast("安装命令已复制")}>
                      复制命令
                    </button>
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
cd ./llama3-8b-sft
oras push zot.axisml.internal/llm-lab/llama3-8b-sft:v5 \\
  --artifact-type application/vnd.axisml.model.v1 \\
  ./*:application/octet-stream`}</pre>
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
