import { useMemo, useState } from "react";
import { Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { useDataVolumes, useStorageClasses } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { FieldSection } from "@/components/field-section";
import { SearchInput } from "@/components/search-input";
import { DataTable, type Column } from "@/components/data-table";
import { StatusDot } from "@/components/phase-tag";
import { phaseTone, type PhaseTone } from "@/lib/phase";
import { fmtDateTime, fmtBytes, parseQty } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { FormDrawer } from "@/components/form-drawer";
import { Field, FieldLabel, FieldDescription, FieldGroup } from "@/components/ui/field";
import { cn } from "@/lib/utils";

// Access modes: the API carries the canonical Kubernetes values; the UI shows the
// short RWO/RWX/ROX labels (enum values are never translated, only their labels).
const ACCESS_MODES = [
  { value: "ReadWriteOnce", labelKey: "volumes.rwo", short: "RWO" },
  { value: "ReadWriteMany", labelKey: "volumes.rwx", short: "RWX" },
  { value: "ReadOnlyMany", labelKey: "volumes.rox", short: "ROX" },
] as const;
const SHORT: Record<string, string> = { ReadWriteOnce: "RWO", ReadWriteMany: "RWX", ReadOnlyMany: "ROX" };

// PVC phases map to the shared tone palette (Bound = healthy, Pending = warming,
// Lost = failed); anything unknown falls back to the generic phaseTone.
const STATUS_TONE: Record<string, PhaseTone> = { Bound: "success", Pending: "pending", Lost: "failed" };
function volumeTone(phase?: string | null): PhaseTone {
  return phase ? (STATUS_TONE[phase] ?? phaseTone(phase)) : "stopped";
}

// Capacity / usage cell — a meter (success/warn/danger by fill) + "used / total"
// when usage is reported, otherwise just the requested size. No fabricated usage:
// when usedBytes is absent (e.g. metrics unavailable) only the capacity shows; a
// reported usedBytes of 0 (genuinely empty) still renders a 0% meter.
function UsageMeter({ v }: { v: sdk.DataVolume }) {
  if (!v.size) return <span className="text-muted-foreground">—</span>;
  const total = parseQty(v.status?.boundCapacity || v.size);
  const used = v.status?.usedBytes;
  if (!total || used == null) return <span className="font-mono text-sm">{v.size}</span>;
  const pct = Math.min(100, Math.round((used / total) * 100));
  const color = pct >= 80 ? "bg-destructive" : pct >= 60 ? "bg-warning" : "bg-success";
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-16 shrink-0 overflow-hidden rounded-full bg-muted">
        <div className={cn("h-full rounded-full", color)} style={{ width: `${pct}%` }} />
      </div>
      <span className="font-mono text-xs whitespace-nowrap text-muted-foreground">
        {fmtBytes(used)} / {v.status?.boundCapacity || v.size}
      </span>
    </div>
  );
}

export default function DataVolumes() {
  const q = useDataVolumes();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [accessFilter, setAccessFilter] = useState("all");
  const [creating, setCreating] = useState(false);
  const [manageName, setManageName] = useState<string | null>(null);

  const delVolume = useApiMutation((name: string) => sdk.deleteDataVolume({ path: { name } }), {
    invalidate: [["datavolumes"]],
    success: t("volumes.deleted"),
  });

  const allRows = q.data?.items ?? [];
  const rows = useMemo(
    () =>
      allRows.filter(
        (v) =>
          (!search || v.name.includes(search) || (v.description ?? "").includes(search)) &&
          (statusFilter === "all" || v.status?.phase === statusFilter) &&
          (accessFilter === "all" || (v.accessModes ?? []).includes(accessFilter)),
      ),
    [allRows, search, statusFilter, accessFilter],
  );
  const manageVolume = manageName ? allRows.find((v) => v.name === manageName) : undefined;

  const onDelete = (v: sdk.DataVolume) =>
    confirm({
      title: t("volumes.deleteTitle", { name: v.name }),
      desc: t("volumes.deleteDesc"),
      info: t("volumes.deleteInfo"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delVolume.mutate(v.name),
    });

  const columns: Column<sdk.DataVolume>[] = [
    {
      key: "name",
      title: t("volumes.colName"),
      render: (v) => (
        <button
          type="button"
          className="font-mono font-medium text-foreground hover:text-info hover:underline"
          onClick={() => setManageName(v.name)}
        >
          {v.name}
        </button>
      ),
    },
    {
      key: "description",
      title: t("volumes.colDesc"),
      render: (v) => v.description || <span className="text-muted-foreground">—</span>,
    },
    {
      key: "storageClass",
      title: t("volumes.colStorageClass"),
      render: (v) =>
        v.storageClass ? (
          <span className="font-mono text-sm">{v.storageClass}</span>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      key: "accessModes",
      title: t("volumes.colAccessModes"),
      render: (v) => (
        <div className="flex flex-wrap gap-1">
          {(v.accessModes ?? []).map((m) => (
            <Badge key={m} variant="outline" className="font-mono">
              {SHORT[m] ?? m}
            </Badge>
          ))}
        </div>
      ),
    },
    {
      key: "capacity",
      title: t("volumes.colCapacity"),
      width: 200,
      render: (v) => <UsageMeter v={v} />,
    },
    {
      key: "status",
      title: t("volumes.colStatus"),
      width: 120,
      render: (v) => (
        <StatusDot tone={volumeTone(v.status?.phase)}>
          {v.status?.phase || "—"}
        </StatusDot>
      ),
    },
    {
      key: "createdAt",
      title: t("volumes.colCreated"),
      width: 170,
      render: (v) => <span className="text-muted-foreground">{fmtDateTime(v.createdAt)}</span>,
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 140,
      align: "right",
      render: (v) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" onClick={() => setManageName(v.name)}>
            {t("volumes.manage")}
          </Button>
          <Button variant="link" size="sm" className="text-destructive" onClick={() => onDelete(v)}>
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.systemMgmt"), t("nav.volumes")]}
      title={t("volumes.title")}
      subtitle={t("volumes.subtitle")}
      extra={
        <Button onClick={() => setCreating(true)}>
          <Plus data-icon="inline-start" />
          {t("volumes.newVolume")}
        </Button>
      }
    >
      <Card className="overflow-hidden p-0">
        <div className="flex flex-wrap items-center gap-3 border-b p-4">
          <SearchInput
            className="max-w-xs flex-1"
            placeholder={t("volumes.searchPlaceholder")}
            value={search}
            onChange={setSearch}
          />
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-36">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("volumes.filterAllStatus")}</SelectItem>
              <SelectItem value="Bound">Bound</SelectItem>
              <SelectItem value="Pending">Pending</SelectItem>
              <SelectItem value="Lost">Lost</SelectItem>
            </SelectContent>
          </Select>
          <Select value={accessFilter} onValueChange={setAccessFilter}>
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("volumes.filterAllAccess")}</SelectItem>
              {ACCESS_MODES.map((m) => (
                <SelectItem key={m.value} value={m.value}>
                  {m.short}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            onClick={() => {
              setSearch("");
              setStatusFilter("all");
              setAccessFilter("all");
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          rowKey={(v) => v.name}
          loading={q.isLoading}
          error={q.isError}
        />
      </Card>

      {creating && <VolumeCreateDrawer onClose={() => setCreating(false)} />}
      {manageVolume && (
        <ManageVolumeDrawer key={manageVolume.name} volume={manageVolume} onClose={() => setManageName(null)} />
      )}
    </PageContainer>
  );
}

// ── Create drawer ─────────────────────────────────────────────────────────────
function VolumeCreateDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const [submitted, setSubmitted] = useState(false);
  const scQ = useStorageClasses();
  const scOptions = scQ.data?.items ?? [];
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [size, setSize] = useState("");
  const [storageClass, setStorageClass] = useState("");
  const [accessMode, setAccessMode] = useState<string>("ReadWriteOnce");

  const create = useApiMutation((body: sdk.DataVolumeCreateRequest) => sdk.createDataVolume({ body }), {
    invalidate: [["datavolumes"]],
    success: t("volumes.created"),
  });

  const submit = () => {
    setSubmitted(true);
    const n = name.trim();
    const s = size.trim();
    if (!n || !s) return;
    create.mutate(
      {
        name: n,
        size: s,
        description: description.trim() || undefined,
        storageClass: storageClass.trim() || undefined,
        accessModes: [accessMode],
      },
      { onSuccess: onClose },
    );
  };

  return (
    <FormDrawer
      title={t("volumes.drawerNew")}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("volumes.createVolume")}
      submitting={create.isPending}
    >
      <FieldSection n={1} title={t("volumes.fsBasic")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="vol-name">
            {t("volumes.fName")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <Input
            id="vol-name"
            className="font-mono"
            placeholder={t("volumes.fNamePlaceholder")}
            value={name}
            aria-invalid={submitted && !name.trim()}
            onChange={(e) => setName(e.target.value)}
          />
          <FieldDescription>{t("volumes.fNameHelp")}</FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="vol-desc">{t("volumes.fDesc")}</FieldLabel>
          <Textarea
            id="vol-desc"
            rows={2}
            placeholder={t("volumes.fDescPlaceholder")}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </Field>
      </FieldGroup>

      <FieldSection n={2} title={t("volumes.fsSpec")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="vol-size">
            {t("volumes.fSize")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <Input
            id="vol-size"
            className="font-mono"
            placeholder={t("volumes.fSizePlaceholder")}
            value={size}
            aria-invalid={submitted && !size.trim()}
            onChange={(e) => setSize(e.target.value)}
          />
          <FieldDescription>{t("volumes.fSizeHelp")}</FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="vol-sc">{t("volumes.fStorageClass")}</FieldLabel>
          <Select
            value={storageClass || "__default"}
            onValueChange={(val) => setStorageClass(val === "__default" ? "" : val)}
          >
            <SelectTrigger id="vol-sc" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__default">{t("volumes.scClusterDefault")}</SelectItem>
              {scOptions.map((sc) => (
                <SelectItem key={sc.name} value={sc.name}>
                  <span className="font-mono">{sc.name}</span>
                  {sc.default ? ` · ${t("volumes.scDefault")}` : ""}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FieldDescription>
            {scQ.isError ? t("common.loadFailed") : t("volumes.fStorageClassHelp")}
          </FieldDescription>
        </Field>
        <Field>
          <FieldLabel>{t("volumes.fAccessMode")}</FieldLabel>
          <div className="grid grid-cols-3 gap-2">
            {ACCESS_MODES.map((m) => {
              const on = accessMode === m.value;
              const desc = t(m.labelKey).split("·").pop()?.trim() ?? "";
              return (
                <button
                  type="button"
                  key={m.value}
                  onClick={() => setAccessMode(m.value)}
                  className={cn(
                    "rounded-md border p-2.5 text-left transition-colors",
                    on ? "border-primary bg-primary/5 ring-1 ring-primary/30" : "hover:border-primary/40",
                  )}
                >
                  <div className="font-mono text-sm font-semibold">{m.short}</div>
                  <div className="mt-0.5 text-xs text-muted-foreground">{desc}</div>
                </button>
              );
            })}
          </div>
          <FieldDescription>{t("volumes.fAccessModeHelp")}</FieldDescription>
        </Field>
      </FieldGroup>
    </FormDrawer>
  );
}

// ── Manage drawer: basics (description) + storage spec (size expand) + mounts ──
function ManageVolumeDrawer({ volume, onClose }: { volume: sdk.DataVolume; onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [description, setDescription] = useState(volume.description ?? "");
  const [size, setSize] = useState(volume.size ?? "");

  // Mount occupancy + usedBytes are populated on the detail GET, not the list,
  // so fetch the single volume here. Fall back to the list row on error.
  const detailQ = useQuery({
    queryKey: ["datavolume", tenant, volume.name],
    queryFn: async () => {
      const { data, error } = await sdk.getDataVolume({ path: { name: volume.name } });
      if (error) throw error;
      return data;
    },
  });
  const detail = detailQ.data ?? volume;

  const update = useApiMutation(
    (body: sdk.DataVolumePatchRequest) => sdk.updateDataVolume({ path: { name: volume.name }, body }),
    { invalidate: [["datavolumes"]], success: t("volumes.saved") },
  );

  const submit = () => {
    const body: sdk.DataVolumePatchRequest = { description: description.trim() };
    const s = size.trim();
    if (s && s !== volume.size) body.size = s;
    update.mutate(body, { onSuccess: onClose });
  };

  const mounts = detail.status?.mounts ?? [];

  return (
    <FormDrawer
      title={<span className="font-mono">{volume.name}</span>}
      subtitle={t("volumes.accessImmutable")}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("common.save")}
      submitting={update.isPending}
    >
      <FieldSection n={1} title={t("volumes.fsBasic")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="mvol-name">{t("volumes.fName")}</FieldLabel>
          <Input id="mvol-name" className="font-mono" value={volume.name} readOnly aria-readonly />
        </Field>
        <Field>
          <FieldLabel htmlFor="mvol-desc">{t("volumes.fDesc")}</FieldLabel>
          <Textarea
            id="mvol-desc"
            rows={2}
            placeholder={t("volumes.fDescPlaceholder")}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </Field>
      </FieldGroup>

      <FieldSection n={2} title={t("volumes.fsSpec")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="mvol-sc">{t("volumes.fStorageClass")}</FieldLabel>
          <Input id="mvol-sc" className="font-mono" value={volume.storageClass || "—"} readOnly aria-readonly />
        </Field>
        <Field>
          <FieldLabel>{t("volumes.fAccessMode")}</FieldLabel>
          <div className="flex flex-wrap gap-1">
            {(volume.accessModes ?? []).map((m) => (
              <Badge key={m} variant="outline" className="font-mono">
                {SHORT[m] ?? m}
              </Badge>
            ))}
          </div>
        </Field>
        <Field>
          <FieldLabel htmlFor="mvol-size">{t("volumes.fSize")}</FieldLabel>
          <Input
            id="mvol-size"
            className="font-mono"
            placeholder={t("volumes.fSizePlaceholder")}
            value={size}
            onChange={(e) => setSize(e.target.value)}
          />
          <FieldDescription>{t("volumes.fSizeHelp")}</FieldDescription>
        </Field>
        {detail.status?.usedBytes ? (
          <Field>
            <FieldLabel>{t("volumes.used")}</FieldLabel>
            <Input className="font-mono" value={fmtBytes(detail.status.usedBytes)} readOnly aria-readonly />
          </Field>
        ) : null}
      </FieldGroup>

      <FieldSection n={3} title={t("volumes.fsMounts")} />
      {mounts.length === 0 ? (
        <p className="px-0.5 text-sm text-muted-foreground">{t("volumes.mountsEmpty")}</p>
      ) : (
        <div className="overflow-hidden rounded-md border">
          <table className="w-full text-sm">
            <thead className="bg-muted/40 text-xs text-muted-foreground">
              <tr>
                <th className="px-3 py-2 text-left font-medium">{t("volumes.mountWorkload")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("volumes.mountKind")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("volumes.mountPath")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("volumes.mountStatus")}</th>
              </tr>
            </thead>
            <tbody>
              {mounts.map((m, i) => (
                <tr key={`${m.workload}-${i}`} className="border-t">
                  <td className="px-3 py-2 font-mono">{m.workload}</td>
                  <td className="px-3 py-2 text-muted-foreground">{m.kind || "—"}</td>
                  <td className="px-3 py-2 font-mono">{m.mountPath || "—"}</td>
                  <td className="px-3 py-2">
                    <StatusDot tone={m.running ? "running" : "stopped"}>
                      {m.running ? t("volumes.mountRunning") : t("volumes.mountStopped")}
                    </StatusDot>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </FormDrawer>
  );
}
