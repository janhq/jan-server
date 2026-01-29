import { create } from "zustand";
import { persist } from "zustand/middleware";
import { extractUserFromTokens, decodeJWT } from "@/lib/oauth";
import { fetchJsonWithAuth } from "@/lib/api-client";
import { analytics } from "@/lib/analytics";

declare const JAN_API_BASE_URL: string;

interface RegisterTokens {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isGuest: boolean;
  accessToken: string | null;
  refreshToken: string | null;
  login: (user: User) => void;
  loginWithOAuth: (tokens: OAuthTokenResponse) => void;
  loginWithRegisterTokens: (tokens: RegisterTokens) => void;
  logout: () => Promise<void>;
  guestLogin: () => Promise<void>;
  refreshAccessToken: () => Promise<void>;
}

export const useAuth = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      isGuest: false,
      accessToken: null,
      refreshToken: null,
      login: (user) => set({ user, isAuthenticated: true, isGuest: false }),
      loginWithOAuth: (tokens) => {
        const userData = extractUserFromTokens(tokens);
        set({
          user: {
            id: userData.id,
            name: userData.name,
            email: userData.email,
            avatar: userData.avatar,
            pro: userData.pro,
          },
          isAuthenticated: true,
          isGuest: false,
          accessToken: userData.accessToken,
          refreshToken: userData.refreshToken,
        });

        analytics.identify(userData.id);
        analytics.capture("user_logged_in", {
          method: "google",
          is_new_user: false,
        });
      },
      loginWithRegisterTokens: (tokens) => {
        // Decode user info from access token (same as OAuth but without id_token)
        const claims = decodeJWT(tokens.access_token);
        const userData = {
          id: (claims.sub as string) || "",
          name:
            (claims.name as string) ||
            (claims.preferred_username as string) ||
            "",
          email: (claims.email as string) || "",
          avatar: (claims.picture as string) || undefined,
          pro: false,
        };

        set({
          user: userData,
          isAuthenticated: true,
          isGuest: false,
          accessToken: tokens.access_token,
          refreshToken: tokens.refresh_token,
        });

        analytics.identify(userData.id);
        analytics.capture("user_logged_in", {
          method: "email",
          is_new_user: true,
        });
      },
      logout: async () => {
        try {
          const refreshToken = useAuth.getState().refreshToken;
          await fetchJsonWithAuth<{ status: string }>(
            `${JAN_API_BASE_URL}auth/logout`,
            {
              method: "POST",
              body: JSON.stringify({ refresh_token: refreshToken }),
            },
          );

          analytics.capture("user_logged_out");
          analytics.reset();

          // Clear all stores
          const { useProjects } = await import("@/stores/projects-store");
          const { useConversations } =
            await import("@/stores/conversation-store");
          const { useAdminStore } = await import("@/stores/admin-store");
          useProjects.getState().clearProjects();
          useConversations.getState().clearConversations();
          useAdminStore.getState().clearAdminStatus();
          set({
            user: null,
            isAuthenticated: false,
            isGuest: false,
            accessToken: null,
            refreshToken: null,
          });
        } catch (error) {
          console.error("Logout error:", error);
        }
      },
      guestLogin: async () => {
        try {
          const response = await fetch(`${JAN_API_BASE_URL}auth/guest-login`, {
            method: "POST",
            credentials: "include",
            headers: {
              "Content-Type": "application/json",
            },
          });

          if (!response.ok) {
            throw new Error(
              `Guest login failed: ${response.status} ${response.statusText}`,
            );
          }
          const data: GuestLoginResponse = await response.json();
          const guestUser: User = {
            id: data.user_id,
            name: data.username,
            email: "",
          };

          set({
            user: guestUser,
            isAuthenticated: true,
            isGuest: true,
            accessToken: data.access_token,
            refreshToken: data.refresh_token,
          });
        } catch (error) {
          console.error("Guest login error:", error);
          throw error;
        }
      },
      refreshAccessToken: async () => {
        try {
          const response = await fetch(
            `${JAN_API_BASE_URL}auth/refresh-token`,
            {
              method: "POST",
              credentials: "include",
              headers: {
                "Content-Type": "application/json",
              },
              body: JSON.stringify({
                refresh_token: useAuth.getState().refreshToken,
              }),
            },
          );

          if (!response.ok) {
            throw new Error(
              `Token refresh failed: ${response.status} ${response.statusText}`,
            );
          }

          const data: RefreshTokenResponse = await response.json();

          set({
            accessToken: data.access_token,
            refreshToken: data.refresh_token,
          });
        } catch (error) {
          console.error("Token refresh error:", error);
          set({
            user: null,
            isAuthenticated: false,
            isGuest: false,
            accessToken: null,
            refreshToken: null,
          });
          throw error;
        }
      },
    }),
    {
      name: "auth-storage",
    },
  ),
);
