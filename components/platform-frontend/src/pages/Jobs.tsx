import { useState } from "react";
import { Link } from "react-router-dom";
import { useJobs } from "@/api/hooks";
import { useApiMutation, tenantHeader } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { TableState } from "@/components/states";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { RunBar, type RunState } from "@/components/RunBar";
import { FieldsetTitle, VolList } from "@/components/forms";

interface JobRow {
  name: string;
  desc: string;
  runs: RunState[];
  runCount: number;
  owner: string;
  updated: string;
  deletable?: boolean;
}

type DrawerMode = "new" | "run" | "edit";

export default function Jobs() {
  const q = useJobs();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; name?: string } | null>(null);

  const delJob = useApiMutation(
    (name: string) => sdk.deleteJob({ path: { name }, headers: tenantHeader() }),
    { invalidate: [["jobs"]], success: "Job 已删除" },
  );

  const rows: JobRow[] =
    q.data?.items?.map((j) => ({
      name: j.name,
      desc: j.description ?? j.displayName ?? "",
      runs: ["none", "none", "none", "none", "none"],
      runCount: 0,
      owner: j.owner ?? "—",
      updated: j.updatedAt ?? j.createdAt ?? "",
    })) ?? [];

  const onDelete = (r: JobRow) =>
    confirm({
      title: `删除 Job ${r.name}？`,
      desc: "删除后模板不可恢复；若存在历史 Run 将一并级联软删。",
      info: "有活跃 Run 时删除会被阻断（409 job-has-active-runs）。",
      okLabel: "确认删除",
      danger: true,
      onConfirm: () => delJob.mutate(r.name),
    });

  return (
    <main className="page">
      <div className="breadcrumb">
        <span>训练中心</span>
        <span className="sep">/</span>
        <span>自定义任务</span>
      </div>
      <div className="page-head">
        <div>
          <h1>自定义任务</h1>
          <p className="sub">
            支持按业务需求灵活创建自定义任务，适配多样化训练、评测与推理场景。通过可配置流程降低重复操作成本，提升任务执行效率。
          </p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setDrawer({ mode: "new" })}>
            <Icon name="plus" />
            新建 Job
          </button>
        </div>
      </div>

      <div className="toolbar">
        <div className="field-search">
          <Icon name="search" />
          <input placeholder="名称搜索" />
        </div>
        <select className="select">
          <option>创建人：全部</option>
          <option>张伟</option>
          <option>李娜</option>
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
                    <Link className="t-name mono" to={`/jobs/${r.name}`}>
                      {r.name}
                    </Link>
                    <div className="t-sub">{r.desc}</div>
                  </td>
                  <td>
                    <RunBar states={r.runs} to={`/jobs/${r.name}`} label={`${r.name} 最近运行`} />
                  </td>
                  <td className="num-col">{r.runCount}</td>
                  <td>{r.owner}</td>
                  <td className="muted">{r.updated}</td>
                  <td>
                    <div className="row-actions">
                      <button className="act act-run" aria-label="运行" onClick={() => setDrawer({ mode: "run", name: r.name })} />
                      <Link className="act" to={`/jobs/${r.name}`} aria-label="详情" />
                      <button className="act" aria-label="编辑" onClick={() => setDrawer({ mode: "edit", name: r.name })} />
                      <button className="act act-danger" aria-label="删除" onClick={() => onDelete(r)} />
                    </div>
                  </td>
                </tr>
              ))}
              <TableState q={q} cols={6} isEmpty={rows.length === 0} />
            </tbody>
          </table>
        </div>
        <div className="pagination">
          <span>共 {rows.length} 个 Job</span>
          <div className="pages">
            <span className="pg">‹</span>
            <span className="pg on">1</span>
            <span className="pg">›</span>
          </div>
          <span>每页 20 条</span>
        </div>
      </div>

      {drawer && <JobDrawer mode={drawer.mode} name={drawer.name} onClose={() => setDrawer(null)} />}
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
const POOLS = ["gpu-a100", "gpu-h100"];
const CMD = `torchrun --nproc_per_node=4 train.py \\
  --model_name llama-7b --lr 2e-5 --epochs 3 \\
  --batch_size 16 --data /data/sft.jsonl`;

// Controlled radio-card grid (mirrors PickGrid styling) so the form can submit
// the chosen value. PickGrid itself is uncontrolled and exposes no callback.
function ControlledPickGrid({
  options,
  value,
  onChange,
}: {
  options: { title: string; spec: string }[];
  value: string;
  onChange: (title: string) => void;
}) {
  return (
    <div className="pick-grid">
      {options.map((o) => (
        <div key={o.title} className={"pick" + (o.title === value ? " on" : "")} onClick={() => onChange(o.title)}>
          <div className="p-title">{o.title}</div>
          <div className="p-spec">{o.spec}</div>
        </div>
      ))}
    </div>
  );
}

// Parse "KEY=VALUE" lines into EnvVar[] (skips blank lines).
function parseEnv(text: string): sdk.EnvVar[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)
    .map((l) => {
      const [name, ...rest] = l.split("=");
      return { name: name.trim(), value: rest.join("=") };
    })
    .filter((e) => e.name);
}

// Split a multi-line / multi-word command into a shell-style command array.
function parseCommand(text: string): string[] {
  return text
    .split(/\s+/)
    .map((s) => s.trim())
    .filter(Boolean);
}

function JobDrawer({ mode, name: initialName, onClose }: { mode: DrawerMode; name?: string; onClose: () => void }) {
  const locked = mode === "run";

  const [name, setName] = useState(mode === "new" ? "" : initialName ?? "");
  const [description, setDescription] = useState("");
  const [image, setImage] = useState(IMAGES[0].title);
  const [poolName, setPoolName] = useState(POOLS[0]);
  const [unitName, setUnitName] = useState(UNITS[0].title);
  const [replicas, setReplicas] = useState("4");
  const [command, setCommand] = useState(CMD);
  const [env, setEnv] = useState("WANDB_DISABLED=true\nNCCL_DEBUG=INFO");
  const [timeout, setTimeoutS] = useState("86400");
  const [retries, setRetries] = useState("2");

  // Assemble a minimal-but-valid JobSpec from the controlled fields.
  const buildSpec = (): sdk.JobSpec => {
    const reps = Number(replicas);
    const role: sdk.MlRunRole = {
      name: "worker",
      replicas: Number.isFinite(reps) && reps > 0 ? reps : 1,
      template: {
        image: image.trim() || undefined,
        command: parseCommand(command),
        env: parseEnv(env),
      },
    };
    const deadline = Number(timeout);
    const backoff = Number(retries);
    return {
      backend: { name: "native", engine: "job" },
      poolName: poolName.trim() || undefined,
      unitName: unitName.trim() || undefined,
      roles: [role],
      runPolicy: {
        activeDeadlineSeconds: Number.isFinite(deadline) && deadline > 0 ? deadline : undefined,
        backoffLimit: Number.isFinite(backoff) && backoff >= 0 ? backoff : undefined,
      },
    };
  };

  const create = useApiMutation(
    (body: sdk.JobCreateInput) => sdk.createJob({ body, headers: tenantHeader() }),
    { invalidate: [["jobs"]], success: "Job 模板已保存" },
  );
  const update = useApiMutation(
    (vars: { name: string; body: sdk.JobPatchInput }) => sdk.updateJob({ path: { name: vars.name }, body: vars.body, headers: tenantHeader() }),
    { invalidate: [["jobs"]], success: "Job 已保存" },
  );
  const trigger = useApiMutation(
    (vars: { name: string; body: sdk.RunTriggerInput }) => sdk.triggerRun({ path: { name: vars.name }, body: vars.body, headers: tenantHeader() }),
    { invalidate: [["jobs"]], success: "已创建运行" },
  );

  const pending = create.isPending || update.isPending || trigger.isPending;

  const submit = () => {
    if (mode === "new") {
      create.mutate(
        {
          name: name.trim(),
          displayName: name.trim() || undefined,
          description: description.trim() || undefined,
          spec: buildSpec(),
        },
        { onSuccess: onClose },
      );
    } else if (mode === "edit") {
      update.mutate(
        { name: name.trim(), body: { description: description.trim() || undefined, spec: buildSpec() } },
        { onSuccess: onClose },
      );
    } else {
      // run: trigger a new Run, optionally overriding pool/unit/role inputs.
      trigger.mutate(
        {
          name: name.trim(),
          body: {
            poolName: poolName.trim() || undefined,
            unitName: unitName.trim() || undefined,
            roles: [{ name: "worker", args: parseCommand(command), env: parseEnv(env) }],
          },
        },
        { onSuccess: onClose },
      );
    }
  };

  const title = mode === "new" ? "新建 Job" : mode === "run" ? "触发运行" : "编辑 Job";
  const sub =
    mode === "new" ? "保存即写模板，不触发运行" : <span className="mono">{initialName ?? name}</span>;
  const submitLabel =
    mode === "new"
      ? pending
        ? "保存中…"
        : "保存模板"
      : mode === "run"
        ? pending
          ? "运行中…"
          : "确认运行"
        : pending
          ? "保存中…"
          : "保存";

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
            disabled={!name.trim() || pending}
            onClick={submit}
          >
            {submitLabel}
          </button>
        </>
      }
    >
      <FieldsetTitle n={1}>基本信息</FieldsetTitle>
      <div className="form-grid">
        <div className="field">
          <label>
            名称 {mode !== "run" && <span className="req">*</span>}
          </label>
          <input
            className="input mono"
            placeholder="train-llm-7b"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={locked || mode === "edit"}
          />
          {mode !== "run" && <span className="help">DNS-1123，同时作为显示名</span>}
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea
            className="textarea"
            placeholder="任务用途说明"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={locked}
          />
        </div>
      </div>

      <FieldsetTitle n={2}>镜像</FieldsetTitle>
      <div className="field">
        <label>
          训练镜像 <span className="req">*</span>
        </label>
        <ControlledPickGrid options={IMAGES} value={image} onChange={setImage} />
      </div>

      <FieldsetTitle n={3}>资源选择</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>
            资源池 <span className="req">*</span>
          </label>
          <select className="input" value={poolName} onChange={(e) => setPoolName(e.target.value)}>
            {POOLS.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>
      </div>
      <div className="field" style={{ marginTop: "var(--space-4)" }}>
        <label>
          资源单元 <span className="req">*</span>
        </label>
        <ControlledPickGrid options={UNITS} value={unitName} onChange={setUnitName} />
      </div>
      <div className="form-grid" style={{ marginTop: "var(--space-4)" }}>
        <div className="field">
          <label>
            副本数 <span className="req">*</span>
          </label>
          <input className="input num" value={replicas} onChange={(e) => setReplicas(e.target.value)} />
        </div>
      </div>

      <FieldsetTitle n={4}>启动命令 / 环境变量</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>启动命令 / 参数</label>
          <textarea className="textarea" value={command} onChange={(e) => setCommand(e.target.value)} />
          {mode !== "run" && <span className="help">训练超参即命令 / 参数 / 环境变量，平台不单独建模。</span>}
        </div>
        <div className="field full">
          <label>环境变量</label>
          <textarea
            className="textarea"
            style={{ minHeight: 60 }}
            value={env}
            onChange={(e) => setEnv(e.target.value)}
          />
          {mode !== "run" && <span className="help">每行一个 KEY=VALUE，注入到训练容器。</span>}
        </div>
      </div>

      <FieldsetTitle n={5}>数据卷</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>数据卷</label>
          <VolList />
        </div>
      </div>

      <details style={{ marginTop: "var(--space-6)" }}>
        <summary style={{ cursor: "pointer", fontWeight: 600, fontSize: 13, color: "var(--accent)" }}>
          高级设置
        </summary>
        <div className="form-grid" style={{ marginTop: "var(--space-4)" }}>
          <div className="field">
            <label>超时 (s)</label>
            <input className="input num" value={timeout} onChange={(e) => setTimeoutS(e.target.value)} />
          </div>
          <div className="field">
            <label>重试次数</label>
            <input className="input num" value={retries} onChange={(e) => setRetries(e.target.value)} />
          </div>
        </div>
      </details>
    </Drawer>
  );
}
