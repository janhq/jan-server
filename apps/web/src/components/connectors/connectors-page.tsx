import { useEffect, useState } from "react";
import { AppSidebar } from "@/components/sidebar/app-sidebar";
import { SidebarInset } from "@/components/sidebar/sidebar";
import { NavHeader } from "@/components/sidebar/nav-header";
import { useConnectorStore } from "@/stores/connector-store";
import type { ConnectorType } from "@/services/connector-service";
import { cn } from "@/lib/utils";
import {
  Loader2,
  Check,
  X,
  ExternalLink,
  RefreshCw,
  Plug,
  Search,
} from "lucide-react";

// Connector icons mapping
const connectorIcons: Record<ConnectorType, React.ReactNode> = {
  github: (
    <svg viewBox="0 0 24 24" className="h-8 w-8" fill="currentColor">
      <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
    </svg>
  ),
  gmail: (
    <svg viewBox="0 0 24 24" className="h-8 w-8" fill="currentColor">
      <path d="M24 5.457v13.909c0 .904-.732 1.636-1.636 1.636h-3.819V11.73L12 16.64l-6.545-4.91v9.273H1.636A1.636 1.636 0 0 1 0 19.366V5.457c0-2.023 2.309-3.178 3.927-1.964L5.455 4.64 12 9.548l6.545-4.91 1.528-1.145C21.69 2.28 24 3.434 24 5.457z" />
    </svg>
  ),
  google_drive: (
    <svg viewBox="0 0 24 24" className="h-8 w-8" fill="currentColor">
      <path d="M12.01 1.485c-2.082 0-3.754.02-3.743.047.01.02 1.708 3.001 3.774 6.62l3.76 6.574h3.76c2.081 0 3.753-.02 3.742-.047-.005-.02-1.708-3.001-3.775-6.62l-3.76-6.574h-3.758zm-4.76 1.73a789.828 789.828 0 0 0-3.63 6.319L0 15.868l1.89 3.298 1.885 3.297 3.62-6.335 3.618-6.33-1.876-3.294c-1.03-1.81-1.893-3.29-1.897-3.289zm4.65 8.421-3.618 6.33-3.618 6.33h7.24l3.617-6.33c1.988-3.48 3.617-6.34 3.617-6.36s-1.63-.04-3.62-.04h-3.618z" />
    </svg>
  ),
  google_calendar: (
    <svg viewBox="0 0 24 24" className="h-8 w-8" fill="currentColor">
      <path d="M18.316 5.684H24v12.632h-5.684V5.684zM5.684 24h12.632v-5.684H5.684V24zM18.316 5.684V0H1.895A1.894 1.894 0 0 0 0 1.895v16.421h5.684V5.684h12.632zm-7.207 6.25v-.065c.272-.144.5-.349.687-.617s.279-.595.279-.982c0-.379-.099-.72-.3-1.025a2.05 2.05 0 0 0-.832-.714 2.703 2.703 0 0 0-1.197-.257c-.6 0-1.094.156-1.481.467s-.581.74-.58 1.283h1.164c.001-.24.088-.429.256-.568s.394-.21.668-.21c.266 0 .483.082.645.246s.242.37.242.617c0 .267-.08.473-.242.617s-.367.216-.61.216h-.515v.936h.515c.337 0 .6.084.787.25s.28.39.28.658c0 .254-.091.456-.272.607s-.417.226-.706.226c-.273 0-.5-.082-.68-.243s-.27-.374-.27-.64h-1.181c0 .576.202 1.041.604 1.392s.937.528 1.602.528c.637 0 1.163-.164 1.576-.491s.62-.748.62-1.262c0-.325-.094-.617-.282-.876s-.451-.451-.789-.576zm4.063-4.08H24v2.842h-8.828V7.854z" />
    </svg>
  ),
};

// Connector descriptions
const connectorDescriptions: Record<ConnectorType, string> = {
  github:
    "Search repositories, manage issues, create pull requests, and commit code directly from chat.",
  gmail:
    "Search and read emails from your Gmail inbox to provide context in conversations.",
  google_drive:
    "Search and access files from your Google Drive for reference in conversations.",
  google_calendar:
    "View, create, and manage calendar events directly from chat.",
};

// Connector categories
const connectorCategories: Record<ConnectorType, string> = {
  github: "Development",
  gmail: "Productivity",
  google_drive: "Productivity",
  google_calendar: "Productivity",
};

interface ConnectorCardProps {
  type: ConnectorType;
  displayName: string;
  enabled: boolean;
}

function ConnectorCard({ type, displayName, enabled }: ConnectorCardProps) {
  const {
    statuses,
    isConnecting,
    initiateConnect,
    disconnect,
    refreshTokens,
    fetchStatus,
  } = useConnectorStore();

  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isDisconnecting, setIsDisconnecting] = useState(false);

  const status = statuses[type];
  const isConnected = status?.connected ?? false;
  const connecting = isConnecting[type] ?? false;

  useEffect(() => {
    if (enabled) {
      fetchStatus(type);
    }
  }, [enabled, type, fetchStatus]);

  const handleConnect = async () => {
    await initiateConnect(type);
  };

  const handleDisconnect = async () => {
    setIsDisconnecting(true);
    try {
      await disconnect(type);
    } finally {
      setIsDisconnecting(false);
    }
  };

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try {
      await refreshTokens(type);
    } finally {
      setIsRefreshing(false);
    }
  };

  if (!enabled) {
    return (
      <div className="group relative rounded-xl border border-border bg-muted/20 p-5 opacity-50 cursor-not-allowed">
        <div className="flex items-start gap-4">
          <div className="flex-shrink-0 rounded-lg bg-muted/50 p-2.5 text-muted-foreground">
            {connectorIcons[type]}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="font-semibold text-foreground">{displayName}</h3>
              <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
                Not available
              </span>
            </div>
            <p className="mt-1.5 text-sm text-muted-foreground line-clamp-2">
              {connectorDescriptions[type]}
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "group relative rounded-xl border p-5 transition-all duration-200",
        isConnected
          ? "border-green-500/40 bg-green-500/5 hover:border-green-500/60 hover:bg-green-500/10"
          : "border-border bg-card hover:border-primary/30 hover:bg-muted/30",
      )}
    >
      <div className="flex items-start gap-4">
        <div
          className={cn(
            "flex-shrink-0 rounded-lg p-2.5 transition-colors",
            isConnected
              ? "bg-green-500/10 text-green-500"
              : "bg-muted/50 text-muted-foreground group-hover:bg-primary/10 group-hover:text-primary",
          )}
        >
          {connectorIcons[type]}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="font-semibold text-foreground">{displayName}</h3>
            {isConnected && (
              <span className="flex items-center gap-1 rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-500">
                <Check className="h-3 w-3" />
                Connected
              </span>
            )}
          </div>
          <p className="mt-1.5 text-sm text-muted-foreground line-clamp-2">
            {connectorDescriptions[type]}
          </p>

          {isConnected && status && (
            <div className="mt-3 flex items-center gap-4 text-xs text-muted-foreground">
              {status.username && (
                <span>
                  <span className="text-foreground font-medium">
                    {status.username}
                  </span>
                </span>
              )}
              {status.email && !status.username && (
                <span>
                  <span className="text-foreground font-medium">
                    {status.email}
                  </span>
                </span>
              )}
            </div>
          )}

          <div className="mt-4 flex items-center gap-2">
            {isConnected ? (
              <>
                <button
                  onClick={handleRefresh}
                  disabled={isRefreshing}
                  className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-50"
                >
                  {isRefreshing ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <RefreshCw className="h-3.5 w-3.5" />
                  )}
                  Refresh
                </button>
                <button
                  onClick={handleDisconnect}
                  disabled={isDisconnecting}
                  className="inline-flex items-center gap-1.5 rounded-lg border border-red-500/30 bg-red-500/5 px-3 py-1.5 text-sm font-medium text-red-500 transition-colors hover:bg-red-500/10 disabled:opacity-50"
                >
                  {isDisconnecting ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <X className="h-3.5 w-3.5" />
                  )}
                  Disconnect
                </button>
              </>
            ) : (
              <button
                onClick={handleConnect}
                disabled={connecting}
                className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
              >
                {connecting ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <ExternalLink className="h-3.5 w-3.5" />
                )}
                Connect
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export function ConnectorsPage() {
  const {
    connectors,
    isLoading,
    error,
    fetchConnectors,
    fetchAllStatuses,
    clearError,
  } = useConnectorStore();

  const [searchQuery, setSearchQuery] = useState("");
  const [activeCategory, setActiveCategory] = useState<string>("all");

  useEffect(() => {
    fetchConnectors();
  }, [fetchConnectors]);

  // Listen for OAuth callback messages from popup window
  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;

      if (
        event.data?.type === "CONNECTOR_OAUTH_SUCCESS" ||
        event.data?.type === "CONNECTOR_OAUTH_ERROR"
      ) {
        fetchAllStatuses();
      }
    };

    window.addEventListener("message", handleMessage);
    return () => window.removeEventListener("message", handleMessage);
  }, [fetchAllStatuses]);

  // Get unique categories
  const categories = [
    "all",
    ...new Set(
      connectors.map(
        (c) => connectorCategories[c.type as ConnectorType] || "Other",
      ),
    ),
  ];

  // Filter connectors
  const filteredConnectors = connectors.filter((connector) => {
    const matchesSearch =
      connector.display_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      connectorDescriptions[connector.type as ConnectorType]
        ?.toLowerCase()
        .includes(searchQuery.toLowerCase());

    const matchesCategory =
      activeCategory === "all" ||
      connectorCategories[connector.type as ConnectorType] === activeCategory;

    return matchesSearch && matchesCategory;
  });

  // Count connected
  const { statuses } = useConnectorStore();
  const connectedCount = connectors.filter(
    (c) => statuses[c.type as ConnectorType]?.connected,
  ).length;

  return (
    <>
      <AppSidebar />
      <SidebarInset>
        <NavHeader showModelSelector={false} />
        <div className="flex-1 overflow-auto">
          <div className="container max-w-screen-xl py-8 px-4 md:px-6">
            {/* Header */}
            <div className="mb-8">
              <div className="flex items-center gap-3 mb-2">
                <div className="p-2 rounded-lg bg-primary/10">
                  <Plug className="h-6 w-6 text-primary" />
                </div>
                <div>
                  <h1 className="text-2xl font-bold">Connectors</h1>
                  <p className="text-sm text-muted-foreground">
                    Connect external services to enhance your AI assistant
                  </p>
                </div>
              </div>
              {connectedCount > 0 && (
                <p className="text-sm text-muted-foreground mt-2">
                  {connectedCount} service{connectedCount !== 1 ? "s" : ""}{" "}
                  connected
                </p>
              )}
            </div>

            {/* Search and Filter */}
            <div className="flex flex-col sm:flex-row gap-4 mb-6">
              <div className="relative flex-1 max-w-md">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <input
                  type="text"
                  placeholder="Search connectors..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full rounded-lg border border-border bg-background pl-10 pr-4 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
                />
              </div>
              <div className="flex gap-2 flex-wrap">
                {categories.map((category) => (
                  <button
                    key={category}
                    onClick={() => setActiveCategory(category)}
                    className={cn(
                      "px-3 py-1.5 rounded-lg text-sm font-medium transition-colors",
                      activeCategory === category
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground",
                    )}
                  >
                    {category === "all"
                      ? "All"
                      : category.charAt(0).toUpperCase() + category.slice(1)}
                  </button>
                ))}
              </div>
            </div>

            {/* Error */}
            {error && (
              <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-4 mb-6">
                <div className="flex items-center justify-between">
                  <p className="text-sm text-red-500">{error}</p>
                  <button
                    onClick={clearError}
                    className="text-red-500 hover:text-red-400"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>
            )}

            {/* Connectors Grid */}
            {isLoading && connectors.length === 0 ? (
              <div className="flex items-center justify-center py-20">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : filteredConnectors.length > 0 ? (
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {filteredConnectors.map((connector) => (
                  <ConnectorCard
                    key={connector.type}
                    type={connector.type as ConnectorType}
                    displayName={connector.display_name}
                    enabled={connector.enabled}
                  />
                ))}
              </div>
            ) : connectors.length > 0 ? (
              <div className="text-center py-20">
                <Search className="h-12 w-12 mx-auto text-muted-foreground/50 mb-4" />
                <h3 className="text-lg font-medium mb-1">No matches found</h3>
                <p className="text-sm text-muted-foreground">
                  Try adjusting your search or filter
                </p>
              </div>
            ) : (
              <div className="text-center py-20 rounded-xl border border-dashed border-border">
                <Plug className="h-12 w-12 mx-auto text-muted-foreground/50 mb-4" />
                <h3 className="text-lg font-medium mb-1">
                  No connectors available
                </h3>
                <p className="text-sm text-muted-foreground">
                  Connectors have not been configured on this server
                </p>
              </div>
            )}

            {/* Info Section */}
            <div className="mt-12 rounded-xl border border-border bg-muted/20 p-6">
              <h3 className="font-semibold text-foreground mb-2">
                About Connectors
              </h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Connectors allow the AI assistant to interact with external
                services on your behalf. When you connect a service, you grant
                the AI permission to read data and, in some cases, perform
                actions like creating issues or managing calendar events. Your
                credentials are encrypted and stored securely. You can
                disconnect a service at any time to revoke these permissions.
              </p>
            </div>
          </div>
        </div>
      </SidebarInset>
    </>
  );
}
