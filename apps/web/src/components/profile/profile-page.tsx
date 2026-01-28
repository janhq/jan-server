import { useState } from "react";
import { AppSidebar } from "@/components/sidebar/app-sidebar";
import { SidebarInset } from "@/components/sidebar/sidebar";
import { NavHeader } from "@/components/sidebar/nav-header";
import { ApiKeysTab } from "./api-keys-tab";
import { UserTab } from "./user-tab";
import { UsageTab } from "./usage-tab";
import { ConnectorsTab } from "./connectors-tab";
import { cn } from "@/lib/utils";

export function ProfilePage() {
  const [activeTab, setActiveTab] = useState<
    "user" | "api-keys" | "usage" | "connectors"
  >("user");

  return (
    <>
      <AppSidebar />
      <SidebarInset>
        <NavHeader showModelSelector={false} />
        <div className="flex-1 overflow-auto">
          <div className="container max-w-screen-lg py-10 px-4 md:px-6">
            <h1 className="mb-8 text-3xl font-bold">Profile</h1>

            <div className="mb-8 border-b">
              <nav className="-mb-px flex space-x-8" aria-label="Tabs">
                <button
                  onClick={() => setActiveTab("user")}
                  className={cn(
                    "whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium transition-colors",
                    activeTab === "user"
                      ? "border-primary text-foreground"
                      : "border-transparent text-muted-foreground hover:border-border hover:text-foreground",
                  )}
                >
                  User
                </button>
                <button
                  onClick={() => setActiveTab("api-keys")}
                  className={cn(
                    "whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium transition-colors",
                    activeTab === "api-keys"
                      ? "border-primary text-foreground"
                      : "border-transparent text-muted-foreground hover:border-border hover:text-foreground",
                  )}
                >
                  API Keys
                </button>
                <button
                  onClick={() => setActiveTab("usage")}
                  className={cn(
                    "whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium transition-colors",
                    activeTab === "usage"
                      ? "border-primary text-foreground"
                      : "border-transparent text-muted-foreground hover:border-border hover:text-foreground",
                  )}
                >
                  Usage
                </button>
                <button
                  onClick={() => setActiveTab("connectors")}
                  className={cn(
                    "whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium transition-colors",
                    activeTab === "connectors"
                      ? "border-primary text-foreground"
                      : "border-transparent text-muted-foreground hover:border-border hover:text-foreground",
                  )}
                >
                  Connectors
                </button>
              </nav>
            </div>

            {activeTab === "user" && <UserTab />}
            {activeTab === "api-keys" && <ApiKeysTab />}
            {activeTab === "usage" && <UsageTab />}
            {activeTab === "connectors" && <ConnectorsTab />}
          </div>
        </div>
      </SidebarInset>
    </>
  );
}
