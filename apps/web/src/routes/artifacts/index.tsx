import { createFileRoute } from "@tanstack/react-router";
import { AppSidebar } from "@/components/sidebar/app-sidebar";
import { SidebarInset, SidebarProvider } from "@/components/sidebar/sidebar";
import { NavHeader } from "@/components/sidebar/nav-header";
import { ArtifactGallery } from "@/components/artifacts/artifact-gallery";

function ArtifactsPage() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <NavHeader />
        <ArtifactGallery />
      </SidebarInset>
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/artifacts/")({
  component: ArtifactsPage,
});
