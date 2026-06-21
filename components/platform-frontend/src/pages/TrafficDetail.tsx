import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  Card,
  Tabs,
  Descriptions,
  Table,
  Tag,
  Button,
  Space,
  Slider,
  InputNumber,
  Progress,
  Tooltip,
  Breadcrumb,
  Spin,
  Result,
  Alert,
  List,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { ArrowLeftOutlined, CopyOutlined, DeleteOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";

interface BackendView {
  serviceName: string;
  role?: sdk.TrafficPolicyBackendRole;
  weight: number;
  actualPct: number;
  ready?: boolean;
}

export default function TrafficDetail() {
  const { name = "" } = useParams();
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { confirm } = useUI();

  const q = useQuery({
    queryKey: ["trafficpolicies", tenant, name],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getTrafficPolicy({ path: { name } });
      if (error) throw error;
      return data;
    },
  });

  const del = useApiMutation(() => sdk.deleteTrafficPolicy({ path: { name } }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.deleted"),
  });

  const breadcrumb = (
    <Breadcrumb
      className="mb-3"
      items={[{ title: t("nav.serviceCenter") }, { title: <Link to="/traffic">{t("nav.traffic")}</Link> }, { title: name }]}
    />
  );

  const back = (
    <Link to="/traffic" className="mb-3 inline-flex items-center gap-1 text-sm text-muted hover:text-accent">
      <ArrowLeftOutlined />
      {t("traffic.backToList")}
    </Link>
  );

  if (q.isLoading) {
    return (
      <div className="mx-auto max-w-[1200px] p-6">
        {breadcrumb}
        {back}
        <div className="grid place-items-center py-24">
          <Spin />
        </div>
      </div>
    );
  }

  if (q.isError || !q.data) {
    return (
      <div className="mx-auto max-w-[1200px] p-6">
        {breadcrumb}
        {back}
        <Result status="error" title={t("traffic.loadFailedTitle")} subTitle={t("common.loadFailed")} />
      </div>
    );
  }

  const p = q.data;
  const backends: BackendView[] = (p.backends ?? []).map((b) => ({
    serviceName: b.serviceName,
    role: b.role,
    weight: b.weight,
    actualPct: b.actualPct ?? b.weight,
    ready: b.ready,
  }));
  const modeLabel = p.mode === "weighted" ? t("traffic.modeWeighted") : t("traffic.modeCanary");

  const onDelete = () =>
    confirm({
      title: t("traffic.deleteTitle", { name: p.name }),
      desc: t("traffic.detailDeleteDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate(undefined),
    });

  return (
    <PageContainer
      breadcrumb={[t("nav.serviceCenter"), t("nav.traffic"), p.name]}
      title={
        <Space size="middle" align="center">
          <span className="font-mono">{p.name}</span>
          <PhaseTag phase={p.phase} />
          <Tag>{modeLabel}</Tag>
        </Space>
      }
      subtitle={p.description ?? p.displayName ?? undefined}
      extra={
        <Button danger icon={<DeleteOutlined />} loading={del.isPending} onClick={onDelete}>
          {t("traffic.delete")}
        </Button>
      }
    >
      {back}
      <Tabs
        items={[
          { key: "overview", label: t("traffic.tabOverview"), children: <Overview policy={p} backendCount={backends.length} /> },
          {
            key: "dist",
            label: t("traffic.tabDistribution"),
            children:
              p.mode === "canary" ? (
                <CanaryDistribution name={p.name} initial={p.canaryPercent ?? 0} backends={backends} />
              ) : (
                <WeightedDistribution name={p.name} backends={backends} />
              ),
          },
          { key: "events", label: t("traffic.tabEvents"), children: <Events name={p.name} /> },
        ]}
      />
    </PageContainer>
  );
}

// ── Overview ──────────────────────────────────────────────────────────────────
function Overview({ policy, backendCount }: { policy: sdk.TrafficPolicy; backendCount: number }) {
  const { t } = useTranslation();
  const { toast } = useUI();
  const endpoint = policy.accessUrl ?? policy.endpoint?.path;
  const modeLabel = policy.mode === "weighted" ? t("traffic.modeWeighted") : t("traffic.modeCanary");

  const copy = () => {
    if (!endpoint) return;
    void navigator.clipboard?.writeText(endpoint);
    toast(t("traffic.endpointCopied"));
  };

  return (
    <Card title={t("traffic.policyInfo")}>
      <Descriptions column={1} size="middle" colon={false} labelStyle={{ width: 120 }}>
        <Descriptions.Item label={t("traffic.fieldName")}>
          <span className="font-mono">{policy.name}</span>
        </Descriptions.Item>
        <Descriptions.Item label={t("traffic.fieldDesc")}>
          {policy.description ?? policy.displayName ?? "—"}
        </Descriptions.Item>
        <Descriptions.Item label={t("traffic.fieldMode")}>
          <Tag>{modeLabel}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t("traffic.fieldEndpoint")}>
          {endpoint ? (
            <Space size="small">
              <span className="font-mono">{endpoint}</span>
              <Tooltip title={t("traffic.copyEndpoint")}>
                <Button type="text" size="small" icon={<CopyOutlined />} onClick={copy} />
              </Tooltip>
            </Space>
          ) : (
            "—"
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t("traffic.fieldBackendCount")}>
          <span className="font-mono">{backendCount}</span>
        </Descriptions.Item>
        <Descriptions.Item label={t("traffic.fieldOwner")}>{policy.owner || "—"}</Descriptions.Item>
        <Descriptions.Item label={t("traffic.fieldCreatedAt")}>
          <span className="font-mono">{policy.createdAt ? dayjs(policy.createdAt).format("YYYY-MM-DD HH:mm:ss") : "—"}</span>
        </Descriptions.Item>
      </Descriptions>
    </Card>
  );
}

// ── Backend distribution table (shared) ───────────────────────────────────────
function roleLabel(t: (k: string) => string, role?: sdk.TrafficPolicyBackendRole): string {
  if (role === "stable") return t("traffic.roleStable");
  if (role === "canary") return t("traffic.roleCanary");
  return t("traffic.roleMember");
}

function backendColumns(
  t: (k: string) => string,
  weightCol: ColumnsType<BackendView>[number],
): ColumnsType<BackendView> {
  return [
    {
      title: t("traffic.colService"),
      dataIndex: "serviceName",
      render: (v: string) => (
        <Link to={`/services/${v}`} className="font-mono">
          {v}
        </Link>
      ),
    },
    {
      title: t("traffic.colRole"),
      dataIndex: "role",
      width: 90,
      render: (r?: sdk.TrafficPolicyBackendRole) => <Tag>{roleLabel(t, r)}</Tag>,
    },
    weightCol,
    {
      title: t("traffic.colActualPct"),
      dataIndex: "actualPct",
      width: 220,
      render: (v: number) => (
        <div className="flex items-center gap-2">
          <Progress percent={v} showInfo={false} size="small" strokeColor="var(--accent)" className="flex-1" />
          <span className="w-10 text-right font-mono text-xs">{v}%</span>
        </div>
      ),
    },
    {
      title: t("traffic.colBackendStatus"),
      dataIndex: "ready",
      width: 110,
      render: (ready?: boolean) =>
        ready ? <Tag color="success">{t("traffic.backendReady")}</Tag> : <Tag>{t("traffic.backendNotReady")}</Tag>,
    },
  ];
}

// ── Canary distribution ───────────────────────────────────────────────────────
function CanaryDistribution({ name, initial, backends }: { name: string; initial: number; backends: BackendView[] }) {
  const { t } = useTranslation();
  const [canary, setCanary] = useState(initial);
  const stable = 100 - canary;

  const split = useApiMutation(
    (body: sdk.TrafficPolicySplitRequest) => sdk.splitTrafficPolicy({ path: { name }, body }),
    { invalidate: [["trafficpolicies"]], success: t("traffic.splitApplied") },
  );
  const promote = useApiMutation(() => sdk.promoteTrafficPolicy({ path: { name } }), {
    invalidate: [["trafficpolicies"]],
    success: t("traffic.promoted"),
  });

  // Reflect the live slider value into the table's actual share for an immediate preview.
  const rows: BackendView[] = backends.map((b) =>
    b.role === "canary"
      ? { ...b, weight: canary, actualPct: canary }
      : b.role === "stable"
        ? { ...b, weight: stable, actualPct: stable }
        : b,
  );

  const weightCol: ColumnsType<BackendView>[number] = {
    title: t("traffic.colTargetWeight"),
    dataIndex: "weight",
    width: 110,
    align: "right",
    render: (v: number) => <span className="font-mono">{v}</span>,
  };

  return (
    <div className="flex flex-col gap-5">
      <Card title={t("traffic.canaryPercentTitle")}>
        <div className="mb-3 flex items-center justify-between">
          <span className="text-sm text-fg-2">{t("traffic.canaryShare")}</span>
          <span className="font-mono text-2xl font-semibold">
            {canary}
            <span className="text-sm text-muted">%</span>
          </span>
        </div>
        <Slider min={0} max={100} value={canary} onChange={setCanary} />
        <div className="mt-2 flex gap-7 text-sm text-fg-2">
          <span>
            {t("traffic.stableShare")} <b className="font-mono text-info">{stable}%</b>
          </span>
          <span>
            {t("traffic.canaryShare")} <b className="font-mono text-warn">{canary}%</b>
          </span>
        </div>
        <div className="mt-4 flex gap-2">
          <Button
            type="primary"
            loading={split.isPending}
            onClick={() => split.mutate({ canaryPercent: canary })}
          >
            {t("traffic.applyCanary")}
          </Button>
          <Button loading={promote.isPending} onClick={() => promote.mutate(undefined)}>
            {t("traffic.promoteToStable")}
          </Button>
        </div>
      </Card>

      <Card title={t("traffic.backendDist")} styles={{ body: { padding: 0 } }}>
        <Table<BackendView>
          rowKey="serviceName"
          columns={backendColumns(t, weightCol)}
          dataSource={rows}
          pagination={false}
        />
      </Card>
    </div>
  );
}

// ── Weighted distribution ─────────────────────────────────────────────────────
function WeightedDistribution({ name, backends }: { name: string; backends: BackendView[] }) {
  const { t } = useTranslation();
  const [weights, setWeights] = useState<Record<string, number>>(() =>
    Object.fromEntries(backends.map((b) => [b.serviceName, b.weight])),
  );

  const split = useApiMutation(
    (body: sdk.TrafficPolicySplitRequest) => sdk.splitTrafficPolicy({ path: { name }, body }),
    { invalidate: [["trafficpolicies"]], success: t("traffic.weightsApplied") },
  );

  const sum = Object.values(weights).reduce((a, b) => a + (b || 0), 0);
  const ok = sum === 100;

  const rows: BackendView[] = backends.map((b) => {
    const w = weights[b.serviceName] ?? 0;
    return { ...b, weight: w, actualPct: sum ? Math.round((w / sum) * 100) : 0 };
  });

  const weightCol: ColumnsType<BackendView>[number] = {
    title: t("traffic.colTargetWeight"),
    dataIndex: "weight",
    width: 130,
    align: "right",
    render: (_: number, r) => (
      <InputNumber
        min={0}
        max={100}
        value={weights[r.serviceName] ?? 0}
        onChange={(v) => setWeights((prev) => ({ ...prev, [r.serviceName]: v ?? 0 }))}
        className="!w-24"
      />
    ),
  };

  const apply = () =>
    split.mutate({
      backends: backends.map((b) => ({
        serviceName: b.serviceName,
        role: "member" as const,
        weight: weights[b.serviceName] ?? 0,
      })),
    });

  return (
    <Card
      title={t("traffic.backendDist")}
      extra={<span className="text-xs text-muted">{t("traffic.weightedHint")}</span>}
      styles={{ body: { padding: 0 } }}
    >
      <Table<BackendView> rowKey="serviceName" columns={backendColumns(t, weightCol)} dataSource={rows} pagination={false} />
      <div className="flex items-center gap-4 border-t border-border-soft p-4">
        <Alert
          type={ok ? "success" : "warning"}
          showIcon
          message={ok ? t("traffic.sumOk", { sum }) : t("traffic.sumBad", { sum })}
          className="!py-1"
        />
        <span className="flex-1" />
        <Button type="primary" disabled={!ok} loading={split.isPending} onClick={apply}>
          {t("traffic.applyWeights")}
        </Button>
      </div>
    </Card>
  );
}

// ── Events ────────────────────────────────────────────────────────────────────
function Events({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const q = useQuery({
    queryKey: ["trafficpolicies", tenant, name, "events"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getTrafficPolicyEvents({ path: { name } });
      if (error) throw error;
      return data;
    },
  });

  return (
    <Card styles={{ body: { padding: 0 } }}>
      <List
        loading={q.isLoading}
        dataSource={q.data?.items ?? []}
        locale={{ emptyText: q.isError ? t("common.loadFailed") : t("traffic.noEvents") }}
        renderItem={(e) => (
          <List.Item className="!px-4">
            <List.Item.Meta
              title={
                <Space size="small">
                  <span className="font-medium">{e.reason}</span>
                  <Tag color={e.type === "Warning" ? "warning" : "default"}>{e.type}</Tag>
                </Space>
              }
              description={e.message}
            />
            <span className="font-mono text-xs text-muted">
              {e.lastTimestamp ? dayjs(e.lastTimestamp).format("YYYY-MM-DD HH:mm:ss") : "—"}
            </span>
          </List.Item>
        )}
      />
    </Card>
  );
}
