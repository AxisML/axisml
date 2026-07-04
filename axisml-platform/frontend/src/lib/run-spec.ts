import * as sdk from "@/api/generated";

// Shared spec builders for the training run drawers (Jobs & Experiments — both
// are native Job-backed Runs with the same form shape). Pools + units now come
// from the live ResourcePool list (see usePoolUnitOptions). The training-image
// catalog is still a sample list pending an image+version picker.
export const TRAINING_IMAGES = [
  { value: "pytorch:2.3-cu121", title: "pytorch:2.3-cu121", desc: "PyTorch 训练镜像" },
  { value: "megatron:24.05", title: "megatron:24.05", desc: "Megatron-LM 训练镜像" },
];

// Parse a `KEY=value` lines blob into EnvVar[] (blank lines / keyless skipped).
export function parseEnv(text: string): sdk.EnvVar[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)
    .map((l) => {
      const [name, ...rest] = l.split("=");
      return { name: name.trim(), value: rest.join("=").trim() };
    })
    .filter((e) => e.name);
}

// Split a shell command string into argv, tolerating `\`-continued lines.
export function parseCommand(text: string): string[] {
  return text
    .replace(/\\\s*\n/g, " ")
    .split(/\s+/)
    .map((s) => s.trim())
    .filter((s) => s && s !== "\\");
}

export interface VolumeInput {
  name?: string;
  mountPath?: string;
}

export interface RunSpecInput {
  image?: string;
  command?: string;
  env?: string;
  replicas: number;
  poolName?: string;
  unitName?: string;
  volumes?: VolumeInput[];
  timeout: number;
  retries: number;
}

// Build the native Job-backed run spec shared by the Job & Experiment drawers.
export function buildRunSpec(v: RunSpecInput): sdk.JobSpec {
  const reps = Number(v.replicas);
  const picked = (v.volumes ?? []).filter((vol) => vol.name?.trim() && vol.mountPath?.trim());
  const mounts = picked.map((vol) => ({ name: vol.name!.trim(), mountPath: vol.mountPath!.trim() }));
  // Each volumeMount needs a matching pod `volumes` entry, else the mount
  // references a volume that doesn't exist. The selected name is a DataVolume,
  // backed by a PVC of the same name.
  const volumes = picked.map((vol) => ({
    name: vol.name!.trim(),
    persistentVolumeClaim: { claimName: vol.name!.trim() },
  }));
  const role: sdk.MlRunRole = {
    name: "worker",
    replicas: Number.isFinite(reps) && reps > 0 ? reps : 1,
    template: {
      image: v.image?.trim() || undefined,
      command: parseCommand(v.command || ""),
      env: parseEnv(v.env || ""),
      volumes: volumes.length ? volumes : undefined,
      volumeMounts: mounts.length ? mounts : undefined,
    },
  };
  return {
    backend: { name: "native", engine: "job" },
    poolName: v.poolName?.trim() || undefined,
    unitName: v.unitName?.trim() || undefined,
    roles: [role],
    runPolicy: {
      activeDeadlineSeconds: v.timeout > 0 ? v.timeout : undefined,
      backoffLimit: v.retries >= 0 ? v.retries : undefined,
    },
  };
}
