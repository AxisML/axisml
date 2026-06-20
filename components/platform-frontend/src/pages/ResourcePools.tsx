import { useState } from "react";
import { useResourcePools } from "@/api/hooks";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { FieldsetTitle } from "@/components/forms";

interface PoolRow {
  name: string;
  desc: string;
  selectors: { text: string; mono?: boolean }[];
  units: number;
  created: string;
  // delete behavior variants from prototype
  del:
    | { kind: "plain" }
    | { kind: "blocked"; info: string }
    | { kind: "confirm"; desc: string; info: string };
}

// Faithful demo rows from prototype/resource-pools.html — rendered when the
// backend (contract-only shell) returns no items.
const FALLBACK: PoolRow[] = [
  {
    name: "gpu-a100",
    desc: "A100 训练池",
    selectors: [{ text: "gpu.product=A100", mono: true }, { text: "+1" }],
    units: 3,
    created: "2026-03-08",
    del: {
      kind: "blocked",
      info: "池内 3 个资源单元将随资源池级联删除（不阻断）：a100-1x-large / a100-4x-xlarge / a100-8x-xlarge-ib",
    },
  },
  {
    name: "gpu-h100",
    desc: "H100 训练/推理池",
    selectors: [
      { text: "product=H100", mono: true },
      { text: "network=ib", mono: true },
      { text: "+1" },
    ],
    units: 4,
    created: "2026-04-02",
    del: { kind: "plain" },
  },
  {
    name: "gpu-l40s",
    desc: "L40S 推理池",
    selectors: [{ text: "gpu.product=L40S", mono: true }],
    units: 2,
    created: "2026-04-19",
    del: {
      kind: "confirm",
      desc: "该池暂无活跃负载引用。",
      info: "池内 2 个资源单元将随资源池级联删除：l40s-1x / l40s-2x",
    },
  },
  {
    name: "cpu-large",
    desc: "大内存 CPU 池",
    selectors: [
      { text: "arch=amd64", mono: true },
      { text: "memory-tier=high", mono: true },
    ],
    units: 1,
    created: "2026-01-22",
    del: { kind: "plain" },
  },
  {
    name: "cpu-arm-edge",
    desc: "ARM 边缘推理池",
    selectors: [{ text: "arch=arm64", mono: true }],
    units: 0,
    created: "2026-05-15",
    del: { kind: "plain" },
  },
];

const GearIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
    <circle cx="12" cy="12" r="3" />
    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
  </svg>
);

const TrashIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
    <path d="M3 6h18" />
    <path d="M8 6V4h8v2" />
    <path d="M19 6l-1 14H6L5 6" />
    <path d="M10 11v6M14 11v6" />
  </svg>
);

type DrawerKind =
  | { kind: "pool" }
  | { kind: "manage"; pool: string; desc: string }
  | { kind: "unit"; pool: string; editing?: UnitData };

export default function ResourcePools() {
  const { data } = useResourcePools();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<DrawerKind | null>(null);

  const rows: PoolRow[] =
    data?.items?.map((p) => ({
      name: p.name,
      desc: p.description ?? "",
      selectors: Object.entries(p.nodeSelector ?? {}).map(([k, v]) => ({ text: `${k}=${v}`, mono: true })),
      units: p.units?.length ?? 0,
      created: p.createdAt,
      del: { kind: "plain" },
    })) ?? FALLBACK;

  const openManage = (r: PoolRow) => setDrawer({ kind: "manage", pool: r.name, desc: r.desc });

  const onDelete = (r: PoolRow) => {
    if (r.del.kind === "blocked") {
      confirm({
        title: `确定删除资源池 ${r.name}？`,
        info: r.del.info,
        block: (
          <div>
            5 个活跃任务、2 个活跃服务正在引用本池
            <ul>
              <li className="mono">llm-lab / train-llm-7b-12</li>
              <li className="mono">llm-lab / svc-chat-api</li>
              <li>…等 5 项</li>
            </ul>
            请先清空活跃负载后重试。
          </div>
        ),
        blocked: true,
      });
    } else if (r.del.kind === "confirm") {
      confirm({
        title: `确定删除资源池 ${r.name}？`,
        desc: r.del.desc,
        info: r.del.info,
        okLabel: "确认删除",
        toast: `资源池 ${r.name} 已删除`,
      });
    } else {
      confirm({ title: `确定删除资源池 ${r.name}？` });
    }
  };

  return (
    <main className="page">
      <div className="breadcrumb">
        <span>系统管理</span>
        <span className="sep">/</span>
        <span>资源池管理</span>
      </div>
      <div className="page-head">
        <div>
          <h1>资源池管理</h1>
          <p className="sub">
            统一调度和管理计算、存储等资源池，提升资源利用率与任务运行稳定性。支持资源分配、监控与隔离，保障多业务场景下的高效运行。
          </p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setDrawer({ kind: "pool" })}>
            <Icon name="plus" />
            新建资源池
          </button>
        </div>
      </div>

      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="关键字搜索" />
        </div>
        <button className="btn btn-ghost">重置</button>
      </div>

      <div className="panel">
        <div className="table-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>名称</th>
                <th>描述</th>
                <th>节点选择器</th>
                <th className="num-col">资源单元</th>
                <th>创建时间</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.name}>
                  <td>
                    <a
                      className="t-name mono"
                      href="#"
                      onClick={(e) => {
                        e.preventDefault();
                        openManage(r);
                      }}
                    >
                      {r.name}
                    </a>
                  </td>
                  <td>{r.desc}</td>
                  <td>
                    <div className="chip-row">
                      {r.selectors.map((s) => (
                        <span className={s.mono ? "tag mono" : "tag"} key={s.text}>
                          {s.text}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="num-col">
                    <a
                      className="link"
                      href="#"
                      onClick={(e) => {
                        e.preventDefault();
                        openManage(r);
                      }}
                    >
                      {r.units}
                    </a>
                  </td>
                  <td className="muted mono">{r.created}</td>
                  <td>
                    <div className="row-actions">
                      <button className="act" title="管理" aria-label="管理" onClick={() => openManage(r)}>
                        <GearIcon />
                      </button>
                      <button className="act act-danger" title="删除" aria-label="删除" onClick={() => onDelete(r)}>
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
          <span>共 {rows.length} 个资源池</span>
          <div className="pages">
            <span className="pg">‹</span>
            <span className="pg on">1</span>
            <span className="pg">›</span>
          </div>
          <span>每页 20 条</span>
        </div>
      </div>

      {drawer?.kind === "pool" && <PoolDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "manage" && (
        <ManagePoolDrawer
          pool={drawer.pool}
          desc={drawer.desc}
          onNewUnit={() => setDrawer({ kind: "unit", pool: drawer.pool })}
          onEditUnit={(u) => setDrawer({ kind: "unit", pool: drawer.pool, editing: u })}
          onClose={() => setDrawer(null)}
        />
      )}
      {drawer?.kind === "unit" && (
        <UnitFormDrawer pool={drawer.pool} editing={drawer.editing} onClose={() => setDrawer(null)} />
      )}
    </main>
  );
}

// ── Tolerations list (add/remove rows; Exists operator disables value) ────────
interface TolRow {
  id: number;
  key: string;
  op: string;
  value: string;
  effect: string;
}
let tolSeq = 0;
function blankTol(): TolRow {
  return { id: ++tolSeq, key: "", op: "Equal", value: "", effect: "NoSchedule" };
}

function TolList({ initial }: { initial?: TolRow[] }) {
  const [rows, setRows] = useState<TolRow[]>(() => initial ?? [blankTol()]);
  const add = () => setRows((r) => [...r, blankTol()]);
  // Keep at least one editable row (matches the prototype: app.js refuses to
  // delete the last vol-row), so the form always shows a toleration template.
  const remove = (id: number) => setRows((r) => (r.length <= 1 ? r : r.filter((x) => x.id !== id)));
  const update = (id: number, patch: Partial<TolRow>) =>
    setRows((r) => r.map((x) => (x.id === id ? { ...x, ...patch } : x)));

  return (
    <>
      <div className="tol-head">
        <span>key</span>
        <span>operator</span>
        <span>value</span>
        <span>effect</span>
        <span />
      </div>
      <div className="vol-list tol-list">
        {rows.map((row) => (
          <div className="vol-row" key={row.id}>
            <input
              className="input mono"
              placeholder="污点键 如 nvidia.com/gpu"
              aria-label="key"
              value={row.key}
              onChange={(e) => update(row.id, { key: e.target.value })}
            />
            <select
              className="input"
              aria-label="operator"
              value={row.op}
              onChange={(e) => update(row.id, { op: e.target.value })}
            >
              <option>Equal</option>
              <option>Exists</option>
            </select>
            <input
              className="input mono"
              placeholder="如 true"
              aria-label="value"
              value={row.value}
              disabled={row.op === "Exists"}
              onChange={(e) => update(row.id, { value: e.target.value })}
            />
            <select
              className="input"
              aria-label="effect"
              value={row.effect}
              onChange={(e) => update(row.id, { effect: e.target.value })}
            >
              <option>NoSchedule</option>
              <option>PreferNoSchedule</option>
              <option>NoExecute</option>
            </select>
            <button type="button" className="icon-btn" title="移除" onClick={() => remove(row.id)}>
              <Icon name="x" />
            </button>
          </div>
        ))}
      </div>
      <a className="link vol-add" role="button" tabIndex={0} onClick={add}>
        <Icon name="plus" />
        添加容忍
      </a>
    </>
  );
}

function PoolDrawer({ onClose }: { onClose: () => void }) {
  const { toast } = useUI();
  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title="新建资源池"
      sub="按节点标签划分一组计算资源 · 创建后维护资源单元"
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("资源池已创建，请添加资源单元");
              onClose();
            }}
          >
            创建资源池
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
          <input className="input mono" placeholder="gpu-a100" />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" placeholder="用途说明（可选）" />
        </div>
      </div>

      <FieldsetTitle n={2}>节点调度</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>节点选择器（K=V）</label>
          <div className="chip-row">
            <span className="tag mono">gpu.product=A100 ✕</span>
            <span className="tag mono" style={{ borderStyle: "dashed", color: "var(--muted)" }}>
              + 添加
            </span>
          </div>
        </div>

        <div className="field full">
          <label>容忍配置（tolerations）</label>
          <TolList />
        </div>
      </div>
    </Drawer>
  );
}

// ── Resource units shown inside the manage drawer ─────────────────────────────
interface UnitData {
  name: string;
  cpu: string;
  mem: string;
  gpu: string;
  spec: string;
}
const MANAGE_UNITS: UnitData[] = [
  { name: "a100-1x-large", cpu: "8", mem: "64", gpu: "1", spec: "1×A100 · 8 vCPU · 64 GiB" },
  { name: "a100-4x-xlarge", cpu: "32", mem: "256", gpu: "4", spec: "4×A100 · 32 vCPU · 256 GiB" },
  { name: "a100-8x-xlarge-ib", cpu: "64", mem: "512", gpu: "8", spec: "8×A100 · 64 vCPU · 512 GiB" },
];

function ManagePoolDrawer({
  pool,
  desc,
  onNewUnit,
  onEditUnit,
  onClose,
}: {
  pool: string;
  desc: string;
  onNewUnit: () => void;
  onEditUnit: (u: UnitData) => void;
  onClose: () => void;
}) {
  const { toast } = useUI();
  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title={<span className="mono">{pool}</span>}
      sub={`${desc} · 管理基本信息、节点调度与资源单元`}
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            onClick={() => {
              toast("资源池已保存");
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
          <label>名称</label>
          <input className="input mono" defaultValue={pool} readOnly aria-readonly="true" />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" defaultValue={desc} />
        </div>
      </div>

      <FieldsetTitle n={2}>节点调度</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>节点选择器（K=V）</label>
          <div className="chip-row">
            <span className="tag mono">gpu.product=A100 ✕</span>
            <span className="tag mono">arch=amd64 ✕</span>
            <span className="tag mono" style={{ borderStyle: "dashed", color: "var(--muted)" }}>
              + 添加
            </span>
          </div>
        </div>

        <div className="field full">
          <label>容忍配置（tolerations）</label>
          <TolList
            initial={[{ id: ++tolSeq, key: "nvidia.com/gpu", op: "Exists", value: "", effect: "NoSchedule" }]}
          />
        </div>
      </div>

      <FieldsetTitle n={3}>资源单元</FieldsetTitle>
      <div className="unit-grid">
        {MANAGE_UNITS.map((u) => (
          <div className="unit-card" key={u.name}>
            <div className="uc-top">
              <span className="uc-name">{u.name}</span>
              <div className="uc-act">
                <button className="act" aria-label="编辑" onClick={() => onEditUnit(u)} />
                <button className="act act-danger" aria-label="删除" />
              </div>
            </div>
            <div className="uc-spec">{u.spec}</div>
          </div>
        ))}
        <button type="button" className="unit-add" title="新建资源单元" aria-label="新建资源单元" onClick={onNewUnit}>
          <Icon name="plus" />
        </button>
      </div>
    </Drawer>
  );
}

// ── Resource-unit form drawer (new + edit reuse) with requests/limits lock ────
function UnitFormDrawer({ pool, editing, onClose }: { pool: string; editing?: UnitData; onClose: () => void }) {
  const { toast } = useUI();
  const [lock, setLock] = useState(true);
  const [reqCpu, setReqCpu] = useState(editing?.cpu ?? "");
  const [reqMem, setReqMem] = useState(editing?.mem ?? "");
  const [limCpu, setLimCpu] = useState(editing?.cpu ?? "");
  const [limMem, setLimMem] = useState(editing?.mem ?? "");
  const [gpu, setGpu] = useState(editing?.gpu ?? "");

  const onReqCpu = (v: string) => {
    setReqCpu(v);
    if (lock) setLimCpu(v);
  };
  const onReqMem = (v: string) => {
    setReqMem(v);
    if (lock) setLimMem(v);
  };
  const toggleLock = (on: boolean) => {
    setLock(on);
    if (on) {
      setLimCpu(reqCpu);
      setLimMem(reqMem);
    }
  };

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title={editing ? "编辑资源单元" : "新建资源单元"}
      sub={`${pool} · 定义一种可被任务申请的资源规格`}
      footer={
        <>
          <span className="grow" />
          <button className="btn" onClick={onClose}>
            取消
          </button>
          <button
            className="btn btn-primary"
            onClick={() => {
              toast(editing ? "资源单元已保存" : "资源单元已创建");
              onClose();
            }}
          >
            {editing ? "保存" : "创建资源单元"}
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
          <input className="input mono" placeholder="a100-1x-large" defaultValue={editing?.name ?? ""} />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" placeholder="规格用途说明（可选），如：单卡训练 / 4 卡分布式" />
        </div>
      </div>

      <FieldsetTitle
        n={2}
        extra={
          <label className="rm-lock">
            <input type="checkbox" checked={lock} onChange={(e) => toggleLock(e.target.checked)} />
            <span>limits 与 requests 保持一致</span>
          </label>
        }
      >
        资源规格
      </FieldsetTitle>
      <div className="res-matrix">
        <div className="rm-grid">
          <span className="rm-h" />
          <span className="rm-h">requests</span>
          <span className="rm-h">limits</span>
          <span className="rm-name">
            CPU <span className="req">*</span>
          </span>
          <div className="input-affix">
            <input
              inputMode="numeric"
              placeholder="8"
              aria-label="CPU requests"
              value={reqCpu}
              onChange={(e) => onReqCpu(e.target.value)}
            />
            <span className="suf">核</span>
          </div>
          <div className="input-affix">
            <input
              inputMode="numeric"
              placeholder="8"
              aria-label="CPU limits"
              value={limCpu}
              disabled={lock}
              onChange={(e) => setLimCpu(e.target.value)}
            />
            <span className="suf">核</span>
          </div>
          <span className="rm-name">
            内存 <span className="req">*</span>
          </span>
          <div className="input-affix">
            <input
              inputMode="numeric"
              placeholder="64"
              aria-label="内存 requests"
              value={reqMem}
              onChange={(e) => onReqMem(e.target.value)}
            />
            <span className="suf">GiB</span>
          </div>
          <div className="input-affix">
            <input
              inputMode="numeric"
              placeholder="64"
              aria-label="内存 limits"
              value={limMem}
              disabled={lock}
              onChange={(e) => setLimMem(e.target.value)}
            />
            <span className="suf">GiB</span>
          </div>
          <span className="rm-name">GPU</span>
          <div className="input-affix">
            <input
              inputMode="numeric"
              placeholder="1"
              aria-label="GPU 卡数"
              value={gpu}
              onChange={(e) => setGpu(e.target.value)}
            />
            <span className="suf">卡</span>
          </div>
          <span className="rm-eq">requests = limits</span>
        </div>
      </div>

      <FieldsetTitle n={3}>节点调度</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>额外节点选择器（K=V）</label>
          <div className="chip-row">
            <span className="tag mono" style={{ borderStyle: "dashed", color: "var(--muted)" }}>
              + 添加
            </span>
          </div>
        </div>
        <div className="field full">
          <label>容忍配置（tolerations）</label>
          <TolList />
        </div>
      </div>
    </Drawer>
  );
}
