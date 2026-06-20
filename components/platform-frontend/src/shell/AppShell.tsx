import { Outlet } from "react-router-dom";
import { useApp } from "@/app/store";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";

export function AppShell() {
  const { collapsed } = useApp();
  return (
    <div className={"app-shell" + (collapsed ? " collapsed" : "")}>
      <Sidebar />
      <div className="app-main">
        <Topbar />
        <Outlet />
      </div>
    </div>
  );
}
