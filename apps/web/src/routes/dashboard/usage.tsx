import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { DashboardLayout } from "@/components/dashboard/dashboard-layout";
import { UsageTab } from "@/components/profile/usage-tab";

function DashboardUsageRoute() {
  return (
    <SidebarProvider>
      <DashboardLayout>
        <UsageTab />
      </DashboardLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/dashboard/usage" as "/")({
  component: DashboardUsageRoute,
});
