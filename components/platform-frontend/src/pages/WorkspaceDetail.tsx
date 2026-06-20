import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useUI } from "@/app/ui";
import { Tabs } from "@/components/Tabs";
import { Drawer } from "@/components/Drawer";

// Faithful port of prototype/workspace-detail.html. Detail page → static demo
// content; the :name param drives the title. Brand glyphs (Jupyter / VS Code)
// inlined verbatim and referenced via <use href="#ic-..."/>.
function BrandDefs() {
  return (
    <svg width="0" height="0" style={{ position: "absolute" }} aria-hidden="true" focusable="false">
      <symbol id="ic-jupyter" viewBox="0 0 24 24">
        <path fill="#F37726" d="M7.157 22.201A1.784 1.784 0 0 1 5.374 24a1.784 1.784 0 0 1-1.784-1.799 1.784 1.784 0 0 1 1.784-1.799 1.784 1.784 0 0 1 1.783 1.799zM20.582 1.427a1.415 1.415 0 0 1-1.415 1.428 1.415 1.415 0 0 1-1.416-1.428A1.415 1.415 0 0 1 19.167 0a1.415 1.415 0 0 1 1.415 1.427zM4.992 3.336A1.781 1.781 0 0 1 3.21 5.135 1.781 1.781 0 0 1 1.427 3.336 1.781 1.781 0 0 1 3.21 1.537a1.781 1.781 0 0 1 1.782 1.799zM12 18.694c-3.945 0-7.394-1.417-9.191-3.506a9.799 9.799 0 0 0 18.382 0c-1.797 2.089-5.246 3.506-9.191 3.506zM12 5.306c3.945 0 7.394 1.417 9.191 3.506a9.799 9.799 0 0 0-18.382 0C4.606 6.723 8.055 5.306 12 5.306z" />
      </symbol>
      <symbol id="ic-vscode" viewBox="0 0 24 24">
        <path fill="#007ACC" d="M23.15 2.587L18.21.21a1.494 1.494 0 0 0-1.705.29l-9.46 8.63-4.12-3.128a.999.999 0 0 0-1.276.057L.327 7.261A1 1 0 0 0 .326 8.74L3.899 12 .326 15.26a1 1 0 0 0 .001 1.479L1.65 17.94a.999.999 0 0 0 1.276.057l4.12-3.128 9.46 8.63a1.492 1.492 0 0 0 1.704.29l4.942-2.377A1.5 1.5 0 0 0 24 20.06V3.939a1.5 1.5 0 0 0-.85-1.352zm-5.146 14.861L10.826 12l7.178-5.448v10.896z" />
      </symbol>
    </svg>
  );
}

export default function WorkspaceDetail() {
  const { name = "ws-dev-zhang" } = useParams();
  const { toast, confirm } = useUI();
  const [edit, setEdit] = useState(false);

  return (
    <main className="page">
      <BrandDefs />
      <Link className="back-link" to="/workspaces">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M15 18l-6-6 6-6" />
        </svg>
        返回工作区列表
      </Link>

      <div className="page-head">
        <div>
          <h1 className="detail-title">
            {name}{" "}
            <span className="spill run">
              <span className="dot" />
              运行中
            </span>
          </h1>
          <div className="detail-sub">开发调试环境 · 创建人 张伟</div>
        </div>
        <div className="actions">
          <a className="btn" href="#" onClick={(e) => { e.preventDefault(); toast("正在打开 Jupyter…"); }}>
            <svg className="brand" style={{ width: 15, height: 15 }}>
              <use href="#ic-jupyter" />
            </svg>
            打开 Jupyter
          </a>
          <a className="btn" href="#" onClick={(e) => { e.preventDefault(); toast("正在打开 VS Code…"); }}>
            <svg className="brand" style={{ width: 15, height: 15 }}>
              <use href="#ic-vscode" />
            </svg>
            VS Code
          </a>
          <button className="btn" onClick={() => toast("工作区已停止")}>
            停止
          </button>
          <button
            className="btn btn-danger"
            onClick={() =>
              confirm({
                title: `删除工作区 ${name}？`,
                desc: "工作区将被删除，环境内未持久化的内容会丢失。",
                info: (
                  <label style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 4 }}>
                    <input type="checkbox" defaultChecked /> 一并删除数据卷 PVC（50 GiB）
                  </label>
                ),
                okLabel: "确认删除",
                toast: "工作区已删除",
              })
            }
          >
            删除
          </button>
        </div>
      </div>

      <Tabs
        tabs={[
          { key: "info", label: "概览", content: <InfoPane onEdit={() => setEdit(true)} /> },
          { key: "log", label: "日志", content: <LogPane /> },
          { key: "ev", label: "事件", content: <EventsPane /> },
        ]}
      />

      {edit && <EditDrawer name={name} onClose={() => setEdit(false)} />}
    </main>
  );
}

function InfoPane({ onEdit }: { onEdit: () => void }) {
  const { toast } = useUI();
  return (
    <div className="panel">
      <div className="panel-head">
        <h3>配置信息</h3>
        <button className="btn btn-sm" onClick={onEdit}>
          编辑
        </button>
      </div>
      <div className="panel-body">
        <dl className="kv kv-lg">
          <dt>名称</dt>
          <dd>
            <span className="cchip">ws-dev-zhang</span>
          </dd>
          <dt>描述</dt>
          <dd>开发调试环境</dd>
          <dt>资源池</dt>
          <dd>
            <span className="cchip">cpu-medium · 通用 CPU 池</span>
          </dd>
          <dt>资源单元</dt>
          <dd>
            <span className="cchip">cpu-medium</span> 8 vCPU · 32 GiB
          </dd>
          <dt>镜像</dt>
          <dd>
            <span className="cchip">jupyter-ds:2024.3</span>
          </dd>
          <dt>访问地址</dt>
          <dd>
            <span className="cchip">/ws/llm-lab/ws-dev-zhang/</span>
            <button className="icon-mini" title="复制" aria-label="复制访问地址" onClick={() => toast("访问地址已复制")}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <rect x="9" y="9" width="11" height="11" rx="2" />
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
              </svg>
            </button>
          </dd>
          <dt>数据卷</dt>
          <dd>
            <span className="cchip">ws-dev-zhang-data</span> 50 GiB · standard-rwo
          </dd>
          <dt>挂载路径</dt>
          <dd>
            <span className="cchip">/workspace</span>
          </dd>
          <dt>环境变量</dt>
          <dd>
            <div className="chip-row">
              <span className="cchip">HF_HOME=/data/hf</span>
              <span className="cchip">JUPYTER_TOKEN=••••</span>
            </div>
          </dd>
          <dt>创建人</dt>
          <dd>
            张伟 · <span className="mono">2026-06-13</span>
          </dd>
        </dl>
      </div>
    </div>
  );
}

function LogPane() {
  const [follow, setFollow] = useState(true);
  return (
    <div className="panel">
      <div className="panel-body">
        <div className="log-bar">
          <div className="pod-pick">
            <span className="pp-tag">POD</span>
            <select>
              <option>ws-dev-zhang-0</option>
            </select>
          </div>
          <div className="grow" />
          <label className="follow">
            实时跟随{" "}
            <button className={"toggle" + (follow ? " on" : "")} aria-label="实时跟随" onClick={() => setFollow((f) => !f)} />
          </label>
        </div>
        <pre className="logbox">
          <span className="l-time">02:14:10</span>
          <span className="l-info">[I]</span> Jupyter server started on :8888{"\n"}
          <span className="l-time">02:14:11</span>
          <span className="l-info">[I]</span> Mounted PVC ws-dev-zhang-data at /workspace{"\n"}
          <span className="l-time">09:02:33</span>
          <span className="l-info">[I]</span> Kernel restarted by user{"\n"}
          <span className="l-time">09:31:18</span>
          <span className="l-warn">[W]</span> high memory usage 27.4 / 32 GiB{"\n"}
          <span className="l-time">09:48:12</span>
          <span className="l-info">[I]</span> Saved checkpoint train_sft.ipynb
        </pre>
      </div>
    </div>
  );
}

function EventsPane() {
  return (
    <div className="panel">
      <div className="panel-body">
        <div className="timeline">
          <div className="tl-item">
            <span className="tl-dot" />
            <div className="tl-head">
              <span className="tl-name">Started</span>
              <span className="tl-tag">NORMAL</span>
              <span className="tl-time">2026-06-13 02:14:10</span>
            </div>
            <div className="tl-desc">Pod ws-dev-zhang-0 scheduled to node-cpu-12 and started</div>
          </div>
          <div className="tl-item">
            <span className="tl-dot" />
            <div className="tl-head">
              <span className="tl-name">VolumeBound</span>
              <span className="tl-tag">NORMAL</span>
              <span className="tl-time">2026-06-13 02:14:02</span>
            </div>
            <div className="tl-desc">PVC ws-dev-zhang-data (50 GiB · standard-rwo) bound</div>
          </div>
          <div className="tl-item is-muted">
            <span className="tl-dot" />
            <div className="tl-head">
              <span className="tl-name">Created</span>
              <span className="tl-tag">NORMAL</span>
              <span className="tl-time">2026-06-13 02:13:58</span>
            </div>
            <div className="tl-desc">Workspace created from image jupyter-ds:2024.3</div>
          </div>
        </div>
      </div>
    </div>
  );
}

// Edit drawer (running workspace → only name editable).
function EditDrawer({ name, onClose }: { name: string; onClose: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer
      open
      onClose={onClose}
      title="编辑工作区"
      sub={`${name} · 运行中`}
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("工作区已更新");
              onClose();
            }}
          >
            保存修改
          </button>
        </>
      }
    >
      <div className="form-notice">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="9" />
          <path d="M12 8h.01M11 12h1v4h1" />
        </svg>
        <span>
          工作区运行中，仅可修改<b>名称</b>。镜像、资源、数据卷、环境变量需先<b>停止</b>工作区后才能修改。
        </span>
      </div>
      <div className="form-grid">
        <div className="field full">
          <label>
            名称 <span className="req">*</span>
          </label>
          <input className="input" defaultValue="开发调试环境" />
          <span className="help">用于在列表与详情中展示</span>
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" disabled defaultValue="开发调试用交互式容器" />
          <span className="help">需停机后修改</span>
        </div>
        <div className="field full">
          <label>镜像</label>
          <input className="input" defaultValue="jupyter-ds:2024.3" disabled />
          <span className="help">需停机后修改</span>
        </div>
        <div className="field full">
          <label>资源单元</label>
          <input className="input" defaultValue="cpu-medium / 1x · 8 vCPU · 32 GiB" disabled />
          <span className="help">需停机后修改</span>
        </div>
        <div className="field full">
          <label>数据卷</label>
          <div className="vol-list">
            <div className="vol-row" data-only="">
              <select className="input" disabled>
                <option>ws-dev-zhang-data（50 GiB · standard-rwo）</option>
              </select>
              <input className="input mono" defaultValue="/workspace" disabled />
              <button type="button" className="icon-btn" disabled aria-hidden="true">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M18 6 6 18M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
          <span className="help">需停机后修改</span>
        </div>
        <div className="field full">
          <label>环境变量</label>
          <textarea className="textarea" disabled defaultValue={"HF_HOME=/data/hf\nJUPYTER_TOKEN=••••"} />
          <span className="help">需停机后修改</span>
        </div>
      </div>
    </Drawer>
  );
}
