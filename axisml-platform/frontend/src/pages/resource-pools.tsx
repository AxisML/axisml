import { useMemo, useState } from "react";
import { Plus, Trash2, Pencil, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useResourcePools } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { FieldSection } from "@/components/field-section";
import { SearchInput } from "@/components/search-input";
import { DataTable, type Column } from "@/components/data-table";
import { fmtDateTime } from "@/lib/format";
import { unitSpecLine, memGiB } from "@/lib/units";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  InputGroupText,
} from "@/components/ui/input-group";
import { FormDrawer } from "@/components/form-drawer";
import { Field, FieldLabel, FieldDescription, FieldGroup } from "@/components/ui/field";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";

// ── Node-selector / resource-map value helpers ────────────────────────────────
// Both nodeSelector and capacity are flat string maps. Edit them as ordered
// [key,value] pairs so values remain easy to enter and duplicate keys replace
// deterministically.
type Pair = [string, string];

function toPairs(sel?: sdk.StringMap): Pair[] {
  return Object.entries(sel ?? {});
}

function pairsToSelector(pairs: Pair[]): sdk.StringMap | undefined {
  const out: sdk.StringMap = {};
  for (const [k, v] of pairs) {
    const key = k.trim();
    if (key) out[key] = v.trim();
  }
  return Object.keys(out).length ? out : undefined;
}

function pairsToResources(pairs: Pair[]): sdk.ResourceMap {
  const out: sdk.ResourceMap = {};
  for (const [k, v] of pairs) {
    const key = k.trim();
    if (key) out[key] = v.trim();
  }
  return out;
}

const num = (m: sdk.ResourceMap | undefined, k: string): number | undefined => {
  const v = m?.[k];
  if (v == null) return undefined;
  const n = parseFloat(String(v));
  return Number.isFinite(n) ? n : undefined;
};

export default function ResourcePools() {
  const q = useResourcePools();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [search, setSearch] = useState("");
  const [creating, setCreating] = useState(false);
  const [manageName, setManageName] = useState<string | null>(null);

  const delPool = useApiMutation((pool: string) => sdk.deleteResourcePool({ path: { pool } }), {
    invalidate: [["resourcepools"]],
    success: t("pools.deleted"),
  });

  const allRows = q.data?.items ?? [];
  const rows = useMemo(
    () => allRows.filter((p) => !search || p.name.includes(search)),
    [allRows, search],
  );
  // The manage drawer re-derives its pool from live query data so unit edits
  // (which invalidate the list) reflect immediately without a stale snapshot.
  const managePool = manageName ? allRows.find((p) => p.name === manageName) : undefined;

  const onDelete = (p: sdk.ResourcePool) =>
    confirm({
      title: t("pools.deleteTitle", { name: p.name }),
      desc: t("pools.deleteDesc"),
      info: t("pools.deleteInfo"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delPool.mutate(p.name),
    });

  const columns: Column<sdk.ResourcePool>[] = [
    {
      key: "name",
      title: t("pools.colName"),
      render: (p) => (
        <button
          type="button"
          className="font-mono font-medium text-foreground hover:text-info hover:underline"
          onClick={() => setManageName(p.name)}
        >
          {p.name}
        </button>
      ),
    },
    {
      key: "description",
      title: t("pools.colDesc"),
      render: (p) => p.description || <span className="text-muted-foreground">—</span>,
    },
    {
      key: "selector",
      title: t("pools.colSelector"),
      render: (p) => {
        const pairs = toPairs(p.nodeSelector);
        if (!pairs.length) return <span className="text-muted-foreground">{t("pools.noSelector")}</span>;
        const shown = pairs.slice(0, 2);
        const overflow = pairs.length - shown.length;
        return (
          <div className="flex flex-wrap items-center gap-1">
            {shown.map(([k, v]) => (
              <Badge key={k} variant="outline" className="font-mono">
                {k}={v}
              </Badge>
            ))}
            {overflow > 0 && <Badge variant="outline">{t("pools.more", { count: overflow })}</Badge>}
          </div>
        );
      },
    },
    {
      key: "units",
      title: t("pools.colUnits"),
      width: 100,
      align: "right",
      render: (p) => (
        <button
          type="button"
          className="font-mono text-info hover:underline"
          onClick={() => setManageName(p.name)}
        >
          {p.units?.length ?? 0}
        </button>
      ),
    },
    {
      key: "createdAt",
      title: t("pools.colCreated"),
      width: 180,
      render: (p) => <span className="text-muted-foreground">{fmtDateTime(p.createdAt)}</span>,
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 140,
      align: "right",
      render: (p) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" onClick={() => setManageName(p.name)}>
            {t("pools.manage")}
          </Button>
          <Button variant="link" size="sm" className="text-destructive" onClick={() => onDelete(p)}>
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.systemMgmt"), t("nav.pools")]}
      title={t("pools.title")}
      subtitle={t("pools.subtitle")}
      extra={
        <Button onClick={() => setCreating(true)}>
          <Plus data-icon="inline-start" />
          {t("pools.newPool")}
        </Button>
      }
    >
      <Card className="overflow-hidden p-0">
        <div className="flex flex-wrap items-center gap-3 border-b p-4">
          <SearchInput
            className="max-w-xs flex-1"
            placeholder={t("pools.searchPlaceholder")}
            value={search}
            onChange={setSearch}
          />
          <Button variant="outline" onClick={() => setSearch("")}>
            {t("common.reset")}
          </Button>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          rowKey={(p) => p.name}
          loading={q.isLoading}
          error={q.isError}
        />
      </Card>

      {creating && <PoolCreateDrawer onClose={() => setCreating(false)} />}
      {managePool && (
        <ManagePoolDrawer
          key={managePool.name}
          pool={managePool}
          onClose={() => setManageName(null)}
        />
      )}
    </PageContainer>
  );
}

// ── Key/value chip editor ─────────────────────────────────────────────────────
function KeyValueChips({
  pairs,
  onChange,
  keyPlaceholder,
  valuePlaceholder,
  addLabel,
}: {
  pairs: Pair[];
  onChange: (next: Pair[]) => void;
  keyPlaceholder: string;
  valuePlaceholder: string;
  addLabel: string;
}) {
  const { t } = useTranslation();
  const [k, setK] = useState("");
  const [v, setV] = useState("");

  const add = () => {
    const key = k.trim();
    if (!key) return;
    onChange([...pairs.filter(([pk]) => pk !== key), [key, v.trim()]]);
    setK("");
    setV("");
  };

  return (
    <div className="flex flex-col gap-2">
      {pairs.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {pairs.map(([pk, pv], i) => (
            <Badge key={`${pk}-${i}`} variant="outline" className="gap-1 font-mono">
              {pk}={pv}
              <button
                type="button"
                className="text-muted-foreground hover:text-foreground"
                aria-label={t("common.delete")}
                onClick={() => onChange(pairs.filter((_, j) => j !== i))}
              >
                <X className="size-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}
      <div className="flex items-center gap-2">
        <Input
          className="font-mono"
          placeholder={keyPlaceholder}
          value={k}
          onChange={(e) => setK(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), add())}
        />
        <span className="text-muted-foreground">=</span>
        <Input
          className="font-mono"
          placeholder={valuePlaceholder}
          value={v}
          onChange={(e) => setV(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), add())}
        />
        <Button type="button" variant="outline" onClick={add} disabled={!k.trim()}>
          {addLabel}
        </Button>
      </div>
    </div>
  );
}

// ── Create-pool drawer (basics + node scheduling; units added after) ──────────
function PoolCreateDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const [submitted, setSubmitted] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [pairs, setPairs] = useState<Pair[]>([]);
  const [capacityEnabled, setCapacityEnabled] = useState(false);
  const [capacityPairs, setCapacityPairs] = useState<Pair[]>([]);

  const create = useApiMutation(
    (body: sdk.ResourcePoolCreateRequest) => sdk.createResourcePool({ body }),
    { invalidate: [["resourcepools"]], success: t("pools.created2") },
  );

  const submit = () => {
    setSubmitted(true);
    const n = name.trim();
    if (!n) return;
    create.mutate(
      {
        name: n,
        description: description.trim() || undefined,
        nodeSelector: pairsToSelector(pairs),
        capacity: capacityEnabled ? pairsToResources(capacityPairs) : undefined,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <FormDrawer
      title={t("pools.drawerNew")}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("pools.createPool")}
      submitting={create.isPending}
    >
      <FieldSection n={1} title={t("pools.fsBasic")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="pool-name">
            {t("pools.fName")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <Input
            id="pool-name"
            className="font-mono"
            placeholder={t("pools.fNamePlaceholder")}
            value={name}
            aria-invalid={submitted && !name.trim()}
            onChange={(e) => setName(e.target.value)}
          />
          <FieldDescription>{t("pools.fNameHelp")}</FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="pool-desc">{t("pools.fDesc")}</FieldLabel>
          <Textarea
            id="pool-desc"
            rows={2}
            placeholder={t("pools.fDescPlaceholder")}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </Field>
      </FieldGroup>

      <FieldSection n={2} title={t("pools.fsSchedule")} />
      <FieldGroup>
        <Field>
          <FieldLabel>{t("pools.fSelector")}</FieldLabel>
          <KeyValueChips
            pairs={pairs}
            onChange={setPairs}
            keyPlaceholder={t("pools.selectorKey")}
            valuePlaceholder={t("pools.selectorValue")}
            addLabel={t("pools.selectorAdd")}
          />
        </Field>
        <Field>
          <div className="flex items-center gap-2">
            <Checkbox
              id="pool-capacity-enabled"
              checked={capacityEnabled}
              onCheckedChange={(checked) => setCapacityEnabled(checked === true)}
            />
            <FieldLabel htmlFor="pool-capacity-enabled">{t("pools.fCapacity")}</FieldLabel>
          </div>
          <FieldDescription>{t("pools.fCapacityHelp")}</FieldDescription>
          {capacityEnabled && (
            <KeyValueChips
              pairs={capacityPairs}
              onChange={setCapacityPairs}
              keyPlaceholder={t("pools.capacityResource")}
              valuePlaceholder={t("pools.capacityQuantity")}
              addLabel={t("pools.selectorAdd")}
            />
          )}
        </Field>
      </FieldGroup>
    </FormDrawer>
  );
}

// ── Manage-pool drawer: basics + scheduling (staged, saved together) +
//    resource-unit cards (live CRUD via the nested unit-form drawer) ───────────
type UnitDrawer = { kind: "new" } | { kind: "edit"; unit: sdk.ResourceUnit };

function ManagePoolDrawer({ pool, onClose }: { pool: sdk.ResourcePool; onClose: () => void }) {
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [description, setDescription] = useState(pool.description ?? "");
  const [pairs, setPairs] = useState<Pair[]>(() => toPairs(pool.nodeSelector));
  const [capacityEnabled, setCapacityEnabled] = useState(
    () => Object.keys(pool.capacity ?? {}).length > 0,
  );
  const [capacityPairs, setCapacityPairs] = useState<Pair[]>(() => toPairs(pool.capacity));
  const [unitDrawer, setUnitDrawer] = useState<UnitDrawer | null>(null);

  const units = pool.units ?? [];

  const update = useApiMutation(
    (body: sdk.ResourcePoolPatchRequest) =>
      sdk.updateResourcePool({ path: { pool: pool.name }, body }),
    { invalidate: [["resourcepools"]], success: t("pools.saved") },
  );
  const delUnit = useApiMutation(
    (unit: string) => sdk.deleteResourceUnit({ path: { pool: pool.name, unit } }),
    { invalidate: [["resourcepools"]], success: t("pools.unitDeleted") },
  );

  const save = () =>
    update.mutate(
      {
        description: description.trim() || undefined,
        nodeSelector: pairsToSelector(pairs),
        capacity: capacityEnabled ? pairsToResources(capacityPairs) : {},
      },
      { onSuccess: onClose },
    );

  const removeUnit = (u: sdk.ResourceUnit) =>
    confirm({
      title: t("pools.unitDeleteTitle", { name: u.name }),
      desc: t("pools.unitDeleteDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delUnit.mutate(u.name),
    });

  return (
    <>
      <FormDrawer
        title={<span className="font-mono">{pool.name}</span>}
        onClose={onClose}
        onSubmit={save}
        submitLabel={t("common.save")}
        submitting={update.isPending}
      >
        <FieldSection n={1} title={t("pools.fsBasic")} />
        <FieldGroup>
          <Field>
            <FieldLabel>{t("pools.fName")}</FieldLabel>
            <Input className="font-mono" value={pool.name} readOnly aria-readonly />
          </Field>
          <Field>
            <FieldLabel htmlFor="mp-desc">{t("pools.fDesc")}</FieldLabel>
            <Textarea
              id="mp-desc"
              rows={2}
              placeholder={t("pools.fDescPlaceholder")}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </Field>
        </FieldGroup>

        <FieldSection n={2} title={t("pools.fsSchedule")} />
        <FieldGroup>
          <Field>
            <FieldLabel>{t("pools.fSelector")}</FieldLabel>
            <KeyValueChips
              pairs={pairs}
              onChange={setPairs}
              keyPlaceholder={t("pools.selectorKey")}
              valuePlaceholder={t("pools.selectorValue")}
              addLabel={t("pools.selectorAdd")}
            />
          </Field>
          <Field>
            <div className="flex items-center gap-2">
              <Checkbox
                id="manage-pool-capacity-enabled"
                checked={capacityEnabled}
                onCheckedChange={(checked) => setCapacityEnabled(checked === true)}
              />
              <FieldLabel htmlFor="manage-pool-capacity-enabled">{t("pools.fCapacity")}</FieldLabel>
            </div>
            <FieldDescription>{t("pools.fCapacityHelp")}</FieldDescription>
            {capacityEnabled && (
              <KeyValueChips
                pairs={capacityPairs}
                onChange={setCapacityPairs}
                keyPlaceholder={t("pools.capacityResource")}
                valuePlaceholder={t("pools.capacityQuantity")}
                addLabel={t("pools.selectorAdd")}
              />
            )}
          </Field>
        </FieldGroup>

        <FieldSection n={3} title={t("pools.fsUnits")} />
        {units.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyTitle>{t("pools.unitsEmpty")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {units.map((u) => (
              <Card key={u.name} className="group gap-0 p-3">
                <div className="flex items-start justify-between gap-2">
                  <span className="truncate font-mono text-sm font-semibold">{u.name}</span>
                  <div className="flex shrink-0 items-center gap-0.5">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={t("common.edit")}
                      onClick={() => setUnitDrawer({ kind: "edit", unit: u })}
                    >
                      <Pencil />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-destructive"
                      aria-label={t("common.delete")}
                      onClick={() => removeUnit(u)}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                </div>
                <div className="mt-1.5 font-mono text-xs text-muted-foreground">
                  {unitSpecLine(u, t("pools.uGpu"))}
                </div>
              </Card>
            ))}
          </div>
        )}
        <Button
          variant="outline"
          className="mt-3 w-full border-dashed"
          onClick={() => setUnitDrawer({ kind: "new" })}
        >
          <Plus data-icon="inline-start" />
          {t("pools.newUnit")}
        </Button>
      </FormDrawer>

      {unitDrawer && (
        <UnitFormDrawer
          poolName={pool.name}
          unit={unitDrawer.kind === "edit" ? unitDrawer.unit : undefined}
          onClose={() => setUnitDrawer(null)}
        />
      )}
    </>
  );
}

// ── Unit form drawer: basics + requests/limits matrix + node scheduling ───────
interface UnitForm {
  name: string;
  description: string;
  cpuReq?: number;
  cpuLim?: number;
  memReq?: number;
  memLim?: number;
  gpu?: number;
  lock: boolean;
  pairs: Pair[];
}

function unitToForm(u?: sdk.ResourceUnit): UnitForm {
  const cpuReq = num(u?.requests, "cpu");
  const cpuLim = num(u?.limits, "cpu");
  const memReq = memGiB(u?.requests, "memory");
  const memLim = memGiB(u?.limits, "memory");
  return {
    name: u?.name ?? "",
    description: u?.description ?? "",
    cpuReq,
    cpuLim,
    memReq,
    memLim,
    gpu: num(u?.requests, "nvidia.com/gpu"),
    lock: !u || (cpuReq === cpuLim && memReq === memLim),
    pairs: toPairs(u?.nodeSelector),
  };
}

function formToMaps(f: UnitForm): { requests: sdk.ResourceMap; limits: sdk.ResourceMap } {
  const requests: sdk.ResourceMap = {};
  const limits: sdk.ResourceMap = {};
  if (f.cpuReq != null) requests.cpu = String(f.cpuReq);
  if (f.memReq != null) requests.memory = `${f.memReq}Gi`;
  const cpuLim = f.lock ? f.cpuReq : f.cpuLim;
  const memLim = f.lock ? f.memReq : f.memLim;
  if (cpuLim != null) limits.cpu = String(cpuLim);
  if (memLim != null) limits.memory = `${memLim}Gi`;
  if (f.gpu != null && f.gpu > 0) {
    requests["nvidia.com/gpu"] = String(f.gpu);
    limits["nvidia.com/gpu"] = String(f.gpu);
  }
  return { requests, limits };
}

function UnitFormDrawer({
  poolName,
  unit,
  onClose,
}: {
  poolName: string;
  unit?: sdk.ResourceUnit;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const editing = !!unit;
  const [submitted, setSubmitted] = useState(false);
  const [f, setF] = useState<UnitForm>(() => unitToForm(unit));
  const set = <K extends keyof UnitForm>(k: K, v: UnitForm[K]) =>
    setF((prev) => ({ ...prev, [k]: v }));
  const numInput = (v: string) => (v === "" ? undefined : Number(v));

  const create = useApiMutation(
    (body: sdk.ResourceUnitCreateRequest) =>
      sdk.createResourceUnit({ path: { pool: poolName }, body }),
    { invalidate: [["resourcepools"]], success: t("pools.unitCreated") },
  );
  const update = useApiMutation(
    (vars: { unit: string; body: sdk.ResourceUnitPatchRequest }) =>
      sdk.updateResourceUnit({ path: { pool: poolName, unit: vars.unit }, body: vars.body }),
    { invalidate: [["resourcepools"]], success: t("pools.unitSaved") },
  );
  const pending = create.isPending || update.isPending;

  const submit = () => {
    setSubmitted(true);
    const name = f.name.trim();
    if (!name || f.cpuReq == null || f.memReq == null) return;
    const { requests, limits } = formToMaps(f);
    const nodeSelector = pairsToSelector(f.pairs);
    const description = f.description.trim() || undefined;
    if (editing) {
      update.mutate(
        { unit: unit!.name, body: { description, requests, limits, nodeSelector } },
        { onSuccess: onClose },
      );
    } else {
      create.mutate(
        { name, description, requests, limits, nodeSelector },
        { onSuccess: onClose },
      );
    }
  };

  const affix = (
    value: number | undefined,
    onChange: (v: number | undefined) => void,
    suffix: string,
    invalid?: boolean,
  ) => (
    <InputGroup aria-invalid={invalid}>
      <InputGroupInput
        type="number"
        min={0}
        inputMode="numeric"
        value={value ?? ""}
        onChange={(e) => onChange(numInput(e.target.value))}
      />
      <InputGroupAddon align="inline-end">
        <InputGroupText>{suffix}</InputGroupText>
      </InputGroupAddon>
    </InputGroup>
  );

  return (
    <FormDrawer
      title={editing ? t("pools.unitDrawerEdit") : t("pools.unitDrawerNew")}
      subtitle={<span className="font-mono">{poolName}</span>}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={editing ? t("common.save") : t("pools.createUnit")}
      submitting={pending}
    >
      <FieldSection n={1} title={t("pools.fsBasic")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="uf-name">
            {t("pools.uName")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <Input
            id="uf-name"
            className="font-mono"
            placeholder={t("pools.uNamePlaceholder")}
            value={f.name}
            disabled={editing}
            aria-invalid={submitted && !editing && !f.name.trim()}
            onChange={(e) => set("name", e.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="uf-desc">{t("pools.uDesc")}</FieldLabel>
          <Textarea
            id="uf-desc"
            rows={2}
            placeholder={t("pools.uDescPlaceholder")}
            value={f.description}
            onChange={(e) => set("description", e.target.value)}
          />
        </Field>
      </FieldGroup>

      <FieldSection n={2} title={t("pools.fsSpec")}>
        <label className="ml-auto flex cursor-pointer items-center gap-1.5 text-xs font-normal text-muted-foreground">
          <Checkbox checked={f.lock} onCheckedChange={(c) => set("lock", c === true)} />
          {t("pools.lockLabel")}
        </label>
      </FieldSection>
      {f.lock ? (
        <div className="grid grid-cols-[56px_1fr] items-center gap-x-4 gap-y-3">
          <span className="text-sm font-medium">
            {t("pools.uCpu")}
            <span className="text-destructive">*</span>
          </span>
          {affix(f.cpuReq, (v) => set("cpuReq", v), t("pools.uCpuUnit"), submitted && f.cpuReq == null)}
          <span className="text-sm font-medium">
            {t("pools.uMem")}
            <span className="text-destructive">*</span>
          </span>
          {affix(f.memReq, (v) => set("memReq", v), t("pools.uMemUnit"), submitted && f.memReq == null)}
          <span className="text-sm font-medium">{t("pools.uGpu")}</span>
          {affix(f.gpu, (v) => set("gpu", v), t("pools.uGpuUnit"))}
          <span />
          <span className="font-mono text-xs text-muted-foreground">{t("pools.reqEqLim")}</span>
        </div>
      ) : (
        <div className="grid grid-cols-[56px_1fr_1fr] items-center gap-x-4 gap-y-3">
          <span />
          <span className="text-xs text-muted-foreground">{t("pools.uReq")}</span>
          <span className="text-xs text-muted-foreground">{t("pools.uLim")}</span>
          <span className="text-sm font-medium">
            {t("pools.uCpu")}
            <span className="text-destructive">*</span>
          </span>
          {affix(f.cpuReq, (v) => set("cpuReq", v), t("pools.uCpuUnit"), submitted && f.cpuReq == null)}
          {affix(f.cpuLim, (v) => set("cpuLim", v), t("pools.uCpuUnit"))}
          <span className="text-sm font-medium">
            {t("pools.uMem")}
            <span className="text-destructive">*</span>
          </span>
          {affix(f.memReq, (v) => set("memReq", v), t("pools.uMemUnit"), submitted && f.memReq == null)}
          {affix(f.memLim, (v) => set("memLim", v), t("pools.uMemUnit"))}
          <span className="text-sm font-medium">{t("pools.uGpu")}</span>
          {affix(f.gpu, (v) => set("gpu", v), t("pools.uGpuUnit"))}
          <span className="self-center font-mono text-xs text-muted-foreground">
            {t("pools.reqEqLim")}
          </span>
        </div>
      )}

      <FieldSection n={3} title={t("pools.fsSchedule")} />
      <FieldGroup>
        <Field>
          <FieldLabel>{t("pools.uSelector")}</FieldLabel>
          <KeyValueChips
            pairs={f.pairs}
            onChange={(p) => set("pairs", p)}
            keyPlaceholder={t("pools.selectorKey")}
            valuePlaceholder={t("pools.selectorValue")}
            addLabel={t("pools.selectorAdd")}
          />
        </Field>
      </FieldGroup>
    </FormDrawer>
  );
}
