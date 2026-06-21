// Shared async-state UI. Pages render genuine loading / error / empty states
// straight from the react-query result — there is NO fallback to demo data, so a
// broken backend call shows up as an error the user (and we) can see and fix.

export interface QueryLike {
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  refetch?: () => unknown;
}

// Best-effort human message out of a backend Problem / Error / unknown throw.
export function errorText(err: unknown): string {
  if (!err) return "未知错误";
  if (typeof err === "string") return err;
  const e = err as { detail?: string; title?: string; message?: string };
  return e.detail || e.title || e.message || "请求失败";
}

function RetryButton({ q }: { q: QueryLike }) {
  if (!q.refetch) return null;
  return (
    <button className="btn btn-sm" style={{ marginLeft: 12 }} onClick={() => q.refetch?.()}>
      重试
    </button>
  );
}

// Table variant: returns a full-width <tr> for the current state, or null when
// data is present and non-empty (so the caller's mapped rows render instead).
export function TableState({
  q,
  cols,
  isEmpty,
  empty = "暂无数据",
}: {
  q: QueryLike;
  cols: number;
  isEmpty: boolean;
  empty?: string;
}) {
  let body: React.ReactNode = null;
  if (q.isLoading) body = <span className="muted">加载中…</span>;
  else if (q.isError)
    body = (
      <span className="data-state-err">
        加载失败：{errorText(q.error)}
        <RetryButton q={q} />
      </span>
    );
  else if (isEmpty) body = <span className="muted">{empty}</span>;
  if (!body) return null;
  return (
    <tr>
      <td colSpan={cols} className="data-state">
        {body}
      </td>
    </tr>
  );
}

// Block variant for card grids / non-table layouts. Returns a centered state
// block, or null when data is present and non-empty.
export function BlockState({
  q,
  isEmpty,
  empty = "暂无数据",
}: {
  q: QueryLike;
  isEmpty: boolean;
  empty?: string;
}) {
  if (q.isLoading) return <div className="data-state muted">加载中…</div>;
  if (q.isError)
    return (
      <div className="data-state data-state-err">
        加载失败：{errorText(q.error)}
        <RetryButton q={q} />
      </div>
    );
  if (isEmpty) return <div className="data-state muted">{empty}</div>;
  return null;
}
