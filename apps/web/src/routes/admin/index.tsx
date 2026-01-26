import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { AdminLayout } from "@/components/admin/admin-layout";
import { AdminOverview } from "@/components/admin/admin-overview";

function AdminIndexRoute() {
  return (
    <SidebarProvider>
      <AdminLayout>
        <AdminOverview />
      </AdminLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/admin/")({
  component: AdminIndexRoute,
});
