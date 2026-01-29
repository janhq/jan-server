// Image viewer - full screen, rendered via portal
import { createPortal } from "react-dom";
import { useEffect } from "react";
import { useNavigate } from "@tanstack/react-router";
import { XIcon, DownloadIcon, ExternalLinkIcon, Loader2 } from "lucide-react";
import { Button } from "@janhq/interfaces/button";

export const ImageViewer = ({
  src,
  title,
  onDownload,
  conversationId,
  isLoading,
  onClose,
}: {
  src: string;
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

  // Render via portal to escape any parent constraints
  return createPortal(
    <div className="fixed inset-0 z-100 flex items-center justify-center bg-black/80 backdrop-blur-sm">
      {/* Backdrop click to close */}
      <div className="absolute inset-0" onClick={onClose} />

      {/* Header - floating */}
      <div className="absolute top-4 right-4 z-10 flex items-center gap-2 bg-background/90 backdrop-blur-sm rounded-lg px-3 py-2 border shadow-lg">
        {conversationId && (
          <Button variant="ghost" size="sm" onClick={handleViewThread}>
            <ExternalLinkIcon className="size-4 mr-2" />
            View Chat
          </Button>
        )}
        {onDownload && (
          <Button variant="ghost" size="icon-sm" onClick={handleDownload}>
            <DownloadIcon className="size-4" />
          </Button>
        )}
        <div className="w-px h-6 bg-border" />
        <Button variant="ghost" size="icon-sm" onClick={onClose}>
          <XIcon className="size-4" />
        </Button>
      </div>

      {/* Title - floating */}
      {title && (
        <div className="absolute top-4 left-4 z-10 bg-background/90 backdrop-blur-sm rounded-lg px-3 py-2 border shadow-lg">
          <h2 className="font-medium text-sm">{title}</h2>
        </div>
      )}

      {/* Main content */}
      <div className="relative max-w-[90%] max-h-[90%] flex items-center justify-center">
        {isLoading ? (
          <div className="flex items-center justify-center">
            <Loader2 className="size-8 animate-spin text-white" />
          </div>
        ) : (
          <img
            src={src}
            alt={title || "Image"}
            className="max-w-full max-h-[85vh] object-contain rounded-lg shadow-2xl"
          />
        )}
      </div>
    </div>,
    document.body
  );
};
