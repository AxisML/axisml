import { Link, useParams } from "react-router-dom";
import { useUI } from "@/app/ui";
import { Tabs } from "@/components/Tabs";

type RunStatus = "running" | "success" | "failed";

interface RunRow {
  run: string;
  status: RunStatus;
  statusLabel: string;
  unit: string;
  replicas: number;
  by: string;
  cost: string;
  cancelable?: boolean;
}

const RUNS: RunRow[] = [
  { run: "llama3-sft-lr-sweep-8", status: "running", statusLabel: "运行中", unit: "a100-8x-xlarge-ib", replicas: 2, by: "张伟", cost: "01:12:30", cancelable: true },
  { run: "llama3-sft-lr-sweep-7", status: "success", statusLabel: "成功", unit: "a100-4x-xlarge", replicas: 2, by: "张伟", cost: "03:40:18" },
  { run: "llama3-sft-lr-sweep-6", status: "success", statusLabel: "成功", unit: "a100-4x-xlarge", replicas: 2, by: "李娜", cost: "03:33:40" },
  { run: "llama3-sft-lr-sweep-5", status: "success", statusLabel: "成功", unit: "a100-4x-xlarge", replicas: 2, by: "张伟", cost: "03:38:02" },
  { run: "llama3-sft-lr-sweep-4", status: "failed", statusLabel: "失败", unit: "a100-4x-xlarge", replicas: 2, by: "李娜", cost: "00:00:09" },
  { run: "llama3-sft-lr-sweep-3", status: "success", statusLabel: "成功", unit: "a100-4x-xlarge", replicas: 2, by: "张伟", cost: "03:51:22" },
  { run: "llama3-sft-lr-sweep-2", status: "failed", statusLabel: "失败", unit: "a100-4x-xlarge", replicas: 2, by: "张伟", cost: "00:00:06" },
  { run: "llama3-sft-lr-sweep-1", status: "success", statusLabel: "成功", unit: "a100-4x-xlarge", replicas: 2, by: "张伟", cost: "03:47:55" },
];

// One-off glyphs not present in the shared icon map (Icon.tsx).
function EyeIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}
function LogIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <path d="M14 3v5h5M8 13h8M8 17h6" />
    </svg>
  );
}
function MonitorIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
    </svg>
  );
}
function CancelIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <circle cx="12" cy="12" r="9" />
      <path d="M15 9l-6 6M9 9l6 6" />
    </svg>
  );
}
function DeleteIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path d="M3 6h18" />
      <path d="M8 6V4h8v2" />
      <path d="M19 6l-1 14H6L5 6" />
      <path d="M10 11v6M14 11v6" />
    </svg>
  );
}

export default function ExperimentDetail() {
  const { name } = useParams<{ name: string }>();
  const expName = name ?? "llama3-sft-lr-sweep";
  const { toast, confirm } = useUI();

  const infoPane = (
    <div className="panel">
      <div className="panel-head">
        <h3>实验信息</h3>
        <button className="btn btn-sm" onClick={() => toast("进入编辑（编辑只影响之后触发的运行）")}>
          编辑
        </button>
      </div>
      <div className="panel-body">
        <dl className="kv kv-lg">
          <dt>实验名</dt>
          <dd>
            <span className="cchip">{expName}</span>
          </dd>
          <dt>描述</dt>
          <dd>LLaMA3-8B SFT 学习率扫描</dd>
          <dt>训练镜像</dt>
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
            <span className="mono">2</span>
            <span className="muted" style={{ marginLeft: 8 }}>
              多机多卡
            </span>
          </dd>
          <dt>数据卷</dt>
          <dd>
            <span className="cchip">team-datasets · 1 TiB → /data</span>
            <span className="cchip" style={{ marginLeft: 6 }}>
              ckpt-store · 500 GiB → /output
            </span>
          </dd>
          <dt>运行策略</dt>
          <dd>
            超时 <span className="mono">48h</span> · 重试 <span className="mono">1</span>
          </dd>
          <dt>创建人</dt>
          <dd>
            张伟 · <span className="mono">2026-06-10</span>
          </dd>
        </dl>
        <div
          style={{
            marginTop: "var(--space-6)",
            paddingTop: "var(--space-5)",
            borderTop: "1px solid var(--border-soft)",
          }}
        >
          <label className="muted" style={{ fontSize: 12 }}>
            启动命令（超参写在命令 / 参数中）
          </label>
          <pre className="logbox" style={{ maxHeight: "none", margin: "6px 0 18px" }}>
            {`torchrun --nproc_per_node=4 sft.py \\
  --base llama3-8b-base --lr {{lr}} --epochs 3`}
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

  const runsPane = (
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
        <div className="panel-head">
          <h3>运行记录</h3>
          <span className="hint">按触发时间倒序 · 共 8 条</span>
        </div>
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
              {RUNS.map((r) => (
                <tr key={r.run}>
                  <td>
                    <Link className="t-name mono" to={`/experiments/${expName}/runs/${r.run}`}>
                      {r.run}
                    </Link>
                  </td>
                  <td>
                    <span className={"status status-" + r.status}>
                      <span className="dot" />
                      {r.statusLabel}
                    </span>
                  </td>
                  <td className="mono">{r.unit}</td>
                  <td className="num-col">{r.replicas}</td>
                  <td>{r.by}</td>
                  <td className="num-col">{r.cost}</td>
                  <td>
                    <div className="row-actions">
                      <Link className="act" to={`/experiments/${expName}/runs/${r.run}`} title="详情" aria-label="详情">
                        <EyeIcon />
                      </Link>
                      <button className="act" title="日志" aria-label="日志">
                        <LogIcon />
                      </button>
                      <button className="act" title="监控" aria-label="监控">
                        <MonitorIcon />
                      </button>
                      {r.cancelable ? (
                        <button
                          className="act"
                          title="取消"
                          aria-label="取消"
                          onClick={() =>
                            confirm({
                              title: `取消运行 ${r.run}？`,
                              desc: "取消后正在执行的 Pod 将被终止，已产出的 checkpoint 保留。",
                              okLabel: "确认取消",
                              toast: "运行已请求取消",
                            })
                          }
                        >
                          <CancelIcon />
                        </button>
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
                        >
                          <DeleteIcon />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="pagination">
          <span>共 8 个运行</span>
          <div className="pages">
            <span className="pg">‹</span>
            <span className="pg on">1</span>
            <span className="pg">›</span>
          </div>
          <span>每页 20 条</span>
        </div>
      </div>
    </>
  );

  return (
    <main className="page">
      <Link className="back-link" to="/experiments">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M15 18l-6-6 6-6" />
        </svg>
        返回实验
      </Link>

      <div className="page-head">
        <div>
          <h1 className="detail-title mono">
            {expName}{" "}
            <span className="spill run">
              <span className="dot" />
              运行中
            </span>
          </h1>
          <div className="detail-sub">LLaMA3-8B SFT 学习率扫描 · 8 个运行 · 创建人 张伟</div>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => toast(`已触发运行 ${expName}-9`)}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path d="M6 4l14 8-14 8z" />
            </svg>
            运行
          </button>
          <button
            className="btn"
            style={{ padding: 0, width: 34 }}
            title="启动 TensorBoard"
            aria-label="启动 TensorBoard"
            onClick={() => toast("正在启动 TensorBoard…")}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="#FF6F00" strokeLinecap="round" strokeLinejoin="round">
              <path d="M4 19V5M4 19h16M8 16v-5M12 16V8M16 16v-3" />
            </svg>
          </button>
          <button
            className="btn btn-danger"
            onClick={() =>
              confirm({
                title: `删除实验 ${expName}？`,
                desc: "删除后实验模板不可恢复；历史运行将一并级联软删。",
                info: "有活跃运行时删除会被阻断。",
                okLabel: "确认删除",
                toast: `实验 ${expName} 已删除`,
              })
            }
          >
            删除
          </button>
        </div>
      </div>

      <Tabs
        tabs={[
          { key: "info", label: "实验信息", content: infoPane },
          { key: "runs", label: "运行记录", count: 8, content: runsPane },
        ]}
      />
    </main>
  );
}
