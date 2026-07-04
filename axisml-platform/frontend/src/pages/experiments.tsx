import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { usePagedList } from "@/api/hooks";
import { useDebouncedValue } from "@/lib/use-debounced";
import { LoadMore } from "@/components/load-more";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { RunStrip } from "@/components/run-strip";
import { SearchInput } from "@/components/search-input";
import { FilterSelect } from "@/components/filter-select";
import { DataTable, type Column } from "@/components/data-table";
import { ExpDrawer, type DrawerMode } from "@/components/exp-drawer";
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
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; name?: string } | null>(null);
  const [search, setSearch] = useState("");
  const [creator, setCreator] = useState<string>("");

  const dq = useDebouncedValue(search, 300);
  const q = usePagedList<sdk.Experiment>(["experiments", dq, creator], (page) =>
    sdk.listExperiments({ query: { q: dq || undefined, owner: creator || undefined, ...page } }),
  );

  const delExp = useApiMutation((name: string) => sdk.deleteExperiment({ path: { name } }), {
    invalidate: [["experiments"]],
    success: t("experiments.deleted"),
  });

  const rows: ExpRow[] = useMemo(
    () =>
      q.items.map((e) => {
        const summary = e.runSummary;
        return {
          name: e.name,
          desc: e.description ?? e.displayName ?? "",
          runCount: summary?.count ?? 0,
          recent: summary?.recent ?? [],
          owner: e.owner ?? "—",
          updated: e.updatedAt ?? e.createdAt ?? "",
        };
      }),
    [q.items],
  );

  const creatorOptions = useMemo(
    () => Array.from(new Set(rows.map((r) => r.owner).filter((o) => o && o !== "—"))),
    [rows],
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
      <LoadMore hasMore={q.hasMore} loading={q.isFetchingMore} onClick={q.loadMore} />

      {drawer && <ExpDrawer mode={drawer.mode} name={drawer.name} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}
