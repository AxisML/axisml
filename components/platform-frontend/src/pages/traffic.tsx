import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Search, MinusCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useTrafficPolicies, useServices } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { FieldSection } from "@/components/field-section";
import { CardRadio } from "@/components/card-radio";
import { PhaseTag } from "@/components/phase-tag";
import { DataTable, type Column } from "@/components/data-table";
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
import { Slider } from "@/components/ui/slider";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";

interface SplitBackend {
  serviceName: string;
  weight: number;
  actualPct: number;
  role?: sdk.TrafficPolicyBackendRole;
}

interface TrafficRow {
  name: string;
  desc: string;
  mode: sdk.TrafficPolicyMode;
  phase?: sdk.TrafficPolicyPhase;
  split: SplitBackend[];
  endpoint?: string;
}

const ALL = "__all__";

// Compact per-row split bars: one ink bar (first backend) + muted bars for the
// rest, each filled to the backend's actual traffic share.
function MiniSplit({ split }: { split: SplitBackend[] }) {
  if (!split.length) return <span className="text-xs text-muted-foreground">—</span>;
  return (
    <div className="flex min-w-[200px] flex-col gap-1.5">
      {split.map((b, i) => (
        <div key={b.serviceName} className="flex items-center gap-2 text-xs">
          <span className="w-24 truncate font-mono text-muted-foreground">{b.serviceName}</span>
          <span className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
            <span
              className={cn("block h-full rounded-full", i === 0 ? "bg-foreground" : "bg-muted-foreground")}
              style={{ width: `${b.actualPct}%` }}
            />
          </span>
          <span className="w-8 text-right font-mono">{b.weight}</span>
        </div>
      ))}
    </div>
  );
}

export default function Traffic() {
  const q = useTrafficPolicies();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ kind: "create" } | { kind: "split"; row: TrafficRow } | null>(null);
  const [search, setSearch] = useState("");
  const [mode, setMode] = useState<sdk.TrafficPolicyMode | "">("");
  const [phase, setPhase] = useState<sdk.TrafficPolicyPhase | "">("");

  const del = useApiMutation((name: string) => sdk.deleteTrafficPolicy({ path: { name } }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.deleted"),
  });

  const allRows: TrafficRow[] = useMemo(
    () =>
      (q.data?.items ?? []).map((p) => ({
        name: p.name,
        desc: p.description ?? p.displayName ?? "",
        mode: p.mode,
        phase: p.phase,
        split: (p.backends ?? []).map((b) => ({
          serviceName: b.serviceName,
          weight: b.weight,
          actualPct: b.actualPct ?? b.weight,
          role: b.role,
        })),
        endpoint: p.accessUrl,
      })),
    [q.data],
  );

  const rows = allRows.filter(
    (r) => (!search || r.name.includes(search)) && (!mode || r.mode === mode) && (!phase || r.phase === phase),
  );

  const modeLabel = (m: sdk.TrafficPolicyMode) => (m === "weighted" ? t("traffic.modeWeighted") : t("traffic.modeCanary"));

  const onDelete = (r: TrafficRow) =>
    confirm({
      title: t("traffic.deleteTitle", { name: r.name }),
      desc: t("traffic.deleteDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate(r.name),
    });

  const columns: Column<TrafficRow>[] = [
    {
      key: "name",
      title: t("traffic.colName"),
      render: (r) => (
        <div className="min-w-0">
          <Link
            to={`/traffic/${r.name}`}
            className="font-mono font-medium text-foreground hover:text-info hover:underline"
          >
            {r.name}
          </Link>
          {r.desc && <div className="truncate text-xs text-muted-foreground">{r.desc}</div>}
        </div>
      ),
    },
    {
      key: "mode",
      title: t("traffic.colMode"),
      width: 90,
      render: (r) => <span className="text-muted-foreground">{modeLabel(r.mode)}</span>,
    },
    {
      key: "phase",
      title: t("traffic.colStatus"),
      width: 110,
      render: (r) => <PhaseTag phase={r.phase} />,
    },
    {
      key: "split",
      title: t("traffic.colBackends"),
      width: 280,
      render: (r) => <MiniSplit split={r.split} />,
    },
    {
      key: "endpoint",
      title: t("traffic.colEndpoint"),
      render: (r) =>
        r.endpoint ? (
          <span className="font-mono text-xs text-muted-foreground">{r.endpoint}</span>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 180,
      align: "right",
      render: (r) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" asChild>
            <Link to={`/traffic/${r.name}`}>{t("common.detail")}</Link>
          </Button>
          <Button variant="link" size="sm" onClick={() => setDrawer({ kind: "split", row: r })}>
            {r.mode === "canary" ? t("traffic.actSplitCanary") : t("traffic.actSplitWeighted")}
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
      breadcrumb={[t("nav.serviceCenter"), t("nav.traffic")]}
      title={t("traffic.title")}
      subtitle={t("traffic.subtitle")}
      extra={
        <Button onClick={() => setDrawer({ kind: "create" })}>
          <Plus data-icon="inline-start" />
          {t("traffic.newPolicy")}
        </Button>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-8"
            placeholder={t("traffic.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <Select value={mode || ALL} onValueChange={(v) => setMode(v === ALL ? "" : (v as sdk.TrafficPolicyMode))}>
          <SelectTrigger className="min-w-36">
            <SelectValue placeholder={t("traffic.modeAll")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("traffic.modeAll")}</SelectItem>
            <SelectItem value="weighted">{t("traffic.modeWeighted")}</SelectItem>
            <SelectItem value="canary">{t("traffic.modeCanary")}</SelectItem>
          </SelectContent>
        </Select>
        <Select value={phase || ALL} onValueChange={(v) => setPhase(v === ALL ? "" : (v as sdk.TrafficPolicyPhase))}>
          <SelectTrigger className="min-w-36">
            <SelectValue placeholder={t("traffic.statusAll")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("traffic.statusAll")}</SelectItem>
            {(["Ready", "Pending", "Creating", "Degraded", "Failed"] as sdk.TrafficPolicyPhase[]).map((p) => (
              <SelectItem key={p} value={p}>
                {t(`phase.${p}`, { defaultValue: p })}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          onClick={() => {
            setSearch("");
            setMode("");
            setPhase("");
          }}
        >
          {t("common.reset")}
        </Button>
      </div>
      <Card className="overflow-hidden p-0">
        <DataTable
          columns={columns}
          data={rows}
          rowKey={(r) => r.name}
          loading={q.isLoading}
          error={q.isError}
        />
      </Card>

      {drawer?.kind === "create" && <TrafficDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "split" && <SplitDrawer row={drawer.row} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// Ready services for the current tenant, as backend dropdown options.
function useReadyServiceNames(): string[] {
  const sq = useServices();
  return useMemo(() => (sq.data?.items ?? []).filter((s) => s.phase === "Ready").map((s) => s.name), [sq.data]);
}

// ── Create drawer ─────────────────────────────────────────────────────────────
interface WeightRow {
  service?: string;
  weight: number;
}

interface TrafficFormValues {
  name: string;
  description: string;
  mode: sdk.TrafficPolicyMode;
  path: string;
  stable?: string;
  canary?: string;
  canaryPercent: number;
  weights: WeightRow[];
}

function TrafficDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const services = useReadyServiceNames();
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<TrafficFormValues>({
    name: "",
    description: "",
    mode: "canary",
    path: "",
    canaryPercent: 5,
    weights: [
      { weight: 50 },
      { weight: 50 },
    ],
  });
  const set = <K extends keyof TrafficFormValues>(k: K, val: TrafficFormValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));
  const mode = v.mode;

  const create = useApiMutation((body: sdk.TrafficPolicyCreateRequest) => sdk.createTrafficPolicy({ body }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.created"),
  });

  const submit = () => {
    setSubmitted(true);
    if (!v.name.trim()) return;
    const endpoint = v.path?.trim() ? { path: v.path.trim() } : undefined;
    let body: sdk.TrafficPolicyCreateRequest;
    if (v.mode === "canary") {
      if (!v.stable || !v.canary) return;
      body = {
        name: v.name.trim(),
        mode: "canary",
        description: v.description?.trim() || undefined,
        endpoint,
        canaryPercent: v.canaryPercent,
        backends: [
          { serviceName: v.stable, role: "stable" },
          { serviceName: v.canary, role: "canary" },
        ],
      };
    } else {
      const backends = (v.weights ?? [])
        .filter((row) => row.service)
        .map((row) => ({ serviceName: row.service!, role: "member" as const, weight: Number(row.weight) || 0 }));
      if (!backends.length) return;
      body = {
        name: v.name.trim(),
        mode: "weighted",
        description: v.description?.trim() || undefined,
        endpoint,
        backends,
      };
    }
    create.mutate(body, { onSuccess: onClose });
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[560px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("traffic.drawerNew")}</SheetTitle>
          <p className="text-xs text-muted-foreground">{t("traffic.drawerNewSub")}</p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("traffic.fsBasic")} />
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="tp-name">
                {t("traffic.fName")}
                <span className="text-destructive">*</span>
              </FieldLabel>
              <Input
                id="tp-name"
                className="font-mono"
                placeholder={t("traffic.fNamePlaceholder")}
                value={v.name}
                aria-invalid={submitted && !v.name.trim()}
                onChange={(e) => set("name", e.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="tp-desc">{t("traffic.fDesc")}</FieldLabel>
              <Textarea
                id="tp-desc"
                rows={2}
                placeholder={t("traffic.fDescPlaceholder")}
                value={v.description}
                onChange={(e) => set("description", e.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel>
                {t("traffic.fMode")}
                <span className="text-destructive">*</span>
              </FieldLabel>
              <CardRadio
                options={[
                  { value: "canary", title: t("traffic.modeCanary"), desc: t("traffic.modeCanaryDesc") },
                  { value: "weighted", title: t("traffic.modeWeighted"), desc: t("traffic.modeWeightedDesc") },
                ]}
                value={v.mode}
                onChange={(val) => set("mode", val as sdk.TrafficPolicyMode)}
              />
            </Field>
          </FieldGroup>

          <FieldSection n={2} title={t("traffic.fsEndpoint")} />
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="tp-path">{t("traffic.fPath")}</FieldLabel>
              <Input
                id="tp-path"
                className="font-mono"
                placeholder={t("traffic.fPathPlaceholder")}
                value={v.path}
                onChange={(e) => set("path", e.target.value)}
              />
            </Field>
          </FieldGroup>

          {mode === "canary" ? (
            <>
              <FieldSection n={3} title={t("traffic.fsBackendCanary")} />
              <FieldGroup>
                <div className="flex gap-4">
                  <Field className="flex-1">
                    <FieldLabel>
                      {t("traffic.fStable")}
                      <span className="text-destructive">*</span>
                    </FieldLabel>
                    <Select value={v.stable} onValueChange={(val) => set("stable", val)}>
                      <SelectTrigger className="w-full" aria-invalid={submitted && !v.stable}>
                        <SelectValue placeholder={t("traffic.pickService")} />
                      </SelectTrigger>
                      <SelectContent>
                        {services.map((s) => (
                          <SelectItem key={s} value={s}>
                            {t("traffic.serviceReady", { name: s })}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </Field>
                  <Field className="flex-1">
                    <FieldLabel>
                      {t("traffic.fCanary")}
                      <span className="text-destructive">*</span>
                    </FieldLabel>
                    <Select value={v.canary} onValueChange={(val) => set("canary", val)}>
                      <SelectTrigger className="w-full" aria-invalid={submitted && !v.canary}>
                        <SelectValue placeholder={t("traffic.pickService")} />
                      </SelectTrigger>
                      <SelectContent>
                        {services.map((s) => (
                          <SelectItem key={s} value={s}>
                            {t("traffic.serviceReady", { name: s })}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </Field>
                </div>
                <Field>
                  <FieldLabel htmlFor="tp-canary-pct">{t("traffic.fCanaryPercent")}</FieldLabel>
                  <Input
                    id="tp-canary-pct"
                    type="number"
                    min={0}
                    max={100}
                    className="w-40"
                    value={v.canaryPercent}
                    onChange={(e) => set("canaryPercent", Number(e.target.value))}
                  />
                  <FieldDescription>{t("traffic.fCanaryHelp")}</FieldDescription>
                </Field>
              </FieldGroup>
            </>
          ) : (
            <>
              <FieldSection n={3} title={t("traffic.fsBackendWeighted")} />
              <FieldGroup>
                <Field>
                  <FieldLabel>
                    {t("traffic.fBackendWeights")}
                    <span className="text-destructive">*</span>
                  </FieldLabel>
                  <div className="flex flex-col gap-2">
                    {v.weights.map((row, i) => (
                      <div key={i} className="flex items-center gap-2">
                        <Select
                          value={row.service}
                          onValueChange={(val) =>
                            set("weights", v.weights.map((x, j) => (j === i ? { ...x, service: val } : x)))
                          }
                        >
                          <SelectTrigger className="flex-1" aria-invalid={submitted && !row.service}>
                            <SelectValue placeholder={t("traffic.pickService")} />
                          </SelectTrigger>
                          <SelectContent>
                            {services.map((s) => (
                              <SelectItem key={s} value={s}>
                                {t("traffic.serviceReady", { name: s })}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <Input
                          type="number"
                          min={0}
                          max={100}
                          className="w-32"
                          placeholder={t("traffic.weightPlaceholder")}
                          value={row.weight}
                          onChange={(e) =>
                            set("weights", v.weights.map((x, j) => (j === i ? { ...x, weight: Number(e.target.value) } : x)))
                          }
                        />
                        <Button
                          variant="ghost"
                          size="icon"
                          disabled={v.weights.length <= 1}
                          onClick={() => set("weights", v.weights.filter((_, j) => j !== i))}
                        >
                          <MinusCircle />
                        </Button>
                      </div>
                    ))}
                    <Button
                      variant="link"
                      className="self-start px-0"
                      onClick={() => set("weights", [...v.weights, { weight: 0 }])}
                    >
                      <Plus data-icon="inline-start" />
                      {t("traffic.addBackend")}
                    </Button>
                  </div>
                  <FieldDescription>{t("traffic.fWeightHelp")}</FieldDescription>
                </Field>
              </FieldGroup>
            </>
          )}
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={create.isPending}>
            {create.isPending && <Spinner data-icon="inline-start" />}
            {t("traffic.createPolicy")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ── Split drawer (切流 / 调整权重) ──────────────────────────────────────────────
function SplitDrawer({ row, onClose }: { row: TrafficRow; onClose: () => void }) {
  const { t } = useTranslation();
  const [canaryPercent, setCanaryPercent] = useState<number>(row.split[1]?.weight ?? 5);
  const [weights, setWeights] = useState<{ serviceName: string; weight: number }[]>(() =>
    row.split.map((b) => ({ serviceName: b.serviceName, weight: b.weight })),
  );

  const split = useApiMutation(
    (body: sdk.TrafficPolicySplitRequest) => sdk.splitTrafficPolicy({ path: { name: row.name }, body }),
    { invalidate: [["trafficpolicies"]], success: t("traffic.splitApplied") },
  );

  const submit = () => {
    const body: sdk.TrafficPolicySplitRequest =
      row.mode === "canary"
        ? { canaryPercent }
        : {
            backends: weights.map((w) => ({ serviceName: w.serviceName, role: "member" as const, weight: Number(w.weight) || 0 })),
          };
    split.mutate(body, { onSuccess: onClose });
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[480px]">
        <SheetHeader className="border-b">
          <SheetTitle>
            {row.mode === "canary" ? t("traffic.drawerSplitCanary") : t("traffic.drawerSplitWeighted")}
          </SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{row.name}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          {row.mode === "canary" ? (
            <>
              <FieldSection n={1} title={t("traffic.fsCanaryPercent")} />
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="sp-canary-pct">{t("traffic.fCanaryPercentLabel")}</FieldLabel>
                  <div className="flex items-center gap-4">
                    <Slider
                      id="sp-canary-pct"
                      min={0}
                      max={100}
                      step={1}
                      value={[canaryPercent]}
                      onValueChange={(v) => setCanaryPercent(v[0])}
                      className="flex-1"
                    />
                    <span className="w-12 shrink-0 text-right font-mono text-sm tabular-nums">
                      {canaryPercent}%
                    </span>
                  </div>
                  <FieldDescription>
                    {t("traffic.canarySplitHint", { canary: canaryPercent, stable: 100 - canaryPercent })}
                  </FieldDescription>
                </Field>
              </FieldGroup>
            </>
          ) : (
            <>
              <FieldSection n={1} title={t("traffic.fsBackendWeight")} />
              <FieldGroup>
                <div className="flex flex-col gap-2">
                  {weights.map((w, i) => (
                    <div key={w.serviceName} className="flex items-center gap-2">
                      <Input className="flex-1 font-mono" value={w.serviceName} readOnly />
                      <Input
                        type="number"
                        min={0}
                        max={100}
                        className="w-32"
                        value={w.weight}
                        onChange={(e) =>
                          setWeights((prev) => prev.map((x, j) => (j === i ? { ...x, weight: Number(e.target.value) } : x)))
                        }
                      />
                    </div>
                  ))}
                </div>
              </FieldGroup>
            </>
          )}
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={split.isPending}>
            {split.isPending && <Spinner data-icon="inline-start" />}
            {t("traffic.splitApply")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
