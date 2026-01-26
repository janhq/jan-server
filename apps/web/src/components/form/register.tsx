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
import { useAuth } from "@/stores/auth-store";

declare const JAN_API_BASE_URL: string;

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
  const { loginWithRegisterTokens } = useAuth();

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

      // Call backend registration API
      const response = await fetch(`${JAN_API_BASE_URL}auth/register`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email,
          username,
          password,
          first_name: firstName || undefined,
          last_name: lastName || undefined,
        }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || data.error || "Registration failed");
      }

      // Check if tokens are returned for auto-login
      if (data.access_token && data.refresh_token) {
        // Auto-login with the tokens
        loginWithRegisterTokens({
          access_token: data.access_token,
          refresh_token: data.refresh_token,
          token_type: data.token_type || "Bearer",
          expires_in: data.expires_in || 3600,
        });

        // Clear the modal parameter and redirect to home
        const url = new URL(window.location.href);
        url.searchParams.delete(URL_PARAM.MODAL);
        router.navigate({ to: url.pathname + url.search });

        // Call onSuccess callback if provided
        onSuccess?.();
      } else {
        // Fallback: tokens not available, user needs to login manually
        setSuccess("Account created successfully! You can now sign in.");

        // Clear form
        setEmail("");
        setUsername("");
        setPassword("");
        setConfirmPassword("");
        setFirstName("");
        setLastName("");

        // Switch to login after a short delay
        setTimeout(() => {
          handleSwitchToLogin();
        }, 2000);
      }
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
