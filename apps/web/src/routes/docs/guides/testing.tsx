import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { DocsLayout } from "@/components/docs/docs-layout";
import { DocsPage, getStaticDoc } from "@/components/docs/docs-page";

function DocsGuidesTestingRoute() {
  const doc = getStaticDoc("/docs/guides/testing");

  return (
    <SidebarProvider>
      <DocsLayout>
        <DocsPage
          title={doc?.title || "Testing"}
          description={doc?.description}
          content={doc?.content}
        />
      </DocsLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/docs/guides/testing")({
  component: DocsGuidesTestingRoute,
});
