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
import { useRouter } from "@tanstack/react-router";
import { URL_PARAM, URL_PARAM_VALUE } from "@/constants";

declare const VITE_AUTH_URL: string;
declare const VITE_AUTH_REALM: string;
declare const VITE_AUTH_CLIENT_ID: string;

export function RegisterForm({
  className,
  onSuccess,
  ...props
}: React.ComponentProps<"div"> & { onSuccess?: () => void }) {
  const [isGoogleLoading, setIsGoogleLoading] = useState(false);
  const [isRegisterLoading, setIsRegisterLoading] = useState(false);
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
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

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    // Validation
    if (!email || !username || !password || !confirmPassword) {
      setError("Please fill in all required fields");
      return;
    }

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    if (password.length < 8) {
      setError("Password must be at least 8 characters long");
      return;
    }

    try {
      setIsRegisterLoading(true);

      // Call Keycloak registration endpoint
      const registrationEndpoint = `${VITE_AUTH_URL}/realms/${VITE_AUTH_REALM}/protocol/openid-connect/registrations`;

      // For Keycloak, we need to redirect to the registration page
      // or use the admin API if self-registration is disabled
      // Let's try using the registration action URL
      const params = new URLSearchParams({
        client_id: VITE_AUTH_CLIENT_ID,
        redirect_uri: window.location.origin + "/auth/callback",
        response_type: "code",
        scope: "openid profile email",
        kc_action: "REGISTER",
      });

      // Try direct registration if Keycloak supports it
      // Otherwise redirect to Keycloak registration page
      const registerUrl = `${VITE_AUTH_URL}/realms/${VITE_AUTH_REALM}/protocol/openid-connect/auth?${params.toString()}`;

      window.location.href = registerUrl;
    } catch (error) {
      console.error("Registration error:", error);
      setError(
        error instanceof Error ? error.message : "Registration failed. Please try again."
      );
    } finally {
      setIsRegisterLoading(false);
    }
  };

  const handleSwitchToLogin = () => {
    const url = new URL(window.location.href);
    url.searchParams.set(URL_PARAM.MODAL, URL_PARAM_VALUE.LOGIN);
    router.navigate({ to: url.pathname + url.search });
  };

  const isLoading = isGoogleLoading || isRegisterLoading;

  return (
    <div className={cn("flex flex-col gap-4", className)} {...props}>
      <DialogHeader className="mb-2 text-left">
        <DialogTitle>Create an account</DialogTitle>
        <DialogDescription>
          Register with your email or Google account
        </DialogDescription>
      </DialogHeader>

      {error && (
        <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {success && (
        <div className="rounded-md bg-green-500/10 p-3 text-sm text-green-600">
          {success}
        </div>
      )}

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
          <span className="bg-background px-2 text-muted-foreground">or register with email</span>
        </div>
      </div>

      <form onSubmit={handleRegister} className="flex flex-col gap-3">
        <div className="grid grid-cols-2 gap-2">
          <Input
            type="text"
            placeholder="First name"
            value={firstName}
            onChange={(e) => setFirstName(e.target.value)}
            disabled={isLoading}
            autoComplete="given-name"
          />
          <Input
            type="text"
            placeholder="Last name"
            value={lastName}
            onChange={(e) => setLastName(e.target.value)}
            disabled={isLoading}
            autoComplete="family-name"
          />
        </div>
        <Input
          type="text"
          placeholder="Username *"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          disabled={isLoading}
          autoComplete="username"
          required
        />
        <Input
          type="email"
          placeholder="Email *"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          disabled={isLoading}
          autoComplete="email"
          required
        />
        <Input
          type="password"
          placeholder="Password *"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          disabled={isLoading}
          autoComplete="new-password"
          required
        />
        <Input
          type="password"
          placeholder="Confirm password *"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          disabled={isLoading}
          autoComplete="new-password"
          required
        />
        <Button type="submit" disabled={isLoading}>
          {isRegisterLoading ? "Creating account..." : "Create account"}
        </Button>
      </form>

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
