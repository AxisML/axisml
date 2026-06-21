import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  Table,
  Button,
  Input,
  Select,
  Space,
  Card,
  Divider,
  Drawer,
  Form,
  InputNumber,
  Progress,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  SearchOutlined,
  MinusCircleOutlined,
  CloseOutlined,
} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useTrafficPolicies, useServices } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { FieldSection } from "@/components/FieldSection";
import { CardRadio } from "@/components/CardRadio";
import { PhaseTag } from "@/components/PhaseTag";

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

// Compact per-row split bars: one accent bar (first backend) + muted bars for the
// rest, each filled to the backend's actual traffic share.
function MiniSplit({ split }: { split: SplitBackend[] }) {
  if (!split.length) return <span className="text-xs text-muted">—</span>;
  return (
    <div className="flex min-w-[200px] flex-col gap-1.5">
      {split.map((b, i) => (
        <div key={b.serviceName} className="flex items-center gap-2 text-xs">
          <span className="w-24 truncate font-mono text-fg-2">{b.serviceName}</span>
          <Progress
            percent={b.actualPct}
            showInfo={false}
            size="small"
            strokeColor={i === 0 ? "var(--accent)" : "var(--muted)"}
            className="flex-1"
          />
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
  const promote = useApiMutation((name: string) => sdk.promoteTrafficPolicy({ path: { name } }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.promoted"),
  });
  const rollback = useApiMutation((name: string) => sdk.rollbackTrafficPolicy({ path: { name } }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.rolledBack"),
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

  const onPromote = (r: TrafficRow) =>
    confirm({
      title: t("traffic.promoteTitle", { name: r.name }),
      desc: t("traffic.promoteDesc"),
      okLabel: t("traffic.promoteOk"),
      danger: false,
      onConfirm: () => promote.mutate(r.name),
    });

  const onRollback = (r: TrafficRow) =>
    confirm({
      title: t("traffic.rollbackTitle", { name: r.name }),
      desc: t("traffic.rollbackDesc"),
      okLabel: t("traffic.rollbackOk"),
      danger: false,
      onConfirm: () => rollback.mutate(r.name),
    });

  const columns: ColumnsType<TrafficRow> = [
    {
      title: t("traffic.colName"),
      dataIndex: "name",
      render: (_, r) => (
        <div className="min-w-0">
          <Link to={`/traffic/${r.name}`} className="font-mono font-medium">
            {r.name}
          </Link>
          {r.desc && <div className="truncate text-xs text-muted">{r.desc}</div>}
        </div>
      ),
    },
    {
      title: t("traffic.colMode"),
      dataIndex: "mode",
      width: 90,
      render: (m: sdk.TrafficPolicyMode) => <span className="text-fg-2">{modeLabel(m)}</span>,
    },
    {
      title: t("traffic.colStatus"),
      dataIndex: "phase",
      width: 110,
      render: (p?: string) => <PhaseTag phase={p} />,
    },
    {
      title: t("traffic.colBackends"),
      key: "split",
      width: 280,
      render: (_, r) => <MiniSplit split={r.split} />,
    },
    {
      title: t("traffic.colEndpoint"),
      dataIndex: "endpoint",
      render: (v?: string) =>
        v ? <span className="font-mono text-xs text-muted">{v}</span> : <span className="text-muted">—</span>,
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 260,
      align: "right",
      render: (_, r) => (
        <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
          <Link to={`/traffic/${r.name}`}>
            <Button type="link" size="small" className="!px-1">
              {t("common.detail")}
            </Button>
          </Link>
          <Button type="link" size="small" className="!px-1" onClick={() => setDrawer({ kind: "split", row: r })}>
            {r.mode === "canary" ? t("traffic.actSplitCanary") : t("traffic.actSplitWeighted")}
          </Button>
          {r.mode === "canary" && (
            <Button type="link" size="small" className="!px-1" onClick={() => onPromote(r)}>
              {t("traffic.actPromote")}
            </Button>
          )}
          {r.mode === "canary" && (
            <Button type="link" size="small" className="!px-1" onClick={() => onRollback(r)}>
              {t("traffic.actRollback")}
            </Button>
          )}
          <Button type="link" size="small" danger className="!px-1" onClick={() => onDelete(r)}>
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.serviceCenter"), t("nav.traffic")]}
      title={t("traffic.title")}
      subtitle={t("traffic.subtitle")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawer({ kind: "create" })}>
          {t("traffic.newPolicy")}
        </Button>
      }
    >
      <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 border-b border-border-soft p-4">
          <Input
            allowClear
            prefix={<SearchOutlined className="text-muted" />}
            placeholder={t("traffic.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
          <Select
            value={mode || undefined}
            onChange={(v) => setMode(v ?? "")}
            placeholder={t("traffic.modeAll")}
            allowClear
            className="min-w-36"
            options={[
              { label: t("traffic.modeWeighted"), value: "weighted" },
              { label: t("traffic.modeCanary"), value: "canary" },
            ]}
          />
          <Select
            value={phase || undefined}
            onChange={(v) => setPhase(v ?? "")}
            placeholder={t("traffic.statusAll")}
            allowClear
            className="min-w-36"
            options={(["Ready", "Pending", "Creating", "Degraded", "Failed"] as sdk.TrafficPolicyPhase[]).map((p) => ({
              label: t(`phase.${p}`, { defaultValue: p }),
              value: p,
            }))}
          />
          <Button
            onClick={() => {
              setSearch("");
              setMode("");
              setPhase("");
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <Table<TrafficRow>
          rowKey="name"
          columns={columns}
          dataSource={rows}
          loading={q.isLoading}
          pagination={{ pageSize: 20, showTotal: (n) => t("traffic.total", { count: n }), hideOnSinglePage: false }}
          locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
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
  id: number;
  service?: string;
  weight: number;
}

interface TrafficFormValues {
  name: string;
  description?: string;
  mode: sdk.TrafficPolicyMode;
  path?: string;
  stable?: string;
  canary?: string;
  canaryPercent?: number;
  weights: WeightRow[];
}

function TrafficDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const [form] = Form.useForm<TrafficFormValues>();
  const services = useReadyServiceNames();
  const mode = Form.useWatch("mode", form) ?? "canary";

  const serviceOptions = services.map((s) => ({ label: t("traffic.serviceReady", { name: s }), value: s }));

  const create = useApiMutation((body: sdk.TrafficPolicyCreateRequest) => sdk.createTrafficPolicy({ body }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.created"),
  });

  const onFinish = (v: TrafficFormValues) => {
    const endpoint = v.path?.trim() ? { path: v.path.trim() } : undefined;
    let body: sdk.TrafficPolicyCreateRequest;
    if (v.mode === "canary") {
      body = {
        name: v.name.trim(),
        mode: "canary",
        description: v.description?.trim() || undefined,
        endpoint,
        canaryPercent: v.canaryPercent,
        backends: [
          { serviceName: v.stable!, role: "stable" },
          { serviceName: v.canary!, role: "canary" },
        ],
      };
    } else {
      const backends = (v.weights ?? [])
        .filter((row) => row.service)
        .map((row) => ({ serviceName: row.service!, role: "member" as const, weight: Number(row.weight) || 0 }));
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
    <Drawer
      open
      width={640}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("traffic.drawerNew")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">{t("traffic.drawerNewSub")}</div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={create.isPending} onClick={() => form.submit()}>
            {t("traffic.createPolicy")}
          </Button>
        </div>
      }
    >
      <Form<TrafficFormValues>
        form={form}
        layout="vertical"
        size="large"
        onFinish={onFinish}
        initialValues={{
          name: "",
          mode: "canary",
          canaryPercent: 5,
          weights: [
            { id: 0, weight: 50 },
            { id: 1, weight: 50 },
          ],
        }}
      >
        <FieldSection n={1} title={t("traffic.fsBasic")} />
        <Form.Item name="name" label={t("traffic.fName")} rules={[{ required: true }]}>
          <Input className="font-mono" placeholder={t("traffic.fNamePlaceholder")} />
        </Form.Item>
        <Form.Item name="description" label={t("traffic.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("traffic.fDescPlaceholder")} />
        </Form.Item>
        <Form.Item name="mode" label={t("traffic.fMode")} rules={[{ required: true }]}>
          <CardRadio
            options={[
              { value: "canary", title: t("traffic.modeCanary"), desc: t("traffic.modeCanaryDesc") },
              { value: "weighted", title: t("traffic.modeWeighted"), desc: t("traffic.modeWeightedDesc") },
            ]}
          />
        </Form.Item>

        <FieldSection n={2} title={t("traffic.fsEndpoint")} />
        <Form.Item name="path" label={t("traffic.fPath")}>
          <Input className="font-mono" placeholder={t("traffic.fPathPlaceholder")} />
        </Form.Item>

        {mode === "canary" ? (
          <>
            <FieldSection n={3} title={t("traffic.fsBackendCanary")} />
            <div className="flex gap-4">
              <Form.Item name="stable" label={t("traffic.fStable")} rules={[{ required: true }]} className="flex-1">
                <Select placeholder={t("traffic.pickService")} options={serviceOptions} />
              </Form.Item>
              <Form.Item name="canary" label={t("traffic.fCanary")} rules={[{ required: true }]} className="flex-1">
                <Select placeholder={t("traffic.pickService")} options={serviceOptions} />
              </Form.Item>
            </div>
            <Form.Item name="canaryPercent" label={t("traffic.fCanaryPercent")} extra={t("traffic.fCanaryHelp")}>
              <InputNumber min={0} max={100} className="!w-40" />
            </Form.Item>
          </>
        ) : (
          <>
            <FieldSection n={3} title={t("traffic.fsBackendWeighted")} />
            <Form.Item label={t("traffic.fBackendWeights")} required extra={t("traffic.fWeightHelp")}>
              <Form.List name="weights">
                {(fields, { add, remove }) => (
                  <div className="flex flex-col gap-2">
                    {fields.map((field) => (
                      <div key={field.key} className="flex items-center gap-2">
                        <Form.Item name={[field.name, "service"]} noStyle rules={[{ required: true }]}>
                          <Select placeholder={t("traffic.pickService")} options={serviceOptions} className="flex-1" />
                        </Form.Item>
                        <Form.Item name={[field.name, "weight"]} noStyle>
                          <InputNumber min={0} max={100} placeholder={t("traffic.weightPlaceholder")} className="!w-32" />
                        </Form.Item>
                        <Button
                          type="text"
                          icon={<MinusCircleOutlined />}
                          disabled={fields.length <= 1}
                          onClick={() => remove(field.name)}
                        />
                      </div>
                    ))}
                    <Button type="link" icon={<PlusOutlined />} className="self-start !px-0" onClick={() => add({ weight: 0 })}>
                      {t("traffic.addBackend")}
                    </Button>
                  </div>
                )}
              </Form.List>
            </Form.Item>
          </>
        )}
      </Form>
    </Drawer>
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
    <Drawer
      open
      width={480}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">
            {row.mode === "canary" ? t("traffic.drawerSplitCanary") : t("traffic.drawerSplitWeighted")}
          </div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            <span className="font-mono">{row.name}</span>
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={split.isPending} onClick={submit}>
            {t("traffic.splitApply")}
          </Button>
        </div>
      }
    >
      {row.mode === "canary" ? (
        <>
          <FieldSection n={1} title={t("traffic.fsCanaryPercent")} />
          <label className="mb-1 block text-sm text-fg-2">{t("traffic.fCanaryPercentLabel")}</label>
          <InputNumber
            min={0}
            max={100}
            value={canaryPercent}
            onChange={(v) => setCanaryPercent(v ?? 0)}
            className="!w-40"
          />
          <p className="mt-2 text-xs text-muted">{t("traffic.canaryPercentHelp")}</p>
        </>
      ) : (
        <>
          <FieldSection n={1} title={t("traffic.fsBackendWeight")} />
          <div className="flex flex-col gap-2">
            {weights.map((w, i) => (
              <div key={w.serviceName} className="flex items-center gap-2">
                <Input className="flex-1 font-mono" value={w.serviceName} readOnly />
                <InputNumber
                  min={0}
                  max={100}
                  value={w.weight}
                  onChange={(v) =>
                    setWeights((prev) => prev.map((x, j) => (j === i ? { ...x, weight: v ?? 0 } : x)))
                  }
                  className="!w-32"
                />
              </div>
            ))}
          </div>
        </>
      )}
    </Drawer>
  );
}
