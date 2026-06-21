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
    displayName: "黄振洛",
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
    { userId: "u-admin", username: "admin", displayName: "黄振洛", email: "admin@axisml.io", roleName: "tenant-admin", addedAt: ago(24 * 90) },
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
function job(name: string, displayName: string, description: string, engine: string): Job {
  return {
    id: `job-${name}`,
    name,
    displayName,
    description,
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner: "zhangwei",
    createdAt: ago(24 * 10),
    updatedAt: ago(5),
    spec: {
      backend: { name: "native", engine },
      poolName: "a100-pool",
      unitName: "a100-8x",
      roles: [
        {
          name: "worker",
          replicas: 4,
          template: { image: "harbor.axisml.io/llm/pytorch-train:2.3", resources: { "nvidia.com/gpu": "8" } },
        },
      ],
    },
  };
}

export const jobs: Job[] = [
  job("train-llm-7b", "LLaMA 7B 预训练", "7B 基座模型全参数预训练", "pytorchjob"),
  job("sft-baseline", "SFT 基线", "指令微调基线任务", "pytorchjob"),
  job("qwen-vl-finetune", "Qwen-VL 微调", "多模态视觉语言模型微调", "pytorchjob"),
  job("embedding-distill", "向量模型蒸馏", "BGE embedding 蒸馏任务", "pytorchjob"),
];

// ── experiments ─────────────────────────────────────────────────────────────────
function experiment(name: string, displayName: string, description: string): Experiment {
  return {
    id: `exp-${name}`,
    name,
    displayName,
    description,
    namespace: "tenant-llm-lab",
    tenantName: "llm-lab",
    owner: "liuyang",
    createdAt: ago(24 * 8),
    updatedAt: ago(2),
    spec: {
      backend: nativeJob,
      poolName: "a100-pool",
      unitName: "a100-4x",
      roles: [{ name: "worker", replicas: 1, template: { image: "harbor.axisml.io/llm/pytorch-train:2.3" } }],
    },
  };
}

export const experiments: Experiment[] = [
  experiment("llama3-sft-lr-sweep", "LLaMA3 SFT 学习率搜索", "对 SFT 学习率做网格搜索"),
  experiment("resnet-aug-search", "ResNet 数据增强搜索", "图像增强策略对比实验"),
  experiment("qwen-vl-finetune", "Qwen-VL 微调对比", "不同微调策略效果评估"),
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
      owner: kind === "job" ? "zhangwei" : "liuyang",
      phase,
      backend: nativeJob,
      poolName: "a100-pool",
      unitName: "a100-8x",
      createdAt: ago(25 - i * 4),
      updatedAt: ago(i),
      startedAt: started,
      finishedAt: finished,
      resources: { "nvidia.com/gpu": "8", cpu: "96", memory: "1Ti" },
      roles: [{ name: "worker", replicas: 4, readyReplicas: phase === "Running" ? 4 : 0, succeededReplicas: phase === "Succeeded" ? 4 : 0 }],
      message: phase === "Failed" ? "OOMKilled on worker-2" : undefined,
    };
  });
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
    owner: "zhangwei",
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
    owner: "liuyang",
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
    owner: "zhangwei",
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
    owner: "liuyang",
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
    owner: "zhangwei",
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
    owner: "zhangwei",
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
    owner: "liuyang",
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
    owner: "zhangwei",
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
    owner: "liuyang",
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
