import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { AppSidebar } from "@/components/sidebar/app-sidebar";
import { SidebarInset, SidebarProvider } from "@/components/sidebar/sidebar";
import { NavHeader } from "@/components/sidebar/nav-header";
import { ArtifactGallery } from "@/components/artifacts/artifact-gallery";
import { useProfile } from "@/stores/profile-store";

function ArtifactsPage() {
  const navigate = useNavigate();
  const preferences = useProfile((state) => state.preferences);
  const hideArtifacts = preferences?.preferences?.hide_artifacts ?? false;

  useEffect(() => {
    if (hideArtifacts) {
      navigate({ to: "/" });
    }
  }, [hideArtifacts, navigate]);

  if (hideArtifacts) {
    return null;
  }

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
