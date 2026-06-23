import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useExperiments } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { RunStrip } from "@/components/run-strip";
import { SearchInput } from "@/components/search-input";
import { FilterSelect } from "@/components/filter-select";
import { DataTable, type Column } from "@/components/data-table";
import { ExpDrawer, type DrawerMode } from "@/components/exp-drawer";
import { USE_MOCK } from "@/api/mock";
import { runSummary } from "@/api/mock/data";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

interface ExpRow {
  name: string;
  desc: string;
  runCount: number;
  recent: string[];
  owner: string;
  updated: string;
}

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
          <Button variant="link" size="sm" onClick={() => setDrawer({ mode: "run", name: r.name })}>
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
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <SearchInput
          className="max-w-xs flex-1"
          placeholder={t("experiments.searchPlaceholder")}
          value={search}
          onChange={setSearch}
        />
        <FilterSelect
          value={creator}
          onChange={setCreator}
          options={creatorOptions}
          allLabel={t("experiments.creatorAll")}
        />
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
      <Card className="overflow-hidden p-0">
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
