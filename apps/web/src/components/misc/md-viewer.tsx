// Markdown viewer - full screen, rendered via portal
import { createPortal } from "react-dom";
import { useEffect } from "react";
import { XIcon, DownloadIcon } from "lucide-react";
import { Button } from "@janhq/interfaces/button";
import { MessageResponse } from "@janhq/interfaces/ai-elements/message";

export const MdViewer = ({
  content,
  title,
  downloadUrl,
  onClose,
}: {
  content: string;
  title?: string;
  downloadUrl?: string;
  onClose: () => void;
}) => {
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
    if (downloadUrl) {
      window.open(downloadUrl, "_blank");
    }
  };

  // Render via portal to escape any parent constraints
  return createPortal(
    <div className="fixed inset-0 z-100 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      {/* Backdrop click to close */}
      <div className="absolute inset-0" onClick={onClose} />

      {/* Modal container */}
      <div className="relative w-[90%] max-w-4xl h-[80%] bg-background rounded-xl border shadow-2xl flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between border-b px-4 py-3 shrink-0">
          <div>
            <h2 className="font-medium">{title || "Research Report"}</h2>
          </div>
          <div className="flex items-center gap-2">
            {downloadUrl && (
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
        <div className="flex-1 overflow-auto">
          <div className="max-w-3xl mx-auto p-6">
            <div className="prose prose-sm dark:prose-invert max-w-none [&>*:first-child]:mt-0 [&>*:last-child]:mb-0">
              <MessageResponse>{content}</MessageResponse>
            </div>
          </div>
        </div>
      </div>
    </div>,
    document.body
  );
};
