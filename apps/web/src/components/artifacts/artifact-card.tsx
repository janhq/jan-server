import { useState, useEffect, useMemo } from "react";
import { artifactService, type ArtifactResponse } from "@/services/artifact-service";
import { useArtifactGalleryStore } from "@/stores/artifact-gallery-store";
import { useNavigate } from "@tanstack/react-router";
import { Button } from "@janhq/interfaces/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@janhq/interfaces/dropdown-menu";
import { MessageResponse } from "@janhq/interfaces/ai-elements/message";
import {
  MoreVertical,
  Download,
  Eye,
  Presentation,
  FileText,
  FileSearch,
  Code,
  Image,
  File,
  ExternalLink,
  FileArchive,
  Globe,
} from "lucide-react";
import { cn, formatFileSize } from "@/lib/utils";

// Simple relative time formatter (avoids date-fns dependency)
// Handles both Unix timestamps (seconds) and ISO date strings
function formatRelativeTime(dateValue: string | number): string {
  let date: Date;
  if (typeof dateValue === "number") {
    // Unix timestamp in seconds - convert to milliseconds
    date = new Date(dateValue * 1000);
  } else {
    // Check if it's a numeric string (Unix timestamp)
    const num = Number(dateValue);
    if (!isNaN(num) && num > 1000000000 && num < 10000000000) {
      // Looks like a Unix timestamp in seconds
      date = new Date(num * 1000);
    } else {
      // ISO date string or other format
      date = new Date(dateValue);
    }
  }

  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffDay > 30) {
    return date.toLocaleDateString();
  } else if (diffDay > 0) {
    return `${diffDay} day${diffDay > 1 ? "s" : ""} ago`;
  } else if (diffHour > 0) {
    return `${diffHour} hour${diffHour > 1 ? "s" : ""} ago`;
  } else if (diffMin > 0) {
    return `${diffMin} minute${diffMin > 1 ? "s" : ""} ago`;
  } else {
    return "just now";
  }
}

interface ArtifactCardProps {
  artifact: ArtifactResponse;
}

function getArtifactIcon(contentType: string) {
  switch (contentType) {
    case "slides":
      return Presentation;
    case "document":
      return FileText;
    case "research":
      return FileSearch;
    case "code":
      return Code;
    case "image":
      return Image;
    case "markdown":
      return FileText;
    case "html":
      return Globe;
    case "zip":
    case "archive":
      return FileArchive;
    default:
      return File;
  }
}

function getContentTypeLabel(contentType: string): string {
  switch (contentType) {
    case "slides":
      return "Presentation";
    case "document":
      return "Document";
    case "research":
      return "Research";
    case "code":
      return "Code";
    case "image":
      return "Image";
    case "markdown":
      return "Markdown";
    case "html":
      return "HTML";
    case "json":
      return "JSON";
    case "zip":
    case "archive":
      return "Archive";
    default:
      return contentType;
  }
}

// Get truncated markdown content for preview (keeps markdown syntax for rendering)
function getMarkdownPreview(content?: string, maxLength = 300): string | null {
  if (!content) return null;
  // Truncate at a reasonable length for preview
  const truncated = content.slice(0, maxLength);
  // Try to end at a natural break point
  const lastNewline = truncated.lastIndexOf("\n");
  if (lastNewline > maxLength * 0.6) {
    return truncated.slice(0, lastNewline);
  }
  return truncated;
}

export function ArtifactCard({ artifact }: ArtifactCardProps) {
  const { openViewer } = useArtifactGalleryStore();
  const navigate = useNavigate();
  const Icon = getArtifactIcon(artifact.content_type);

  // State for lazy-loaded content preview
  const [fetchedContent, setFetchedContent] = useState<string | null>(null);

  // Fetch content from storage_path for markdown preview if content is null
  useEffect(() => {
    const shouldFetch = artifact.content_type === "markdown" ||
      artifact.content_type === "research" ||
      artifact.mime_type === "text/markdown";
    if (!artifact.content && artifact.storage_path && shouldFetch) {
      fetch(artifact.storage_path)
        .then((res) => res.ok ? res.text() : null)
        .then((text) => {
          if (text) setFetchedContent(text);
        })
        .catch(() => {/* ignore fetch errors */});
    }
  }, [artifact.content, artifact.storage_path, artifact.content_type, artifact.mime_type]);

  const handleDownload = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await artifactService.download(artifact.id, artifact.title);
    } catch (err) {
      console.error("Failed to download artifact:", err);
    }
  };

  const handleGoToConversation = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (artifact.conversation_id) {
      navigate({ to: "/threads/$conversationId", params: { conversationId: artifact.conversation_id } });
    }
  };

  const relativeDate = formatRelativeTime(artifact.updated_at);

  // Get thumbnail from metadata if available (for slides)
  const metadata = artifact.metadata as Record<string, unknown> | undefined;
  const slidesImages = metadata?.slides_images as Array<{ thumb: string }> | undefined;
  const thumbnailUrl = slidesImages?.[0]?.thumb;

  // Check if this artifact should show a markdown preview
  const isMarkdown = artifact.content_type === "markdown" ||
    artifact.content_type === "research" ||
    artifact.mime_type === "text/markdown";

  // Check if this artifact has a supported viewer
  // Documents only have viewer if they're text-based (not PDF/DOCX)
  const hasViewer = (() => {
    if (["slides", "research", "markdown", "html", "code", "image"].includes(artifact.content_type)) {
      return true;
    }
    if (artifact.content_type === "document" && artifact.mime_type?.startsWith("text/")) {
      return true;
    }
    return false;
  })();

  // Get markdown preview content
  const markdownPreview = useMemo(() => {
    if (!isMarkdown) return null;
    const content = artifact.content || fetchedContent;
    if (!content) return null;
    return getMarkdownPreview(content);
  }, [isMarkdown, artifact.content, fetchedContent]);

  // Open viewer for supported types, download for others
  const handleCardClick = () => {
    if (hasViewer) {
      openViewer(artifact);
    } else {
      artifactService.download(artifact.id, artifact.title);
    }
  };

  return (
    <div
      className={cn(
        "group relative bg-card rounded-xl border shadow-sm",
        "hover:shadow-md hover:border-primary/50 transition-all cursor-pointer"
      )}
      onClick={handleCardClick}
    >
      {/* Thumbnail/Preview Area */}
      <div className="h-32 bg-muted/30 rounded-t-xl flex items-center justify-center overflow-hidden relative">
        {thumbnailUrl ? (
          <img
            src={thumbnailUrl}
            alt={artifact.title}
            className="w-full h-full object-cover"
          />
        ) : markdownPreview ? (
          <div className="w-full h-full p-3 overflow-hidden">
            <div className="prose prose-sm dark:prose-invert max-w-none
              [&>*]:!my-1 [&>*]:!leading-snug
              [&_h1]:!text-sm [&_h1]:!font-semibold
              [&_h2]:!text-[13px] [&_h2]:!font-semibold
              [&_h3]:!text-xs [&_h3]:!font-medium
              [&_p]:!text-xs [&_p]:!text-muted-foreground/90
              [&_li]:!text-xs [&_li]:!text-muted-foreground/90
              [&_ul]:!pl-4 [&_ol]:!pl-4
              [&_code]:!text-[11px] [&_code]:!px-1
              [&_pre]:!hidden">
              <MessageResponse>{markdownPreview}</MessageResponse>
            </div>
          </div>
        ) : (
          <Icon className="size-10 text-muted-foreground/50" />
        )}
        {/* Gradient overlay for markdown preview */}
        {markdownPreview && !thumbnailUrl && (
          <div className="absolute inset-x-0 bottom-0 h-10 bg-gradient-to-t from-muted/80 to-transparent" />
        )}
      </div>

      {/* Content */}
      <div className="p-3">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <h3 className="font-medium text-sm line-clamp-1" title={artifact.title}>
              {artifact.title}
            </h3>
            <p className="text-xs text-muted-foreground mt-1">
              Last edited {relativeDate}
            </p>
            <div className="flex items-center gap-2 mt-2">
              <span className="text-xs bg-secondary px-2 py-0.5 rounded-full">
                {getContentTypeLabel(artifact.content_type)}
              </span>
              <span className="text-xs text-muted-foreground">
                {formatFileSize(artifact.size_bytes)}
              </span>
            </div>
          </div>

          {/* Actions Menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
              <Button
                variant="ghost"
                size="icon"
                className="size-7 opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <MoreVertical className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {hasViewer ? (
                <DropdownMenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    openViewer(artifact);
                  }}
                >
                  <Eye className="size-4 mr-2" />
                  View
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    artifactService.download(artifact.id, artifact.title);
                  }}
                >
                  <Download className="size-4 mr-2" />
                  Open
                </DropdownMenuItem>
              )}
              {artifact.conversation_id && (
                <DropdownMenuItem onClick={handleGoToConversation}>
                  <ExternalLink className="size-4 mr-2" />
                  View Chat
                </DropdownMenuItem>
              )}
              <DropdownMenuItem onClick={handleDownload}>
                <Download className="size-4 mr-2" />
                Download
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  );
}
