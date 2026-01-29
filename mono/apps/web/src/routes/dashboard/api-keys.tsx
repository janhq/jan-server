import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { DashboardLayout } from "@/components/dashboard/dashboard-layout";
import { ApiKeysTab } from "@/components/profile/api-keys-tab";

function DashboardApiKeysRoute() {
  return (
    <SidebarProvider>
      <DashboardLayout>
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-bold">API Keys</h1>
            <p className="text-muted-foreground">
              Manage your API keys for programmatic access
            </p>
          </div>
          <ApiKeysTab />
        </div>
      </DashboardLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/dashboard/api-keys" as "/")({
  component: DashboardApiKeysRoute,
});
