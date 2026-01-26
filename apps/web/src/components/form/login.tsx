import { cn } from "@/lib/utils";
import { Button } from "@janhq/interfaces/button";
import {
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@janhq/interfaces/dialog";
import { Input } from "@janhq/interfaces/input";
import { Google } from "@janhq/interfaces/svgs/google";
import { buildGoogleAuthUrl } from "@/lib/oauth";
import { useState } from "react";
import { useAuth } from "@/stores/auth-store";
import { useRouter } from "@tanstack/react-router";
import { URL_PARAM, URL_PARAM_VALUE } from "@/constants";

declare const VITE_AUTH_URL: string;
declare const VITE_AUTH_REALM: string;
declare const VITE_AUTH_CLIENT_ID: string;

export function LoginForm({
  className,
  onSuccess,
  ...props
}: React.ComponentProps<"div"> & { onSuccess?: () => void }) {
  const [isGoogleLoading, setIsGoogleLoading] = useState(false);
  const [isPasswordLoading, setIsPasswordLoading] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const { loginWithOAuth } = useAuth();
  const router = useRouter();

  const handleGoogleLogin = async () => {
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
      console.error("Google login error:", error);
      setError("Failed to initiate Google login. Please try again.");
      setIsGoogleLoading(false);
    }
  };

  const handlePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!email || !password) {
      setError("Please enter both email and password");
      return;
    }

    try {
      setIsPasswordLoading(true);

      // Call Keycloak token endpoint with password grant
      const tokenEndpoint = `${VITE_AUTH_URL}/realms/${VITE_AUTH_REALM}/protocol/openid-connect/token`;

      const params = new URLSearchParams({
        grant_type: "password",
        client_id: VITE_AUTH_CLIENT_ID,
        username: email,
        password: password,
      });

      const response = await fetch(tokenEndpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
        body: params.toString(),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        if (errorData.error === "invalid_grant") {
          throw new Error("Invalid email or password");
        }
        throw new Error(errorData.error_description || "Login failed");
      }

      const tokens: OAuthTokenResponse = await response.json();
      loginWithOAuth(tokens);
      onSuccess?.();
    } catch (error) {
      console.error("Password login error:", error);
      setError(
        error instanceof Error ? error.message : "Login failed. Please try again."
      );
    } finally {
      setIsPasswordLoading(false);
    }
  };

  const isLoading = isGoogleLoading || isPasswordLoading;

  return (
    <div className={cn("flex flex-col gap-4", className)} {...props}>
      <DialogHeader className="mb-2 text-left">
        <DialogTitle>Login to your account</DialogTitle>
        <DialogDescription>
          Sign in with your credentials or Google account
        </DialogDescription>
      </DialogHeader>

      {error && (
        <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <form onSubmit={handlePasswordLogin} className="flex flex-col gap-3">
        <div className="flex flex-col gap-2">
          <Input
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            disabled={isLoading}
            autoComplete="email"
          />
          <Input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={isLoading}
            autoComplete="current-password"
          />
        </div>
        <Button type="submit" disabled={isLoading}>
          {isPasswordLoading ? "Signing in..." : "Sign in"}
        </Button>
      </form>

      <div className="relative">
        <div className="absolute inset-0 flex items-center">
          <span className="w-full border-t" />
        </div>
        <div className="relative flex justify-center text-xs uppercase">
          <span className="bg-background px-2 text-muted-foreground">or</span>
        </div>
      </div>

      <Button
        variant="outline"
        type="button"
        onClick={handleGoogleLogin}
        disabled={isLoading}
      >
        <Google className="size-4" />
        {isGoogleLoading ? "Redirecting..." : "Continue with Google"}
      </Button>

      <div className="text-center text-sm text-muted-foreground">
        Don't have an account?{" "}
        <button
          type="button"
          onClick={() => {
            const url = new URL(window.location.href);
            url.searchParams.set(URL_PARAM.MODAL, URL_PARAM_VALUE.REGISTER);
            router.navigate({ to: url.pathname + url.search });
          }}
          className="text-primary hover:underline font-medium"
        >
          Register
        </button>
      </div>
    </div>
  );
}
