import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useApiMutation } from "@/api/mutations";
import { FieldSection } from "@/components/field-section";
import { CardRadio } from "@/components/card-radio";
import { FormDrawer } from "@/components/form-drawer";
import { DetailError } from "@/components/page-state";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Spinner } from "@/components/ui/spinner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Field, FieldLabel, FieldDescription, FieldGroup } from "@/components/ui/field";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  TRAINING_IMAGES as IMAGES,
  TRAINING_UNITS as UNITS,
  TRAINING_POOLS as POOLS,
  parseEnv,
  parseCommand,
  buildRunSpec,
} from "@/lib/run-spec";

export type DrawerMode = "new" | "run" | "edit";

const VOLUMES = [
  { value: "team-datasets", label: "team-datasets · 1 TiB" },
  { value: "ckpt-store", label: "ckpt-store · 500 GiB" },
];
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
  volumes: { name?: string; mountPath?: string }[];
  timeout: number;
  retries: number;
}

const DEFAULTS: ExpFormValues = {
  name: "",
  description: "",
  image: IMAGES[0].value,
  poolName: POOLS[0],
  unitName: UNITS[0].value,
  replicas: 2,
  command: CMD,
  env: "WANDB_DISABLED=true\nNCCL_DEBUG=INFO",
  volumes: [
    { name: "team-datasets", mountPath: "/data" },
    { name: "ckpt-store", mountPath: "/output" },
  ],
  timeout: 172800,
  retries: 1,
};

// Map a fetched experiment onto the form so run / edit drawers reflect the real
// saved configuration instead of placeholder defaults.
function expToForm(exp: sdk.Experiment): ExpFormValues {
  const role = exp.spec.roles?.[0];
  const tpl = role?.template;
  const mounts = (tpl?.volumeMounts ?? []) as { name?: string; mountPath?: string }[];
  const cmd = [...(tpl?.command ?? []), ...(tpl?.args ?? [])].join(" ");
  const env = (tpl?.env ?? []).map((e) => `${e.name}=${e.value ?? ""}`).join("\n");
  return {
    name: exp.name,
    description: exp.description ?? "",
    image: tpl?.image || DEFAULTS.image,
    poolName: exp.spec.poolName || DEFAULTS.poolName,
    unitName: exp.spec.unitName || DEFAULTS.unitName,
    replicas: role?.replicas ?? DEFAULTS.replicas,
    command: cmd || DEFAULTS.command,
    env,
    volumes: mounts.length ? mounts : DEFAULTS.volumes,
    timeout: exp.spec.runPolicy?.activeDeadlineSeconds ?? DEFAULTS.timeout,
    retries: exp.spec.runPolicy?.backoffLimit ?? DEFAULTS.retries,
  };
}

// Create / Run / Edit experiment drawer, shared by the list and detail pages.
// `new` starts from defaults; `run` / `edit` fetch the experiment first so the
// form is pre-filled with its current image, resources, command, and volumes.
export function ExpDrawer({
  mode,
  name,
  onClose,
}: {
  mode: DrawerMode;
  name?: string;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const needsFetch = mode !== "new" && !!name;

  const q = useQuery({
    queryKey: ["experiments", tenant, name],
    enabled: needsFetch && tenant !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getExperiment({ path: { name: name! } });
      if (error) throw error;
      return data;
    },
  });

  if (needsFetch && (q.isLoading || q.isError || !q.data)) {
    return (
      <FormDrawer
        title={mode === "run" ? t("experiments.drawerRun") : t("experiments.drawerEdit")}
        subtitle={<span className="font-mono">{name}</span>}
        onClose={onClose}
      >
        {q.isError ? (
          <DetailError message={t("common.loadFailed")} />
        ) : (
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        )}
      </FormDrawer>
    );
  }

  return (
    <ExpForm mode={mode} initial={needsFetch ? expToForm(q.data!) : DEFAULTS} onClose={onClose} />
  );
}

function ExpForm({
  mode,
  initial,
  onClose,
}: {
  mode: DrawerMode;
  initial: ExpFormValues;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  // Run mode locks the experiment's identity + template (name, description, image,
  // replicas, volumes, run policy). Only the fields a trigger can actually override
  // stay editable — pool, unit, command (args) and env — per RunTriggerRequest.
  const locked = mode === "run";
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<ExpFormValues>(initial);
  const set = <K extends keyof ExpFormValues>(k: K, val: ExpFormValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

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
          spec: buildRunSpec(v),
        },
        { onSuccess: onClose },
      );
    } else if (mode === "edit") {
      update.mutate(
        { name: v.name.trim(), body: { description: v.description.trim() || undefined, spec: buildRunSpec(v) } },
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

  const title =
    mode === "new" ? t("experiments.drawerNew") : mode === "run" ? t("experiments.drawerRun") : t("experiments.drawerEdit");
  const submitLabel =
    mode === "new" ? t("experiments.createExperiment") : mode === "run" ? t("experiments.confirmRun") : t("experiments.save");

  return (
    <FormDrawer
      title={title}
      subtitle={mode === "new" ? undefined : <span className="font-mono">{initial.name}</span>}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={submitLabel}
      submitting={pending}
    >
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
          <Select value={v.poolName} onValueChange={(val) => set("poolName", val)}>
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
          <CardRadio options={UNITS} value={v.unitName} onChange={(val) => set("unitName", val)} />
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
            onChange={(e) => set("env", e.target.value)}
          />
          {mode !== "run" && <FieldDescription>{t("experiments.fEnvHelp")}</FieldDescription>}
        </Field>
      </FieldGroup>

      <FieldSection n={5} title={t("experiments.fsVolume")} />
      <FieldGroup>
        <Field>
          <FieldLabel>
            {t("experiments.fVolume")}
            {mode !== "run" && <span className="text-destructive">*</span>}
          </FieldLabel>
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
                    <SelectValue placeholder={t("experiments.fVolume")} />
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
                  placeholder={t("experiments.fMountPath")}
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
              + {t("experiments.addVolume")}
            </Button>
          </div>
          <FieldDescription>{t("experiments.fVolumeHelp")}</FieldDescription>
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
    </FormDrawer>
  );
}
