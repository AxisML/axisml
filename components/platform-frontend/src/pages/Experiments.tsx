import { useState } from "react";
import { Link } from "react-router-dom";
import { useExperiments } from "@/api/hooks";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { RunBar, type RunState } from "@/components/RunBar";
import { PickGrid, FieldsetTitle, VolList } from "@/components/forms";

interface ExpRow {
  name: string;
  desc: string;
  runs: RunState[];
  runLabel: string;
  runCount: number;
  owner: string;
  updated: string;
}

// Faithful demo rows from prototype/experiments.html — rendered when the backend
// (contract-only shell) returns no items.
const FALLBACK: ExpRow[] = [
  {
    name: "llama3-sft-lr-sweep",
    desc: "LLaMA3-8B SFT 学习率扫描",
    runs: ["ok", "fail", "ok", "run", "none"],
    runLabel: "查看 llama3-sft-lr-sweep 历史运行",
    runCount: 8,
    owner: "张伟",
    updated: "12 分钟前",
  },
  {
    name: "resnet-aug-search",
    desc: "ResNet 数据增强搜索",
    runs: ["ok", "ok", "fail", "ok", "ok"],
    runLabel: "查看 resnet-aug-search 历史运行",
    runCount: 12,
    owner: "李娜",
    updated: "2 小时前",
  },
  {
    name: "qwen-vl-finetune",
    desc: "Qwen2-VL 指令微调",
    runs: ["none", "fail", "ok", "ok", "ok"],
    runLabel: "查看 qwen-vl-finetune 历史运行",
    runCount: 5,
    owner: "陈曦",
    updated: "1 天前",
  },
  {
    name: "embed-contrastive",
    desc: "向量模型对比学习",
    runs: ["none", "none", "ok", "ok", "pend"],
    runLabel: "查看 embed-contrastive 历史运行",
    runCount: 3,
    owner: "王磊",
    updated: "3 天前",
  },
];

type DrawerMode = "new" | "run" | "edit";

export default function Experiments() {
  const { data } = useExperiments();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; name?: string } | null>(null);

  const rows: ExpRow[] =
    data?.items?.map((e) => ({
      name: e.name,
      desc: e.description ?? e.displayName ?? "",
      runs: ["none", "none", "none", "none", "none"],
      runLabel: `查看 ${e.name} 历史运行`,
      runCount: 0,
      owner: e.owner ?? "—",
      updated: e.updatedAt ?? e.createdAt ?? "",
    })) ?? FALLBACK;

  return (
    <main className="page">
      <div className="breadcrumb">
        <span>训练中心</span>
        <span className="sep">/</span>
        <span>实验</span>
      </div>
      <div className="page-head">
        <div>
          <h1>实验</h1>
          <p className="sub">
            统一管理实验创建、运行、对比与追踪流程，帮助团队高效沉淀实验结果。支持实验过程可视化与版本留痕，让模型迭代更清晰可控。
          </p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setDrawer({ mode: "new" })}>
            <Icon name="plus" />
            新建实验
          </button>
        </div>
      </div>

      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="实验名搜索" />
        </div>
        <select className="select">
          <option>创建人：全部</option>
          <option>张伟</option>
          <option>李娜</option>
          <option>陈曦</option>
          <option>王磊</option>
        </select>
        <button className="btn btn-ghost">重置</button>
      </div>

      <div className="panel">
        <div className="table-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>名称</th>
                <th>最近运行状态</th>
                <th className="num-col">运行数</th>
                <th>创建人</th>
                <th>更新时间</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.name}>
                  <td>
                    <Link className="t-name mono" to={`/experiments/${r.name}`}>
                      {r.name}
                    </Link>
                    <div className="t-sub">{r.desc}</div>
                  </td>
                  <td>
                    <RunBar states={r.runs} to={`/experiments/${r.name}`} label={r.runLabel} />
                  </td>
                  <td className="num-col">{r.runCount}</td>
                  <td>{r.owner}</td>
                  <td className="muted">{r.updated}</td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="act act-run"
                        title="运行"
                        aria-label="运行"
                        onClick={() => setDrawer({ mode: "run", name: r.name })}
                      />
                      <Link className="act" to={`/experiments/${r.name}`} title="详情" aria-label="详情" />
                      <button
                        className="act"
                        title="编辑"
                        aria-label="编辑"
                        onClick={() => setDrawer({ mode: "edit", name: r.name })}
                      />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="pagination">
          <span>共 {rows.length} 个实验</span>
          <div className="pages">
            <span className="pg">‹</span>
            <span className="pg on">1</span>
            <span className="pg">›</span>
          </div>
          <span>每页 20 条</span>
        </div>
      </div>

      {drawer && <ExpDrawer mode={drawer.mode} name={drawer.name} onClose={() => setDrawer(null)} />}
    </main>
  );
}

const IMAGES = [
  { title: "pytorch:2.3-cu121", spec: "PyTorch 训练镜像" },
  { title: "megatron:24.05", spec: "Megatron-LM 训练镜像" },
];
const UNITS = [
  { title: "a100-4x-xlarge", spec: "4×A100 · 32 vCPU · 256 GiB" },
  { title: "a100-8x-xlarge-ib", spec: "8×A100 · IB · 64 vCPU · 512 GiB" },
];
const CMD_TPL = `torchrun --nproc_per_node=4 sft.py \\
  --base llama3-8b-base --lr {{lr}} --epochs 3`;
const CMD_RUN = `torchrun --nproc_per_node=4 sft.py \\
  --base llama3-8b-base --lr 2e-5 --epochs 3`;

// Data-volume rows for the experiment drawer (team-datasets → /data,
// ckpt-store → /output), using the shared VolList's `initial` prop so add/remove
// actually work instead of being faked with toasts.
const EXP_VOLS = [
  { options: ["team-datasets · 1 TiB", "ckpt-store · 500 GiB", "新建数据卷…"], path: "/data" },
  { options: ["ckpt-store · 500 GiB", "team-datasets · 1 TiB", "新建数据卷…"], path: "/output" },
];

function ExpDrawer({ mode, name, onClose }: { mode: DrawerMode; name?: string; onClose: () => void }) {
  const { toast } = useUI();
  const expName = name ?? "llama3-sft-lr-sweep";
  const title = mode === "new" ? "新建实验" : mode === "run" ? "触发运行" : "编辑实验";
  const sub =
    mode === "new" ? "保存模板，不触发运行" : <span className="mono">{expName}</span>;
  const locked = mode === "run";
  const submit =
    mode === "new"
      ? { label: "创建实验", toast: "实验已创建" }
      : mode === "run"
        ? { label: "确认运行", toast: `已触发运行 ${expName}-9` }
        : { label: "保存", toast: "实验已保存" };
  const cmd = mode === "run" ? CMD_RUN : CMD_TPL;

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
            实验名 {mode !== "run" && <span className="req">*</span>}
          </label>
          <input
            className="input mono"
            placeholder="llama3-sft-lr-sweep"
            defaultValue={mode === "new" ? "" : expName}
            disabled={locked}
          />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="围绕同一训练目标，扫描不同学习率以择优"
            disabled={locked}
            defaultValue={mode === "new" ? "" : "LLaMA3-8B SFT 学习率扫描"}
          />
        </div>
      </div>

      <FieldsetTitle n={2}>镜像</FieldsetTitle>
      <div className="field">
        <label>
          训练镜像 <span className="req">*</span>
        </label>
        <PickGrid options={IMAGES} />
      </div>

      <FieldsetTitle n={3}>资源选择</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            资源池 <span className="req">*</span>
          </label>
          <select className="input">
            <option>gpu-a100 · A100 训练池</option>
            <option>gpu-h100 · H100 训练/推理池</option>
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
          <input className="input num" defaultValue="2" />
        </div>
      </div>

      <FieldsetTitle n={4}>启动命令 / 环境变量</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            {mode === "run" ? "启动命令（本次运行可覆盖超参）" : "启动命令（超参写在命令 / 参数中）"}
          </label>
          <textarea className="textarea" defaultValue={cmd} />
          <span className="help">
            {mode === "run" ? "将 {{lr}} 替换为本次运行的取值。" : "触发运行时可对 lr 等参数做本次覆盖。"}
          </span>
        </div>
        <div className="field full">
          <label>环境变量</label>
          <textarea
            className="textarea"
            style={{ minHeight: 60 }}
            placeholder={mode === "new" ? "WANDB_DISABLED=true\nNCCL_DEBUG=INFO" : undefined}
            defaultValue={mode === "new" ? "" : "WANDB_DISABLED=true\nNCCL_DEBUG=INFO"}
          />
          {mode !== "run" && <span className="help">每行一个 KEY=VALUE，注入到训练容器。</span>}
        </div>
      </div>

      <FieldsetTitle n={5}>数据卷</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            数据卷 <span className="req">*</span>
          </label>
          <VolList initial={EXP_VOLS} />
          {mode === "new" && (
            <span className="help">挂载训练数据与产出目录，每次运行（Run）继承此挂载，支持挂载多个。</span>
          )}
        </div>
      </div>

      <details style={{ marginTop: "var(--space-6)" }}>
        <summary style={{ cursor: "pointer", fontWeight: 600, fontSize: 13, color: "var(--accent)" }}>
          高级设置
        </summary>
        <div className="form-grid" style={{ marginTop: "var(--space-4)" }}>
          <div className="field">
            <label>超时 (s)</label>
            <input className="input num" defaultValue="172800" />
          </div>
          <div className="field">
            <label>重试次数</label>
            <input className="input num" defaultValue="1" />
          </div>
        </div>
      </details>
    </Drawer>
  );
}
