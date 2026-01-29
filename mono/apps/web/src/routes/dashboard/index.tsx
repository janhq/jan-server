import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { DashboardLayout } from "@/components/dashboard/dashboard-layout";
import { UserTab } from "@/components/profile/user-tab";

function DashboardProfileRoute() {
  return (
    <SidebarProvider>
      <DashboardLayout>
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-bold">Profile</h1>
            <p className="text-muted-foreground">
              View your account information
            </p>
          </div>
          <UserTab />
        </div>
      </DashboardLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/dashboard/" as "/")({
  component: DashboardProfileRoute,
});
