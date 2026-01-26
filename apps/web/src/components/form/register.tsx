import { cn } from "@/lib/utils";
import { Button } from "@janhq/interfaces/button";
import {
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@janhq/interfaces/dialog";
import { Google } from "@janhq/interfaces/svgs/google";
import { Mail } from "lucide-react";
import { buildGoogleAuthUrl, buildRegistrationAuthUrl } from "@/lib/oauth";
import { useState } from "react";
import { useRouter } from "@tanstack/react-router";
import { URL_PARAM, URL_PARAM_VALUE } from "@/constants";

export function RegisterForm({
  className,
  onSuccess,
  ...props
}: React.ComponentProps<"div"> & { onSuccess?: () => void }) {
  const [isGoogleLoading, setIsGoogleLoading] = useState(false);
  const [isEmailLoading, setIsEmailLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const router = useRouter();

  const handleGoogleRegister = async () => {
    try {
      setIsGoogleLoading(true);
      setError(null);

      // Store the current URL to redirect back after OAuth
      const currentUrl = window.location.pathname + window.location.search;

      // Build Keycloak authorization URL with Google IdP
      const authUrl = await buildGoogleAuthUrl(currentUrl);
      // Redirect to Keycloak for Google OAuth
      window.location.href = authUrl;
    } catch (error) {
      console.error("Google register error:", error);
      setError("Failed to initiate Google registration. Please try again.");
      setIsGoogleLoading(false);
    }
  };

  const handleEmailRegister = async () => {
    try {
      setIsEmailLoading(true);
      setError(null);

      // Store the current URL to redirect back after OAuth
      const currentUrl = window.location.pathname + window.location.search;

      // Build Keycloak authorization URL with proper PKCE flow
      // This redirects to Keycloak where users can register
      const authUrl = await buildRegistrationAuthUrl(currentUrl);

      // Redirect to Keycloak for registration
      window.location.href = authUrl;
    } catch (error) {
      console.error("Registration error:", error);
      setError(
        error instanceof Error ? error.message : "Registration failed. Please try again."
      );
      setIsEmailLoading(false);
    }
  };

  const handleSwitchToLogin = () => {
    const url = new URL(window.location.href);
    url.searchParams.set(URL_PARAM.MODAL, URL_PARAM_VALUE.LOGIN);
    router.navigate({ to: url.pathname + url.search });
  };

  const isLoading = isGoogleLoading || isEmailLoading;

  return (
    <div className={cn("flex flex-col gap-4", className)} {...props}>
      <DialogHeader className="mb-2 text-left">
        <DialogTitle>Create an account</DialogTitle>
        <DialogDescription>
          Choose how you'd like to register
        </DialogDescription>
      </DialogHeader>

      {error && (
        <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <div className="flex flex-col gap-3">
        <Button
          variant="outline"
          type="button"
          onClick={handleGoogleRegister}
          disabled={isLoading}
        >
          <Google className="size-4" />
          {isGoogleLoading ? "Redirecting..." : "Continue with Google"}
        </Button>

        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <span className="w-full border-t" />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background px-2 text-muted-foreground">or</span>
          </div>
        </div>

        <Button
          type="button"
          onClick={handleEmailRegister}
          disabled={isLoading}
        >
          <Mail className="size-4" />
          {isEmailLoading ? "Redirecting..." : "Register with Email"}
        </Button>
      </div>

      <div className="text-center text-sm text-muted-foreground">
        Already have an account?{" "}
        <button
          type="button"
          onClick={handleSwitchToLogin}
          className="text-primary hover:underline font-medium"
        >
          Sign in
        </button>
      </div>
    </div>
  );
}
