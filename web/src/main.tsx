import "@/styles/index.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, Outlet } from "react-router";
import { RouterProvider } from "react-router/dom";
import { AppShell } from "@/components/app-shell";
import { InstallContext } from "@/components/install-button";
import { ToastProvider } from "@/components/toaster";
import { useAuth } from "@/data/hooks";
import { DataProvider } from "@/data/provider";
import { InstallPrompt } from "@/lib/install";
import GoalScreen from "@/routes/goal";
import Goals from "@/routes/goals";
import Inbox from "@/routes/inbox";
import Login from "@/routes/login";
import Plan from "@/routes/plan";
import PlanProjects from "@/routes/plan-projects";
import ProjectScreen from "@/routes/project";
import Reminders from "@/routes/reminders";
import Review from "@/routes/review";
import Settings from "@/routes/settings";
import Today from "@/routes/today";

function SignedIn() {
  const { auth } = useAuth();
  return auth === "signedIn" ? <Outlet /> : <Navigate to="/login" replace />;
}

function SignedOut() {
  const { auth } = useAuth();
  return auth === "signedIn" ? <Navigate to="/" replace /> : <Outlet />;
}

const router = createBrowserRouter([
  {
    element: <SignedOut />,
    children: [{ path: "/login", element: <Login /> }],
  },
  {
    element: <SignedIn />,
    children: [
      {
        element: <AppShell />,
        children: [
          { path: "/", element: <Today /> },
          { path: "/pick", element: <Navigate to="/?pick=1" replace /> },
          { path: "/inbox", element: <Inbox /> },
          { path: "/plan", element: <Plan /> },
          { path: "/plan/projects", element: <PlanProjects /> },
          { path: "/projects/:id", element: <ProjectScreen /> },
          { path: "/goals", element: <Goals /> },
          { path: "/goals/:id", element: <GoalScreen /> },
          { path: "/reminders", element: <Reminders /> },
          { path: "/review", element: <Review /> },
          { path: "/settings", element: <Settings /> },
          { path: "*", element: <Navigate to="/" replace /> },
        ],
      },
    ],
  },
]);

const install = new InstallPrompt();

if ("serviceWorker" in navigator && import.meta.env.PROD) {
  void navigator.serviceWorker.register("/sw.js", { scope: "/" });
}

const root = document.getElementById("root");
if (!root) {
  throw new Error("Missing #root element");
}

createRoot(root).render(
  <StrictMode>
    <InstallContext value={install}>
      <ToastProvider>
        <DataProvider fallback={<div className="min-h-dvh bg-bg" />}>
          <RouterProvider router={router} />
        </DataProvider>
      </ToastProvider>
    </InstallContext>
  </StrictMode>,
);
