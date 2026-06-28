import { createBrowserRouter, Navigate, useLocation } from "react-router-dom";
import { lazy, Suspense, type ReactNode } from "react";
import { Spinner } from "@/components/ui/spinner";
import { useSession } from "@/app/session";
import { AppShell } from "@/components/layout/app-shell";

// Pages are code-split (React.lazy) so the initial bundle stays small and the
// chart-heavy pages (Dashboard) load on demand. AppShell renders the Suspense
// boundary around the routed <Outlet>; /login gets its own below.
const Login = lazy(() => import("@/pages/login"));
const Dashboard = lazy(() => import("@/pages/dashboard"));
const Workspaces = lazy(() => import("@/pages/workspaces"));
const WorkspaceDetail = lazy(() => import("@/pages/workspace-detail"));
const Experiments = lazy(() => import("@/pages/experiments"));
const ExperimentDetail = lazy(() => import("@/pages/experiment-detail"));
const Jobs = lazy(() => import("@/pages/jobs"));
const JobDetail = lazy(() => import("@/pages/job-detail"));
const RunDetail = lazy(() => import("@/pages/run-detail"));
const Services = lazy(() => import("@/pages/services"));
const ServiceDetail = lazy(() => import("@/pages/service-detail"));
const Traffic = lazy(() => import("@/pages/traffic"));
const TrafficDetail = lazy(() => import("@/pages/traffic-detail"));
const Models = lazy(() => import("@/pages/models"));
const Images = lazy(() => import("@/pages/images"));
const Tenants = lazy(() => import("@/pages/tenants"));
const ResourcePools = lazy(() => import("@/pages/resource-pools"));
const DataVolumes = lazy(() => import("@/pages/data-volumes"));

function PageFallback() {
  return (
    <div className="grid h-full place-items-center py-24">
      <Spinner className="size-7 text-muted-foreground" />
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
      { path: "/data-volumes", element: <DataVolumes /> },
    ],
  },
]);
