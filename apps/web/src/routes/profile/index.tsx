import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { ProfilePage } from "@/components/profile/profile-page";

function ProfileRoute() {
  return (
    <SidebarProvider>
      <ProfilePage />
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/profile/")({
  component: ProfileRoute,
});
