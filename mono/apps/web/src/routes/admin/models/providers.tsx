import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { AdminLayout } from "@/components/admin/admin-layout";
import { ProvidersManagement } from "@/components/admin/providers-management";

function AdminProvidersRoute() {
  return (
    <SidebarProvider>
      <AdminLayout>
        <ProvidersManagement />
      </AdminLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/admin/models/providers")({
  component: AdminProvidersRoute,
});
