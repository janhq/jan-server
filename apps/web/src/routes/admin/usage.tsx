import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { AdminLayout } from "@/components/admin/admin-layout";
import { AdminUsage } from "@/components/admin/admin-usage";

function AdminUsageRoute() {
  return (
    <SidebarProvider>
      <AdminLayout>
        <AdminUsage />
      </AdminLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/admin/usage" as "/")({
  component: AdminUsageRoute,
});
