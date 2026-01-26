import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { AdminLayout } from "@/components/admin/admin-layout";
import { ProviderModelsManagement } from "@/components/admin/provider-models-management";

function AdminProviderModelsRoute() {
  return (
    <SidebarProvider>
      <AdminLayout>
        <ProviderModelsManagement />
      </AdminLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/admin/models/provider-models")({
  component: AdminProviderModelsRoute,
});
