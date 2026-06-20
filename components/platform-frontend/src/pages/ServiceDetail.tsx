import { useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { Segmented } from "@/components/Segmented";
import { Tabs } from "@/components/Tabs";
import { PickGrid, FieldsetTitle } from "@/components/forms";

export default function ServiceDetail() {
  const { name } = useParams<{ name: string }>();
  const svcName = name ?? "svc-chat-api";
  const { toast, confirm } = useUI();
  const [drawer, setDrawer] = useState<"edit" | "scale" | null>(null);

  return (
    <main className="page">
      <Link className="back-link" to="/services">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M15 18l-6-6 6-6" />
        </svg>
        返回服务列表
      </Link>

      <div className="page-head">
        <div>
          <h1 className="detail-title">
            {svcName}{" "}
            <span className="spill ok">
              <span className="dot" />
              就绪
            </span>
          </h1>
          <div className="detail-sub">对话推理服务</div>
        </div>
        <div className="actions">
          <button className="btn" onClick={() => setDrawer("edit")}>
            编辑
          </button>
          <button className="btn btn-primary" onClick={() => setDrawer("scale")}>
            扩缩容
          </button>
          <button className="btn" onClick={() => toast("服务正在停止…")}>
            停止
          </button>
          <button
            className="btn btn-danger"
            onClick={() =>
              confirm({
                title: `删除服务 ${svcName}？`,
                desc: "将下线服务并回收副本，访问路由一并移除。该操作不可恢复。",
                okLabel: "确认删除",
                toast: `服务 ${svcName} 已删除`,
              })
            }
          >
            删除
          </button>
        </div>
      </div>

      <Tabs
        tabs={[
          { key: "info", label: "概览", content: <InfoPane /> },
          { key: "mon", label: "监控", content: <MonPane /> },
          { key: "pods", label: "实例", content: <PodsPane /> },
          { key: "log", label: "日志", content: <LogPane /> },
          { key: "ev", label: "事件", content: <EventPane /> },
        ]}
      />

      {drawer === "edit" && <EditSvcDrawer name={svcName} onClose={() => setDrawer(null)} />}
      {drawer === "scale" && <ScaleDrawer name={svcName} onClose={() => setDrawer(null)} />}
    </main>
  );
}

// ── 概览 ─────────────────────────────────────────────────────────────────────
function InfoPane() {
  const { toast } = useUI();
  return (
    <div className="panel">
      <div className="panel-head">
        <h3>配置信息</h3>
      </div>
      <div className="panel-body">
        <dl className="kv kv-lg">
          <dt>名称</dt>
          <dd>
            <span className="cchip">svc-chat-api</span>
          </dd>
          <dt>描述</dt>
          <dd>线上对话接口，承接 App 端流量</dd>
          <dt>模型版本</dt>
          <dd>
            <span className="cchip">llama3-8b-sft@v4</span>
          </dd>
          <dt>镜像</dt>
          <dd>
            <span className="cchip">triton-infer@sha256:5c4d3e</span>
          </dd>
          <dt>资源池</dt>
          <dd>
            <span className="cchip">gpu-h100 · H100 推理池</span>
          </dd>
          <dt>资源单元</dt>
          <dd>
            <span className="cchip">h100-1x-large</span>
          </dd>
          <dt>副本</dt>
          <dd>
            <span className="mono">2 / 2</span> 就绪
          </dd>
          <dt>端口</dt>
          <dd>
            <div className="chip-row">
              <span className="cchip">http : 8000</span>
              <span className="cchip">grpc : 8001</span>
              <span className="cchip">metrics : 8002</span>
            </div>
          </dd>
          <dt>访问地址</dt>
          <dd>
            <span className="cchip">/services/team-a/chat-api/</span>
            <button
              className="icon-mini"
              title="复制"
              aria-label="复制访问地址"
              onClick={() => toast("访问地址已复制")}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <rect x="9" y="9" width="11" height="11" rx="2" />
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
              </svg>
            </button>
          </dd>
          <dt>创建人</dt>
          <dd>
            张伟 · <span className="mono muted">zhang.wei</span>
          </dd>
          <dt>创建时间</dt>
          <dd className="mono">2026-05-28 11:09:44</dd>
        </dl>
      </div>
    </div>
  );
}

// ── 监控 ─────────────────────────────────────────────────────────────────────
function MonPane() {
  return (
    <>
      <div className="toolbar">
        <span className="hint">来自 Prometheus · 按副本聚合</span>
        <div className="grow" />
        <Segmented options={["5m", "1h", "24h"]} defaultValue="1h" />
      </div>
      <div className="grid cols-2">
        <div className="mcard">
          <div className="mc-head">
            <span className="mc-title">QPS</span>
            <span className="mc-val">128 req/s</span>
          </div>
          <svg className="mc-chart chart" viewBox="0 0 600 132" preserveAspectRatio="none">
            <line className="grid-line" x1="0" y1="44" x2="600" y2="44" />
            <line className="grid-line" x1="0" y1="88" x2="600" y2="88" />
            <defs>
              <linearGradient id="g-qps" x1="0" x2="0" y1="0" y2="1">
                <stop offset="0" stopColor="var(--info)" stopOpacity="0.18" />
                <stop offset="1" stopColor="var(--info)" stopOpacity="0" />
              </linearGradient>
            </defs>
            <path
              d="M0 105 L50 96 L100 88 L150 92 L200 74 L250 78 L300 66 L350 70 L400 58 L450 64 L500 52 L550 58 L600 50 L600 132 L0 132 Z"
              fill="url(#g-qps)"
            />
            <path
              d="M0 105 L50 96 L100 88 L150 92 L200 74 L250 78 L300 66 L350 70 L400 58 L450 64 L500 52 L550 58 L600 50"
              fill="none"
              stroke="var(--info)"
              strokeWidth="2"
            />
          </svg>
        </div>
        <div className="mcard">
          <div className="mc-head">
            <span className="mc-title">延迟</span>
            <div className="legend">
              <span>
                <i style={{ background: "var(--muted)" }} />
                p50
              </span>
              <span>
                <i style={{ background: "var(--info)" }} />
                p95
              </span>
              <span>
                <i style={{ background: "var(--danger)" }} />
                p99
              </span>
            </div>
          </div>
          <svg className="mc-chart chart" viewBox="0 0 600 132" preserveAspectRatio="none">
            <line className="grid-line" x1="0" y1="44" x2="600" y2="44" />
            <line className="grid-line" x1="0" y1="88" x2="600" y2="88" />
            <path
              d="M0 84 L60 72 L120 88 L180 66 L240 92 L300 62 L360 86 L420 60 L480 90 L540 68 L600 80"
              fill="none"
              stroke="var(--muted)"
              strokeWidth="1.6"
            />
            <path
              d="M0 76 L60 62 L120 82 L180 56 L240 86 L300 52 L360 80 L420 50 L480 84 L540 60 L600 72"
              fill="none"
              stroke="var(--info)"
              strokeWidth="1.8"
            />
            <path
              d="M0 70 L60 54 L120 78 L180 48 L240 82 L300 44 L360 76 L420 44 L480 80 L540 54 L600 66"
              fill="none"
              stroke="var(--danger)"
              strokeWidth="1.8"
            />
          </svg>
        </div>
        <div className="mcard">
          <div className="mc-head">
            <span className="mc-title">错误率 (5xx)</span>
            <span className="mc-val">0.04%</span>
          </div>
          <svg className="mc-chart chart" viewBox="0 0 600 132" preserveAspectRatio="none">
            <line className="grid-line" x1="0" y1="44" x2="600" y2="44" />
            <line className="grid-line" x1="0" y1="88" x2="600" y2="88" />
            <defs>
              <linearGradient id="g-err" x1="0" x2="0" y1="0" y2="1">
                <stop offset="0" stopColor="var(--danger)" stopOpacity="0.16" />
                <stop offset="1" stopColor="var(--danger)" stopOpacity="0" />
              </linearGradient>
            </defs>
            <path
              d="M0 112 L60 108 L120 104 L180 112 L240 92 L300 84 L360 72 L420 64 L480 84 L540 102 L600 106 L600 132 L0 132 Z"
              fill="url(#g-err)"
            />
            <path
              d="M0 112 L60 108 L120 104 L180 112 L240 92 L300 84 L360 72 L420 64 L480 84 L540 102 L600 106"
              fill="none"
              stroke="var(--danger)"
              strokeWidth="1.9"
            />
          </svg>
        </div>
        <div className="mcard">
          <div className="mc-head">
            <span className="mc-title">CPU 利用率</span>
            <span className="mc-val">62%</span>
          </div>
          <svg className="mc-chart chart" viewBox="0 0 600 132" preserveAspectRatio="none">
            <line className="grid-line" x1="0" y1="44" x2="600" y2="44" />
            <line className="grid-line" x1="0" y1="88" x2="600" y2="88" />
            <defs>
              <linearGradient id="g-cpu" x1="0" x2="0" y1="0" y2="1">
                <stop offset="0" stopColor="var(--info)" stopOpacity="0.18" />
                <stop offset="1" stopColor="var(--info)" stopOpacity="0" />
              </linearGradient>
            </defs>
            <path
              d="M0 80 L60 72 L120 92 L180 58 L240 74 L300 54 L360 70 L420 48 L480 66 L540 46 L600 58 L600 132 L0 132 Z"
              fill="url(#g-cpu)"
            />
            <path
              d="M0 80 L60 72 L120 92 L180 58 L240 74 L300 54 L360 70 L420 48 L480 66 L540 46 L600 58"
              fill="none"
              stroke="var(--info)"
              strokeWidth="2"
            />
          </svg>
        </div>
      </div>
    </>
  );
}

// ── 实例 (Pods) ──────────────────────────────────────────────────────────────
function PodsPane() {
  return (
    <div className="panel">
      <div className="table-wrap">
        <table className="tbl">
          <thead>
            <tr>
              <th>POD 名称</th>
              <th>阶段</th>
              <th>节点</th>
              <th className="num-col">重启</th>
              <th>启动时间</th>
              <th style={{ textAlign: "right" }}>操作</th>
            </tr>
          </thead>
          <tbody>
            <PodRow name="svc-chat-api-7b9c-x4d2" node="node-h100-01" started="2026-06-13 02:11:30" />
            <PodRow name="svc-chat-api-7b9c-m8k1" node="node-h100-03" started="2026-06-13 02:11:30" />
          </tbody>
        </table>
      </div>
    </div>
  );
}

function PodRow({ name, node, started }: { name: string; node: string; started: string }) {
  return (
    <tr>
      <td className="mono">{name}</td>
      <td>
        <span className="spill run">
          <span className="dot" />
          Running
        </span>
      </td>
      <td className="mono muted">{node}</td>
      <td className="num-col">0</td>
      <td className="muted mono">{started}</td>
      <td>
        <div className="row-actions">
          <button className="act" title="日志" aria-label="日志">
            日志
          </button>
          <button className="act" title="事件" aria-label="事件">
            事件
          </button>
        </div>
      </td>
    </tr>
  );
}

// ── 日志 ─────────────────────────────────────────────────────────────────────
function LogPane() {
  return (
    <div className="panel">
      <div className="panel-body">
        <div className="log-bar">
          <div className="pod-pick">
            <span className="pp-tag">POD</span>
            <select>
              <option>svc-chat-api-7b9c-x4d2</option>
              <option>svc-chat-api-7b9c-m8k1</option>
            </select>
          </div>
          <div className="grow" />
          <FollowToggle />
        </div>
        <pre className="logbox">
          <span className="l-time">02:11:42</span>
          <span className="l-info">[I]</span> Triton server started, model chat-7b loaded on GPU 0
          {"\n"}
          <span className="l-time">02:11:43</span>
          <span className="l-info">[I]</span> HTTP/gRPC endpoints ready on :8000 / :8001
          {"\n"}
          <span className="l-time">09:14:22</span>
          <span className="l-info">[I]</span> inference id=8841 tokens=512 latency=86ms
          {"\n"}
          <span className="l-time">09:14:23</span>
          <span className="l-info">[I]</span> inference id=8842 tokens=128 latency=41ms
          {"\n"}
          <span className="l-time">09:15:01</span>
          <span className="l-warn">[W]</span> request id=8901 queued 120ms (batch full)
          {"\n"}
          <span className="l-time">09:15:44</span>
          <span className="l-info">[I]</span> inference id=8950 tokens=256 latency=63ms
        </pre>
      </div>
    </div>
  );
}

// ── 事件 ─────────────────────────────────────────────────────────────────────
function EventPane() {
  return (
    <>
      <div className="panel">
        <div className="panel-head">
          <h3>服务事件</h3>
        </div>
        <div className="panel-body">
          <div className="timeline">
            <TlItem name="ScaledUp" time="2026-06-13 02:11:30" desc="服务扩容至 2 副本" />
            <TlItem name="RouteReady" time="2026-06-13 02:11:46" desc="对外路由 /services/team-a/chat-api/ 已生效" />
            <TlItem name="Pulled" time="2026-06-13 02:11:14" desc="镜像 triton-infer@sha256:5c4d3e 拉取完成" muted />
          </div>
        </div>
      </div>
      <div className="panel" style={{ marginTop: "var(--space-5)" }}>
        <div className="panel-head">
          <h3>实例事件</h3>
        </div>
        <div className="panel-body">
          <div className="log-bar">
            <div className="pod-pick">
              <span className="pp-tag">POD</span>
              <select>
                <option>svc-chat-api-7b9c-x4d2</option>
                <option>svc-chat-api-7b9c-m8k1</option>
              </select>
            </div>
            <div className="grow" />
            <FollowToggle />
          </div>
          <div className="timeline" style={{ marginTop: "var(--space-5)" }}>
            <TlItem name="Scheduled" time="2026-06-13 02:11:14" desc="x4d2 分配到节点 node-h100-01" />
            <TlItem name="Pulled" time="2026-06-13 02:11:14" desc="镜像 triton-infer@sha256:5c4d3e 拉取完成" />
            <TlItem name="Started" time="2026-06-13 02:11:30" desc="容器启动，加载模型 chat-7b 到 GPU 0" />
            <TlItem name="Ready" time="2026-06-13 02:11:43" desc="就绪探针通过，HTTP/gRPC 端点开始接流" />
          </div>
        </div>
      </div>
    </>
  );
}

function TlItem({ name, time, desc, muted }: { name: string; time: string; desc: string; muted?: boolean }) {
  return (
    <div className={"tl-item" + (muted ? " is-muted" : "")}>
      <span className="tl-dot" />
      <div className="tl-head">
        <span className="tl-name">{name}</span>
        <span className="tl-tag">NORMAL</span>
        <span className="tl-time">{time}</span>
      </div>
      <div className="tl-desc">{desc}</div>
    </div>
  );
}

function FollowToggle() {
  const [on, setOn] = useState(true);
  return (
    <label className="follow">
      实时跟随 <button className={"toggle" + (on ? " on" : "")} aria-label="实时跟随" onClick={() => setOn((v) => !v)} />
    </label>
  );
}

// ── drawers ──────────────────────────────────────────────────────────────────
function PortRow({ name, port }: { name: string; port: string }) {
  return (
    <div className="vol-row">
      <input className="input mono" defaultValue={name} placeholder="名称，如 http" aria-label="端口名" maxLength={15} />
      <input className="input mono" defaultValue={port} placeholder="端口号" aria-label="端口号" inputMode="numeric" />
      <button type="button" className="icon-btn" title="移除">
        <Icon name="x" />
      </button>
    </div>
  );
}

const UNITS = [
  { title: "h100-1x-large", spec: "1×H100 · 16 vCPU · 128 GiB" },
  { title: "l40s-1x", spec: "1×L40S · 8 vCPU · 64 GiB" },
];

function EditSvcDrawer({ name, onClose }: { name: string; onClose: () => void }) {
  const { toast } = useUI();
  const sub: ReactNode = <span className="mono">{name}</span>;
  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="编辑服务"
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
              toast("服务配置已保存");
              onClose();
            }}
          >
            保存
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
          <input className="input mono" defaultValue={name} />
          <span className="help">用于在列表与详情中展示</span>
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" defaultValue="线上对话接口，承接 App 端流量" />
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
            <option>triton-infer:24.05</option>
            <option>vllm-serve:0.5.1</option>
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
          <div className="vol-list">
            <PortRow name="http" port="8000" />
            <PortRow name="grpc" port="8001" />
            <PortRow name="metrics" port="8002" />
          </div>
          <a className="link vol-add" role="button" tabIndex={0}>
            <Icon name="plus" />
            添加端口
          </a>
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
          <input className="input mono" defaultValue="/services/team-a/chat-api/" />
        </div>
      </div>
    </Drawer>
  );
}

function RouteToggle() {
  const [on, setOn] = useState(true);
  return <button className={"toggle" + (on ? " on" : "")} aria-label="启用对外路由" onClick={() => setOn((v) => !v)} />;
}

function ScaleDrawer({ name, onClose }: { name: string; onClose: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer
      open
      onClose={onClose}
      title="扩缩容"
      sub={<span className="mono">{name}</span>}
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
      <div className="field">
        <label>目标副本数</label>
        <input className="input num" defaultValue="3" />
        <span className="help">当前 2 / 2 就绪 · 配额上限 gpu-h100 max=8</span>
      </div>
    </Drawer>
  );
}
