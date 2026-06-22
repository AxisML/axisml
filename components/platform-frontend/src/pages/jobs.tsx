import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Search, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useJobs } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { FieldSection } from "@/components/field-section";
import { CardRadio } from "@/components/card-radio";
import { RunStrip } from "@/components/run-strip";
import { DataTable, type Column } from "@/components/data-table";
import { USE_MOCK } from "@/api/mock";
import { runSummary } from "@/api/mock/data";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Field, FieldLabel, FieldDescription } from "@/components/ui/field";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Spinner } from "@/components/ui/spinner";

interface JobRow {
  name: string;
  desc: string;
  runCount: number;
  recent: string[];
  owner: string;
  updated: string;
}

type DrawerMode = "new" | "run" | "edit";
const ALL = "__all__";

export default function Jobs() {
  const q = useJobs();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; name?: string } | null>(null);
  const [search, setSearch] = useState("");
  const [creator, setCreator] = useState<string>("");

  const delJob = useApiMutation((name: string) => sdk.deleteJob({ path: { name } }), {
    invalidate: [["jobs"]],
    success: t("jobs.deleted"),
  });

  const allRows: JobRow[] = useMemo(
    () =>
      q.data?.items?.map((j) => {
        const summary = USE_MOCK ? runSummary(j.name) : { count: 0, recent: [] as string[] };
        return {
          name: j.name,
          desc: j.description ?? j.displayName ?? "",
          runCount: summary.count,
          recent: summary.recent,
          owner: j.owner ?? "—",
          updated: j.updatedAt ?? j.createdAt ?? "",
        };
      }) ?? [],
    [q.data],
  );

  const creatorOptions = useMemo(
    () => Array.from(new Set(allRows.map((r) => r.owner).filter((o) => o && o !== "—"))),
    [allRows],
  );

  const rows = allRows.filter(
    (r) => (!search || r.name.includes(search)) && (!creator || r.owner === creator),
  );

  const onDelete = (r: JobRow) =>
    confirm({
      title: t("jobs.deleteTitle", { name: r.name }),
      desc: t("jobs.deleteDesc"),
      info: t("jobs.deleteInfo"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delJob.mutate(r.name),
    });

  const columns: Column<JobRow>[] = [
    {
      key: "name",
      title: t("jobs.colName"),
      render: (r) => (
        <div className="min-w-0">
          <Link to={`/jobs/${r.name}`} className="font-mono font-medium text-foreground hover:text-info hover:underline">
            {r.name}
          </Link>
          {r.desc && <div className="truncate text-xs text-muted-foreground">{r.desc}</div>}
        </div>
      ),
    },
    {
      key: "runs",
      title: t("jobs.colStatus"),
      width: 150,
      render: (r) => <RunStrip phases={r.recent} to={`/jobs/${r.name}`} />,
    },
    {
      key: "runCount",
      title: t("jobs.colRuns"),
      width: 90,
      align: "right",
      render: (r) => <span className="font-mono">{r.runCount}</span>,
    },
    { key: "owner", title: t("jobs.colCreator"), width: 140, dataIndex: "owner" },
    {
      key: "updated",
      title: t("jobs.colUpdated"),
      width: 150,
      render: (r) => (
        <span className="text-muted-foreground">{r.updated ? dayjs(r.updated).fromNow() : "—"}</span>
      ),
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 200,
      align: "right",
      render: (r) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" onClick={() => setDrawer({ mode: "run", name: r.name })}>
            {t("common.run")}
          </Button>
          <Button variant="link" size="sm" asChild>
            <Link to={`/jobs/${r.name}`}>{t("common.detail")}</Link>
          </Button>
          <Button variant="link" size="sm" onClick={() => setDrawer({ mode: "edit", name: r.name })}>
            {t("common.edit")}
          </Button>
          <Button
            variant="link"
            size="sm"
            className="text-destructive"
            onClick={() => onDelete(r)}
          >
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.jobs")]}
      title={t("jobs.title")}
      subtitle={t("jobs.subtitle")}
      extra={
        <Button onClick={() => setDrawer({ mode: "new" })}>
          <Plus data-icon="inline-start" />
          {t("jobs.newJob")}
        </Button>
      }
    >
      <Card className="overflow-hidden p-0">
        <div className="flex flex-wrap items-center gap-3 border-b p-4">
          <div className="relative max-w-xs flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-8"
              placeholder={t("jobs.searchPlaceholder")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <Select value={creator || ALL} onValueChange={(v) => setCreator(v === ALL ? "" : v)}>
            <SelectTrigger className="min-w-44">
              <SelectValue placeholder={t("jobs.creatorAll")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t("jobs.creatorAll")}</SelectItem>
              {creatorOptions.map((o) => (
                <SelectItem key={o} value={o}>
                  {o}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            onClick={() => {
              setSearch("");
              setCreator("");
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          rowKey={(r) => r.name}
          loading={q.isLoading}
          error={q.isError}
        />
      </Card>

      {drawer && <JobDrawer mode={drawer.mode} name={drawer.name} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Create / Run / Edit drawer ────────────────────────────────────────────────
const IMAGES = [
  { value: "pytorch:2.3-cu121", title: "pytorch:2.3-cu121", desc: "PyTorch 训练镜像" },
  { value: "megatron:24.05", title: "megatron:24.05", desc: "Megatron-LM 训练镜像" },
];
const UNITS = [
  { value: "a100-4x-xlarge", title: "a100-4x-xlarge", desc: "4×A100 · 32 vCPU · 256 GiB" },
  { value: "a100-8x-xlarge-ib", title: "a100-8x-xlarge-ib", desc: "8×A100 · IB · 64 vCPU · 512 GiB" },
];
const POOLS = ["gpu-a100", "gpu-h100"];
const VOLUMES = [
  { value: "training-data", label: "training-data · 200 GiB" },
  { value: "shared-cache", label: "shared-cache · 500 GiB" },
];
const CMD = `torchrun --nproc_per_node=4 train.py \\
  --model_name llama-7b --lr 2e-5 --epochs 3 \\
  --batch_size 16 --data /data/sft.jsonl`;

interface JobFormValues {
  name: string;
  description: string;
  image: string;
  poolName: string;
  unitName: string;
  replicas: number;
  command: string;
  env: string;
  volumes: { name?: string; mountPath?: string }[];
  timeout: number;
  retries: number;
}

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

function parseCommand(text: string): string[] {
  return text.split(/\s+/).map((s) => s.trim()).filter(Boolean);
}

function JobDrawer({ mode, name: initialName, onClose }: { mode: DrawerMode; name?: string; onClose: () => void }) {
  const { t } = useTranslation();
  const locked = mode === "run";
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<JobFormValues>({
    name: mode === "new" ? "" : (initialName ?? ""),
    description: "",
    image: IMAGES[0].value,
    poolName: POOLS[0],
    unitName: UNITS[0].value,
    replicas: 4,
    command: CMD,
    env: "WANDB_DISABLED=true\nNCCL_DEBUG=INFO",
    volumes: [{ name: "training-data", mountPath: "/data" }],
    timeout: 86400,
    retries: 2,
  });
  const set = <K extends keyof JobFormValues>(k: K, val: JobFormValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

  const buildSpec = (): sdk.JobSpec => {
    const reps = Number(v.replicas);
    const role: sdk.MlRunRole = {
      name: "worker",
      replicas: Number.isFinite(reps) && reps > 0 ? reps : 1,
      template: {
        image: v.image?.trim() || undefined,
        command: parseCommand(v.command || ""),
        env: parseEnv(v.env || ""),
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
  };

  const create = useApiMutation((body: sdk.JobCreateRequest) => sdk.createJob({ body }), {
    invalidate: [["jobs"]],
    success: t("jobs.savedTemplate"),
  });
  const update = useApiMutation(
    (vars: { name: string; body: sdk.JobPatchRequest }) =>
      sdk.updateJob({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["jobs"]], success: t("jobs.saved") },
  );
  const trigger = useApiMutation(
    (vars: { name: string; body: sdk.RunTriggerRequest }) =>
      sdk.triggerRun({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["jobs"]], success: t("jobs.runCreated") },
  );
  const pending = create.isPending || update.isPending || trigger.isPending;

  const submit = () => {
    setSubmitted(true);
    if (mode !== "run" && !v.name.trim()) return;
    if (mode === "new") {
      create.mutate(
        {
          name: v.name.trim(),
          displayName: v.name.trim() || undefined,
          description: v.description.trim() || undefined,
          spec: buildSpec(),
        },
        { onSuccess: onClose },
      );
    } else if (mode === "edit") {
      update.mutate(
        { name: v.name.trim(), body: { description: v.description.trim() || undefined, spec: buildSpec() } },
        { onSuccess: onClose },
      );
    } else {
      trigger.mutate(
        {
          name: v.name.trim(),
          body: {
            poolName: v.poolName?.trim() || undefined,
            unitName: v.unitName?.trim() || undefined,
            roles: [{ name: "worker", args: parseCommand(v.command || ""), env: parseEnv(v.env || "") }],
          },
        },
        { onSuccess: onClose },
      );
    }
  };

  const title = mode === "new" ? t("jobs.drawerNew") : mode === "run" ? t("jobs.drawerRun") : t("jobs.drawerEdit");
  const submitLabel = mode === "new" ? t("jobs.saveTemplate") : mode === "run" ? t("jobs.confirmRun") : t("common.save");

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[760px]">
        <SheetHeader className="border-b">
          <SheetTitle>{title}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            {mode === "new" ? t("jobs.drawerNewSub") : <span className="font-mono">{initialName}</span>}
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("jobs.fsBasic")} />
          <Field className="mb-4">
            <FieldLabel htmlFor="job-name">
              {t("jobs.fName")}
              {mode !== "run" && <span className="text-destructive">*</span>}
            </FieldLabel>
            <Input
              id="job-name"
              className="font-mono"
              placeholder={t("jobs.fNamePlaceholder")}
              value={v.name}
              disabled={locked || mode === "edit"}
              aria-invalid={submitted && mode !== "run" && !v.name.trim()}
              onChange={(e) => set("name", e.target.value)}
            />
            {mode !== "run" && <FieldDescription>{t("jobs.fNameHelp")}</FieldDescription>}
          </Field>
          <Field className="mb-4">
            <FieldLabel htmlFor="job-desc">{t("jobs.fDesc")}</FieldLabel>
            <Textarea
              id="job-desc"
              rows={2}
              placeholder={t("jobs.fDescPlaceholder")}
              value={v.description}
              disabled={locked}
              onChange={(e) => set("description", e.target.value)}
            />
          </Field>

          <FieldSection n={2} title={t("jobs.fsImage")} />
          <Field className="mb-4">
            <FieldLabel>{t("jobs.fImage")}</FieldLabel>
            <CardRadio options={IMAGES} value={v.image} onChange={(val) => set("image", val)} disabled={locked} />
          </Field>

          <FieldSection n={3} title={t("jobs.fsResource")} />
          <Field className="mb-4">
            <FieldLabel>{t("jobs.fPool")}</FieldLabel>
            <Select value={v.poolName} onValueChange={(val) => set("poolName", val)} disabled={locked}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {POOLS.map((p) => (
                  <SelectItem key={p} value={p}>
                    {p}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field className="mb-4">
            <FieldLabel>{t("jobs.fUnit")}</FieldLabel>
            <CardRadio options={UNITS} value={v.unitName} onChange={(val) => set("unitName", val)} disabled={locked} />
          </Field>
          <Field className="mb-4">
            <FieldLabel htmlFor="job-replicas">{t("jobs.fReplicas")}</FieldLabel>
            <Input
              id="job-replicas"
              type="number"
              min={1}
              className="w-40"
              value={v.replicas}
              disabled={locked}
              onChange={(e) => set("replicas", Number(e.target.value))}
            />
          </Field>

          <FieldSection n={4} title={t("jobs.fsCommand")} />
          <Field className="mb-4">
            <FieldLabel htmlFor="job-cmd">{t("jobs.fCommand")}</FieldLabel>
            <Textarea
              id="job-cmd"
              rows={3}
              className="font-mono"
              value={v.command}
              disabled={locked}
              onChange={(e) => set("command", e.target.value)}
            />
            {mode !== "run" && <FieldDescription>{t("jobs.fCommandHelp")}</FieldDescription>}
          </Field>
          <Field className="mb-4">
            <FieldLabel htmlFor="job-env">{t("jobs.fEnv")}</FieldLabel>
            <Textarea
              id="job-env"
              rows={2}
              className="font-mono"
              value={v.env}
              disabled={locked}
              onChange={(e) => set("env", e.target.value)}
            />
            {mode !== "run" && <FieldDescription>{t("jobs.fEnvHelp")}</FieldDescription>}
          </Field>

          <FieldSection n={5} title={t("jobs.fsVolume")} />
          <div className="flex flex-col gap-2.5">
            {v.volumes.map((vol, i) => (
              <div key={i} className="flex items-start gap-2">
                <Select
                  value={vol.name}
                  onValueChange={(val) =>
                    set("volumes", v.volumes.map((x, j) => (j === i ? { ...x, name: val } : x)))
                  }
                  disabled={locked}
                >
                  <SelectTrigger className="flex-1">
                    <SelectValue placeholder={t("jobs.fVolume")} />
                  </SelectTrigger>
                  <SelectContent>
                    {VOLUMES.map((o) => (
                      <SelectItem key={o.value} value={o.value}>
                        {o.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  className="flex-1 font-mono"
                  placeholder={t("jobs.fMountPath")}
                  value={vol.mountPath ?? ""}
                  disabled={locked}
                  onChange={(e) =>
                    set("volumes", v.volumes.map((x, j) => (j === i ? { ...x, mountPath: e.target.value } : x)))
                  }
                />
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive"
                  disabled={locked}
                  onClick={() => set("volumes", v.volumes.filter((_, j) => j !== i))}
                >
                  <Trash2 />
                </Button>
              </div>
            ))}
            <Button
              variant="outline"
              className="w-full border-dashed"
              disabled={locked}
              onClick={() => set("volumes", [...v.volumes, { mountPath: "/data" }])}
            >
              + {t("jobs.addVolume")}
            </Button>
          </div>

          <Collapsible className="mt-4">
            <CollapsibleTrigger className="text-sm font-semibold text-info hover:underline">
              {t("common.advanced")}
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-3 flex gap-4">
              <Field className="flex-1">
                <FieldLabel htmlFor="job-timeout">{t("jobs.fTimeout")}</FieldLabel>
                <Input
                  id="job-timeout"
                  type="number"
                  min={0}
                  value={v.timeout}
                  disabled={locked}
                  onChange={(e) => set("timeout", Number(e.target.value))}
                />
              </Field>
              <Field className="flex-1">
                <FieldLabel htmlFor="job-retries">{t("jobs.fRetries")}</FieldLabel>
                <Input
                  id="job-retries"
                  type="number"
                  min={0}
                  value={v.retries}
                  disabled={locked}
                  onChange={(e) => set("retries", Number(e.target.value))}
                />
              </Field>
            </CollapsibleContent>
          </Collapsible>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={pending}>
            {pending && <Spinner data-icon="inline-start" />}
            {submitLabel}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
