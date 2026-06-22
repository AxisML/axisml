// In-memory mock fixtures for VITE_USE_MOCK_API mode. Values are grounded in the
// product prototype (docs/product_design/prototype) so the demo reads like the
// real product. Every shape here is the generated API type — the mock router
// (router.ts) returns these verbatim, so pages render exactly as they would
// against a live backend.
import type {
  ArtifactDefinition,
  Backend,
  Event,
  Experiment,
  Image,
  Job,
  Member,
  MeResponse,
  MlService,
  Model,
  Pod,
  Quota,
  ResourcePool,
  Run,
  Tenant,
  TrafficPolicy,
  Workspace,
} from "../generated";

// ── time helpers ───────────────────────────────────────────────────────────────
const hour = 3600_000;
export const ago = (hours: number) => new Date(Date.now() - hours * hour).toISOString();
export const now = () => new Date().toISOString();

const nativeJob: Backend = { name: "native", engine: "pytorchjob" };
const nativeDeploy: Backend = { name: "native", engine: "deployment" };

// ── tenants ────────────────────────────────────────────────────────────────────
export const tenants: Tenant[] = [
  {
    identifier: "llm-lab",
    displayName: "大模型研究院",
    description: "大语言模型预训练与微调团队",
    kubernetesNamespace: "tenant-llm-lab",
    phase: "Active",
    suspended: false,
    memberCount: 12,
    activeJobRuns: 5,
    activeExperimentRuns: 3,
    onlineServices: 2,
    owner: "zhenluo",
    createdAt: ago(24 * 90),
    updatedAt: ago(3),
    quotas: [
      { pool: "a100-pool", units: [{ unitName: "a100-8x", quantity: 6 }] },
      { pool: "cpu-pool", units: [{ unitName: "cpu-large", quantity: 20 }] },
    ],
  },
  {
    identifier: "rec-alg",
    displayName: "推荐算法团队",
    description: "推荐与排序模型训练",
    kubernetesNamespace: "tenant-rec-alg",
    phase: "Active",
    suspended: false,
    memberCount: 8,
    activeJobRuns: 2,
    activeExperimentRuns: 1,
    onlineServices: 3,
    owner: "limei",
    createdAt: ago(24 * 60),
    updatedAt: ago(8),
    quotas: [{ pool: "a100-pool", units: [{ unitName: "a100-4x", quantity: 4 }] }],
  },
  {
    identifier: "av-perc",
    displayName: "视觉感知团队",
    description: "自动驾驶视觉感知模型",
    kubernetesNamespace: "tenant-av-perc",
    phase: "Active",
    suspended: false,
    memberCount: 15,
    activeJobRuns: 4,
    activeExperimentRuns: 6,
    onlineServices: 1,
    owner: "wangqiang",
    createdAt: ago(24 * 45),
    updatedAt: ago(12),
    quotas: [{ pool: "h100-pool", units: [{ unitName: "h100-8x", quantity: 8 }] }],
  },
  {
    identifier: "risk-ai",
    displayName: "风控AI团队",
    description: "风险控制与反欺诈模型",
    kubernetesNamespace: "tenant-risk-ai",
    phase: "Suspended",
    suspended: true,
    memberCount: 5,
    activeJobRuns: 0,
    activeExperimentRuns: 0,
    onlineServices: 0,
    owner: "zhaolei",
    createdAt: ago(24 * 30),
    updatedAt: ago(48),
    quotas: [],
  },
];

// ── current user (system-admin so the full nav + system management is visible) ──
export const me: MeResponse = {
  isSystemAdmin: true,
  user: {
    id: "u-admin",
    username: "admin",
    displayName: "Admin",
    email: "admin@axisml.io",
    createdAt: ago(24 * 120),
    updatedAt: ago(24),
  },
  tenantRoles: tenants.map((t) => ({ tenantName: t.identifier, roleName: "tenant-admin" })),
  permissions: ["*"],
};

// ── resource pools ──────────────────────────────────────────────────────────────
export const pools: ResourcePool[] = [
  {
    name: "a100-pool",
    description: "A100 80GB GPU 资源池",
    nodeCount: 8,
    createdAt: ago(24 * 120),
    updatedAt: ago(24),
    labels: { "gpu.type": "a100" },
    nodeSelector: { "gpu.product": "A100", arch: "amd64", "network": "ib" },
    tolerations: [{ key: "nvidia.com/gpu", operator: "Exists", effect: "NoSchedule" }],
    units: [
      {
        name: "a100-8x",
        description: "整机 8 卡 A100",
        requests: { cpu: "96", memory: "1Ti", "nvidia.com/gpu": "8" },
        limits: { cpu: "128", memory: "1.5Ti", "nvidia.com/gpu": "8" },
      },
      {
        name: "a100-4x",
        description: "半机 4 卡 A100",
        requests: { cpu: "48", memory: "512Gi", "nvidia.com/gpu": "4" },
        limits: { cpu: "64", memory: "768Gi", "nvidia.com/gpu": "4" },
      },
    ],
  },
  {
    name: "h100-pool",
    description: "H100 SXM GPU 资源池",
    nodeCount: 4,
    createdAt: ago(24 * 90),
    updatedAt: ago(24),
    labels: { "gpu.type": "h100" },
    nodeSelector: { "gpu.product": "H100", network: "ib" },
    tolerations: [{ key: "nvidia.com/gpu", operator: "Exists", effect: "NoSchedule" }],
    units: [
      {
        name: "h100-8x",
        description: "整机 8 卡 H100",
        requests: { cpu: "128", memory: "2Ti", "nvidia.com/gpu": "8" },
        limits: { cpu: "192", memory: "2Ti", "nvidia.com/gpu": "8" },
      },
    ],
  },
  {
    name: "cpu-pool",
    description: "通用 CPU 资源池（数据处理 / 推理）",
    nodeCount: 16,
    createdAt: ago(24 * 120),
    updatedAt: ago(48),
    nodeSelector: { arch: "amd64" },
    units: [
      {
        name: "cpu-large",
        description: "16C 64G",
        requests: { cpu: "16", memory: "64Gi" },
        limits: { cpu: "16", memory: "64Gi" },
      },
    ],
  },
];

// ── members & quotas (keyed by tenant identifier) ───────────────────────────────
export const membersByTenant: Record<string, Member[]> = {
  "llm-lab": [
    { userId: "u-admin", username: "admin", displayName: "Admin", email: "admin@axisml.io", roleName: "tenant-admin", addedAt: ago(24 * 90) },
    { userId: "u-zhang", username: "zhangwei", displayName: "张伟", email: "zhangwei@axisml.io", roleName: "user", addedAt: ago(24 * 40) },
    { userId: "u-liu", username: "liuyang", displayName: "刘洋", email: "liuyang@axisml.io", roleName: "user", addedAt: ago(24 * 20) },
  ],
  "rec-alg": [
    { userId: "u-limei", username: "limei", displayName: "李梅", email: "limei@axisml.io", roleName: "tenant-admin", addedAt: ago(24 * 60) },
    { userId: "u-chen", username: "chenjie", displayName: "陈杰", email: "chenjie@axisml.io", roleName: "user", addedAt: ago(24 * 15) },
  ],
};

export const quotasByTenant: Record<string, Quota[]> = Object.fromEntries(
  tenants.map((t) => [t.identifier, t.quotas ?? []]),
);

// ── jobs ────────────────────────────────────────────────────────────────────────
function job(name: string, displayName: string, description: string, engine: string, owner: string, updatedH: number): Job {
  return {
    id: `job-${name}`,
    name,
    displayName,
    description,
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner,
    createdAt: ago(24 * 10),
    updatedAt: ago(updatedH),
    spec: {
      backend: { name: "native", engine },
      poolName: "a100-pool",
      unitName: "a100-8x",
      roles: [
        {
          name: "worker",
          replicas: 4,
          template: {
            image: "harbor.axisml.io/llm/pytorch-train:2.3",
            command: ["torchrun", "--nproc_per_node=4", "train.py", "--model_name", "llama-7b", "--lr", "2e-5", "--epochs", "3", "--batch_size", "16", "--data", "/data/sft.jsonl"],
            env: [
              { name: "WANDB_DISABLED", value: "true" },
              { name: "NCCL_DEBUG", value: "INFO" },
            ],
            resources: { "nvidia.com/gpu": "8" },
          },
        },
      ],
    },
  };
}

export const jobs: Job[] = [
  job("train-llm-7b", "LLaMA 7B 预训练", "7B 基座模型全参数预训练", "pytorchjob", "张伟", 48),
  job("eval-recall", "召回离线评估", "召回模型离线评估", "job", "李娜", 24 * 5),
  job("data-clean-etl", "训练数据清洗", "训练数据清洗 ETL", "job", "陈曦", 6),
  job("sft-baseline", "SFT 基线", "SFT 基线训练", "pytorchjob", "王磊", 1),
];

// ── experiments ─────────────────────────────────────────────────────────────────
function experiment(name: string, displayName: string, description: string, owner = "刘洋", updatedH = 2): Experiment {
  return {
    id: `exp-${name}`,
    name,
    displayName,
    description,
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner,
    createdAt: ago(24 * 8),
    updatedAt: ago(updatedH),
    spec: {
      backend: nativeJob,
      poolName: "a100-pool",
      unitName: "a100-4x",
      roles: [
        {
          name: "worker",
          replicas: 1,
          template: {
            image: "harbor.axisml.io/llm/pytorch-train:2.3",
            command: ["torchrun", "--nproc_per_node=4", "sft.py", "--base", "llama3-8b-base", "--lr", "{{lr}}", "--epochs", "3"],
            env: [
              { name: "WANDB_DISABLED", value: "true" },
              { name: "NCCL_DEBUG", value: "INFO" },
            ],
          },
        },
      ],
    },
  };
}

export const experiments: Experiment[] = [
  experiment("llama3-sft-lr-sweep", "LLaMA3 SFT 学习率搜索", "对 SFT 学习率做网格搜索", "刘洋", 2),
  experiment("resnet-aug-search", "ResNet 数据增强搜索", "图像增强策略对比实验", "李娜", 24 * 3),
  experiment("qwen-vl-finetune", "Qwen-VL 微调对比", "不同微调策略效果评估", "张伟", 9),
];

// ── runs (shared generator for jobs & experiments) ──────────────────────────────
const runPhases = ["Running", "Succeeded", "Failed", "Pending"] as const;
export function runsFor(parent: string, kind: "job" | "experiment"): Run[] {
  return [0, 1, 2, 3].map((i) => {
    const phase = runPhases[i % runPhases.length];
    const started = phase === "Pending" ? null : ago(24 - i * 4);
    const finished = phase === "Running" || phase === "Pending" ? null : ago(20 - i * 4);
    return {
      id: `run-${parent}-${i + 1}`,
      name: `${parent}-${i + 1}`,
      displayName: `${parent} #${i + 1}`,
      runNumber: i + 1,
      jobName: parent,
      namespace: "tenant-llm-lab",
      tenantName: "llm-lab",
      tenantDisplayName: "大模型研究院",
      owner: kind === "job" ? "张伟" : "刘洋",
      phase,
      backend: nativeJob,
      poolName: "a100-pool",
      unitName: "a100-8x",
      createdAt: ago(25 - i * 4),
      updatedAt: ago(i),
      startedAt: started,
      finishedAt: finished,
      resources: { "nvidia.com/gpu": "8", cpu: "96", memory: "1Ti" },
      roles: [
        {
          name: "worker",
          replicas: 4,
          readyReplicas: phase === "Running" ? 4 : 0,
          succeededReplicas: phase === "Succeeded" ? 4 : 0,
          template: {
            image: "harbor.axisml.io/llm/pytorch-train:2.3",
            command: ["torchrun", "--nproc_per_node=4", "train.py", "--lr", "2e-5", "--epochs", "3"],
            env: [{ name: "NCCL_DEBUG", value: "INFO" }],
          },
        },
      ],
      message: phase === "Failed" ? "OOMKilled on worker-2" : undefined,
    };
  });
}

// Deterministic run roll-up for list pages (run count + last-5 phases for the
// status-dot strip). Demo-only: real list endpoints don't carry run history, so
// the Jobs/Experiments lists fall back to an empty strip outside mock mode.
const RUN_SUMMARY: Record<string, { count: number; recent: string[] }> = {
  "train-llm-7b": { count: 4, recent: ["Succeeded", "Failed", "Succeeded", "Running"] },
  "eval-recall": { count: 3, recent: ["Succeeded", "Succeeded", "Succeeded"] },
  "data-clean-etl": { count: 7, recent: ["Succeeded", "Failed", "Succeeded", "Succeeded", "Failed"] },
  "sft-baseline": { count: 0, recent: [] },
  "llama3-sft-lr-sweep": { count: 5, recent: ["Succeeded", "Running", "Succeeded", "Failed", "Succeeded"] },
  "resnet-aug-search": { count: 2, recent: ["Succeeded", "Succeeded"] },
  "qwen-vl-finetune": { count: 1, recent: ["Running"] },
};
export function runSummary(name: string): { count: number; recent: string[] } {
  return RUN_SUMMARY[name] ?? { count: 0, recent: [] };
}

// ── workspaces ──────────────────────────────────────────────────────────────────
export const workspaces: Workspace[] = [
  {
    id: "ws-1",
    name: "notebook-llm",
    displayName: "LLM 开发环境",
    description: "Jupyter + CUDA 调试环境",
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner: "张伟",
    image: "harbor.axisml.io/base/jupyter-cuda:12.1",
    containerPort: 8888,
    phase: "Running",
    desiredState: "Running",
    replicas: 1,
    readyReplicas: 1,
    poolName: "a100-pool",
    unitName: "a100-4x",
    createdAt: ago(24 * 6),
    updatedAt: ago(1),
    lastStartedAt: ago(6),
    endpoint: { accessUrl: "https://notebook-llm.llm-lab.axisml.io", internalDns: "notebook-llm.tenant-llm-lab.svc" },
    resources: { "nvidia.com/gpu": "4", cpu: "48", memory: "512Gi" },
  },
  {
    id: "ws-2",
    name: "data-prep",
    displayName: "数据预处理",
    description: "CPU 数据清洗工作空间",
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner: "刘洋",
    image: "harbor.axisml.io/base/python-data:3.11",
    containerPort: 8080,
    phase: "Stopped",
    desiredState: "Stopped",
    replicas: 0,
    readyReplicas: 0,
    poolName: "cpu-pool",
    unitName: "cpu-large",
    createdAt: ago(24 * 12),
    updatedAt: ago(30),
    lastStoppedAt: ago(30),
    resources: { cpu: "16", memory: "64Gi" },
  },
];

// ── services ────────────────────────────────────────────────────────────────────
export const services: MlService[] = [
  {
    id: "svc-1",
    name: "llama3-8b-sft",
    displayName: "LLaMA3 8B 推理服务",
    description: "在线对话推理服务",
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    tenantDisplayName: "大模型研究院",
    owner: "张伟",
    phase: "Ready",
    backend: { name: "kserve", engine: "llminference" },
    image: "harbor.axisml.io/serve/vllm:0.5",
    modelName: "llama3-8b-sft",
    modelVersion: "v3",
    replicas: 2,
    readyReplicas: 2,
    poolName: "a100-pool",
    unitName: "a100-4x",
    accessUrl: "https://llama3-8b-sft.llm-lab.axisml.io",
    ports: [{ name: "http", port: 8000 }],
    createdAt: ago(24 * 5),
    updatedAt: ago(2),
    resources: { "nvidia.com/gpu": "4", cpu: "32", memory: "256Gi" },
  },
  {
    id: "svc-2",
    name: "bge-embed-svc",
    displayName: "BGE 向量服务",
    description: "文本向量化在线服务",
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    tenantDisplayName: "大模型研究院",
    owner: "刘洋",
    phase: "Degraded",
    backend: nativeDeploy,
    image: "harbor.axisml.io/serve/bge:1.5",
    modelName: "bge-embed",
    modelVersion: "v2",
    replicas: 3,
    readyReplicas: 1,
    poolName: "cpu-pool",
    unitName: "cpu-large",
    accessUrl: "https://bge-embed.llm-lab.axisml.io",
    ports: [{ name: "http", port: 8080 }],
    createdAt: ago(24 * 9),
    updatedAt: ago(4),
    message: "2/3 副本未就绪",
    resources: { cpu: "16", memory: "64Gi" },
  },
];

// ── traffic policies ────────────────────────────────────────────────────────────
export const trafficPolicies: TrafficPolicy[] = [
  {
    id: "tp-1",
    name: "llama3-canary",
    displayName: "LLaMA3 灰度发布",
    description: "v3 灰度 20% 流量",
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    tenantDisplayName: "大模型研究院",
    owner: "张伟",
    phase: "Ready",
    mode: "canary",
    canaryPercent: 20,
    accessUrl: "https://llama3.llm-lab.axisml.io",
    endpoint: { hostname: "llama3.llm-lab.axisml.io", path: "/" },
    backends: [
      { serviceName: "llama3-8b-sft", role: "stable", weight: 80, actualPct: 80, ready: true },
      { serviceName: "llama3-8b-sft-v4", role: "canary", weight: 20, actualPct: 20, ready: true },
    ],
    createdAt: ago(24 * 3),
    updatedAt: ago(1),
  },
  {
    id: "tp-2",
    name: "bge-embed-weighted",
    displayName: "BGE 向量 多版本加权",
    description: "三个版本按权重承接流量，做 A/B/C 对比",
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    tenantDisplayName: "大模型研究院",
    owner: "刘洋",
    phase: "Ready",
    mode: "weighted",
    accessUrl: "https://bge-embed.llm-lab.axisml.io",
    endpoint: { hostname: "bge-embed.llm-lab.axisml.io", path: "/" },
    backends: [
      { serviceName: "bge-embed-svc", role: "member", weight: 50, actualPct: 50, ready: true },
      { serviceName: "bge-embed-svc-v2", role: "member", weight: 30, actualPct: 30, ready: true },
      { serviceName: "bge-embed-svc-v3", role: "member", weight: 20, actualPct: 20, ready: false },
    ],
    createdAt: ago(24 * 5),
    updatedAt: ago(2),
  },
];

// ── model & image definitions + versions ────────────────────────────────────────
function modelDef(name: string, displayName: string, description: string, framework: string): ArtifactDefinition {
  return {
    id: `model-${name}`,
    kind: "model",
    name,
    displayName,
    description,
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner: "张伟",
    labels: { framework },
    createdAt: ago(24 * 20),
    updatedAt: ago(6),
  };
}

export const models: ArtifactDefinition[] = [
  modelDef("llama3-8b-sft", "LLaMA3 8B SFT", "指令微调后的 LLaMA3 8B", "PyTorch"),
  modelDef("qwen2-vl-ft", "Qwen2-VL 微调", "多模态微调模型", "PyTorch"),
  modelDef("bge-embed", "BGE 向量模型", "中文文本向量模型", "PyTorch"),
  modelDef("resnet-cls", "ResNet 分类", "图像分类模型", "PyTorch"),
];

function imageDef(name: string, displayName: string, description: string): ArtifactDefinition {
  return {
    id: `image-${name}`,
    kind: "image",
    name,
    displayName,
    description,
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner: "刘洋",
    createdAt: ago(24 * 25),
    updatedAt: ago(10),
  };
}

export const images: ArtifactDefinition[] = [
  imageDef("pytorch-train", "PyTorch 训练镜像", "CUDA 12.1 + PyTorch 2.3 训练镜像"),
  imageDef("vllm-serve", "vLLM 推理镜像", "vLLM 0.5 推理镜像"),
];

export function modelVersions(name: string): Model[] {
  return ["v3", "v2", "v1"].map((v, i) => ({
    id: `model-${name}-${v}`,
    name,
    version: v,
    displayName: `${name} ${v}`,
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner: "张伟",
    status: "Ready",
    sizeBytes: 16_000_000_000 - i * 2_000_000_000,
    digest: `sha256:${name}${v}`,
    uri: `s3://axisml-models/llm-lab/${name}/${v}`,
    createdAt: ago(24 * (6 - i * 2)),
    updatedAt: ago(6 - i * 2),
    readyAt: ago(6 - i * 2),
  }));
}

export function imageVersions(name: string): Image[] {
  return ["2.3", "2.1"].map((v, i) => ({
    id: `image-${name}-${v}`,
    name,
    version: v,
    displayName: `${name}:${v}`,
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner: "刘洋",
    status: "Ready",
    sizeBytes: 8_000_000_000 - i * 1_000_000_000,
    digest: `sha256:${name}${v}`,
    uri: `harbor.axisml.io/llm/${name}:${v}`,
    createdAt: ago(24 * (10 - i * 5)),
    updatedAt: ago(10 - i * 5),
    readyAt: ago(10 - i * 5),
  }));
}

// ── pods, events, metrics (synthesized on demand) ───────────────────────────────
export function podsFor(owner: string): Pod[] {
  return [0, 1, 2, 3].map((i) => ({
    name: `${owner}-worker-${i}`,
    phase: i === 3 ? "Pending" : "Running",
    role: "worker",
    replicaIndex: i,
    nodeName: `gpu-node-${(i % 4) + 1}`,
    restartCount: i === 2 ? 1 : 0,
    startedAt: ago(6 - i),
    finishedAt: null,
  }));
}

export function eventsFor(name: string): Event[] {
  return [
    { reason: "Scheduled", message: `Successfully assigned ${name} to gpu-node-1`, type: "Normal", lastTimestamp: ago(6), count: 1 },
    { reason: "Pulled", message: "Container image already present on machine", type: "Normal", lastTimestamp: ago(5.9), count: 1 },
    { reason: "Started", message: "Started container worker", type: "Normal", lastTimestamp: ago(5.8), count: 1 },
    { reason: "BackOff", message: "Back-off restarting failed container", type: "Warning", lastTimestamp: ago(2), count: 3 },
  ];
}

export function metricSeries(metric: string) {
  const series = Array.from({ length: 24 }, (_, i) => ({
    timestamp: ago(24 - i),
    value: Math.round((40 + 30 * Math.sin(i / 3) + i) % 100),
  }));
  return { metric, range: "24h", step: "1h", unit: "%", series };
}

// ── cluster resource usage (dashboard) ──────────────────────────────────────────
// Demo-only: there is no cluster-metrics endpoint in the platform contract, so the
// real Dashboard renders an honest zero state. Under VITE_USE_MOCK_API the
// Dashboard reads these fixtures instead, so the landing page looks like the
// product prototype (docs/product_design/prototype/index.html).
export type MeterState = "ok" | "warn" | "hot" | "na";
export interface UsageMetric {
  used: number;
  total: number;
  unit: string;
  pct: number;
  state: MeterState;
  display: string; // formatted "used" value (mem keeps one decimal)
}
export interface ClusterUsage {
  key: string;
  label: string;
  gpu: UsageMetric;
  cpu: UsageMetric;
  mem: UsageMetric;
  /** GPU utilisation (%) and GPU quota usage (%) trend points. */
  trend: { t: string; util: number; quota: number }[];
}

const meter = (used: number, total: number, unit: string, decimals = 0): UsageMetric => {
  const pct = total === 0 ? 0 : Math.round((used / total) * 100);
  const state: MeterState = total === 0 ? "na" : pct >= 80 ? "hot" : pct >= 60 ? "warn" : "ok";
  return { used, total, unit, pct, state, display: decimals ? used.toFixed(decimals) : String(used) };
};

const trendFor = (base: number) =>
  Array.from({ length: 13 }, (_, i) => {
    const util = Math.max(8, Math.round(base - 18 + i * 1.6 + 6 * Math.sin(i / 1.7)));
    return {
      t: `${i * 2}:00`,
      util,
      // 已分配的 GPU 配额(%)，通常略高于实际利用率。
      quota: Math.min(100, util + 8 + Math.round(4 * Math.sin(i / 2.3))),
    };
  });

const clusterUsageMap: Record<string, ClusterUsage> = {
  all: {
    key: "all",
    label: "全部",
    gpu: meter(36, 48, "卡"),
    cpu: meter(740, 1152, "核"),
    mem: { ...meter(3.4, 5.5, "TiB", 1) },
    trend: trendFor(72),
  },
  "a100-pool": {
    key: "a100-pool",
    label: "a100-pool",
    gpu: meter(22, 32, "卡"),
    cpu: meter(240, 384, "核"),
    mem: meter(1.2, 2.0, "TiB", 1),
    trend: trendFor(69),
  },
  "h100-pool": {
    key: "h100-pool",
    label: "h100-pool",
    gpu: meter(14, 16, "卡"),
    cpu: meter(180, 256, "核"),
    mem: meter(1.4, 2.0, "TiB", 1),
    trend: trendFor(86),
  },
  "cpu-pool": {
    key: "cpu-pool",
    label: "cpu-pool",
    gpu: meter(0, 0, "卡"),
    cpu: meter(320, 512, "核"),
    mem: meter(0.8, 1.5, "TiB", 1),
    trend: trendFor(60),
  },
};

export function clusterUsage(pool?: string): ClusterUsage {
  return clusterUsageMap[pool ?? "all"] ?? clusterUsageMap.all;
}

export function podLogs(pod: string): string {
  return [
    `[INFO] ${pod} starting…`,
    "[INFO] loading checkpoint from s3://axisml-models/llm-lab/llama3-8b-sft/v3",
    "[INFO] world_size=4 rank=0 local_rank=0",
    "[INFO] epoch 1/3 step 100 loss=1.842 lr=2.0e-5",
    "[INFO] epoch 1/3 step 200 loss=1.611 lr=2.0e-5",
    "[INFO] epoch 1/3 step 300 loss=1.498 lr=2.0e-5",
  ].join("\n");
}
