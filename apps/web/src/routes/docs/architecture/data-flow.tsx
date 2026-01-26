import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { DocsLayout } from "@/components/docs/docs-layout";
import { DocsPage, getStaticDoc } from "@/components/docs/docs-page";

function DocsArchitectureDataFlowRoute() {
  const doc = getStaticDoc("/docs/architecture/data-flow");

  return (
    <SidebarProvider>
      <DocsLayout>
        <DocsPage
          title={doc?.title || "Data Flow"}
          description={doc?.description}
          content={doc?.content}
        />
      </DocsLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/docs/architecture/data-flow")({
  component: DocsArchitectureDataFlowRoute,
});
