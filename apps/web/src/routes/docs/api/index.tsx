import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { DocsLayout } from "@/components/docs/docs-layout";
import { DocsPage, getStaticDoc } from "@/components/docs/docs-page";

function DocsApiIndexRoute() {
  const doc = getStaticDoc("/docs/api");

  return (
    <SidebarProvider>
      <DocsLayout>
        <DocsPage
          title={doc?.title || "API Reference"}
          description={doc?.description}
          content={doc?.content}
        />
      </DocsLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/docs/api/")({
  component: DocsApiIndexRoute,
});
