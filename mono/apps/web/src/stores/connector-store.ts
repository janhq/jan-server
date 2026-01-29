import { create } from "zustand";
import {
  connectorService,
  type ConnectorType,
  type ConnectorInfo,
  type ConnectorStatus,
} from "@/services/connector-service";

interface ConnectorState {
  // State
  connectors: ConnectorInfo[];
  statuses: Record<ConnectorType, ConnectorStatus | null>;
  isLoading: boolean;
  isConnecting: Record<ConnectorType, boolean>;
  error: string | null;

  // Actions
  fetchConnectors: () => Promise<void>;
  fetchStatus: (type: ConnectorType) => Promise<void>;
  fetchAllStatuses: () => Promise<void>;
  initiateConnect: (type: ConnectorType) => Promise<void>;
  completeConnect: (
    type: ConnectorType,
    code: string,
    state: string,
  ) => Promise<void>;
  disconnect: (type: ConnectorType) => Promise<void>;
  refreshTokens: (type: ConnectorType) => Promise<void>;
  clearError: () => void;
}

let fetchConnectorsPromise: Promise<void> | null = null;

export const useConnectorStore = create<ConnectorState>((set, get) => ({
  connectors: [],
  statuses: {
    github: null,
    gmail: null,
    google_drive: null,
    google_calendar: null,
  },
  isLoading: false,
  isConnecting: {
    github: false,
    gmail: false,
    google_drive: false,
    google_calendar: false,
  },
  error: null,

  fetchConnectors: async () => {
    // Deduplicate concurrent fetches
    if (fetchConnectorsPromise) {
      return fetchConnectorsPromise;
    }

    fetchConnectorsPromise = (async () => {
      set({ isLoading: true, error: null });
      try {
        const response = await connectorService.listConnectors();
        set({ connectors: response.connectors, isLoading: false });
      } catch (error) {
        set({
          error:
            error instanceof Error
              ? error.message
              : "Failed to fetch connectors",
          isLoading: false,
        });
      } finally {
        fetchConnectorsPromise = null;
      }
    })();

    return fetchConnectorsPromise;
  },

  fetchStatus: async (type: ConnectorType) => {
    try {
      const status = await connectorService.getStatus(type);
      set((state) => ({
        statuses: {
          ...state.statuses,
          [type]: status,
        },
      }));
    } catch (error) {
      // If connector is not connected, set status as disconnected
      set((state) => ({
        statuses: {
          ...state.statuses,
          [type]: {
            connected: false,
            enabled: true,
            status: "disconnected",
          },
        },
      }));
    }
  },

  fetchAllStatuses: async () => {
    const { connectors } = get();
    const enabledConnectors = connectors.filter((c) => c.enabled);

    await Promise.all(
      enabledConnectors.map((connector) =>
        get().fetchStatus(connector.type as ConnectorType),
      ),
    );
  },

  initiateConnect: async (type: ConnectorType) => {
    set((state) => ({
      isConnecting: { ...state.isConnecting, [type]: true },
      error: null,
    }));

    try {
      const response = await connectorService.getAuthURL(type);
      // Open OAuth window
      window.open(response.auth_url, "_blank", "width=600,height=700");
    } catch (error) {
      set((state) => ({
        isConnecting: { ...state.isConnecting, [type]: false },
        error:
          error instanceof Error ? error.message : "Failed to start connection",
      }));
    }
  },

  completeConnect: async (type: ConnectorType, code: string, state: string) => {
    set((state) => ({
      isConnecting: { ...state.isConnecting, [type]: true },
      error: null,
    }));

    try {
      await connectorService.connect(type, code, state);
      // Refresh status after successful connection
      await get().fetchStatus(type);
      set((state) => ({
        isConnecting: { ...state.isConnecting, [type]: false },
      }));
    } catch (error) {
      set((state) => ({
        isConnecting: { ...state.isConnecting, [type]: false },
        error:
          error instanceof Error
            ? error.message
            : "Failed to complete connection",
      }));
      throw error;
    }
  },

  disconnect: async (type: ConnectorType) => {
    set({ error: null });
    try {
      await connectorService.disconnect(type);
      // Update status to disconnected
      set((state) => ({
        statuses: {
          ...state.statuses,
          [type]: {
            connected: false,
            enabled: true,
            status: "disconnected",
          },
        },
      }));
    } catch (error) {
      set({
        error:
          error instanceof Error ? error.message : "Failed to disconnect",
      });
      throw error;
    }
  },

  refreshTokens: async (type: ConnectorType) => {
    try {
      await connectorService.refreshTokens(type);
      // Refresh status after token refresh
      await get().fetchStatus(type);
    } catch (error) {
      set({
        error:
          error instanceof Error ? error.message : "Failed to refresh tokens",
      });
      throw error;
    }
  },

  clearError: () => {
    set({ error: null });
  },
}));
