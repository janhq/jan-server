import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { ConnectorsPage } from "@/components/connectors/connectors-page";
import { useProfile } from "@/stores/profile-store";

function ConnectorsRoute() {
  const navigate = useNavigate();
  const preferences = useProfile((state) => state.preferences);
  const hideConnectors = preferences?.preferences?.hide_connectors ?? false;

  useEffect(() => {
    if (hideConnectors) {
      navigate({ to: "/" });
    }
  }, [hideConnectors, navigate]);

  if (hideConnectors) {
    return null;
  }

  return (
    <SidebarProvider>
      <ConnectorsPage />
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/connectors/")({
  component: ConnectorsRoute,
});
