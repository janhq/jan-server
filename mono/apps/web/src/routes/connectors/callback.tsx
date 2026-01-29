import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Check, X, Loader2 } from "lucide-react";

export const Route = createFileRoute("/connectors/callback")({
  component: ConnectorCallbackPage,
});

type ConnectorType = "github" | "gmail" | "google_drive" | "google_calendar";

const connectorNames: Record<ConnectorType, string> = {
  github: "GitHub",
  gmail: "Gmail",
  google_drive: "Google Drive",
  google_calendar: "Google Calendar",
};

function ConnectorCallbackPage() {
  const [status, setStatus] = useState<"loading" | "success" | "error">(
    "loading",
  );
  const [message, setMessage] = useState<string>("");

  useEffect(() => {
    // Parse URL parameters
    const params = new URLSearchParams(window.location.search);
    const statusParam = params.get("status");
    const connectorParam = params.get("connector") as ConnectorType | null;
    const messageParam = params.get("message");

    if (statusParam === "success") {
      setStatus("success");
      setMessage(
        `Successfully connected to ${connectorParam ? connectorNames[connectorParam] : "the service"}!`,
      );

      // If opened in a popup, notify the parent window and close
      if (window.opener) {
        try {
          window.opener.postMessage(
            {
              type: "CONNECTOR_OAUTH_SUCCESS",
              connector: connectorParam,
            },
            window.location.origin,
          );
        } catch {
          // Ignore cross-origin errors
        }

        // Close the popup after a short delay
        setTimeout(() => {
          window.close();
        }, 2000);
      }
    } else if (statusParam === "error") {
      setStatus("error");
      setMessage(
        messageParam ||
          `Failed to connect to ${connectorParam ? connectorNames[connectorParam] : "the service"}`,
      );

      // If opened in a popup, notify the parent window
      if (window.opener) {
        try {
          window.opener.postMessage(
            {
              type: "CONNECTOR_OAUTH_ERROR",
              connector: connectorParam,
              message: messageParam,
            },
            window.location.origin,
          );
        } catch {
          // Ignore cross-origin errors
        }

        // Close the popup after a short delay
        setTimeout(() => {
          window.close();
        }, 3000);
      }
    } else {
      // Unknown status
      setStatus("error");
      setMessage("Unknown callback status");
    }
  }, []);

  const handleClose = () => {
    if (window.opener) {
      window.close();
    } else {
      // Navigate to profile page if not a popup
      window.location.href = "/profile";
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-md rounded-lg border border-border bg-card p-8 text-center shadow-lg">
        {status === "loading" && (
          <>
            <Loader2 className="mx-auto h-12 w-12 animate-spin text-primary" />
            <h1 className="mt-4 text-xl font-semibold text-foreground">
              Processing...
            </h1>
            <p className="mt-2 text-sm text-muted-foreground">
              Please wait while we complete the connection.
            </p>
          </>
        )}

        {status === "success" && (
          <>
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-green-500/10">
              <Check className="h-8 w-8 text-green-500" />
            </div>
            <h1 className="mt-4 text-xl font-semibold text-foreground">
              Connected!
            </h1>
            <p className="mt-2 text-sm text-muted-foreground">{message}</p>
            <p className="mt-4 text-xs text-muted-foreground">
              {window.opener
                ? "This window will close automatically..."
                : "You can now return to your profile."}
            </p>
            {!window.opener && (
              <button
                onClick={handleClose}
                className="mt-4 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
              >
                Go to Profile
              </button>
            )}
          </>
        )}

        {status === "error" && (
          <>
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-red-500/10">
              <X className="h-8 w-8 text-red-500" />
            </div>
            <h1 className="mt-4 text-xl font-semibold text-foreground">
              Connection Failed
            </h1>
            <p className="mt-2 text-sm text-red-500">{message}</p>
            <p className="mt-4 text-xs text-muted-foreground">
              {window.opener
                ? "This window will close automatically..."
                : "Please try again from your profile."}
            </p>
            {!window.opener && (
              <button
                onClick={handleClose}
                className="mt-4 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
              >
                Go to Profile
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}
