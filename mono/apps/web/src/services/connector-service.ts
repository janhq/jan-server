import { fetchJsonWithAuth, fetchWithAuth } from "@/lib/api-client";

declare const JAN_API_BASE_URL: string;

export type ConnectorType =
  | "github"
  | "gmail"
  | "google_drive"
  | "google_calendar";

export interface ConnectorInfo {
  type: ConnectorType;
  display_name: string;
  description: string;
  enabled: boolean;
  icon_url?: string;
}

export interface ConnectorStatus {
  connected: boolean;
  enabled: boolean;
  status: string;
  username?: string;
  email?: string;
  connected_at?: string;
  expires_at?: string;
}

export interface ConnectorListResponse {
  connectors: ConnectorInfo[];
}

export interface AuthURLResponse {
  auth_url: string;
}

export interface ConnectRequest {
  code: string;
  state: string;
}

export interface ConnectResponse {
  message: string;
  connector_type: ConnectorType;
  username?: string;
  email?: string;
}

export const connectorService = {
  /**
   * List all available connectors
   */
  listConnectors: async (): Promise<ConnectorListResponse> => {
    return fetchJsonWithAuth<ConnectorListResponse>(
      `${JAN_API_BASE_URL}v1/connectors`,
    );
  },

  /**
   * Get connector info by type
   */
  getConnector: async (type: ConnectorType): Promise<ConnectorInfo> => {
    return fetchJsonWithAuth<ConnectorInfo>(
      `${JAN_API_BASE_URL}v1/connectors/${type}`,
    );
  },

  /**
   * Get connector connection status
   */
  getStatus: async (type: ConnectorType): Promise<ConnectorStatus> => {
    return fetchJsonWithAuth<ConnectorStatus>(
      `${JAN_API_BASE_URL}v1/connectors/${type}/status`,
    );
  },

  /**
   * Get OAuth authorization URL for a connector
   */
  getAuthURL: async (type: ConnectorType): Promise<AuthURLResponse> => {
    return fetchJsonWithAuth<AuthURLResponse>(
      `${JAN_API_BASE_URL}v1/connectors/${type}/auth-url`,
    );
  },

  /**
   * Complete OAuth connection with authorization code
   */
  connect: async (
    type: ConnectorType,
    code: string,
    state: string,
  ): Promise<ConnectResponse> => {
    return fetchJsonWithAuth<ConnectResponse>(
      `${JAN_API_BASE_URL}v1/connectors/${type}/connect`,
      {
        method: "POST",
        body: JSON.stringify({ code, state }),
      },
    );
  },

  /**
   * Disconnect a connector
   */
  disconnect: async (type: ConnectorType): Promise<void> => {
    await fetchWithAuth(`${JAN_API_BASE_URL}v1/connectors/${type}/disconnect`, {
      method: "DELETE",
    });
  },

  /**
   * Refresh connector tokens
   */
  refreshTokens: async (type: ConnectorType): Promise<void> => {
    await fetchWithAuth(`${JAN_API_BASE_URL}v1/connectors/${type}/refresh`, {
      method: "POST",
    });
  },
};
