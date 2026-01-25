import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { AdminLayout } from "@/components/admin/admin-layout";
import { ModelsOverview } from "@/components/admin/models-overview";

function AdminModelsRoute() {
  return (
    <SidebarProvider>
      <AdminLayout>
        <ModelsOverview />
      </AdminLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/admin/models/")({
  component: AdminModelsRoute,
});
