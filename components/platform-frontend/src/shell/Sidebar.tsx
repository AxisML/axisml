import { NavLink } from "react-router-dom";
import { NAV, useApp } from "@/app/store";
import { Icon } from "@/components/Icon";

export function Sidebar() {
  const { canSee } = useApp();
  return (
    <aside className="sidebar" id="sidebar">
      <div className="side-brand">
        <div className="mark">A</div>
        <div className="name">
          Axis<b>ML</b>
        </div>
      </div>
      <div className="side-scroll">
        {NAV.map((blk, i) => {
          const visible = blk.items.filter(canSee);
          if (!visible.length) return null;
          return (
            <div className="nav-group" key={blk.group || i}>
              {blk.group && <div className="grp-label">{blk.group}</div>}
              {visible.map((it) => (
                <NavLink
                  key={it.key}
                  to={it.path}
                  end={it.path === "/"}
                  className={({ isActive }) => "nav-item" + (isActive ? " active" : "")}
                  title={it.label}
                >
                  <Icon name={it.icon} />
                  <span className="lbl">{it.label}</span>
                </NavLink>
              ))}
            </div>
          );
        })}
      </div>
    </aside>
  );
}
