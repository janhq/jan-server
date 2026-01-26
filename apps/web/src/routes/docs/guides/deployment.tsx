import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { DocsLayout } from "@/components/docs/docs-layout";
import { DocsPage, getStaticDoc } from "@/components/docs/docs-page";

function DocsGuidesDeploymentRoute() {
  const doc = getStaticDoc("/docs/guides/deployment");

  return (
    <SidebarProvider>
      <DocsLayout>
        <DocsPage
          title={doc?.title || "Deployment"}
          description={doc?.description}
          content={doc?.content}
        />
      </DocsLayout>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/docs/guides/deployment")({
  component: DocsGuidesDeploymentRoute,
});
