import { useState } from "react";
import { Link } from "react-router-dom";
import { useExperiments } from "@/api/hooks";
import { useApiMutation, tenantHeader } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { RunBar, type RunState } from "@/components/RunBar";
import { FieldsetTitle, VolList } from "@/components/forms";
import { TableState } from "@/components/states";

interface ExpRow {
  name: string;
  desc: string;
  displayName?: string;
  runs: RunState[];
  runLabel: string;
  runCount: number;
  owner: string;
  updated: string;
}

type DrawerMode = "new" | "run" | "edit";

const TrashIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
    <path d="M3 6h18" />
    <path d="M8 6V4h8v2" />
    <path d="M19 6l-1 14H6L5 6" />
    <path d="M10 11v6M14 11v6" />
  </svg>
);

export default function Experiments() {
  const q = useExperiments();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; name?: string } | null>(null);

  const del = useApiMutation(
    (name: string) => sdk.deleteExperiment({ path: { name }, headers: tenantHeader() }),
    { invalidate: [["experiments"]], success: "实验已删除" },
  );
  const trigger = useApiMutation(
    (name: string) => sdk.triggerExperimentRun({ path: { name }, body: {}, headers: tenantHeader() }),
    { invalidate: [["experiments"]], success: "已触发运行" },
  );

  const rows: ExpRow[] =
    q.data?.items?.map((e) => ({
      name: e.name,
      desc: e.description ?? e.displayName ?? "",
      displayName: e.displayName,
      runs: ["none", "none", "none", "none", "none"],
      runLabel: `查看 ${e.name} 历史运行`,
      runCount: 0,
      owner: e.owner ?? "—",
      updated: e.updatedAt ?? e.createdAt ?? "",
    })) ?? [];

  const onDelete = (r: ExpRow) =>
    confirm({
      title: `确定删除实验 ${r.name}？`,
      desc: "删除后该实验及其终态运行将一并移除，且不可恢复。",
      okLabel: "确认删除",
      danger: true,
      onConfirm: () => del.mutate(r.name),
    });

  const onRun = (r: ExpRow) =>
    confirm({
      title: `确定触发运行 ${r.name}？`,
      desc: "将按实验模板创建一次新的运行（Run）。",
      okLabel: "确认运行",
      onConfirm: () => trigger.mutate(r.name),
    });

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
                        onClick={() => onRun(r)}
                      />
                      <Link className="act" to={`/experiments/${r.name}`} title="详情" aria-label="详情" />
                      <button
                        className="act"
                        title="编辑"
                        aria-label="编辑"
                        onClick={() => setDrawer({ mode: "edit", name: r.name })}
                      />
                      <button
                        className="act act-danger"
                        title="删除"
                        aria-label="删除"
                        onClick={() => onDelete(r)}
                      >
                        <TrashIcon />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              <TableState q={q} cols={6} isEmpty={rows.length === 0} />
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

const IMAGE_OPTIONS = [
  { value: "pytorch:2.3-cu121", label: "pytorch:2.3-cu121 · PyTorch 训练镜像" },
  { value: "megatron:24.05", label: "megatron:24.05 · Megatron-LM 训练镜像" },
];
const POOL_OPTIONS = [
  { value: "gpu-a100", label: "gpu-a100 · A100 训练池" },
  { value: "gpu-h100", label: "gpu-h100 · H100 训练/推理池" },
];
const UNIT_OPTIONS = [
  { value: "a100-4x-xlarge", label: "a100-4x-xlarge · 4×A100 · 32 vCPU · 256 GiB" },
  { value: "a100-8x-xlarge-ib", label: "a100-8x-xlarge-ib · 8×A100 · IB · 64 vCPU · 512 GiB" },
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

// Parse a textarea command into argv tokens (whitespace-split, dropping the
// shell line-continuation backslashes the placeholder uses for readability).
function parseCommand(s: string): string[] | undefined {
  const toks = s
    .replace(/\\\s*\n/g, " ")
    .split(/\s+/)
    .filter((t) => t && t !== "\\");
  return toks.length ? toks : undefined;
}

// Parse "KEY=VALUE" lines into env vars (skips blanks).
function parseEnv(s: string): { name: string; value: string }[] | undefined {
  const out: { name: string; value: string }[] = [];
  for (const line of s.split("\n")) {
    const t = line.trim();
    if (!t) continue;
    const [k, ...rest] = t.split("=");
    const name = k.trim();
    if (name) out.push({ name, value: rest.join("=").trim() });
  }
  return out.length ? out : undefined;
}

function ExpDrawer({ mode, name, onClose }: { mode: DrawerMode; name?: string; onClose: () => void }) {
  const expName = name ?? "llama3-sft-lr-sweep";
  const title = mode === "new" ? "新建实验" : mode === "run" ? "触发运行" : "编辑实验";
  const sub =
    mode === "new" ? "保存模板，不触发运行" : <span className="mono">{expName}</span>;
  const locked = mode === "run";

  const [expNameInput, setExpNameInput] = useState(mode === "new" ? "" : expName);
  const [description, setDescription] = useState(mode === "new" ? "" : "LLaMA3-8B SFT 学习率扫描");
  const [image, setImage] = useState(IMAGE_OPTIONS[0].value);
  const [pool, setPool] = useState(POOL_OPTIONS[0].value);
  const [unit, setUnit] = useState(UNIT_OPTIONS[0].value);
  const [replicas, setReplicas] = useState("2");
  const [command, setCommand] = useState(mode === "run" ? CMD_RUN : CMD_TPL);
  const [env, setEnv] = useState(mode === "new" ? "" : "WANDB_DISABLED=true\nNCCL_DEBUG=INFO");
  const [deadline, setDeadline] = useState("172800");
  const [backoff, setBackoff] = useState("1");

  const create = useApiMutation(
    (body: sdk.ExperimentCreateInput) => sdk.createExperiment({ body, headers: tenantHeader() }),
    { invalidate: [["experiments"]], success: "实验已创建" },
  );
  const update = useApiMutation(
    (body: sdk.ExperimentPatchInput) => sdk.updateExperiment({ path: { name: expName }, body, headers: tenantHeader() }),
    { invalidate: [["experiments"]], success: "实验已保存" },
  );
  const triggerRun = useApiMutation(
    (body: sdk.RunTriggerInput) => sdk.triggerExperimentRun({ path: { name: expName }, body, headers: tenantHeader() }),
    { invalidate: [["experiments"]], success: `已触发运行 ${expName}` },
  );

  // Compose the JobSpec common to create + edit. `native/job` is the default
  // MLRun backend; the role template carries image + launch command, and the
  // pool/unit selectors resolve to the ResourcePool addressing the backend uses.
  const buildSpec = (): sdk.JobSpec => ({
    backend: { name: "native", engine: "job" },
    poolName: pool || undefined,
    unitName: unit || undefined,
    roles: [
      {
        name: "worker",
        replicas: Number(replicas) || 1,
        template: {
          image,
          command: parseCommand(command),
          env: parseEnv(env),
        },
      },
    ],
    runPolicy: {
      activeDeadlineSeconds: Number(deadline) || undefined,
      backoffLimit: Number(backoff) || undefined,
    },
  });

  const submit = () => {
    if (mode === "new") {
      const n = expNameInput.trim();
      if (!n) return;
      create.mutate(
        { name: n, description: description.trim() || undefined, spec: buildSpec() },
        { onSuccess: onClose },
      );
    } else if (mode === "edit") {
      update.mutate(
        { description: description.trim() || undefined, spec: buildSpec() },
        { onSuccess: onClose },
      );
    } else {
      // run: override the launch command for this Run only.
      triggerRun.mutate(
        {
          poolName: pool || undefined,
          unitName: unit || undefined,
          roles: [{ name: "worker", args: parseCommand(command) }],
        },
        { onSuccess: onClose },
      );
    }
  };

  const mut = mode === "new" ? create : mode === "edit" ? update : triggerRun;
  const submitLabel =
    mode === "new"
      ? mut.isPending
        ? "创建中…"
        : "创建实验"
      : mode === "run"
        ? mut.isPending
          ? "运行中…"
          : "确认运行"
        : mut.isPending
          ? "保存中…"
          : "保存";
  const disabled = mut.isPending || (mode === "new" && !expNameInput.trim());

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
          <button className="btn btn-primary" disabled={disabled} onClick={submit}>
            {submitLabel}
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
            value={expNameInput}
            onChange={(e) => setExpNameInput(e.target.value)}
            disabled={locked}
          />
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="围绕同一训练目标，扫描不同学习率以择优"
            disabled={locked}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
      </div>

      <FieldsetTitle n={2}>镜像</FieldsetTitle>
      <div className="field">
        <label>
          训练镜像 <span className="req">*</span>
        </label>
        <select
          className="input mono"
          value={image}
          disabled={locked}
          onChange={(e) => setImage(e.target.value)}
        >
          {IMAGE_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>

      <FieldsetTitle n={3}>资源选择</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            资源池 <span className="req">*</span>
          </label>
          <select
            className="input"
            value={pool}
            disabled={locked}
            onChange={(e) => setPool(e.target.value)}
          >
            {POOL_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
      </div>
      <div className="field" style={{ marginTop: "var(--space-4)" }}>
        <label>
          资源单元 <span className="req">*</span>
        </label>
        <select
          className="input mono"
          value={unit}
          disabled={locked}
          onChange={(e) => setUnit(e.target.value)}
        >
          {UNIT_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>
      <div className="form-grid" style={{ marginTop: "var(--space-4)" }}>
        <div className="field">
          <label>
            副本数 <span className="req">*</span>
          </label>
          <input
            className="input num"
            value={replicas}
            disabled={locked}
            onChange={(e) => setReplicas(e.target.value)}
          />
        </div>
      </div>

      <FieldsetTitle n={4}>启动命令 / 环境变量</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            {mode === "run" ? "启动命令（本次运行可覆盖超参）" : "启动命令（超参写在命令 / 参数中）"}
          </label>
          <textarea
            className="textarea"
            value={command}
            onChange={(e) => setCommand(e.target.value)}
          />
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
            value={env}
            disabled={locked}
            onChange={(e) => setEnv(e.target.value)}
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
            <input
              className="input num"
              value={deadline}
              disabled={locked}
              onChange={(e) => setDeadline(e.target.value)}
            />
          </div>
          <div className="field">
            <label>重试次数</label>
            <input
              className="input num"
              value={backoff}
              disabled={locked}
              onChange={(e) => setBackoff(e.target.value)}
            />
          </div>
        </div>
      </details>
    </Drawer>
  );
}
