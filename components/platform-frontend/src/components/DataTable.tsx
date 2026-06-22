import { useMemo, useState, type ReactNode } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

// Generic list table — the workhorse behind every list page. Wraps the shadcn
// Table with client-side pagination plus honest loading / error / empty states,
// keeping a column API close to what the pages previously used.
export interface Column<T> {
  key: string;
  title: ReactNode;
  /** Read a primitive cell value when no `render` is given. */
  dataIndex?: keyof T;
  render?: (row: T, index: number) => ReactNode;
  width?: number | string;
  align?: "left" | "right" | "center";
  className?: string;
}

const alignClass = { left: "text-left", right: "text-right", center: "text-center" } as const;

export function DataTable<T>({
  columns,
  data,
  rowKey,
  loading,
  error,
  empty,
  pageSize = 20,
  pagination = true,
  className,
}: {
  columns: Column<T>[];
  data: T[];
  rowKey: (row: T, index: number) => string;
  loading?: boolean;
  error?: boolean;
  empty?: ReactNode;
  pageSize?: number;
  /** Set false for small inline/sub tables that shouldn't show a pager footer. */
  pagination?: boolean;
  className?: string;
}) {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const effectivePageSize = pagination ? pageSize : Math.max(data.length, 1);
  const pageCount = Math.max(1, Math.ceil(data.length / effectivePageSize));
  const current = Math.min(page, pageCount);
  const rows = useMemo(
    () => data.slice((current - 1) * pageSize, current * pageSize),
    [data, current, pageSize],
  );

  const colCount = columns.length;
  const message = error ? t("common.loadFailed") : (empty ?? t("common.noData"));

  return (
    <div className={cn("flex flex-col", className)}>
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {columns.map((c) => (
              <TableHead
                key={c.key}
                style={c.width ? { width: c.width } : undefined}
                className={cn(c.align && alignClass[c.align], "text-muted-foreground")}
              >
                {c.title}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {loading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <TableRow key={`sk-${i}`} className="hover:bg-transparent">
                {columns.map((c) => (
                  <TableCell key={c.key}>
                    <Skeleton className="h-4 w-full max-w-40" />
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : rows.length === 0 ? (
            <TableRow className="hover:bg-transparent">
              <TableCell colSpan={colCount} className="h-28 text-center text-muted-foreground">
                {message}
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row, i) => (
              <TableRow key={rowKey(row, i)}>
                {columns.map((c) => (
                  <TableCell
                    key={c.key}
                    style={c.width ? { width: c.width } : undefined}
                    className={cn(c.align && alignClass[c.align], c.className)}
                  >
                    {c.render ? c.render(row, i) : c.dataIndex ? (row[c.dataIndex] as ReactNode) : null}
                  </TableCell>
                ))}
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      {!loading && data.length > 0 && (
        <div className="flex items-center justify-between border-t px-4 py-3 text-sm text-muted-foreground">
          <span>{t("common.totalItems", { count: data.length })}</span>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon-sm"
              disabled={current <= 1}
              onClick={() => setPage(current - 1)}
              aria-label="prev"
            >
              <ChevronLeft />
            </Button>
            <span className="min-w-16 text-center text-foreground">
              {current} / {pageCount}
            </span>
            <Button
              variant="outline"
              size="icon-sm"
              disabled={current >= pageCount}
              onClick={() => setPage(current + 1)}
              aria-label="next"
            >
              <ChevronRight />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
