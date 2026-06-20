import { useState } from "react";
import { Link } from "react-router-dom";
import { useJobs } from "@/api/hooks";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";
import { Drawer } from "@/components/Drawer";
import { RunBar, type RunState } from "@/components/RunBar";
import { PickGrid, FieldsetTitle, VolList } from "@/components/forms";

interface JobRow {
  name: string;
  desc: string;
  runs: RunState[];
  runCount: number;
  owner: string;
  updated: string;
  deletable?: boolean;
}

// Faithful demo rows from prototype/jobs.html — rendered when the backend
// (contract-only shell) returns no items.
const FALLBACK: JobRow[] = [
  { name: "train-llm-7b", desc: "LLaMA-7B 全参微调", runs: ["ok", "fail", "ok", "run", "none"], runCount: 4, owner: "张伟", updated: "2 天前" },
  { name: "eval-recall", desc: "召回模型离线评估", runs: ["none", "none", "fail", "ok", "ok"], runCount: 3, owner: "李娜", updated: "5 天前" },
  { name: "data-clean-etl", desc: "训练数据清洗", runs: ["ok", "ok", "fail", "ok", "fail"], runCount: 7, owner: "陈曦", updated: "6 小时前" },
  { name: "sft-baseline", desc: "SFT 基线训练", runs: ["none", "none", "none", "none", "none"], runCount: 0, owner: "王磊", updated: "1 小时前", deletable: true },
];

type DrawerMode = "new" | "run" | "edit";

export default function Jobs() {
  const { data } = useJobs();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; name?: string } | null>(null);

  const rows: JobRow[] =
    data?.items?.map((j) => ({
      name: j.name,
      desc: j.description ?? j.displayName ?? "",
      runs: ["none", "none", "none", "none", "none"],
      runCount: 0,
      owner: j.owner ?? "—",
      updated: j.updatedAt ?? j.createdAt ?? "",
    })) ?? FALLBACK;

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
                      {r.deletable ? (
                        <button
                          className="act act-danger"
                          aria-label="删除"
                          onClick={() =>
                            confirm({
                              title: `删除 Job ${r.name}？`,
                              desc: "该 Job 暂无运行记录。删除后模板不可恢复；若存在历史 Run 将一并级联软删。",
                              info: "有活跃 Run 时删除会被阻断（409 job-has-active-runs）。",
                              okLabel: "确认删除",
                              toast: `Job ${r.name} 已删除`,
                            })
                          }
                        />
                      ) : (
                        <button className="act" aria-label="编辑" onClick={() => setDrawer({ mode: "edit", name: r.name })} />
                      )}
                    </div>
                  </td>
                </tr>
              ))}
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
const CMD = `torchrun --nproc_per_node=4 train.py \\
  --model_name llama-7b --lr 2e-5 --epochs 3 \\
  --batch_size 16 --data /data/sft.jsonl`;

function JobDrawer({ mode, name, onClose }: { mode: DrawerMode; name?: string; onClose: () => void }) {
  const { toast } = useUI();
  const title = mode === "new" ? "新建 Job" : mode === "run" ? "触发运行" : "编辑 Job";
  const sub =
    mode === "new" ? "保存即写模板，不触发运行" : <span className="mono">{name ?? "train-llm-7b"}</span>;
  const locked = mode === "run";
  const submit =
    mode === "new"
      ? { label: "保存模板", toast: "Job 模板已保存" }
      : mode === "run"
        ? { label: "确认运行", toast: `已创建运行 ${name ?? "train-llm-7b"}-13` }
        : { label: "保存", toast: "Job 已保存" };

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
            名称 {mode !== "run" && <span className="req">*</span>}
          </label>
          <input
            className="input mono"
            placeholder="train-llm-7b"
            defaultValue={mode === "new" ? "" : name ?? "train-llm-7b"}
            disabled={locked}
          />
          {mode !== "run" && <span className="help">DNS-1123，同时作为显示名</span>}
        </div>
        <div className="field full">
          <label>描述</label>
          <textarea className="textarea" placeholder="任务用途说明" disabled={locked} defaultValue={mode === "new" ? "" : "LLaMA-7B 全参微调"} />
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
          <input className="input num" defaultValue="4" />
        </div>
      </div>

      <FieldsetTitle n={4}>启动命令 / 环境变量</FieldsetTitle>
      <div className="form-grid">
        <div className="field full">
          <label>启动命令 / 参数</label>
          <textarea className="textarea" defaultValue={CMD} />
          {mode !== "run" && <span className="help">训练超参即命令 / 参数 / 环境变量，平台不单独建模。</span>}
        </div>
        <div className="field full">
          <label>环境变量</label>
          <textarea className="textarea" style={{ minHeight: 60 }} defaultValue={"WANDB_DISABLED=true\nNCCL_DEBUG=INFO"} />
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
            <input className="input num" defaultValue="86400" />
          </div>
          <div className="field">
            <label>重试次数</label>
            <input className="input num" defaultValue="2" />
          </div>
        </div>
      </details>
    </Drawer>
  );
}
