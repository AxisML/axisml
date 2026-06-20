import { Link, useParams } from "react-router-dom";
import { useUI } from "@/app/ui";
import { Tabs } from "@/components/Tabs";

// Faithful port of prototype/job-detail.html. Detail page → static demo content
// (no list hook); the :name param drives the title / run links.
export default function JobDetail() {
  const { name = "train-llm-7b" } = useParams();

  return (
    <main className="page">
      <Link className="back-link" to="/jobs">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M15 18l-6-6 6-6" />
        </svg>
        返回自定义任务
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
          <div className="detail-sub">LLaMA-7B 全参微调</div>
        </div>
        <div className="actions">
          <PageActions name={name} />
        </div>
      </div>

      <Tabs
        tabs={[
          { key: "info", label: "任务信息", content: <InfoPane /> },
          { key: "runs", label: "运行记录", count: 4, content: <RunsPane name={name} /> },
        ]}
      />
    </main>
  );
}

function PageActions({ name }: { name: string }) {
  const { toast, confirm } = useUI();
  return (
    <>
      <button className="btn btn-primary" onClick={() => toast(`已创建运行 ${name}-13`)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path d="M6 4l14 8-14 8z" />
        </svg>
        运行
      </button>
      <button
        className="btn btn-danger"
        onClick={() =>
          confirm({
            title: `删除 Job ${name}？`,
            desc: "删除后模板不可恢复；历史 Run 将一并级联软删。",
            info: "有活跃 Run 时删除会被阻断（409 job-has-active-runs）。",
            okLabel: "确认删除",
            toast: `Job ${name} 已删除`,
          })
        }
      >
        删除
      </button>
    </>
  );
}

function InfoPane() {
  const { toast } = useUI();
  return (
    <div className="panel">
      <div className="panel-head">
        <h3>任务信息</h3>
        <button className="btn btn-sm" onClick={() => toast("进入编辑（编辑只影响之后触发的 Run）")}>
          编辑
        </button>
      </div>
      <div className="panel-body">
        <dl className="kv kv-lg">
          <dt>名称</dt>
          <dd>
            <span className="cchip">train-llm-7b</span>
          </dd>
          <dt>描述</dt>
          <dd>LLaMA-7B 全参微调</dd>
          <dt>镜像</dt>
          <dd>
            <span className="cchip">pytorch:2.3-cu121</span>
          </dd>
          <dt>资源池</dt>
          <dd>
            <span className="cchip">gpu-a100 · A100 训练池</span>
          </dd>
          <dt>资源单元</dt>
          <dd>
            <span className="cchip">a100-4x-xlarge</span>
          </dd>
          <dt>副本数</dt>
          <dd>
            <span className="mono">4</span>
          </dd>
          <dt>数据卷</dt>
          <dd>
            <span className="cchip">training-data · 200 GiB → /data</span>
          </dd>
          <dt>运行策略</dt>
          <dd>
            超时 <span className="mono">24h</span> · 重试 <span className="mono">2</span>
          </dd>
          <dt>创建人</dt>
          <dd>
            张伟 · <span className="mono">2026-06-12</span>
          </dd>
        </dl>
        <div style={{ marginTop: "var(--space-6)", paddingTop: "var(--space-5)", borderTop: "1px solid var(--border-soft)" }}>
          <label className="muted" style={{ fontSize: 12 }}>
            启动命令
          </label>
          <pre className="logbox" style={{ maxHeight: "none", margin: "6px 0 18px" }}>
            {`torchrun --nproc_per_node=4 train.py \\
  --model_name llama-7b --lr 2e-5 \\
  --epochs 3 --batch_size 16 \\
  --data /data/sft.jsonl`}
          </pre>
          <label className="muted" style={{ fontSize: 12 }}>
            环境变量
          </label>
          <div className="chip-row" style={{ marginTop: 8 }}>
            <span className="cchip">WANDB_DISABLED=true</span>
            <span className="cchip">NCCL_DEBUG=INFO</span>
          </div>
        </div>
      </div>
    </div>
  );
}

interface RunRow {
  run: string;
  status: "running" | "success" | "failed";
  label: string;
  unit: string;
  replicas: number;
  owner: string;
  duration: string;
  cancel?: boolean;
}

const RUN_ROWS: RunRow[] = [
  { run: "train-llm-7b-12", status: "running", label: "运行中", unit: "gpu-a100/4x", replicas: 4, owner: "张伟", duration: "02:14:30", cancel: true },
  { run: "train-llm-7b-11", status: "success", label: "成功", unit: "gpu-a100/4x", replicas: 4, owner: "张伟", duration: "03:40:02" },
  { run: "train-llm-7b-10", status: "failed", label: "失败", unit: "gpu-a100/8x", replicas: 8, owner: "李娜", duration: "00:08:22" },
  { run: "train-llm-7b-9", status: "success", label: "成功", unit: "gpu-a100/4x", replicas: 4, owner: "张伟", duration: "03:52:11" },
];

function RunsPane({ name }: { name: string }) {
  const { confirm } = useUI();
  return (
    <>
      <div className="toolbar">
        <div className="field-search">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="11" cy="11" r="7" />
            <path d="m20 20-3.5-3.5" />
          </svg>
          <input placeholder="Run 名 / ID 搜索" />
        </div>
        <select className="select">
          <option>状态：全部</option>
          <option>运行中</option>
          <option>成功</option>
          <option>失败</option>
        </select>
        <select className="select">
          <option>触发人：全部</option>
          <option>张伟</option>
          <option>李娜</option>
        </select>
        <select className="select">
          <option>时间：近 7 天</option>
          <option>近 24 小时</option>
          <option>近 30 天</option>
          <option>全部</option>
        </select>
        <button className="btn btn-ghost">重置</button>
      </div>
      <div className="panel">
        <div className="table-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>Run</th>
                <th>状态</th>
                <th>资源单元</th>
                <th className="num-col">副本</th>
                <th>触发人</th>
                <th className="num-col">耗时</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {RUN_ROWS.map((r) => (
                <tr key={r.run}>
                  <td>
                    <Link className="t-name mono" to={`/jobs/${name}/runs/${r.run}`}>
                      {r.run}
                    </Link>
                  </td>
                  <td>
                    <span className={"status status-" + r.status}>
                      <span className="dot" />
                      {r.label}
                    </span>
                  </td>
                  <td className="mono">{r.unit}</td>
                  <td className="num-col">{r.replicas}</td>
                  <td>{r.owner}</td>
                  <td className="num-col">{r.duration}</td>
                  <td>
                    <div className="row-actions">
                      <Link className="act" to={`/jobs/${name}/runs/${r.run}`} title="详情" aria-label="详情" />
                      <button className="act" title="日志" aria-label="日志" />
                      <button className="act" title="监控" aria-label="监控" />
                      {r.cancel ? (
                        <button
                          className="act"
                          title="取消"
                          aria-label="取消"
                          onClick={() =>
                            confirm({
                              title: `取消运行 ${r.run}？`,
                              desc: "取消后正在执行的 Pod 将被终止，已产出的 checkpoint 保留。",
                              okLabel: "确认取消",
                              toast: "运行已请求取消（Canceling）",
                            })
                          }
                        />
                      ) : (
                        <button
                          className="act act-danger"
                          title="删除"
                          aria-label="删除"
                          onClick={() =>
                            confirm({
                              title: `删除运行 ${r.run}？`,
                              desc: "删除后该 Run 记录与日志不可恢复。",
                              okLabel: "确认删除",
                              toast: `运行 ${r.run} 已删除`,
                            })
                          }
                        />
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}
