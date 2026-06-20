import { useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { useServices } from "@/api/hooks";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { PickGrid, FieldsetTitle } from "@/components/forms";

type SvcStatus = "success" | "pending" | "stopped";

interface SvcRow {
  name: string;
  desc: string;
  status: SvcStatus;
  statusLabel: string;
  replicas: string;
  unit: string;
  url?: string;
  running: boolean; // true → 停止 action; false → 启动 action
}

// Faithful demo rows from prototype/services.html — rendered when the backend
// (contract-only shell) returns no items.
const FALLBACK: SvcRow[] = [
  {
    name: "svc-chat-api",
    desc: "对话推理服务",
    status: "success",
    statusLabel: "就绪",
    replicas: "2 / 2",
    unit: "gpu-h100/1x",
    url: "/services/llm-lab/chat/",
    running: true,
  },
  {
    name: "svc-embed",
    desc: "文本向量服务",
    status: "pending",
    statusLabel: "降级",
    replicas: "1 / 2",
    unit: "gpu-l40s/1x",
    url: "/services/llm-lab/embed/",
    running: true,
  },
  {
    name: "svc-rerank",
    desc: "重排序服务",
    status: "stopped",
    statusLabel: "已停止",
    replicas: "0 / 0",
    unit: "cpu-large/1x",
    running: false,
  },
];

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

export default function Services() {
  const { data } = useServices();
  const { toast, confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; name?: string } | null>(null);

  const rows: SvcRow[] =
    data?.items?.map((s) => {
      const p = PHASE_MAP[s.phase ?? "Pending"] ?? PHASE_MAP.Pending;
      return {
        name: s.name,
        desc: s.description ?? s.displayName ?? "",
        status: p.status,
        statusLabel: p.label,
        replicas: `${s.readyReplicas ?? 0} / ${s.replicas ?? 0}`,
        unit: `${s.poolName ?? "—"}/${s.unitName ?? "—"}`,
        url: s.accessUrl,
        running: p.running,
      };
    }) ?? FALLBACK;

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
                        onClick={() => setDrawer({ mode: "edit", name: r.name })}
                      >
                        <EditIcon />
                      </button>
                      <button
                        className="act"
                        title="扩缩容"
                        aria-label="扩缩容"
                        onClick={() => setDrawer({ mode: "scale", name: r.name })}
                      >
                        <ScaleIcon />
                      </button>
                      {r.running ? (
                        <button
                          className="act"
                          title="停止"
                          aria-label="停止"
                          onClick={() => toast(`服务 ${r.name} 正在停止…`)}
                        >
                          <StopIcon />
                        </button>
                      ) : (
                        <button
                          className="act"
                          title="启动"
                          aria-label="启动"
                          onClick={() => toast(`服务 ${r.name} 正在启动…`)}
                        >
                          <PlayIcon />
                        </button>
                      )}
                      <button
                        className="act act-danger"
                        title="删除"
                        aria-label="删除"
                        onClick={() =>
                          confirm({
                            title: `删除服务 ${r.name}？`,
                            desc: "将下线服务并回收副本，访问路由一并移除。该操作不可恢复。",
                            okLabel: "确认删除",
                            toast: `服务 ${r.name} 已删除`,
                          })
                        }
                      >
                        <TrashIcon />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
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

      {drawer?.mode === "new" && <SvcFormDrawer mode="new" onClose={() => setDrawer(null)} />}
      {drawer?.mode === "edit" && <SvcFormDrawer mode="edit" name={drawer.name} onClose={() => setDrawer(null)} />}
      {drawer?.mode === "scale" && <ScaleDrawer name={drawer.name} onClose={() => setDrawer(null)} />}
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

// ── Port rows (端口) — local repeatable widget, mirrors VolList shape ─────────
interface PortRow {
  id: number;
  name: string;
  port: string;
}
let portSeq = 0;
function PortList({ initial }: { initial: { name: string; port: string }[] }) {
  const [rows, setRows] = useState<PortRow[]>(() => initial.map((r) => ({ ...r, id: ++portSeq })));
  const add = () => setRows((r) => [...r, { id: ++portSeq, name: "", port: "" }]);
  const remove = (id: number) => setRows((r) => (r.length <= 1 ? r : r.filter((x) => x.id !== id)));
  return (
    <>
      <div className="vol-list">
        {rows.map((row) => (
          <div className="vol-row" key={row.id}>
            <input
              className="input mono"
              defaultValue={row.name}
              placeholder="名称，如 http"
              aria-label="端口名"
              maxLength={15}
            />
            <input
              className="input mono"
              defaultValue={row.port}
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

const UNITS = [
  { title: "h100-1x", spec: "1×H100 · 16 vCPU · 128 GiB" },
  { title: "l40s-1x", spec: "1×L40S · 8 vCPU · 64 GiB" },
];

function SvcFormDrawer({ mode, name, onClose }: { mode: "new" | "edit"; name?: string; onClose: () => void }) {
  const { toast } = useUI();
  const isEdit = mode === "edit";
  const svcName = name ?? "svc-chat-api";
  const title = isEdit ? "编辑服务" : "新建在线服务";
  const sub: ReactNode = isEdit ? <span className="mono">{svcName}</span> : "从已注册模型版本部署常驻推理服务";
  const submit = isEdit
    ? { label: "保存", toast: "服务配置已保存" }
    : { label: "上线服务", toast: "服务上线中…" };
  const ports = isEdit
    ? [
        { name: "http", port: "8000" },
        { name: "grpc", port: "8001" },
      ]
    : [{ name: "http", port: "8000" }];

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title={title}
      sub={sub}
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            onClick={() => {
              toast(submit.toast);
              onClose();
            }}
          >
            {submit.label}
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
          <input className="input mono" placeholder="svc-chat-api" defaultValue={isEdit ? svcName : ""} />
          <span className="help">用于在列表与详情中展示</span>
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" placeholder="用途说明（可选）" defaultValue={isEdit ? "对话推理服务" : ""} />
        </div>
      </div>

      <FieldsetTitle n={2}>模型与镜像</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>
            模型版本 <span className="req">*</span>
          </label>
          <select className="input">
            <option>llama3-8b-sft@v4</option>
            <option>llama3-8b-sft@v3</option>
          </select>
        </div>
        <div className="field">
          <label>
            推理镜像 <span className="req">*</span>
          </label>
          <select className="input">
            <option>vllm-serve:0.5.1</option>
            <option>tgi:2.0</option>
          </select>
        </div>
      </div>

      <FieldsetTitle n={3}>资源选择</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            资源池 <span className="req">*</span>
          </label>
          <select className="input">
            <option>gpu-h100 · H100 推理池</option>
            <option>gpu-l40s · L40S 推理池</option>
          </select>
        </div>
      </div>
      <div className="field" style={{ marginTop: "var(--space-4)" }}>
        <label>
          资源单元 <span className="req">*</span>
        </label>
        <PickGrid options={UNITS} />
      </div>
      <div className="form-grid" style={{ marginTop: "var(--space-4)" }}>
        <div className="field">
          <label>
            副本数 <span className="req">*</span>
          </label>
          <input className="input num mono" defaultValue="2" />
        </div>
      </div>

      <FieldsetTitle n={4}>端口与路由</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            端口 <span className="req">*</span>
          </label>
          <PortList initial={ports} />
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
          <RouteToggle />
        </div>
        <div className="field">
          <label>Path</label>
          <input
            className="input mono"
            placeholder="/services/llm-lab/chat/"
            defaultValue={isEdit ? "/services/llm-lab/chat/" : ""}
          />
        </div>
      </div>
    </Drawer>
  );
}

function RouteToggle() {
  const [on, setOn] = useState(true);
  return (
    <button
      className={"toggle" + (on ? " on" : "")}
      aria-label="启用对外路由"
      onClick={() => setOn((v) => !v)}
    />
  );
}

function ScaleDrawer({ name, onClose }: { name?: string; onClose: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer
      open
      onClose={onClose}
      title="扩缩容"
      sub={<span className="mono">{name ?? "svc-chat-api"}</span>}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("已扩容至 3 副本");
              onClose();
            }}
          >
            应用
          </button>
        </>
      }
    >
      <p className="muted" style={{ fontSize: 13, marginBottom: "var(--space-5)" }}>
        扩缩容仅修改副本数。
      </p>
      <div className="field">
        <label>目标副本数</label>
        <input className="input num" defaultValue="3" />
        <span className="help">当前 2 / 2 就绪 · 配额上限 gpu-h100 max=8</span>
      </div>
      <div className="panel" style={{ padding: "var(--space-4)", marginTop: "var(--space-5)" }}>
        <dl className="kv" style={{ gridTemplateColumns: "96px 1fr" }}>
          <dt>资源单元</dt>
          <dd className="mono">gpu-h100 / h100-1x</dd>
          <dt>单副本</dt>
          <dd className="mono">1×H100 · 16 vCPU · 128 GiB</dd>
          <dt>扩容增量</dt>
          <dd>+1 副本 = +1 H100</dd>
        </dl>
      </div>
    </Drawer>
  );
}
