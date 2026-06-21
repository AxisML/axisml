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
  Collapse,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { PlusOutlined, SearchOutlined, DeleteOutlined, CloseOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useJobs } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { FieldSection } from "@/components/FieldSection";
import { CardRadio } from "@/components/CardRadio";
import { RunStrip } from "@/components/RunStrip";
import { USE_MOCK } from "@/api/mock";
import { runSummary } from "@/api/mock/data";

interface JobRow {
  name: string;
  desc: string;
  runCount: number;
  recent: string[];
  owner: string;
  updated: string;
}

type DrawerMode = "new" | "run" | "edit";

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

  const columns: ColumnsType<JobRow> = [
    {
      title: t("jobs.colName"),
      dataIndex: "name",
      render: (_, r) => (
        <div className="min-w-0">
          <Link to={`/jobs/${r.name}`} className="font-mono font-medium">
            {r.name}
          </Link>
          {r.desc && <div className="truncate text-xs text-muted">{r.desc}</div>}
        </div>
      ),
    },
    {
      title: t("jobs.colStatus"),
      key: "runs",
      width: 150,
      render: (_, r) => <RunStrip phases={r.recent} to={`/jobs/${r.name}`} />,
    },
    { title: t("jobs.colRuns"), dataIndex: "runCount", width: 90, align: "right", render: (v: number) => <span className="font-mono">{v}</span> },
    { title: t("jobs.colCreator"), dataIndex: "owner", width: 140 },
    {
      title: t("jobs.colUpdated"),
      dataIndex: "updated",
      width: 150,
      render: (v: string) => <span className="text-muted">{v ? dayjs(v).fromNow() : "—"}</span>,
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 180,
      align: "right",
      render: (_, r) => (
        <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
          <Button type="link" size="small" className="!px-1" onClick={() => setDrawer({ mode: "run", name: r.name })}>
            {t("common.run")}
          </Button>
          <Link to={`/jobs/${r.name}`}>
            <Button type="link" size="small" className="!px-1">
              {t("common.detail")}
            </Button>
          </Link>
          <Button type="link" size="small" className="!px-1" onClick={() => setDrawer({ mode: "edit", name: r.name })}>
            {t("common.edit")}
          </Button>
          <Button type="link" size="small" danger className="!px-1" onClick={() => onDelete(r)}>
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.jobs")]}
      title={t("jobs.title")}
      subtitle={t("jobs.subtitle")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawer({ mode: "new" })}>
          {t("jobs.newJob")}
        </Button>
      }
    >
      <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 border-b border-border-soft p-4">
          <Input
            allowClear
            prefix={<SearchOutlined className="text-muted" />}
            placeholder={t("jobs.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
          <Select
            value={creator || undefined}
            onChange={(v) => setCreator(v ?? "")}
            placeholder={t("jobs.creatorAll")}
            allowClear
            className="min-w-44"
            options={creatorOptions.map((o) => ({ label: o, value: o }))}
          />
          <Button
            onClick={() => {
              setSearch("");
              setCreator("");
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <Table<JobRow>
          rowKey="name"
          columns={columns}
          dataSource={rows}
          loading={q.isLoading}
          pagination={{ pageSize: 20, showTotal: (n) => t("jobs.total", { count: n }), hideOnSinglePage: false }}
          locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
        />
      </Card>

      {drawer && <JobDrawer mode={drawer.mode} name={drawer.name} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Create / Run / Edit drawer ────────────────────────────────────────────────
const IMAGES: { value: string; title: string; desc: string }[] = [
  { value: "pytorch:2.3-cu121", title: "pytorch:2.3-cu121", desc: "PyTorch 训练镜像" },
  { value: "megatron:24.05", title: "megatron:24.05", desc: "Megatron-LM 训练镜像" },
];
const UNITS: { value: string; title: string; desc: string }[] = [
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
  description?: string;
  image: string;
  poolName: string;
  unitName: string;
  replicas: number;
  command: string;
  env?: string;
  volumes?: { name?: string; mountPath?: string }[];
  timeout?: number;
  retries?: number;
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
  const [form] = Form.useForm<JobFormValues>();
  const locked = mode === "run";

  const buildSpec = (v: JobFormValues): sdk.JobSpec => {
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
        activeDeadlineSeconds: v.timeout && v.timeout > 0 ? v.timeout : undefined,
        backoffLimit: v.retries != null && v.retries >= 0 ? v.retries : undefined,
      },
    };
  };

  const create = useApiMutation((body: sdk.JobCreateRequest) => sdk.createJob({ body }), {
    invalidate: [["jobs"]],
    success: t("jobs.savedTemplate"),
  });
  const update = useApiMutation(
    (vars: { name: string; body: sdk.JobPatchRequest }) => sdk.updateJob({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["jobs"]], success: t("jobs.saved") },
  );
  const trigger = useApiMutation(
    (vars: { name: string; body: sdk.RunTriggerRequest }) => sdk.triggerRun({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["jobs"]], success: t("jobs.runCreated") },
  );
  const pending = create.isPending || update.isPending || trigger.isPending;

  const onFinish = (v: JobFormValues) => {
    if (mode === "new") {
      create.mutate(
        { name: v.name.trim(), displayName: v.name.trim() || undefined, description: v.description?.trim() || undefined, spec: buildSpec(v) },
        { onSuccess: onClose },
      );
    } else if (mode === "edit") {
      update.mutate(
        { name: v.name.trim(), body: { description: v.description?.trim() || undefined, spec: buildSpec(v) } },
        { onSuccess: onClose },
      );
    } else {
      trigger.mutate(
        {
          name: v.name.trim(),
          body: { poolName: v.poolName?.trim() || undefined, unitName: v.unitName?.trim() || undefined, roles: [{ name: "worker", args: parseCommand(v.command || ""), env: parseEnv(v.env || "") }] },
        },
        { onSuccess: onClose },
      );
    }
  };

  const title = mode === "new" ? t("jobs.drawerNew") : mode === "run" ? t("jobs.drawerRun") : t("jobs.drawerEdit");
  const submitLabel = mode === "new" ? t("jobs.saveTemplate") : mode === "run" ? t("jobs.confirmRun") : t("common.save");

  return (
    <Drawer
      open
      width={640}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{title}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            {mode === "new" ? t("jobs.drawerNewSub") : <span className="font-mono">{initialName}</span>}
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={pending} onClick={() => form.submit()}>
            {submitLabel}
          </Button>
        </div>
      }
    >
      <Form<JobFormValues>
        form={form}
        layout="vertical"
        size="large"
        disabled={locked}
        onFinish={onFinish}
        initialValues={{
          name: mode === "new" ? "" : initialName,
          image: IMAGES[0].value,
          poolName: POOLS[0],
          unitName: UNITS[0].value,
          replicas: 4,
          command: CMD,
          env: "WANDB_DISABLED=true\nNCCL_DEBUG=INFO",
          volumes: [{ name: "training-data", mountPath: "/data" }],
          timeout: 86400,
          retries: 2,
        }}
      >
        <FieldSection n={1} title={t("jobs.fsBasic")} />
        <Form.Item name="name" label={t("jobs.fName")} rules={[{ required: mode !== "run", message: t("jobs.fNameHelp") }]} extra={mode !== "run" ? t("jobs.fNameHelp") : undefined}>
          <Input className="font-mono" placeholder={t("jobs.fNamePlaceholder")} disabled={locked || mode === "edit"} />
        </Form.Item>
        <Form.Item name="description" label={t("jobs.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("jobs.fDescPlaceholder")} />
        </Form.Item>

        <FieldSection n={2} title={t("jobs.fsImage")} />
        <Form.Item name="image" label={t("jobs.fImage")} rules={[{ required: true }]}>
          <CardRadio options={IMAGES} disabled={locked} />
        </Form.Item>

        <FieldSection n={3} title={t("jobs.fsResource")} />
        <Form.Item name="poolName" label={t("jobs.fPool")} rules={[{ required: true }]}>
          <Select options={POOLS.map((p) => ({ label: p, value: p }))} />
        </Form.Item>
        <Form.Item name="unitName" label={t("jobs.fUnit")} rules={[{ required: true }]}>
          <CardRadio options={UNITS} disabled={locked} />
        </Form.Item>
        <Form.Item name="replicas" label={t("jobs.fReplicas")} rules={[{ required: true }]}>
          <InputNumber min={1} className="!w-40" />
        </Form.Item>

        <FieldSection n={4} title={t("jobs.fsCommand")} />
        <Form.Item name="command" label={t("jobs.fCommand")} extra={mode !== "run" ? t("jobs.fCommandHelp") : undefined}>
          <Input.TextArea rows={3} className="font-mono" />
        </Form.Item>
        <Form.Item name="env" label={t("jobs.fEnv")} extra={mode !== "run" ? t("jobs.fEnvHelp") : undefined}>
          <Input.TextArea rows={2} className="font-mono" />
        </Form.Item>

        <FieldSection n={5} title={t("jobs.fsVolume")} />
        <Form.List name="volumes">
          {(fields, { add, remove }) => (
            <div className="space-y-2.5">
              {fields.map((field) => (
                <div key={field.key} className="flex items-start gap-2">
                  <Form.Item name={[field.name, "name"]} className="!mb-0 flex-1">
                    <Select placeholder={t("jobs.fVolume")} options={VOLUMES} />
                  </Form.Item>
                  <Form.Item name={[field.name, "mountPath"]} className="!mb-0 flex-1">
                    <Input className="font-mono" placeholder={t("jobs.fMountPath")} />
                  </Form.Item>
                  <Button
                    type="text"
                    danger
                    className="!mt-0.5"
                    icon={<DeleteOutlined />}
                    onClick={() => remove(field.name)}
                  />
                </div>
              ))}
              <Button type="dashed" block disabled={locked} onClick={() => add({ mountPath: "/data" })}>
                + {t("jobs.addVolume")}
              </Button>
            </div>
          )}
        </Form.List>

        <Collapse
          ghost
          className="!px-0"
          items={[
            {
              key: "adv",
              label: <span className="text-sm font-semibold text-accent">{t("common.advanced")}</span>,
              children: (
                <div className="flex gap-4">
                  <Form.Item name="timeout" label={t("jobs.fTimeout")} className="flex-1">
                    <InputNumber min={0} className="!w-full" />
                  </Form.Item>
                  <Form.Item name="retries" label={t("jobs.fRetries")} className="flex-1">
                    <InputNumber min={0} className="!w-full" />
                  </Form.Item>
                </div>
              ),
            },
          ]}
        />
      </Form>
    </Drawer>
  );
}
