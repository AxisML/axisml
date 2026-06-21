import { useState } from "react";
import { Link } from "react-router-dom";
import { useWorkspaces } from "@/api/hooks";
import { useApiMutation, tenantHeader } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { FieldsetTitle } from "@/components/forms";
import { BlockState } from "@/components/states";

// Page-local styles ported verbatim from prototype/workspace.html's <head> <style>
// block (the .ws-card grid is page-specific, not part of the shared app.css design
// system). Rendered inline so app.css stays untouched.
const WS_CSS = `
    .ws-cards { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:var(--space-4); }
    @media (max-width:1100px){ .ws-cards { grid-template-columns:repeat(2,minmax(0,1fr)); } }
    @media (max-width:680px){ .ws-cards { grid-template-columns:1fr; } }
    .ws-card { background:var(--bg); border:1px solid var(--border); border-radius:var(--radius-md); padding:var(--space-4); box-shadow:var(--elev-card); transition:border-color var(--motion-fast),box-shadow var(--motion-fast); }
    .ws-card:hover { border-color:color-mix(in oklab,var(--accent) 40%,var(--border)); box-shadow:var(--elev-raised); }
    .ws-card .wc-head { display:flex; align-items:center; gap:10px; margin-bottom:12px; }
    .ws-card .wc-logo { width:38px; height:38px; border-radius:9px; display:grid; place-items:center; background:var(--surface); border:1px solid var(--border-soft); flex:none; }
    .ws-card .wc-logo svg { width:23px; height:23px; }
    .ws-card .wc-id { min-width:0; }
    .ws-card .wc-name { font-family:var(--font-mono); font-weight:600; font-size:14px; color:var(--fg); text-decoration:none; }
    .ws-card a.wc-name:hover { color:var(--accent); }
    .ws-card .wc-meta { color:var(--muted); font-size:12px; line-height:1.65; }
    .ws-card .wc-info { display:flex; align-items:center; gap:8px; margin-top:10px; }
    .ws-card .wc-chip { display:inline-flex; align-items:center; gap:5px; padding:3px 9px 3px 7px; border:1px solid var(--border); border-radius:var(--radius-pill); background:var(--surface); color:var(--fg-2); font-family:var(--font-mono); font-size:11.5px; }
    .ws-card .wc-chip svg { width:13px; height:13px; color:var(--muted); flex:none; }
    .ws-card .wc-owner { display:inline-flex; align-items:center; gap:6px; margin-left:auto; color:var(--fg-2); font-size:12px; }
    .ws-card .wc-ava { width:20px; height:20px; border-radius:50%; display:grid; place-items:center; font-size:10px; font-weight:600; line-height:1; color:var(--accent); background:color-mix(in oklab,var(--accent) 13%,var(--bg)); }
    .ws-card .wc-divider { height:1px; background:var(--border-soft); margin:14px 0 10px; }
    .ws-card .wc-actions { display:flex; align-items:center; gap:6px; }
    .ws-card .wc-spacer { flex:1; }
    .ws-card .wc-act { width:32px; height:32px; display:grid; place-items:center; border-radius:8px; border:1px solid transparent; background:transparent; color:var(--fg-2); cursor:pointer; transition:background var(--motion-fast),color var(--motion-fast),border-color var(--motion-fast); }
    .ws-card .wc-act:hover { background:var(--surface); border-color:var(--border); color:var(--fg); }
    .ws-card .wc-act svg { width:18px; height:18px; }
    .ws-card .wc-act svg.brand { width:17px; height:17px; }
    .ws-card .wc-act.start { color:var(--accent); }
    .ws-card .wc-act.danger:hover { background:color-mix(in oklab,var(--danger) 9%,transparent); border-color:color-mix(in oklab,var(--danger) 35%,var(--border)); color:var(--danger); }
    .ws-card .wc-act[disabled] { opacity:.36; pointer-events:none; }
    .ws-card .wc-act[disabled] svg.brand { filter:grayscale(1); }
    svg.brand { display:block; }
  `;

// Tool brand symbols (Jupyter / VS Code / PyTorch / terminal), inlined verbatim
// from prototype/workspace.html, referenced via <use href="#ic-..."/>.
function BrandDefs() {
  return (
    <svg width="0" height="0" style={{ position: "absolute" }} aria-hidden="true" focusable="false">
      <symbol id="ic-jupyter" viewBox="0 0 24 24">
        <path fill="#F37726" d="M7.157 22.201A1.784 1.784 0 0 1 5.374 24a1.784 1.784 0 0 1-1.784-1.799 1.784 1.784 0 0 1 1.784-1.799 1.784 1.784 0 0 1 1.783 1.799zM20.582 1.427a1.415 1.415 0 0 1-1.415 1.428 1.415 1.415 0 0 1-1.416-1.428A1.415 1.415 0 0 1 19.167 0a1.415 1.415 0 0 1 1.415 1.427zM4.992 3.336A1.781 1.781 0 0 1 3.21 5.135 1.781 1.781 0 0 1 1.427 3.336 1.781 1.781 0 0 1 3.21 1.537a1.781 1.781 0 0 1 1.782 1.799zM12 18.694c-3.945 0-7.394-1.417-9.191-3.506a9.799 9.799 0 0 0 18.382 0c-1.797 2.089-5.246 3.506-9.191 3.506zM12 5.306c3.945 0 7.394 1.417 9.191 3.506a9.799 9.799 0 0 0-18.382 0C4.606 6.723 8.055 5.306 12 5.306z" />
      </symbol>
      <symbol id="ic-vscode" viewBox="0 0 24 24">
        <path fill="#007ACC" d="M23.15 2.587L18.21.21a1.494 1.494 0 0 0-1.705.29l-9.46 8.63-4.12-3.128a.999.999 0 0 0-1.276.057L.327 7.261A1 1 0 0 0 .326 8.74L3.899 12 .326 15.26a1 1 0 0 0 .001 1.479L1.65 17.94a.999.999 0 0 0 1.276.057l4.12-3.128 9.46 8.63a1.492 1.492 0 0 0 1.704.29l4.942-2.377A1.5 1.5 0 0 0 24 20.06V3.939a1.5 1.5 0 0 0-.85-1.352zm-5.146 14.861L10.826 12l7.178-5.448v10.896z" />
      </symbol>
      <symbol id="ic-pytorch" viewBox="0 0 24 24">
        <path fill="#EE4C2C" d="M12.005 0L4.952 7.053a9.865 9.865 0 0 0 0 13.945 9.866 9.866 0 0 0 13.946 0c3.515-3.515 3.515-9.21 0-12.724l-1.508 1.508c2.682 2.682 2.682 7.026 0 9.708a6.865 6.865 0 0 1-9.71 0 6.865 6.865 0 0 1 0-9.708l4.317-4.34.008.008V0zm3.291 4.388a1.184 1.184 0 1 0 0 2.368 1.184 1.184 0 0 0 0-2.368z" />
      </symbol>
      <symbol id="ic-terminal" viewBox="0 0 24 24">
        <rect x="1.5" y="3.5" width="21" height="17" rx="3.5" fill="#2B303B" />
        <path d="M5.6 9.1l3.3 2.9-3.3 2.9" fill="none" stroke="#36D399" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M11.6 15h6" fill="none" stroke="#C8CDD6" strokeWidth="1.9" strokeLinecap="round" />
      </symbol>
    </svg>
  );
}

// One-off glyphs not in the icon map.
function CpuChip() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="6" y="6" width="12" height="12" rx="2" />
      <path d="M9 2v2M15 2v2M9 20v2M15 20v2M2 9h2M2 15h2M20 9h2M20 15h2" />
    </svg>
  );
}
function StopGlyph() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 4v8" />
      <path d="M7.5 7a7 7 0 1 0 9 0" />
    </svg>
  );
}
function TrashGlyph() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2M7 7l1 13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1l1-13" />
    </svg>
  );
}
function PlayFill() {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" stroke="none">
      <path d="M8 5.14v13.72a1 1 0 0 0 1.5.86l11-6.86a1 1 0 0 0 0-1.72l-11-6.86A1 1 0 0 0 8 5.14z" />
    </svg>
  );
}
// Row-action glyphs (1.0 stroke variants used in the list view).
function EyeGlyph() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}
function StopSquare() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <rect x="6" y="6" width="12" height="12" rx="1" />
    </svg>
  );
}
function PlayTri() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M6 4l14 8-14 8z" />
    </svg>
  );
}
function TrashRow() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M3 6h18" />
      <path d="M8 6V4h8v2" />
      <path d="M19 6l-1 14H6L5 6" />
      <path d="M10 11v6M14 11v6" />
    </svg>
  );
}

type WsStatus = "running" | "pending" | "stopped";
const STATUS_LABEL: Record<WsStatus, string> = { running: "运行中", pending: "启动中", stopped: "已停止" };

// Map the backend WorkspacePhase enum to the three UI states the prototype shows.
function phaseToStatus(phase: string | undefined): WsStatus {
  switch (phase) {
    case "Running":
    case "Degraded":
      return "running";
    case "Creating":
    case "Starting":
    case "Pending":
      return "pending";
    default: // Stopped / Failed / Deleting / Deleted / undefined
      return "stopped";
  }
}

interface WsRow {
  name: string;
  desc: string;
  status: WsStatus;
  brand: "jupyter" | "pytorch" | "vscode";
  unit: string;
  image: string;
  owner: string;
  ava: string;
  pvc?: string; // PVC size shown in delete confirm; absent rows have no delete (pending)
}

export default function Workspaces() {
  const q = useWorkspaces();
  const { confirm } = useUI();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [drawer, setDrawer] = useState(false);

  const start = useApiMutation(
    (name: string) => sdk.startWorkspace({ path: { name } }),
    { invalidate: [["workspaces"]], success: "工作区启动中…" },
  );
  const stop = useApiMutation(
    (name: string) => sdk.stopWorkspace({ path: { name } }),
    { invalidate: [["workspaces"]], success: "工作区已停止" },
  );
  const del = useApiMutation(
    ({ name, deletePvc }: { name: string; deletePvc: boolean }) =>
      sdk.deleteWorkspace({ path: { name }, body: { deletePvc } }),
    { invalidate: [["workspaces"]], success: "工作区已删除" },
  );

  const rows: WsRow[] = q.data?.items?.map((w) => ({
    name: w.name,
    desc: w.description ?? w.displayName ?? "",
    status: phaseToStatus(w.phase),
    brand: "jupyter",
    unit: w.unitName ?? "—",
    image: w.image ?? "—",
    owner: w.owner ?? "—",
    ava: (w.owner ?? "—").slice(0, 1),
    pvc: w.volumes?.length ? "—" : undefined,
  })) ?? [];

  // Delete with the "also delete PVC" toggle surfaced in the confirm dialog. The
  // checkbox state is captured in a closure so onConfirm reads the latest value.
  const onDelete = (name: string, desc: string, pvc?: string) => {
    let deletePvc = pvc != null;
    confirm({
      title: `删除工作区 ${name}？`,
      desc,
      info: pvc != null ? (
        <label style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 4 }}>
          <input
            type="checkbox"
            defaultChecked
            onChange={(e) => {
              deletePvc = e.target.checked;
            }}
          />{" "}
          一并删除数据卷 PVC（{pvc}）
        </label>
      ) : undefined,
      okLabel: "确认删除",
      danger: true,
      onConfirm: () => del.mutate({ name, deletePvc }),
    });
  };

  return (
    <main className="page">
      <style>{WS_CSS}</style>
      <BrandDefs />
      <div className="breadcrumb">
        <span>训练中心</span>
        <span className="sep">/</span>
        <span>工作区</span>
      </div>
      <div className="page-head">
        <div>
          <h1>工作区</h1>
          <p className="sub">用于交互式开发的容器环境，支持 Jupyter 和 VSCode，不用时随时停掉。</p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setDrawer(true)}>
            <Icon name="plus" />
            新建工作区
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
          <option>运行中</option>
          <option>启动中</option>
          <option>已停止</option>
        </select>
        <select className="select">
          <option>资源池：全部</option>
          <option>gpu-a100</option>
          <option>gpu-l40s</option>
          <option>cpu-medium</option>
        </select>
        <select className="select">
          <option>创建人：全部</option>
          <option>张伟</option>
          <option>李娜</option>
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

      {/* 卡片视图 */}
      {view === "cards" && (
        <div className="ws-cards">
          {rows.map((r) => (
            <WsCard
              key={r.name}
              row={r}
              onStart={(name) => start.mutate(name)}
              onStop={(name) => stop.mutate(name)}
              onDelete={onDelete}
            />
          ))}
          <BlockState q={q} isEmpty={rows.length === 0} />
        </div>
      )}

      {/* 列表视图 */}
      {view === "list" && (
        <div className="panel">
          <div className="table-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>名称</th>
                  <th>状态</th>
                  <th>资源单元</th>
                  <th>镜像</th>
                  <th>创建人</th>
                  <th style={{ textAlign: "right" }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.name}>
                    <td>
                      <Link className="t-name mono" to={`/workspaces/${r.name}`}>
                        {r.name}
                      </Link>
                    </td>
                    <td>
                      <span className={`status status-${r.status}`}>
                        <span className="dot" />
                        {STATUS_LABEL[r.status]}
                      </span>
                    </td>
                    <td className="mono">{r.unit}</td>
                    <td className="mono muted">{r.image}</td>
                    <td>{r.owner}</td>
                    <td>
                      <div className="row-actions">
                        <Link className="act" to={`/workspaces/${r.name}`} title="详情" aria-label="详情">
                          <EyeGlyph />
                        </Link>
                        {r.status === "stopped" ? (
                          <button className="act" title="启动" aria-label="启动" onClick={() => start.mutate(r.name)}>
                            <PlayTri />
                          </button>
                        ) : (
                          <button className="act" title="停止" aria-label="停止" onClick={() => stop.mutate(r.name)}>
                            <StopSquare />
                          </button>
                        )}
                        <button
                          className="act act-danger"
                          title="删除"
                          aria-label="删除"
                          onClick={() => onDelete(r.name, "删除后不可恢复。", r.pvc)}
                        >
                          <TrashRow />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="pagination">
            <span>共 {rows.length} 个工作区</span>
            <div className="pages">
              <span className="pg">‹</span>
              <span className="pg on">1</span>
              <span className="pg">›</span>
            </div>
            <span>每页 20 条</span>
          </div>
        </div>
      )}

      {drawer && <WsDrawer onClose={() => setDrawer(false)} />}
    </main>
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
  onDelete: (name: string, desc: string, pvc?: string) => void;
}) {
  const running = row.status === "running";
  const stopped = row.status === "stopped";
  return (
    <div className="ws-card">
      <div className="wc-head">
        <div className="wc-logo">
          <svg className="brand">
            <use href={`#ic-${row.brand}`} />
          </svg>
        </div>
        <div className="wc-id">
          <Link className="wc-name" to={`/workspaces/${row.name}`}>
            {row.name}
          </Link>
          <div className="wc-meta">{row.desc}</div>
        </div>
        <span className={`status status-${row.status}`} style={{ marginLeft: "auto" }}>
          <span className="dot" />
          {STATUS_LABEL[row.status]}
        </span>
      </div>
      <div className="wc-info">
        <span className="wc-chip">
          <CpuChip />
          {row.unit}
        </span>
        <span className="wc-owner">
          <span className="wc-ava">{row.ava}</span>
          {row.owner}
        </span>
      </div>
      <div className="wc-divider" />
      <div className="wc-actions">
        {row.status === "pending" ? (
          <>
            <button className="wc-act" title="工作区启动后可用" disabled>
              <svg className="brand">
                <use href="#ic-jupyter" />
              </svg>
            </button>
            <button className="wc-act" title="工作区启动后可用" disabled>
              <svg className="brand">
                <use href="#ic-vscode" />
              </svg>
            </button>
            <button className="wc-act" title="工作区启动后可用" disabled>
              <svg className="brand">
                <use href="#ic-terminal" />
              </svg>
            </button>
            <span className="wc-spacer" />
            <button className="wc-act" title="停止工作区" onClick={() => onStop(row.name)}>
              <StopGlyph />
            </button>
          </>
        ) : stopped ? (
          <>
            <button className="wc-act start" title="启动工作区" onClick={() => onStart(row.name)}>
              <PlayFill />
            </button>
            <span className="wc-spacer" />
            <button
              className="wc-act danger"
              title="删除工作区"
              onClick={() => onDelete(row.name, "工作区已停止，删除后不可恢复。", row.pvc)}
            >
              <TrashGlyph />
            </button>
          </>
        ) : (
          <>
            <Link className="wc-act" title="打开 Jupyter" to={`/workspaces/${row.name}`}>
              <svg className="brand">
                <use href="#ic-jupyter" />
              </svg>
            </Link>
            <Link className="wc-act" title="打开 VS Code" to={`/workspaces/${row.name}`}>
              <svg className="brand">
                <use href="#ic-vscode" />
              </svg>
            </Link>
            <Link className="wc-act" title="打开终端" to={`/workspaces/${row.name}`}>
              <svg className="brand">
                <use href="#ic-terminal" />
              </svg>
            </Link>
            <span className="wc-spacer" />
            <button className="wc-act" title="停止工作区" onClick={() => onStop(row.name)}>
              <StopGlyph />
            </button>
            {running && (
              <button
                className="wc-act danger"
                title="删除工作区"
                onClick={() => onDelete(row.name, "运行中的工作区将先停止再删除，删除后不可恢复。", row.pvc)}
              >
                <TrashGlyph />
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

// Image options for the brand picker — `image` is the value mapped into the
// WorkspaceCreateRequest.image required field.
const WS_IMAGES: { brand: string; image: string; spec: string }[] = [
  { brand: "ic-jupyter", image: "jupyter-ds:2024.3", spec: "Jupyter 开发环境 · 公共" },
  { brand: "ic-pytorch", image: "pytorch:2.3-cu121", spec: "PyTorch 训练镜像" },
  { brand: "ic-vscode", image: "vscode-server:1.90", spec: "VS Code 开发环境 · 公共" },
];

// Resource-unit options — `pool`/`unit` map to the required poolName/unitName.
const WS_UNITS: { pool: string; unit: string; spec: string }[] = [
  { pool: "cpu-medium", unit: "cpu-medium", spec: "8 vCPU · 32 GiB" },
  { pool: "cpu-medium", unit: "cpu-large", spec: "16 vCPU · 64 GiB" },
  { pool: "gpu-a100", unit: "a100-1x", spec: "1×A100 · 8 vCPU · 64 GiB" },
];

// Parse the env textarea (one KEY=VALUE per line) into the EnvVar[] DTO shape.
function parseEnv(s: string): sdk.EnvVar[] {
  const out: sdk.EnvVar[] = [];
  for (const line of s.split("\n")) {
    const t = line.trim();
    if (!t) continue;
    const eq = t.indexOf("=");
    const name = (eq === -1 ? t : t.slice(0, eq)).trim();
    if (!name) continue;
    out.push({ name, value: eq === -1 ? "" : t.slice(eq + 1).trim() });
  }
  return out;
}

function WsDrawer({ onClose }: { onClose: () => void }) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [imageIdx, setImageIdx] = useState(0);
  const [unitIdx, setUnitIdx] = useState(0);
  const [port, setPort] = useState("8888");
  const [env, setEnv] = useState("");
  const [mountPath, setMountPath] = useState("/workspace");
  const [volSize, setVolSize] = useState("50Gi");

  const create = useApiMutation(
    (body: sdk.WorkspaceCreateRequest) => sdk.createWorkspace({ body, headers: tenantHeader() }),
    { invalidate: [["workspaces"]], success: "工作区创建中…" },
  );

  const submit = () => {
    const img = WS_IMAGES[imageIdx];
    const u = WS_UNITS[unitIdx];
    const envVars = parseEnv(env);
    const portNum = Number(port.trim());
    const body: sdk.WorkspaceCreateRequest = {
      name: name.trim(),
      image: img.image,
      poolName: u.pool,
      unitName: u.unit,
      description: description.trim() || undefined,
      containerPort: Number.isFinite(portNum) && portNum > 0 ? portNum : undefined,
      env: envVars.length ? envVars : undefined,
      volumes: mountPath.trim()
        ? [{ mountPath: mountPath.trim(), size: volSize.trim() || undefined }]
        : undefined,
    };
    create.mutate(body, { onSuccess: onClose });
  };

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="新建工作区"
      sub="交互式开发容器 · 隶属当前租户"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            disabled={!name.trim() || create.isPending}
            onClick={submit}
          >
            {create.isPending ? "创建中…" : "创建工作区"}
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
            className="input"
            placeholder="开发调试环境"
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

      <FieldsetTitle n={2}>镜像</FieldsetTitle>
      <div className="field">
        <label>
          开发镜像 <span className="req">*</span>
        </label>
        <div className="pick-grid">
          {WS_IMAGES.map((o, i) => (
            <div
              key={o.image}
              className={"pick" + (i === imageIdx ? " on" : "")}
              style={{ display: "flex", alignItems: "center", gap: 10 }}
              onClick={() => setImageIdx(i)}
            >
              <svg className="brand" style={{ width: 22, height: 22, flex: "none" }}>
                <use href={`#${o.brand}`} />
              </svg>
              <div>
                <div className="p-title">{o.image}</div>
                <div className="p-spec">{o.spec}</div>
              </div>
            </div>
          ))}
        </div>
      </div>

      <FieldsetTitle n={3}>资源选择</FieldsetTitle>
      <div className="field">
        <label>
          资源单元 <span className="req">*</span>
        </label>
        <div className="pick-grid">
          {WS_UNITS.map((o, i) => (
            <div
              key={o.unit}
              className={"pick" + (i === unitIdx ? " on" : "")}
              onClick={() => setUnitIdx(i)}
            >
              <div className="p-title">{o.unit}</div>
              <div className="p-spec">
                {o.pool} · {o.spec}
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="form-grid" style={{ marginTop: "var(--space-4)" }}>
        <div className="field">
          <label>
            容器端口 <span className="req">*</span>
          </label>
          <input
            className="input mono"
            inputMode="numeric"
            placeholder="8888"
            value={port}
            onChange={(e) => setPort(e.target.value)}
          />
          <span className="help">工作区主进程监听的端口。</span>
        </div>
      </div>

      <FieldsetTitle n={4}>环境变量</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>环境变量</label>
          <textarea
            className="textarea"
            placeholder={"HF_HOME=/data/hf\nCUDA_VISIBLE_DEVICES=0"}
            value={env}
            onChange={(e) => setEnv(e.target.value)}
          />
          <span className="help">每行一个 KEY=VALUE，注入到工作区容器。</span>
        </div>
      </div>

      <FieldsetTitle n={5}>数据卷</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>数据卷</label>
          <div className="vol-list">
            <div className="vol-row">
              <input
                className="input mono"
                placeholder="50Gi"
                aria-label="数据卷大小"
                value={volSize}
                onChange={(e) => setVolSize(e.target.value)}
              />
              <input
                className="input mono"
                placeholder="挂载路径"
                aria-label="挂载路径"
                value={mountPath}
                onChange={(e) => setMountPath(e.target.value)}
              />
            </div>
          </div>
          <span className="help">留空挂载路径则不创建数据卷。</span>
        </div>
      </div>
    </Drawer>
  );
}
