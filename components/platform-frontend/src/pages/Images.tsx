import { useMemo, useState } from "react";
import { useImages } from "@/api/hooks";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { Tabs } from "@/components/Tabs";
import { FieldsetTitle } from "@/components/forms";

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

// Faithful demo rows from prototype/images.html — rendered when the backend
// (contract-only shell) returns no items. Each row carries its own update label
// so search/filter never reassigns another image's timestamp.
const FALLBACK: ImageRow[] = [
  { name: "pytorch", desc: "PyTorch 训练镜像", icon: "box", purpose: "training", latest: "2.3-cu121", versions: 6, updated: "更新 4 天前", updatedShort: "4 天前" },
  { name: "vllm-serve", desc: "vLLM 推理服务", icon: "bolt", purpose: "inference", latest: "0.5.1", versions: 8, updated: "更新 1 周前", updatedShort: "1 周前" },
  { name: "jupyter-ds", desc: "Jupyter 开发环境", icon: "code", purpose: "dev", latest: "2024.3", versions: 3, updated: "更新 2 周前", updatedShort: "2 周前" },
  { name: "megatron", desc: "Megatron-LM 训练", icon: "box", purpose: "training", latest: "24.05", versions: 2, updated: "更新 6 天前", updatedShort: "6 天前" },
  { name: "lm-eval-harness", desc: "评测镜像", icon: "chart", purpose: "inference", latest: "0.4", versions: 1, updated: "更新 3 天前", updatedShort: "3 天前" },
];

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

type DrawerKind = "ver" | "pull" | "newImg" | "addVer";

export default function Images() {
  const { data } = useImages();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [query, setQuery] = useState("");
  const [drawer, setDrawer] = useState<DrawerKind | null>(null);

  const rows: ImageRow[] =
    data?.items?.map((m) => ({
      name: m.name,
      desc: m.description ?? m.displayName ?? "",
      icon: "box" as const,
      purpose: (m.labels?.purpose as string) ?? "—",
      latest: "—",
      versions: 0,
      updated: "刚刚",
      updatedShort: "刚刚",
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
          <button className="btn btn-primary" onClick={() => setDrawer("newImg")}>
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
                    <th>用途</th>
                    <th>最新版本</th>
                    <th className="num-col">版本数</th>
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
                              setDrawer("addVer");
                            }}
                          >
                            <UploadGlyph />
                          </button>
                        </div>
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

      <VerDrawer open={drawer === "ver"} onClose={() => setDrawer(null)} onPull={() => setDrawer("pull")} onAdd={() => setDrawer("addVer")} />
      <PullDrawer open={drawer === "pull"} onClose={() => setDrawer(null)} />
      <NewImgDrawer open={drawer === "newImg"} onClose={() => setDrawer(null)} />
      <AddVerDrawer open={drawer === "addVer"} onClose={() => setDrawer(null)} />
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
  { name: "2.3-cu121", status: "success", statusLabel: "就绪", src: "Docker 推送", desc: "PyTorch 2.3 + CUDA 12.1，集成 FlashAttention-2", addr: "zot.axisml.internal/llm-lab/pytorch:2.3-cu121", by: "张伟 · 4 天前" },
  { name: "2.2-cu118", status: "success", statusLabel: "就绪", src: "外部镜像", desc: "从 NGC 同步的基线训练镜像", addr: "zot.axisml.internal/llm-lab/pytorch:2.2-cu118", by: "李娜 · 2 周前" },
  { name: "2.1-cu118", status: "success", statusLabel: "就绪", src: "Docker 推送", desc: "兼容旧版 GPU 驱动", addr: "zot.axisml.internal/llm-lab/pytorch:2.1-cu118", by: "李娜 · 3 周前" },
  { name: "2.4-cu124", status: "pending", statusLabel: "推送中", src: "Docker 推送", desc: "升级至 CUDA 12.4", addr: "推送完成后生成…", by: "张伟 · 刚刚", pending: true },
];

function VerDrawer({ open, onClose, onPull, onAdd }: { open: boolean; onClose: () => void; onPull: () => void; onAdd: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer open={open} wide onClose={onClose} title={<span className="mono">pytorch</span>} sub="PyTorch 训练镜像 · training · 创建人 张伟">
      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="搜索版本名称 / 描述" />
        </div>
        <div className="grow" />
        <button className="btn btn-sm btn-primary" onClick={onAdd}>
          + 添加版本
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
                  <button className="act act-run" aria-label="查看进度" />
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
      sub={<span className="mono">pytorch:2.3-cu121</span>}
      footer={
        <button className="btn btn-primary" onClick={onClose}>
          完成
        </button>
      }
    >
      <p className="muted" style={{ fontSize: 13, marginBottom: 12 }}>
        镜像存储于 OCI（zot），使用以下命令登录并拉取。临时凭证有效期 1 小时。
      </p>
      <pre className="logbox" style={{ maxHeight: "none" }}>{`docker login zot.axisml.internal -u <用户名> -p <token>
docker pull zot.axisml.internal/llm-lab/pytorch:2.3-cu121`}</pre>
      <button className="btn" style={{ marginTop: 14 }} onClick={() => toast("命令已复制到剪贴板")}>
        复制命令
      </button>
    </Drawer>
  );
}

function NewImgDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer
      open={open}
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
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("镜像已创建，可在版本列表添加版本");
              onClose();
            }}
          >
            创建镜像
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
          <input className="input mono" placeholder="my-image" />
        </div>
        <div className="field">
          <label>用途</label>
          <select className="input">
            <option>训练镜像</option>
            <option>推理镜像</option>
            <option>评估镜像</option>
            <option>自定义</option>
          </select>
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" placeholder="简要说明镜像用途、基础环境与适用场景" />
        </div>
      </div>

      <FieldsetTitle n={2}>自定义标签</FieldsetTitle>
      <div className="tag-group">
        <div className="custom-tags">
          <div className="chip-row" />
          <div className="cta-input">
            <input className="input mono" placeholder="键，如 cuda" />
            <span className="cta-sep mono">:</span>
            <input className="input mono" placeholder="值，如 12.1" />
            <button className="btn btn-sm" type="button">
              添加
            </button>
          </div>
        </div>
      </div>
    </Drawer>
  );
}

function AddVerDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer
      open={open}
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
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("已提交，版本将在推送 / 同步完成后就绪");
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
          <label>镜像</label>
          <input className="input mono" value="pytorch" disabled />
        </div>
        <div className="field">
          <label>
            版本 / tag <span className="req">*</span>
          </label>
          <input className="input mono" placeholder="2.4-cu124" />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" placeholder="本次更新内容，如：升级 CUDA、集成 FlashAttention-2" />
        </div>
      </div>

      <FieldsetTitle n={2}>添加方式</FieldsetTitle>
      <Tabs
        tabs={[
          {
            key: "remote",
            label: "添加外部镜像",
            content: (
              <>
                <p className="help" style={{ marginBottom: 14 }}>
                  引用外部仓库中的已有镜像，平台将其同步到镜像仓，适合从 Docker Hub、NGC、Harbor 等公开或私有仓库导入。
                </p>
                <div className="form-grid">
                  <div className="field full">
                    <label>
                      源镜像地址（地址 + tag/digest）<span className="req">*</span>
                    </label>
                    <input className="input mono" placeholder="docker.io/pytorch/pytorch:2.4.0-cuda12.4-cudnn9-runtime" />
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
docker tag <本地镜像>:<tag> zot.axisml.internal/llm-lab/pytorch:2.4-cu124

# 2. 推送到镜像仓
docker push zot.axisml.internal/llm-lab/pytorch:2.4-cu124`}</pre>
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
