import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { DocsLayout } from "@/components/docs/docs-layout";
import { DocsPage, getStaticDoc } from "@/components/docs/docs-page";

function DocsConfigurationProvidersRoute() {
  const doc = getStaticDoc("/docs/configuration/providers");

  return (
    <SidebarProvider>
      <DocsLayout>
        <DocsPage
          title={doc?.title || "Providers"}
          description={doc?.description}
          content={doc?.content}
        />
      </DocsLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/docs/configuration/providers")({
  component: DocsConfigurationProvidersRoute,
});
