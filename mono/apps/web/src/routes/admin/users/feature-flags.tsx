import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { AdminLayout } from "@/components/admin/admin-layout";
import { FeatureFlagsManagement } from "@/components/admin/feature-flags-management";

function AdminFeatureFlagsRoute() {
  return (
    <SidebarProvider>
      <AdminLayout>
        <FeatureFlagsManagement />
      </AdminLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/admin/users/feature-flags")({
  component: AdminFeatureFlagsRoute,
});
