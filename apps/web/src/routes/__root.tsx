import {
  createRootRoute,
  Outlet,
  useLocation,
  useRouter,
} from "@tanstack/react-router";
import { useEffect, useRef } from "react";
import { Dialog, DialogContent } from "@janhq/interfaces/dialog";
import { LoginForm } from "@/components/form/login";
import { RegisterForm } from "@/components/form/register";
import { SettingsDialog } from "@/components/settings/settings-dialog";
import { CreateProject } from "@/components/projects/create-project";
import { SearchDialog } from "@/components/search/search-dialog";
import { useAuth } from "@/stores/auth-store";
import { useRightSidebarStore } from "@/stores/right-sidebar-store";
import { ThemeProvider } from "@/components/themes/theme-provider";
import { Toaster } from "@janhq/interfaces/sonner";
import {
  LOCAL_STORAGE_KEY,
  THEME,
  URL_PARAM,
  URL_PARAM_VALUE,
  SETTINGS_SECTION,
} from "@/constants";
import { AccentColorProvider } from "@/providers/accent-color";
import { initAnalytics, analytics } from "@/lib/analytics";

function RootLayout() {
  const location = useLocation();
  const router = useRouter();
  const accessToken = useAuth((state) => state.accessToken);
  const guestLogin = useAuth((state) => state.guestLogin);
  const hasAttemptedGuestLogin = useRef(false);
  const isAuthenticated = useAuth((state) => state.isAuthenticated);
  const isGuest = useAuth((state) => state.isGuest);
  const clearRightSidebar = useRightSidebarStore(
    (state) => state.clearSelection
  );
  const setSidebarOpen = useRightSidebarStore((state) => state.setSidebarOpen);
  const hasInitializedAnalytics = useRef(false);

  // Initialize analytics and track app_opened
  useEffect(() => {
    if (!hasInitializedAnalytics.current) {
      hasInitializedAnalytics.current = true;
      initAnalytics();
      analytics.capture("app_opened", {
        user_status: analytics.getUserStatus(isAuthenticated),
        referrer_source: document.referrer || null,
      });
    }
  }, []);

  // Close right sidebar on route change
  useEffect(() => {
    clearRightSidebar();
    setSidebarOpen(false);
  }, [location.pathname]);

  const isProtectedPath = (path: string) => {
    if (path === "/login" || path === "/") {
      return false;
    }

    const publicPrefixes = ["/auth", "/share", "/docs"];
    if (publicPrefixes.some((prefix) => path.startsWith(prefix))) {
      return false;
    }

    const protectedPrefixes = [
      "/dashboard",
      "/admin",
      "/profile",
      "/projects",
      "/connectors",
      "/artifacts",
    ];

    return protectedPrefixes.some((prefix) => path.startsWith(prefix));
  };

  const getRedirectTarget = () => {
    const url = new URL(window.location.href);
    const redirectParam = url.searchParams.get(URL_PARAM.REDIRECT);
    if (redirectParam && redirectParam.startsWith("/")) {
      return redirectParam;
    }
    return url.pathname + url.search;
  };

  const redirectToLogin = () => {
    const redirectUrl = getRedirectTarget();
    const params = new URLSearchParams();
    params.set(URL_PARAM.REDIRECT, redirectUrl);
    router.navigate({ to: `/login?${params.toString()}` });
  };

  // Auto guest login if no token exists (skip for protected pages)
  useEffect(() => {
    if (
      !accessToken &&
      !hasAttemptedGuestLogin.current &&
      !isProtectedPath(location.pathname) &&
      location.pathname !== "/login"
    ) {
      hasAttemptedGuestLogin.current = true;
      guestLogin().catch((error) => {
        console.error("Auto guest login failed:", error);
        hasAttemptedGuestLogin.current = false; // Allow retry on failure
      });
    }
  }, [accessToken, guestLogin, location.pathname]);

  // Redirect unauthenticated/guest users to login for protected pages
  useEffect(() => {
    if (!isProtectedPath(location.pathname)) {
      return;
    }

    if (!isAuthenticated || isGuest) {
      redirectToLogin();
    }
  }, [isAuthenticated, isGuest, location.pathname]);

  // Check if modals should be shown via search params
  const searchParams = new URLSearchParams(location.search);
  const isLoginRoute = location.pathname === "/login";
  const isLoginModal =
    isLoginRoute || searchParams.get(URL_PARAM.MODAL) === URL_PARAM_VALUE.LOGIN;
  const isRegisterModal =
    searchParams.get(URL_PARAM.MODAL) === URL_PARAM_VALUE.REGISTER;
  const settingSection = searchParams.get(URL_PARAM.SETTING);
  const isSettingsOpen = !!settingSection;
  const projectsSection = searchParams.get(URL_PARAM.PROJECTS);
  const isProjectsOpen = !!projectsSection;
  const searchSection = searchParams.get(URL_PARAM.SEARCH);
  const isSearchOpen = !!searchSection;

  const isAllowedExternalRedirect = (value: string) => {
    // Allow localhost with any port for development
    return /^http:\/\/localhost:\d+/.test(value);
  };

  const handleCloseModal = (redirectUrl?: string) => {
    // Handle external redirect (e.g., http://localhost:19999)
    if (redirectUrl && isAllowedExternalRedirect(redirectUrl)) {
      const target = new URL(redirectUrl);
      target.searchParams.set("token", `Bearer ${accessToken}`);
      window.location.href = target.toString();
      return;
    }

    if (location.pathname === "/login") {
      if (redirectUrl && redirectUrl.startsWith("/")) {
        router.navigate({ to: redirectUrl });
      }
      return;
    }

    if (redirectUrl && redirectUrl.startsWith("/")) {
      router.navigate({ to: redirectUrl });
      return;
    }

    // Remove the modal search param by navigating without it
    const url = new URL(window.location.href);
    url.searchParams.delete(URL_PARAM.MODAL);
    url.searchParams.delete(URL_PARAM.REDIRECT);
    router.navigate({ to: url.pathname + url.search });
  };

  return (
    <ThemeProvider
      defaultTheme={THEME.SYSTEM}
      storageKey={LOCAL_STORAGE_KEY.THEME}
    >
      <AccentColorProvider>
        {/* Main content - always rendered */}
        <Outlet />

        {/* Login Modal */}
        <Dialog
          open={isLoginModal}
          onOpenChange={(open: boolean) => !open && handleCloseModal()}
        >
          <DialogContent>
            <LoginForm onSuccess={handleCloseModal} />
          </DialogContent>
        </Dialog>

        {/* Register Modal */}
        <Dialog
          open={isRegisterModal}
          onOpenChange={(open: boolean) => !open && handleCloseModal()}
        >
          <DialogContent>
            <RegisterForm onSuccess={handleCloseModal} />
          </DialogContent>
        </Dialog>

        {/* Settings Dialog */}
        <SettingsDialog
          open={isSettingsOpen}
          section={settingSection || SETTINGS_SECTION.GENERAL}
        />

        {/* Projects Dialog */}
        <CreateProject open={isProjectsOpen} />

        {/* Search Dialog */}
        <SearchDialog open={isSearchOpen} />

        {/* Toast Notifications */}
        {/* <Toaster position="top-center" /> */}
        <Toaster richColors position="top-center" />
      </AccentColorProvider>
    </ThemeProvider>
  );
}

export const Route = createRootRoute({
  component: RootLayout,
});
