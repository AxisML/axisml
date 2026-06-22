import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useExperiments } from "@/api/hooks";
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
import { Field, FieldLabel, FieldDescription, FieldGroup } from "@/components/ui/field";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Spinner } from "@/components/ui/spinner";

interface ExpRow {
  name: string;
  desc: string;
  runCount: number;
  recent: string[];
  owner: string;
  updated: string;
}

type DrawerMode = "new" | "run" | "edit";
const ALL = "__all__";

export default function Experiments() {
  const q = useExperiments();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; name?: string } | null>(null);
  const [search, setSearch] = useState("");
  const [creator, setCreator] = useState<string>("");

  const delExp = useApiMutation((name: string) => sdk.deleteExperiment({ path: { name } }), {
    invalidate: [["experiments"]],
    success: t("experiments.deleted"),
  });
  const triggerExp = useApiMutation(
    (name: string) => sdk.triggerExperimentRun({ path: { name }, body: {} }),
    { invalidate: [["experiments"]], success: t("experiments.runTriggered") },
  );

  const allRows: ExpRow[] = useMemo(
    () =>
      q.data?.items?.map((e) => {
        const summary = USE_MOCK ? runSummary(e.name) : { count: 0, recent: [] as string[] };
        return {
          name: e.name,
          desc: e.description ?? e.displayName ?? "",
          runCount: summary.count,
          recent: summary.recent,
          owner: e.owner ?? "—",
          updated: e.updatedAt ?? e.createdAt ?? "",
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

  const onDelete = (r: ExpRow) =>
    confirm({
      title: t("experiments.deleteTitle", { name: r.name }),
      desc: t("experiments.deleteDesc"),
      info: t("experiments.deleteInfo"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delExp.mutate(r.name),
    });

  const onRun = (r: ExpRow) =>
    confirm({
      title: t("experiments.runTitle", { name: r.name }),
      desc: t("experiments.runDesc"),
      okLabel: t("experiments.confirmRun"),
      danger: false,
      onConfirm: () => triggerExp.mutate(r.name),
    });

  const columns: Column<ExpRow>[] = [
    {
      key: "name",
      title: t("experiments.colName"),
      render: (r) => (
        <div className="min-w-0">
          <Link
            to={`/experiments/${r.name}`}
            className="font-mono font-medium text-foreground hover:text-info hover:underline"
          >
            {r.name}
          </Link>
          {r.desc && <div className="truncate text-xs text-muted-foreground">{r.desc}</div>}
        </div>
      ),
    },
    {
      key: "runs",
      title: t("experiments.colStatus"),
      width: 150,
      render: (r) => <RunStrip phases={r.recent} to={`/experiments/${r.name}`} />,
    },
    {
      key: "runCount",
      title: t("experiments.colRuns"),
      width: 90,
      align: "right",
      render: (r) => <span className="font-mono">{r.runCount}</span>,
    },
    { key: "owner", title: t("experiments.colCreator"), width: 140, dataIndex: "owner" },
    {
      key: "updated",
      title: t("experiments.colUpdated"),
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
          <Button variant="link" size="sm" onClick={() => onRun(r)}>
            {t("common.run")}
          </Button>
          <Button variant="link" size="sm" asChild>
            <Link to={`/experiments/${r.name}`}>{t("common.detail")}</Link>
          </Button>
          <Button variant="link" size="sm" onClick={() => setDrawer({ mode: "edit", name: r.name })}>
            {t("common.edit")}
          </Button>
          <Button variant="link" size="sm" className="text-destructive" onClick={() => onDelete(r)}>
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.experiments")]}
      title={t("experiments.title")}
      subtitle={t("experiments.subtitle")}
      extra={
        <Button onClick={() => setDrawer({ mode: "new" })}>
          <Plus data-icon="inline-start" />
          {t("experiments.newExperiment")}
        </Button>
      }
    >
      <Card className="overflow-hidden p-0">
        <div className="flex flex-wrap items-center gap-3 border-b p-4">
          <div className="relative max-w-xs flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-8"
              placeholder={t("experiments.searchPlaceholder")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <Select value={creator || ALL} onValueChange={(v) => setCreator(v === ALL ? "" : v)}>
            <SelectTrigger className="min-w-44">
              <SelectValue placeholder={t("experiments.creatorAll")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t("experiments.creatorAll")}</SelectItem>
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

      {drawer && <ExpDrawer mode={drawer.mode} name={drawer.name} onClose={() => setDrawer(null)} />}
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
const CMD = `torchrun --nproc_per_node=4 sft.py \\
  --base llama3-8b-base --lr {{lr}} --epochs 3`;

interface ExpFormValues {
  name: string;
  description: string;
  image: string;
  poolName: string;
  unitName: string;
  replicas: number;
  command: string;
  env: string;
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
  return text
    .replace(/\\\s*\n/g, " ")
    .split(/\s+/)
    .map((s) => s.trim())
    .filter((s) => s && s !== "\\");
}

function ExpDrawer({ mode, name: initialName, onClose }: { mode: DrawerMode; name?: string; onClose: () => void }) {
  const { t } = useTranslation();
  const locked = mode === "run";
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<ExpFormValues>({
    name: mode === "new" ? "" : (initialName ?? ""),
    description: "",
    image: IMAGES[0].value,
    poolName: POOLS[0],
    unitName: UNITS[0].value,
    replicas: 2,
    command: CMD,
    env: "WANDB_DISABLED=true\nNCCL_DEBUG=INFO",
    timeout: 172800,
    retries: 1,
  });
  const set = <K extends keyof ExpFormValues>(k: K, val: ExpFormValues[K]) =>
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

  const create = useApiMutation((body: sdk.ExperimentCreateRequest) => sdk.createExperiment({ body }), {
    invalidate: [["experiments"]],
    success: t("experiments.created"),
  });
  const update = useApiMutation(
    (vars: { name: string; body: sdk.ExperimentPatchRequest }) =>
      sdk.updateExperiment({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["experiments"]], success: t("experiments.saved") },
  );
  const trigger = useApiMutation(
    (vars: { name: string; body: sdk.RunTriggerRequest }) =>
      sdk.triggerExperimentRun({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["experiments"]], success: t("experiments.runTriggered") },
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

  const title = mode === "new" ? t("experiments.drawerNew") : mode === "run" ? t("experiments.drawerRun") : t("experiments.drawerEdit");
  const submitLabel = mode === "new" ? t("experiments.createExperiment") : mode === "run" ? t("experiments.confirmRun") : t("experiments.save");

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[560px]">
        <SheetHeader className="border-b">
          <SheetTitle>{title}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            {mode === "new" ? t("experiments.drawerNewSub") : <span className="font-mono">{initialName}</span>}
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("experiments.fsBasic")} />
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="exp-name">
                {t("experiments.fName")}
                {mode !== "run" && <span className="text-destructive">*</span>}
              </FieldLabel>
              <Input
                id="exp-name"
                className="font-mono"
                placeholder={t("experiments.fNamePlaceholder")}
                value={v.name}
                disabled={locked || mode === "edit"}
                aria-invalid={submitted && mode !== "run" && !v.name.trim()}
                onChange={(e) => set("name", e.target.value)}
              />
              {mode !== "run" && <FieldDescription>{t("experiments.fNameHelp")}</FieldDescription>}
            </Field>
            <Field>
              <FieldLabel htmlFor="exp-desc">{t("experiments.fDesc")}</FieldLabel>
              <Textarea
                id="exp-desc"
                rows={2}
                placeholder={t("experiments.fDescPlaceholder")}
                value={v.description}
                disabled={locked}
                onChange={(e) => set("description", e.target.value)}
              />
            </Field>
          </FieldGroup>

          <FieldSection n={2} title={t("experiments.fsImage")} />
          <FieldGroup>
            <Field>
              <FieldLabel>{t("experiments.fImage")}</FieldLabel>
              <CardRadio options={IMAGES} value={v.image} onChange={(val) => set("image", val)} disabled={locked} />
            </Field>
          </FieldGroup>

          <FieldSection n={3} title={t("experiments.fsResource")} />
          <FieldGroup>
            <Field>
              <FieldLabel>{t("experiments.fPool")}</FieldLabel>
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
            <Field>
              <FieldLabel>{t("experiments.fUnit")}</FieldLabel>
              <CardRadio options={UNITS} value={v.unitName} onChange={(val) => set("unitName", val)} disabled={locked} />
            </Field>
            <Field>
              <FieldLabel htmlFor="exp-replicas">{t("experiments.fReplicas")}</FieldLabel>
              <Input
                id="exp-replicas"
                type="number"
                min={1}
                className="w-40"
                value={v.replicas}
                disabled={locked}
                onChange={(e) => set("replicas", Number(e.target.value))}
              />
            </Field>
          </FieldGroup>

          <FieldSection n={4} title={t("experiments.fsCommand")} />
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="exp-cmd">
                {mode === "run" ? t("experiments.fCommandRun") : t("experiments.fCommandTpl")}
              </FieldLabel>
              <Textarea
                id="exp-cmd"
                rows={3}
                className="font-mono"
                value={v.command}
                disabled={locked}
                onChange={(e) => set("command", e.target.value)}
              />
              <FieldDescription>
                {mode === "run" ? t("experiments.fCommandHelpRun") : t("experiments.fCommandHelpTpl")}
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="exp-env">{t("experiments.fEnv")}</FieldLabel>
              <Textarea
                id="exp-env"
                rows={2}
                className="font-mono"
                value={v.env}
                disabled={locked}
                onChange={(e) => set("env", e.target.value)}
              />
              {mode !== "run" && <FieldDescription>{t("experiments.fEnvHelp")}</FieldDescription>}
            </Field>

            <Collapsible className="mt-4">
              <CollapsibleTrigger className="text-sm font-semibold text-info hover:underline">
                {t("common.advanced")}
              </CollapsibleTrigger>
              <CollapsibleContent className="mt-3 flex gap-4">
                <Field className="flex-1">
                  <FieldLabel htmlFor="exp-timeout">{t("experiments.fTimeout")}</FieldLabel>
                  <Input
                    id="exp-timeout"
                    type="number"
                    min={0}
                    value={v.timeout}
                    disabled={locked}
                    onChange={(e) => set("timeout", Number(e.target.value))}
                  />
                </Field>
                <Field className="flex-1">
                  <FieldLabel htmlFor="exp-retries">{t("experiments.fRetries")}</FieldLabel>
                  <Input
                    id="exp-retries"
                    type="number"
                    min={0}
                    value={v.retries}
                    disabled={locked}
                    onChange={(e) => set("retries", Number(e.target.value))}
                  />
                </Field>
              </CollapsibleContent>
            </Collapsible>
          </FieldGroup>
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
