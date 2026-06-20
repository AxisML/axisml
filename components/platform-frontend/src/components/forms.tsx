import { useState, type ReactNode } from "react";
import { Icon } from "./Icon";

// ── PickGrid — radio-card group (.pick-grid / .pick) used for image & unit pick ──
export interface PickOption {
  title: string;
  spec: string;
}
export function PickGrid({ options, defaultIndex = 0 }: { options: PickOption[]; defaultIndex?: number }) {
  const [sel, setSel] = useState(defaultIndex);
  return (
    <div className="pick-grid">
      {options.map((o, i) => (
        <div key={o.title} className={"pick" + (i === sel ? " on" : "")} onClick={() => setSel(i)}>
          <div className="p-title">{o.title}</div>
          <div className="p-spec">{o.spec}</div>
        </div>
      ))}
    </div>
  );
}

// ── FieldsetTitle — numbered section header inside drawers (.fieldset-title) ──────
export function FieldsetTitle({ n, children, extra }: { n: number; children: ReactNode; extra?: ReactNode }) {
  return (
    <div className="fieldset-title">
      <span className="n">{n}</span>
      {children}
      {extra}
    </div>
  );
}

// ── VolList — repeatable data-volume mount rows (.vol-list / .vol-row) ────────────
interface VolRow {
  id: number;
  options: string[];
  path: string;
}
let volSeq = 0;
export function VolList({
  initial = [{ options: ["training-data · 200 GiB", "shared-cache · 500 GiB", "新建数据卷…"], path: "/data" }],
}: {
  initial?: { options: string[]; path: string }[];
}) {
  const [rows, setRows] = useState<VolRow[]>(() => initial.map((r) => ({ ...r, id: ++volSeq })));
  const add = () =>
    setRows((r) => [...r, { id: ++volSeq, options: initial[0]?.options ?? ["新建数据卷…"], path: "" }]);
  const remove = (id: number) => setRows((r) => (r.length <= 1 ? r : r.filter((x) => x.id !== id)));
  const only = rows.length <= 1;
  return (
    <>
      <div className="vol-list">
        {rows.map((row) => (
          <div className="vol-row" key={row.id} {...(only ? { "data-only": "" } : {})}>
            <select className="input" aria-label="数据卷" defaultValue={row.options[0]}>
              {row.options.map((o) => (
                <option key={o}>{o}</option>
              ))}
            </select>
            <input className="input mono" defaultValue={row.path} placeholder="挂载路径" aria-label="挂载路径" />
            <button type="button" className="icon-btn" title="移除" onClick={() => remove(row.id)}>
              <Icon name="x" />
            </button>
          </div>
        ))}
      </div>
      <a className="link vol-add" role="button" tabIndex={0} onClick={add}>
        <Icon name="plus" />
        添加数据卷
      </a>
    </>
  );
}
