import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { AdminLayout } from "@/components/admin/admin-layout";
import { PromptTemplatesManagement } from "@/components/admin/prompt-templates-management";

function AdminPromptTemplatesRoute() {
  return (
    <SidebarProvider>
      <AdminLayout>
        <PromptTemplatesManagement />
      </AdminLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/admin/prompt-templates")({
  component: AdminPromptTemplatesRoute,
});
