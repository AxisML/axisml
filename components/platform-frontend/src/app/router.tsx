import { createBrowserRouter, Navigate, useLocation } from "react-router-dom";
import { lazy, Suspense, type ReactNode } from "react";
import { Spin } from "antd";
import { useSession } from "@/app/session";
import { AppShell } from "@/shell/AppShell";

// Pages are code-split (React.lazy) so the initial bundle stays small and the
// chart-heavy pages (Dashboard) load on demand. AppShell renders the Suspense
// boundary around the routed <Outlet>; /login gets its own below.
const Login = lazy(() => import("@/pages/Login"));
const Dashboard = lazy(() => import("@/pages/Dashboard"));
const Workspaces = lazy(() => import("@/pages/Workspaces"));
const WorkspaceDetail = lazy(() => import("@/pages/WorkspaceDetail"));
const Experiments = lazy(() => import("@/pages/Experiments"));
const ExperimentDetail = lazy(() => import("@/pages/ExperimentDetail"));
const Jobs = lazy(() => import("@/pages/Jobs"));
const JobDetail = lazy(() => import("@/pages/JobDetail"));
const RunDetail = lazy(() => import("@/pages/RunDetail"));
const Services = lazy(() => import("@/pages/Services"));
const ServiceDetail = lazy(() => import("@/pages/ServiceDetail"));
const Traffic = lazy(() => import("@/pages/Traffic"));
const TrafficDetail = lazy(() => import("@/pages/TrafficDetail"));
const Models = lazy(() => import("@/pages/Models"));
const Images = lazy(() => import("@/pages/Images"));
const Tenants = lazy(() => import("@/pages/Tenants"));
const ResourcePools = lazy(() => import("@/pages/ResourcePools"));

function PageFallback() {
  return (
    <div className="grid h-full place-items-center py-24">
      <Spin size="large" />
    </div>
  );
}

// Auth gate: hold rendering while the session hydrates, send anonymous users to
// /login (remembering where they were headed), admit authenticated users.
function RequireAuth({ children }: { children: ReactNode }) {
  const { status } = useSession();
  const location = useLocation();
  if (status === "loading") {
    return <PageFallback />;
  }
  if (status === "anon") {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
  }
  return <>{children}</>;
}

export const router = createBrowserRouter([
  {
    path: "/login",
    element: (
      <Suspense fallback={<PageFallback />}>
        <Login />
      </Suspense>
    ),
  },
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
