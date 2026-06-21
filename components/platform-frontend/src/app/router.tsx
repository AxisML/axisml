import { createBrowserRouter, Navigate, useLocation } from "react-router-dom";
import type { ReactNode } from "react";
import { useSession } from "@/app/session";
import { AppShell } from "@/shell/AppShell";
import Login from "@/pages/Login";
import Dashboard from "@/pages/Dashboard";
import Workspaces from "@/pages/Workspaces";
import WorkspaceDetail from "@/pages/WorkspaceDetail";
import Experiments from "@/pages/Experiments";
import ExperimentDetail from "@/pages/ExperimentDetail";
import Jobs from "@/pages/Jobs";
import JobDetail from "@/pages/JobDetail";
import RunDetail from "@/pages/RunDetail";
import Services from "@/pages/Services";
import ServiceDetail from "@/pages/ServiceDetail";
import Traffic from "@/pages/Traffic";
import TrafficDetail from "@/pages/TrafficDetail";
import Models from "@/pages/Models";
import Images from "@/pages/Images";
import Tenants from "@/pages/Tenants";
import ResourcePools from "@/pages/ResourcePools";

// Auth gate: hold rendering while the session hydrates, send anonymous users to
// /login (remembering where they were headed), admit authenticated users.
function RequireAuth({ children }: { children: ReactNode }) {
  const { status } = useSession();
  const location = useLocation();
  if (status === "loading") {
    return <div className="auth-splash" aria-busy="true" />;
  }
  if (status === "anon") {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
  }
  return <>{children}</>;
}

export const router = createBrowserRouter([
  { path: "/login", element: <Login /> },
  {
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      { path: "/", element: <Dashboard /> },
      { path: "/workspaces", element: <Workspaces /> },
      { path: "/workspaces/:name", element: <WorkspaceDetail /> },
      { path: "/experiments", element: <Experiments /> },
      { path: "/experiments/:name", element: <ExperimentDetail /> },
      { path: "/experiments/:name/runs/:run", element: <RunDetail kind="experiment" /> },
      { path: "/jobs", element: <Jobs /> },
      { path: "/jobs/:name", element: <JobDetail /> },
      { path: "/jobs/:name/runs/:run", element: <RunDetail kind="job" /> },
      { path: "/services", element: <Services /> },
      { path: "/services/:name", element: <ServiceDetail /> },
      { path: "/traffic", element: <Traffic /> },
      { path: "/traffic/:name", element: <TrafficDetail /> },
      { path: "/models", element: <Models /> },
      { path: "/images", element: <Images /> },
      { path: "/tenants", element: <Tenants /> },
      { path: "/resource-pools", element: <ResourcePools /> },
    ],
  },
]);
