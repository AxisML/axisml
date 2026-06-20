import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useUI } from "@/app/ui";
import { Tabs } from "@/components/Tabs";

// Faithful port of prototype/run-detail.html. The router passes `kind` so the
// back link returns to the owning Job or Experiment. Static demo content.
export default function RunDetail({ kind }: { kind: "job" | "experiment" }) {
  const { name = "train-llm-7b", run = "train-llm-7b-12" } = useParams();
  const backTo = kind === "job" ? `/jobs/${name}` : `/experiments/${name}`;
  const { confirm } = useUI();

  return (
    <main className="page">
      <Link className="back-link" to={backTo}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M15 18l-6-6 6-6" />
        </svg>
        返回 {name}
      </Link>

      <div className="page-head">
        <div>
          <h1 className="detail-title">
            {run}{" "}
            <span className="spill run">
              <span className="dot" />
              运行中
            </span>
          </h1>
          <div className="detail-sub">第 12 次运行 · 触发人 张伟 · 已运行 02:14:30</div>
        </div>
        <div className="actions">
          <button
            className="btn btn-danger"
            onClick={() =>
              confirm({
                title: `取消运行 ${run}？`,
                desc: "取消后正在执行的 Pod 将被终止，已产出的 checkpoint 保留。",
                okLabel: "确认取消",
                toast: "运行已请求取消（Canceling）",
              })
            }
          >
            取消运行
          </button>
        </div>
      </div>

      <Tabs
        tabs={[
          { key: "info", label: "概览", content: <InfoPane /> },
          { key: "pods", label: "实例", content: <PodsPane /> },
          { key: "log", label: "日志", content: <LogPane /> },
          { key: "ev", label: "事件", content: <EventsPane /> },
        ]}
      />
    </main>
  );
}

function InfoPane() {
  return (
    <div className="panel">
      <div className="panel-head">
        <h3>配置信息</h3>
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

interface PodRow {
  name: string;
  node: string;
  restarts: number;
  started: string;
}

const POD_ROWS: PodRow[] = [
  { name: "train-llm-7b-12-worker-0", node: "node-a100-03", restarts: 0, started: "2026-06-13 02:14:30" },
  { name: "train-llm-7b-12-worker-1", node: "node-a100-03", restarts: 0, started: "2026-06-13 02:14:30" },
  { name: "train-llm-7b-12-worker-2", node: "node-a100-07", restarts: 1, started: "2026-06-13 02:11:02" },
  { name: "train-llm-7b-12-worker-3", node: "node-a100-07", restarts: 0, started: "2026-06-13 02:14:30" },
];

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
              <th className="num-col">退出码</th>
              <th>启动时间</th>
              <th style={{ textAlign: "right" }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {POD_ROWS.map((p) => (
              <tr key={p.name}>
                <td className="mono">{p.name}</td>
                <td>
                  <span className="spill run">
                    <span className="dot" />
                    Running
                  </span>
                </td>
                <td className="mono muted">{p.node}</td>
                <td className="num-col">{p.restarts}</td>
                <td className="num-col muted">—</td>
                <td className="muted mono">{p.started}</td>
                <td>
                  <div className="row-actions">
                    <button className="act" title="日志" aria-label="日志" />
                    <button className="act" title="事件" aria-label="事件" />
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function PodPick() {
  return (
    <div className="pod-pick">
      <span className="pp-tag">POD</span>
      <select>
        <option>worker-0</option>
        <option>worker-1</option>
        <option>worker-2</option>
        <option>worker-3</option>
      </select>
    </div>
  );
}

function FollowToggle() {
  const [on, setOn] = useState(true);
  return (
    <label className="follow">
      实时跟随{" "}
      <button className={"toggle" + (on ? " on" : "")} aria-label="实时跟随" onClick={() => setOn((v) => !v)} />
    </label>
  );
}

function LogPane() {
  return (
    <div className="panel">
      <div className="panel-body">
        <div className="log-bar">
          <PodPick />
          <div className="grow" />
          <FollowToggle />
        </div>
        <pre className="logbox">
          <span className="l-time">02:14:31</span>
          <span className="l-info">[I]</span> Initializing distributed: rank=0 world_size=4 backend=nccl
          {"\n"}
          <span className="l-time">02:14:34</span>
          <span className="l-info">[I]</span> Loaded base model llama-7b-base@v1 (13.1 GB)
          {"\n"}
          <span className="l-time">02:14:52</span>
          <span className="l-info">[I]</span> Dataset sft-dialog@v2 · 124,503 samples · 3 epochs
          {"\n"}
          <span className="l-time">02:31:10</span>
          <span className="l-info">[I]</span> epoch 1 | step 0500 | loss 1.842 | lr 1.98e-5 | 3.2 it/s
          {"\n"}
          <span className="l-time">03:02:44</span>
          <span className="l-info">[I]</span> epoch 1 | step 1000 | loss 1.231 | lr 1.91e-5 | 3.3 it/s
          {"\n"}
          <span className="l-time">03:40:08</span>
          <span className="l-warn">[W]</span> worker-2 nccl timeout retry (1/3), recovered in 4.1s
          {"\n"}
          <span className="l-time">04:18:55</span>
          <span className="l-info">[I]</span> epoch 2 | step 2000 | loss 0.948 | lr 1.62e-5 | 3.3 it/s
          {"\n"}
          <span className="l-time">04:29:01</span>
          <span className="l-info">[I]</span> checkpoint saved → /output/ckpt-step-2200
        </pre>
      </div>
    </div>
  );
}

function EventsPane() {
  return (
    <>
      {/* 运行事件 */}
      <div className="panel">
        <div className="panel-head">
          <h3>运行事件</h3>
        </div>
        <div className="panel-body">
          <div className="timeline">
            <div className="tl-item">
              <span className="tl-dot" />
              <div className="tl-head">
                <span className="tl-name">Scheduled</span>
                <span className="tl-tag">NORMAL</span>
                <span className="tl-time">2026-06-13 02:14:30</span>
              </div>
              <div className="tl-desc">Run 创建，PodGroup gang 调度就绪（koord-scheduler）</div>
            </div>
            <div className="tl-item is-muted">
              <span className="tl-dot" />
              <div className="tl-head">
                <span className="tl-name">ArtifactsMounted</span>
                <span className="tl-tag">NORMAL</span>
                <span className="tl-time">2026-06-13 02:14:52</span>
              </div>
              <div className="tl-desc">制品挂载完成：sft-dialog@v2 / llama-7b-base@v1</div>
            </div>
            <div className="tl-item is-warn">
              <span className="tl-dot" />
              <div className="tl-head">
                <span className="tl-name">BackOff</span>
                <span className="tl-tag warn">WARNING</span>
                <span className="tl-time">2026-06-13 03:40:08</span>
              </div>
              <div className="tl-desc">worker-2 重启一次（nccl timeout），已自愈</div>
            </div>
          </div>
        </div>
      </div>
      {/* 实例事件 */}
      <div className="panel" style={{ marginTop: "var(--space-5)" }}>
        <div className="panel-head">
          <h3>实例事件</h3>
        </div>
        <div className="panel-body">
          <div className="log-bar">
            <PodPick />
            <div className="grow" />
            <FollowToggle />
          </div>
          <div className="timeline" style={{ marginTop: "var(--space-5)" }}>
            <div className="tl-item">
              <span className="tl-dot" />
              <div className="tl-head">
                <span className="tl-name">Scheduled</span>
                <span className="tl-tag">NORMAL</span>
                <span className="tl-time">2026-06-13 02:14:30</span>
              </div>
              <div className="tl-desc">worker-0 分配到节点 node-a100-03</div>
            </div>
            <div className="tl-item">
              <span className="tl-dot" />
              <div className="tl-head">
                <span className="tl-name">Pulled</span>
                <span className="tl-tag">NORMAL</span>
                <span className="tl-time">2026-06-13 02:14:48</span>
              </div>
              <div className="tl-desc">镜像 pytorch:2.3-cu121 拉取完成（15.2s）</div>
            </div>
            <div className="tl-item">
              <span className="tl-dot" />
              <div className="tl-head">
                <span className="tl-name">Created</span>
                <span className="tl-tag">NORMAL</span>
                <span className="tl-time">2026-06-13 02:14:49</span>
              </div>
              <div className="tl-desc">容器创建完成</div>
            </div>
            <div className="tl-item">
              <span className="tl-dot" />
              <div className="tl-head">
                <span className="tl-name">Started</span>
                <span className="tl-tag">NORMAL</span>
                <span className="tl-time">2026-06-13 02:14:50</span>
              </div>
              <div className="tl-desc">容器启动，开始分布式训练 rank=0</div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
