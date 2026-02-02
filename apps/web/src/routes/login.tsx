import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { useAuth } from "@/stores/auth-store";
import { URL_PARAM } from "@/constants";

export const Route = createFileRoute("/login" as "/")({
  component: LoginRoute,
});

function LoginRoute() {
  const navigate = useNavigate();
  const isAuthenticated = useAuth((state) => state.isAuthenticated);
  const isGuest = useAuth((state) => state.isGuest);
  const accessToken = useAuth((state) => state.accessToken);

  const isAllowedExternalRedirect = (value: string) => {
    // Allow localhost with any port for development
    return /^http:\/\/localhost:\d+/.test(value);
  };

  useEffect(() => {
    if (!isAuthenticated || isGuest) {
      return;
    }

    const url = new URL(window.location.href);
    const redirectUrl = url.searchParams.get(URL_PARAM.REDIRECT);

    // Handle external redirect (e.g., http://localhost:19999)
    if (redirectUrl && isAllowedExternalRedirect(redirectUrl)) {
      if (!accessToken) {
        return;
      }
      const bearerToken = `Bearer ${accessToken}`;
      const encodedToken = btoa(bearerToken);
      const target = new URL(redirectUrl);
      target.searchParams.set("token", encodedToken);
      window.location.href = target.toString();
      return;
    }

    // Handle internal redirect (e.g., /dashboard)
    if (redirectUrl && redirectUrl.startsWith("/")) {
      navigate({ to: redirectUrl });
      return;
    }

    navigate({ to: "/" });
  }, [accessToken, isAuthenticated, isGuest, navigate]);

  return null;
}
