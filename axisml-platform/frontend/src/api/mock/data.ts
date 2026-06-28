// In-memory fixtures for VITE_USE_MOCK_API mode.
//
// The entity SHAPES + base values come from the OpenAPI spec's whole-object
// examples (src/api/mock/examples.gen.ts, generated from
// axisml-platform/docs/apis/platform.yaml via `pnpm run gen:mock`). The examples
// are authored on the Go DTOs, so the demo data can never drift from the API
// contract. Here we only:
//   • pull each entity via the typed `ex<T>("SchemaName")` accessor, and
//   • clone it into a few rows (varied name/phase) so list pages show variety.
//
// A handful of helpers (pod logs, metric series, the cluster-usage dashboard)
// have NO counterpart in the contract — there is no such endpoint — so they stay
// synthesized here and are clearly marked demo-only.
import { ex } from "./examples.gen";
import type {
  ArtifactDefinition,
  DataVolume,
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

// Shallow clone of a fixture with field overrides — keeps the example as the base
// and varies just enough for a believable multi-row list.
const v = <T,>(base: T, over: Partial<T>): T => ({ ...base, ...over });

// ── tenants ────────────────────────────────────────────────────────────────────
const tenant0 = ex<Tenant>("Tenant");
export const tenants: Tenant[] = [
  tenant0,
  v(tenant0, {
    identifier: "llm-lab",
    displayName: "LLM Research Institute",
    description: "Large language model pretraining and fine-tuning team.",
    kubernetesNamespace: "axisml-llm-lab",
    owner: "zhang.wei",
    memberCount: 12,
    activeJobRuns: 5,
    activeExperimentRuns: 3,
    onlineServices: 2,
  }),
  v(tenant0, {
    identifier: "rec-alg",
    displayName: "Recommendation Algorithm Team",
    description: "Recommendation and ranking model training.",
    kubernetesNamespace: "axisml-rec-alg",
    owner: "li.mei",
    phase: "Suspended",
    suspended: true,
    memberCount: 5,
    activeJobRuns: 0,
    activeExperimentRuns: 0,
    onlineServices: 0,
  }),
];

// ── current user (system-admin so the full nav + system management is visible) ──
export const me: MeResponse = {
  ...ex<MeResponse>("MeResponse"),
  isSystemAdmin: true,
  tenantRoles: tenants.map((t) => ({ tenantName: t.identifier!, roleName: "tenant-admin" })),
  permissions: ["*"],
};

// ── resource pools ──────────────────────────────────────────────────────────────
const pool0 = ex<ResourcePool>("ResourcePool");
export const pools: ResourcePool[] = [
  pool0,
  v(pool0, {
    name: "h100-pool",
    description: "H100 SXM GPU resource pool.",
    nodeCount: 4,
  }),
  v(pool0, {
    name: "cpu-pool",
    description: "General-purpose CPU resource pool (data processing / inference).",
    nodeCount: 16,
    units: [],
  }),
];

// ── data volumes (tenant-scoped durable PVCs) ───────────────────────────────────
const GiB = 1024 ** 3;
const TiB = 1024 ** 4;
const vol0 = ex<DataVolume>("DataVolume");
export const dataVolumes: DataVolume[] = [
  v(vol0, {
    name: "shared-datasets",
    description: "Shared raw datasets directory.",
    size: "2Ti",
    storageClass: "nfs-rwx",
    accessModes: ["ReadWriteMany"],
    createdAt: ago(24 * 130),
    status: {
      phase: "Bound",
      boundCapacity: "2Ti",
      usedBytes: Math.round(1.32 * TiB),
      mounts: [
        { workload: "ws-jupyter-3", kind: "Deployment", mountPath: "/data/shared", running: true },
        { workload: "preprocess-corpus-21", kind: "Job", mountPath: "/mnt/dataset", running: true },
        { workload: "eval-bench-7", kind: "Job", mountPath: "/data/in", running: false },
      ],
    },
  }),
  v(vol0, {
    name: "llm-checkpoints",
    description: "Training checkpoint persistence.",
    size: "500Gi",
    storageClass: "ceph-rbd",
    accessModes: ["ReadWriteOnce"],
    createdAt: ago(24 * 90),
    status: { phase: "Bound", boundCapacity: "500Gi", usedBytes: 210 * GiB, mounts: [
      { workload: "train-llm-7b-12", kind: "Job", mountPath: "/ckpt", running: true },
    ] },
  }),
  v(vol0, {
    name: "preprocess-cache",
    description: "Data preprocessing scratch cache.",
    size: "200Gi",
    storageClass: "nfs-rwx",
    accessModes: ["ReadWriteMany"],
    createdAt: ago(24 * 60),
    status: { phase: "Bound", boundCapacity: "200Gi", usedBytes: 12 * GiB, mounts: [] },
  }),
  v(vol0, {
    name: "eval-artifacts",
    description: "Read-only shared evaluation outputs.",
    size: "100Gi",
    storageClass: "ceph-rbd",
    accessModes: ["ReadOnlyMany"],
    createdAt: ago(24 * 40),
    status: { phase: "Bound", boundCapacity: "100Gi", usedBytes: 28 * GiB, mounts: [] },
  }),
  v(vol0, {
    name: "scratch-ssd",
    description: "Temporary high-speed scratch disk.",
    size: "1Ti",
    storageClass: "local-ssd",
    accessModes: ["ReadWriteOnce"],
    createdAt: ago(6),
    status: { phase: "Pending", mounts: [] },
  }),
];

// storage classes for the new-volume picker (cluster-scoped)
export const storageClasses = [
  { name: "nfs-rwx", provisioner: "nfs.csi.k8s.io", default: false, allowVolumeExpansion: true },
  { name: "ceph-rbd", provisioner: "rbd.csi.ceph.com", default: true, allowVolumeExpansion: true },
  { name: "local-ssd", provisioner: "kubernetes.io/no-provisioner", default: false, allowVolumeExpansion: false },
];

// ── members & quotas (keyed by tenant identifier) ───────────────────────────────
const member0 = ex<Member>("Member");
export const membersByTenant: Record<string, Member[]> = {
  [tenants[0].identifier!]: [
    member0,
    v(member0, { userId: "u-zhang", username: "zhang.wei", displayName: "Zhang Wei", email: "zhang.wei@axisml.io", roleName: "user" }),
    v(member0, { userId: "u-liu", username: "liu.yang", displayName: "Liu Yang", email: "liu.yang@axisml.io", roleName: "user" }),
  ],
  [tenants[1].identifier!]: [
    v(member0, { userId: "u-limei", username: "li.mei", displayName: "Li Mei", email: "li.mei@axisml.io", roleName: "tenant-admin" }),
  ],
};

export const quotasByTenant: Record<string, Quota[]> = Object.fromEntries(
  tenants.map((t) => [t.identifier, ex<{ items: Quota[] }>("QuotaList").items]),
);

// ── jobs ────────────────────────────────────────────────────────────────────────
const job0 = ex<Job>("Job");
export const jobs: Job[] = [
  job0,
  v(job0, { id: "job-eval-recall", name: "eval-recall", displayName: "Recall offline evaluation", description: "Offline evaluation of the recall model.", owner: "li.na" }),
  v(job0, { id: "job-data-clean", name: "data-clean-etl", displayName: "Training data cleaning", description: "Training-data cleaning ETL.", owner: "chen.xi" }),
];

// ── experiments ─────────────────────────────────────────────────────────────────
const exp0 = ex<Experiment>("Experiment");
export const experiments: Experiment[] = [
  exp0,
  v(exp0, { id: "exp-aug-search", name: "resnet-aug-search", displayName: "ResNet data-augmentation search", description: "Experiment comparing image-augmentation strategies.", owner: "li.na" }),
];

// ── runs (shared generator for jobs & experiments) ──────────────────────────────
const run0 = ex<Run>("Run");
const runPhases = ["Running", "Succeeded", "Failed", "Pending"] as const;
export function runsFor(parent: string, kind: "job" | "experiment"): Run[] {
  return [0, 1, 2, 3].map((i) => {
    const phase = runPhases[i % runPhases.length];
    const started = phase === "Pending" ? null : ago(24 - i * 4);
    const finished = phase === "Running" || phase === "Pending" ? null : ago(20 - i * 4);
    return v(run0, {
      id: `run-${parent}-${i + 1}`,
      name: `${parent}-${i + 1}`,
      displayName: `${parent} #${i + 1}`,
      runNumber: i + 1,
      jobName: parent,
      owner: kind === "job" ? "zhang.wei" : "liu.yang",
      phase,
      startedAt: started ?? undefined,
      finishedAt: finished ?? undefined,
      message: phase === "Failed" ? "OOMKilled on worker-2" : undefined,
    });
  });
}

// Deterministic run roll-up for list pages (run count + recent phases for the
// status-dot strip). Demo-only: real list endpoints don't carry run history, so
// the Jobs/Experiments lists fall back to an empty strip outside mock mode.
const RUN_SUMMARY: Record<string, { count: number; recent: string[] }> = {
  [jobs[0].name]: { count: 4, recent: ["Succeeded", "Failed", "Succeeded", "Running"] },
  [jobs[1].name]: { count: 3, recent: ["Succeeded", "Succeeded", "Succeeded"] },
  [jobs[2].name]: { count: 7, recent: ["Succeeded", "Failed", "Succeeded", "Succeeded", "Failed"] },
  [experiments[0].name]: { count: 5, recent: ["Succeeded", "Running", "Succeeded", "Failed", "Succeeded"] },
  [experiments[1].name]: { count: 2, recent: ["Succeeded", "Succeeded"] },
};
export function runSummary(name: string): { count: number; recent: string[] } {
  return RUN_SUMMARY[name] ?? { count: 0, recent: [] };
}

// ── workspaces ──────────────────────────────────────────────────────────────────
const ws0 = ex<Workspace>("Workspace");
export const workspaces: Workspace[] = [
  ws0,
  v(ws0, {
    id: "ws-data-prep",
    name: "data-prep",
    displayName: "Data preprocessing",
    description: "CPU data-cleaning workspace.",
    owner: "liu.yang",
    phase: "Stopped",
    desiredState: "Stopped",
    replicas: 0,
    readyReplicas: 0,
  }),
];

// ── services ────────────────────────────────────────────────────────────────────
const svc0 = ex<MlService>("MLService");
export const services: MlService[] = [
  svc0,
  v(svc0, {
    id: "svc-bge",
    name: "bge-embed-svc",
    displayName: "BGE embedding service",
    description: "Online text-embedding service.",
    owner: "liu.yang",
    phase: "Degraded",
    replicas: 3,
    readyReplicas: 1,
    message: "2/3 replicas not ready.",
  }),
];

// ── traffic policies ────────────────────────────────────────────────────────────
const tp0 = ex<TrafficPolicy>("TrafficPolicy");
export const trafficPolicies: TrafficPolicy[] = [
  tp0,
  v(tp0, {
    id: "tp-weighted",
    name: "bge-embed-weighted",
    displayName: "BGE embedding multi-version weighting",
    description: "Multiple versions take weighted traffic for A/B comparison.",
    owner: "liu.yang",
    mode: "weighted",
  }),
];

// ── model & image definitions + versions ────────────────────────────────────────
const def0 = ex<ArtifactDefinition>("ArtifactDefinition");
export const models: ArtifactDefinition[] = [
  v(def0, { kind: "model" }),
  v(def0, { id: "model-bge", kind: "model", name: "bge-embed", displayName: "BGE embedding model", description: "Chinese text embedding model." }),
];

export const images: ArtifactDefinition[] = [
  v(def0, { id: "image-pytorch", kind: "image", name: "pytorch-train", displayName: "PyTorch training image", description: "CUDA 12.1 + PyTorch 2.3 training image." }),
  v(def0, { id: "image-vllm", kind: "image", name: "vllm-serve", displayName: "vLLM inference image", description: "vLLM inference image." }),
];

const modelVer0 = ex<Model>("Model");
export function modelVersions(name: string): Model[] {
  return ["1.4.0", "1.3.0", "1.2.0"].map((ver, i) =>
    v(modelVer0, {
      id: `model-${name}-${ver}`,
      name,
      version: ver,
      displayName: `${name} ${ver}`,
      createdAt: ago(24 * (6 - i * 2)),
      updatedAt: ago(6 - i * 2),
    }),
  );
}

const imageVer0 = ex<Image>("Image");
export function imageVersions(name: string): Image[] {
  return ["2.3.0", "2.1.0"].map((ver, i) =>
    v(imageVer0, {
      id: `image-${name}-${ver}`,
      name,
      version: ver,
      displayName: `${name}:${ver}`,
      createdAt: ago(24 * (10 - i * 5)),
      updatedAt: ago(10 - i * 5),
    }),
  );
}

// ── pods & events (sourced from spec examples, expanded for a believable list) ──
const pod0 = ex<Pod>("Pod");
export function podsFor(owner: string): Pod[] {
  return [0, 1, 2, 3].map((i) =>
    v(pod0, {
      name: `${owner}-worker-${i}`,
      phase: i === 3 ? "Pending" : "Running",
      replicaIndex: i,
      nodeName: `gpu-node-${(i % 4) + 1}`,
      restartCount: i === 2 ? 1 : 0,
      startedAt: ago(6 - i),
    }),
  );
}

const event0 = ex<Event>("Event");
export function eventsFor(name: string): Event[] {
  return [
    v(event0, { reason: "Scheduled", message: `Successfully assigned ${name} to gpu-node-1`, type: "Normal", lastTimestamp: ago(6), count: 1 }),
    v(event0, { reason: "Pulled", message: "Container image already present on machine", type: "Normal", lastTimestamp: ago(5.9), count: 1 }),
    v(event0, { reason: "Started", message: "Started container worker", type: "Normal", lastTimestamp: ago(5.8), count: 1 }),
    v(event0, { reason: "BackOff", message: "Back-off restarting failed container", type: "Warning", lastTimestamp: ago(2), count: 3 }),
  ];
}

// ── metric series (demo-only: richer than the illustrative spec example) ─────────
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
// product prototype (axisml-platform/docs/product_design/prototype/index.html).
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
      // Allocated GPU quota (%), usually slightly higher than actual utilisation.
      quota: Math.min(100, util + 8 + Math.round(4 * Math.sin(i / 2.3))),
    };
  });

const clusterUsageMap: Record<string, ClusterUsage> = {
  all: { key: "all", label: "All", gpu: meter(36, 48, "cards"), cpu: meter(740, 1152, "cores"), mem: meter(3.4, 5.5, "TiB", 1), trend: trendFor(72) },
  "gpu-a100": { key: "gpu-a100", label: "gpu-a100", gpu: meter(22, 32, "cards"), cpu: meter(240, 384, "cores"), mem: meter(1.2, 2.0, "TiB", 1), trend: trendFor(69) },
  "h100-pool": { key: "h100-pool", label: "h100-pool", gpu: meter(14, 16, "cards"), cpu: meter(180, 256, "cores"), mem: meter(1.4, 2.0, "TiB", 1), trend: trendFor(86) },
  "cpu-pool": { key: "cpu-pool", label: "cpu-pool", gpu: meter(0, 0, "cards"), cpu: meter(320, 512, "cores"), mem: meter(0.8, 1.5, "TiB", 1), trend: trendFor(60) },
};

export function clusterUsage(pool?: string): ClusterUsage {
  return clusterUsageMap[pool ?? "all"] ?? clusterUsageMap.all;
}

export function podLogs(pod: string): string {
  return [
    `[INFO] ${pod} starting…`,
    "[INFO] loading checkpoint from s3://axisml-models/team-vision/resnet50/1.4.0",
    "[INFO] world_size=4 rank=0 local_rank=0",
    "[INFO] epoch 1/3 step 100 loss=1.842 lr=2.0e-5",
    "[INFO] epoch 1/3 step 200 loss=1.611 lr=2.0e-5",
    "[INFO] epoch 1/3 step 300 loss=1.498 lr=2.0e-5",
  ].join("\n");
}
