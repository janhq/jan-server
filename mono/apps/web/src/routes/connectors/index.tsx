import { createFileRoute } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/sidebar/sidebar";
import { ConnectorsPage } from "@/components/connectors/connectors-page";

function ConnectorsRoute() {
  return (
    <SidebarProvider>
      <ConnectorsPage />
    </SidebarProvider>
  );
}

export const Route = createFileRoute("/connectors/")({
  component: ConnectorsRoute,
});
