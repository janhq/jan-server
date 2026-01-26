import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { AdminLayout } from "@/components/admin/admin-layout";
import { CatalogsManagement } from "@/components/admin/catalogs-management";

function AdminCatalogsRoute() {
  return (
    <SidebarProvider>
      <AdminLayout>
        <CatalogsManagement />
      </AdminLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/admin/models/catalogs")({
  component: AdminCatalogsRoute,
});
