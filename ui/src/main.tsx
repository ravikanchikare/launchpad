import React from "react";
import ReactDOM from "react-dom/client";
import { createRouter, createRootRoute, createRoute, Outlet, redirect, RouterProvider } from "@tanstack/react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { getSettings } from "@/api";
import { SidebarLayout } from "@/components/layout/layout";
import { AppSidebar } from "@/components/AppSidebar";
import { ConnectAppsScreen } from "@/components/ConnectAppsScreen";
import { SettingsScreen } from "@/components/Settings";
import "./index.css";

const queryClient = new QueryClient();

const rootRoute = createRootRoute({
  component: () => {
    // preserve preload parity: fetch settings on root mount (no cloud/chat)
    return (
      <div className="min-h-screen bg-white">
        <Outlet />
      </div>
    );
  },
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  beforeLoad: () => { throw redirect({ to: "/connect" }); },
});

const connectRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/connect",
  component: () => (
    <SidebarLayout title="Apps" sidebar={<AppSidebar current="apps" />}>
      <ConnectAppsScreen />
    </SidebarLayout>
  ),
});

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/settings",
  component: () => (
    <SidebarLayout title="Settings" sidebar={<AppSidebar current="settings" />}>
      <SettingsScreen />
    </SidebarLayout>
  ),
});

const routeTree = rootRoute.addChildren([indexRoute, connectRoute, settingsRoute]);
const router = createRouter({ routeTree, context: { queryClient } });

declare module "@tanstack/react-router" {
  interface Register { router: typeof router }
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} context={{ queryClient }} />
    </QueryClientProvider>
  </React.StrictMode>
);
