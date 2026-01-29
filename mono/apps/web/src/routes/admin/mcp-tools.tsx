import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { AdminLayout } from "@/components/admin/admin-layout";
import { MCPToolsManagement } from "@/components/admin/mcp-tools-management";

function AdminMCPToolsRoute() {
  return (
    <SidebarProvider>
      <AdminLayout>
        <MCPToolsManagement />
      </AdminLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/admin/mcp-tools")({
  component: AdminMCPToolsRoute,
});
