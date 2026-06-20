import { useState, type ReactNode } from "react";

export interface TabDef {
  key: string;
  label: ReactNode;
  count?: number;
  content: ReactNode;
}

// Underline tab strip (.tabs / .tabpane) matching the prototype's [data-tabs].
export function Tabs({ tabs, defaultKey }: { tabs: TabDef[]; defaultKey?: string }) {
  const [active, setActive] = useState(defaultKey ?? tabs[0]?.key);
  return (
    <div>
      <div className="tabs">
        {tabs.map((t) => (
          <button key={t.key} className={t.key === active ? "on" : ""} onClick={() => setActive(t.key)}>
            {t.label}
            {t.count != null && <span className="cnt">{t.count}</span>}
          </button>
        ))}
      </div>
      {tabs.map((t) => (
        <div key={t.key} className={"tabpane" + (t.key === active ? " on" : "")}>
          {t.key === active && t.content}
        </div>
      ))}
    </div>
  );
}
