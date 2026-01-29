// HTML viewer - full screen, rendered via portal with iframe
import { createPortal } from "react-dom";
import { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { XIcon, DownloadIcon, ExternalLinkIcon, Loader2 } from "lucide-react";
import { Button } from "@janhq/interfaces/button";

export const HtmlViewer = ({
  content,
  title,
  onDownload,
  conversationId,
  isLoading,
  onClose,
}: {
  content: string;
  title?: string;
  onDownload?: () => void;
  conversationId?: string;
  isLoading?: boolean;
  onClose: () => void;
}) => {
  const navigate = useNavigate();

  const handleViewThread = () => {
    if (conversationId) {
      onClose();
      navigate({ to: "/threads/$conversationId", params: { conversationId } });
    }
  };

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  const handleDownload = () => {
    if (onDownload) {
      onDownload();
    }
  };

  // Create and manage blob URL for the HTML content
  const [htmlBlobUrl, setHtmlBlobUrl] = useState<string | null>(null);

  useEffect(() => {
    if (!content) {
      setHtmlBlobUrl(null);
      return;
    }
    // Create new blob URL
    const url = URL.createObjectURL(new Blob([content], { type: "text/html" }));
    setHtmlBlobUrl(url);

    // Cleanup on content change or unmount
    return () => {
      URL.revokeObjectURL(url);
    };
  }, [content]);

  // Render via portal to escape any parent constraints
  return createPortal(
    <div className="fixed inset-0 z-100 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      {/* Backdrop click to close */}
      <div className="absolute inset-0" onClick={onClose} />

      {/* Modal container */}
      <div className="relative w-[95%] max-w-6xl h-[90%] bg-background rounded-xl border shadow-2xl flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between border-b px-4 py-3 shrink-0">
          <div>
            <h2 className="font-medium">{title || "HTML Preview"}</h2>
          </div>
          <div className="flex items-center gap-2">
            {conversationId && (
              <Button variant="outline" size="sm" onClick={handleViewThread}>
                <ExternalLinkIcon className="size-4 mr-2" />
                View Chat
              </Button>
            )}
            {onDownload && (
              <Button variant="outline" size="icon-sm" onClick={handleDownload}>
                <DownloadIcon className="size-4" />
              </Button>
            )}
            <div className="w-px h-6 bg-border mx-2" />
            <Button variant="ghost" size="icon-sm" onClick={onClose}>
              <XIcon className="size-4" />
            </Button>
          </div>
        </div>

        {/* Main content */}
        <div className="flex-1 overflow-hidden bg-white">
          {isLoading ? (
            <div className="flex items-center justify-center h-full bg-background">
              <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
          ) : htmlBlobUrl ? (
            <iframe
              src={htmlBlobUrl}
              title={title || "HTML Preview"}
              className="w-full h-full border-0"
              sandbox="allow-scripts allow-same-origin"
            />
          ) : (
            <div className="flex items-center justify-center h-full bg-background">
              <p className="text-muted-foreground">No content to display</p>
            </div>
          )}
        </div>
      </div>
    </div>,
    document.body
  );
};
