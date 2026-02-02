// Slide viewer - full screen, rendered via portal
import { createPortal } from "react-dom";
import { useState, useEffect } from "react"
import { useNavigate } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import {
  type SlideImage,
} from "@/stores/right-sidebar-store";
import { ChevronLeftIcon, ChevronRightIcon, XIcon, DownloadIcon, ExternalLinkIcon } from "lucide-react";
import { Button } from "@janhq/interfaces/button";

export const SlideViewer = ({
  slides,
  initialIndex = 0,
  title,
  onDownload,
  conversationId,
  onClose,
}: {
  slides: SlideImage[];
  initialIndex?: number;
  title?: string;
  onDownload?: () => void;
  conversationId?: string;
  onClose: () => void;
}) => {
  const navigate = useNavigate();
  const [currentIndex, setCurrentIndex] = useState(initialIndex);

  // Guard against empty slides array
  if (!slides || slides.length === 0) {
    return createPortal(
      <div className="fixed inset-0 z-100 flex items-center justify-center bg-background">
        <div className="text-center">
          <p className="text-muted-foreground mb-4">No slides available</p>
          <Button variant="outline" onClick={onClose}>Close</Button>
        </div>
      </div>,
      document.body
    );
  }

  const currentSlide = slides[currentIndex];
  const hasPrev = currentIndex > 0;
  const hasNext = currentIndex < slides.length - 1;

  const handleDownload = () => {
    if (onDownload) {
      onDownload();
    }
  };

  const handleViewThread = () => {
    if (conversationId) {
      onClose();
      navigate({ to: "/threads/$conversationId", params: { conversationId } });
    }
  };

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.key === "ArrowLeft" || e.key === "ArrowUp") && hasPrev) {
        setCurrentIndex((i) => i - 1);
      } else if ((e.key === "ArrowRight" || e.key === "ArrowDown") && hasNext) {
        setCurrentIndex((i) => i + 1);
      } else if (e.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [hasPrev, hasNext, onClose]);

  // Render via portal to escape any parent constraints
  return createPortal(
    <div className="fixed inset-0 z-100 flex bg-background">
      {/* Left sidebar - thumbnails */}
      <div className="w-56 border-r bg-muted/30 flex flex-col shrink-0">
        <div className="flex-1 overflow-auto p-3 space-y-3">
          {slides.map((slide, index) => (
            <div
              key={slide.id}
              className={cn(
                "relative rounded-lg overflow-hidden cursor-pointer transition-all",
                currentIndex === index
                  ? "ring-2 ring-primary shadow-md"
                  : "hover:ring-1 hover:ring-primary/50 opacity-70 hover:opacity-100"
              )}
              onClick={() => setCurrentIndex(index)}
            >
              <div className="absolute top-1.5 left-1.5 bg-background/90 backdrop-blur-sm px-2 py-0.5 rounded text-xs font-semibold z-10">
                {index + 1}
              </div>
              <img
                src={slide.thumb}
                alt={`Slide ${index + 1}`}
                className="w-full h-auto object-cover"
              />
            </div>
          ))}
        </div>
      </div>

      {/* Right - main content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between border-b px-6 py-3 shrink-0">
          <div>
            <h2 className="font-medium">{title || "Presentation"}</h2>
            <p className="text-xs mt-1 text-muted-foreground">
              Slide {currentIndex + 1} of {slides.length}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => setCurrentIndex(i => i - 1)}
              disabled={!hasPrev}
            >
              <ChevronLeftIcon className="size-4 mr-1" />
            </Button>
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => setCurrentIndex(i => i + 1)}
              disabled={!hasNext}
            >
              <ChevronRightIcon className="size-4 ml-1" />
            </Button>
            {conversationId && (
              <>
                <div className="w-px h-6 bg-border mx-2" />
                <Button variant="outline" size="sm" onClick={handleViewThread}>
                  <ExternalLinkIcon className="size-4 mr-2" />
                  View Chat
                </Button>
              </>
            )}
            {onDownload && (
              <>
                <div className="w-px h-6 bg-border mx-2" />
                <Button variant="outline" size="icon-sm" onClick={handleDownload}>
                  <DownloadIcon className="size-4" />
                </Button>
              </>
            )}
            <div className="w-px h-6 bg-border mx-2" />
            <Button variant="ghost" size="icon-sm" onClick={onClose}>
              <XIcon className="size-4" />
            </Button>
          </div>
        </div>

        {/* Main slide view */}
        <div className="flex-1 flex items-center justify-center bg-muted/10 p-8 overflow-auto">
          <img
            src={currentSlide.thumb}
            alt={`Slide ${currentIndex + 1}`}
            className="max-w-full max-h-full w-auto h-auto rounded-xl shadow-2xl object-contain"
          />
        </div>
      </div>
    </div>,
    document.body
  );
};