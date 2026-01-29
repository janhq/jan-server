import { memo, useState, useEffect } from "react";

import {
  useRightSidebarStore,
  type SearchResultItem,
  type ArtifactItem,
  type SlideImage,
} from "@/stores/right-sidebar-store";
import { Button } from "@janhq/interfaces/button";
import { MessageResponse } from "@janhq/interfaces/ai-elements/message";
import {
  XIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ExternalLinkIcon,
  LinkIcon,
  DownloadIcon,
  MaximizeIcon,
  Loader2Icon,
  EyeIcon,
} from "lucide-react";
import { cn, getArtifactIcon, formatFileSize } from "@/lib/utils";
import { Favicon } from "@/components/misc/favicon";
import { SlideViewer } from "@/components/misc/slide-viewer";
import { MdViewer } from "@/components/misc/md-viewer";



// Download-only card for Artifacts section
const ArtifactCard = ({ artifact }: { artifact: ArtifactItem }) => {
  const handleDownload = () => {
    if (!artifact.downloadUrl) return;
    window.open(artifact.downloadUrl, "_blank");
  };
  const Icon = getArtifactIcon(artifact.contentType, artifact.mimeType);
  return (
    <div className="p-3 hover:bg-secondary/50 transition-colors">
      <div className="flex items-start gap-3">
        <div className="size-8 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
          <Icon className="size-4 text-primary" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <h3 className="font-medium text-sm line-clamp-1 text-foreground">
                {artifact.filename}
              </h3>
              <p className="text-xs text-muted-foreground mt-0.5">
                {artifact.contentType} • {formatFileSize(artifact.size)}
              </p>
            </div>
            <Button
              variant="ghost"
              size="icon"
              className="size-7 shrink-0"
              onClick={handleDownload}
            >
              <DownloadIcon className="size-4" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

// Hard-coded slide images for preview fallback
const HARD_CODED_SLIDE_IMAGES = [
  { id: "1", thumb: "/slides/slide-1.svg" },
  { id: "2", thumb: "/slides/slide-2.svg" },
  { id: "3", thumb: "/slides/slide-3.svg" },
  { id: "4", thumb: "/slides/slide-4.svg" },
  { id: "5", thumb: "/slides/slide-5.svg" },
  { id: "6", thumb: "/slides/slide-6.svg" },
];
const USE_HARD_CODED_SLIDES = false;

const needsSlidesFallback = (artifact: ArtifactItem, slidesFallback?: SlideImage[]) => {
  if (!slidesFallback || slidesFallback.length === 0) return false;
  if (artifact.slidesImages && artifact.slidesImages.length > 0) return false;
  const filename = (artifact.filename || "").toLowerCase();
  return filename.includes("slide-images");
};

const withSlidesFallback = (
  artifact: ArtifactItem,
  slidesFallback?: SlideImage[],
  fallbackDownloadUrl?: string,
  fallbackTitle?: string,
) => {
  if (!needsSlidesFallback(artifact, slidesFallback)) return artifact;
  return {
    ...artifact,
    slidesImages: slidesFallback,
    downloadUrl: fallbackDownloadUrl || artifact.downloadUrl,
    filename: fallbackTitle || artifact.filename,
  };
};


// Preview component for bottom Panel content section
const ArtifactPreview = ({ artifact }: { artifact: ArtifactItem }) => {
  const [isViewerOpen, setIsViewerOpen] = useState(false);
  const [mdContent, setMdContent] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Fetch markdown content when artifact is research type
  useEffect(() => {
    if (artifact.contentType === "research" && artifact.downloadUrl && !mdContent) {
      setIsLoading(true);
      fetch(artifact.downloadUrl)
        .then((res) => res.text())
        .then((text) => {
          setMdContent(text);
        })
        .catch((err) => {
          console.error("Failed to fetch markdown content:", err);
          setMdContent("Failed to load content");
        })
        .finally(() => {
          setIsLoading(false);
        });
    }
  }, [artifact.contentType, artifact.downloadUrl, mdContent]);

  // Handle research/markdown content type
  if (artifact.contentType === "research") {
    return (
      <>
        <div className="h-full flex flex-col">
          <div className="flex-1 flex items-start p-3">
            <div
              className="relative w-full rounded-lg border overflow-hidden bg-muted/30 cursor-pointer group"
              onClick={() => !isLoading && setIsViewerOpen(true)}
            >
              {/* Document thumbnail preview - compact horizontal layout */}
              <div className="p-3 flex items-center gap-3">
                {(() => {
                  const Icon = getArtifactIcon(artifact.contentType, artifact.mimeType);
                  return (
                    <div className="size-8 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                      <Icon className="size-4 text-primary" />
                    </div>
                  );
                })()}
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium line-clamp-1">{artifact.filename}</p>
                  <p className="text-xs text-muted-foreground">{formatFileSize(artifact.size)}</p>
                </div>
                <EyeIcon className="size-4 text-muted-foreground shrink-0" />
              </div>
              {/* Loading overlay */}
              {isLoading && (
                <div className="absolute inset-0 bg-background/50 flex items-center justify-center">
                  <Loader2Icon className="size-4 animate-spin text-muted-foreground" />
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Full markdown viewer modal */}
        {isViewerOpen && mdContent && (
          <MdViewer
            content={mdContent}
            title={artifact.filename}
            onDownload={artifact.downloadUrl ? () => window.open(artifact.downloadUrl, "_blank") : undefined}
            onClose={() => setIsViewerOpen(false)}
          />
        )}
      </>
    );
  }

  // Handle slides content type
  if (artifact.contentType === "slides" || (artifact.slidesImages && artifact.slidesImages.length > 0)) {
    const slides = USE_HARD_CODED_SLIDES
      ? HARD_CODED_SLIDE_IMAGES
      : (artifact.slidesImages?.length ? artifact.slidesImages : []);
    if (slides.length === 0) {
      return null;
    }
    const currentSlide = slides[0];
    return (
      <>
        <div className="h-full flex flex-col">
          {/* Slide image - clickable to open full viewer */}
          <div className="flex-1 flex items-center justify-center p-3">
            <div
              className="relative w-full rounded-lg border overflow-hidden bg-muted/50 cursor-pointer group"
              onClick={() => setIsViewerOpen(true)}
            >
              <div className="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition-colors flex items-center justify-center pointer-events-none">
                <MaximizeIcon className="size-8 text-white opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
              <img
                src={currentSlide.thumb}
                alt={`Slide ${currentSlide.id}`}
                className="w-full h-auto object-contain"
              />
            </div>
          </div>
        </div>

        {/* Full slide viewer modal */}
        {isViewerOpen && (
          <SlideViewer
            slides={slides}
            initialIndex={0}
            title={artifact.filename}
            onDownload={artifact.downloadUrl ? () => window.open(artifact.downloadUrl, "_blank") : undefined}
            onClose={() => setIsViewerOpen(false)}
          />
        )}
      </>
    );
  }

  return null;
};

const SearchResult = ({
  result,
  isLast,
  slidesFallback,
  slidesFallbackDownloadUrl,
  slidesFallbackTitle,
}: {
  result: SearchResultItem;
  isLast?: boolean;
  slidesFallback?: SlideImage[];
  slidesFallbackDownloadUrl?: string;
  slidesFallbackTitle?: string;
}) => {


  if (result.type === "link") {
    return (
      <a
        href={result.url}
        target="_blank"
        rel="noopener noreferrer"
        className="block group"
      >
        <div
          className={cn(
            "p-3 bg-background hover:bg-secondary/50 transition-colors",
            !isLast && "border-b"
          )}
        >
          <div className="flex items-start gap-2">
            {result.icon ? (
              <span className="text-base shrink-0">{result.icon}</span>
            ) : result.url ? (
              <Favicon url={result.url} />
            ) : (
              <LinkIcon className="size-4 shrink-0 mt-0.5 text-muted-foreground" />
            )}
            <div className="flex-1 min-w-0 text-muted-foreground">
              <div className="flex items-start justify-between gap-2">
                <h3 className="font-medium text-sm line-clamp-1 transition-colors text-foreground">
                  {result.title || result.url}
                </h3>
                <ExternalLinkIcon className="size-3 shrink-0 ml-2 mt-0.5" />
              </div>
              {result.description && (
                <p className="text-xs mt-1 line-clamp-2">
                  {result.description}
                </p>
              )}
              {result.url && (
                <div className="text-xs bg-secondary px-2 py-0.5 rounded-full inline-flex mt-2 max-w-full">
                  <span className="line-clamp-1 break-all">{result.url}</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </a>
    );
  }

  if (result.type === "image") {
    return (
      <div className="my-1 p-3">
        {result.imageUrl && (
          <div className="rounded-lg border overflow-hidden bg-background">
            <img
              src={result.imageUrl}
              alt={result.title || "Result image"}
              className="w-full h-auto"
            />
          </div>
        )}
        {(result.title || result.description) && (
          <div className="px-1 mt-2">
            {result.title && (
              <h3 className="font-medium text-sm">{result.title}</h3>
            )}
            {result.description && (
              <p className="text-muted-foreground text-xs mt-1">
                {result.description}
              </p>
            )}
          </div>
        )}
      </div>
    );
  }

  if (result.type === "text") {
    return (
      <div className="p-3">
        {result.title && (
          <h3 className="font-medium text-sm mb-2">{result.title}</h3>
        )}
        {result.content && (
          <div className="text-sm leading-relaxed prose prose-sm dark:prose-invert max-w-none [&>*:first-child]:mt-0 [&>*:last-child]:mb-0">
            <MessageResponse>{result.content}</MessageResponse>
          </div>
        )}
      </div>
    );
  }

  if (result.type === "artifact" && result.artifact) {
    const previewArtifact = withSlidesFallback(
      result.artifact,
      slidesFallback,
      slidesFallbackDownloadUrl,
      slidesFallbackTitle,
    );
    return (
      <ArtifactPreview artifact={previewArtifact} />
    );
  }

  return null;
};

export const AppSidebarRight = memo(function AppSidebarRight() {
  const rightSidebarOpen = useRightSidebarStore((state) => state.isOpen);
  const toggleSidebar = useRightSidebarStore((state) => state.toggleSidebar);
  const allSteps = useRightSidebarStore((state) => state.allSteps);
  const currentStepIndex = useRightSidebarStore(
    (state) => state.currentStepIndex
  );
  const artifacts = useRightSidebarStore((state) => state.artifacts);
  const clearSelection = useRightSidebarStore((state) => state.clearSelection);
  const nextStep = useRightSidebarStore((state) => state.nextStep);
  const prevStep = useRightSidebarStore((state) => state.prevStep);

  const handleClose = () => {
    clearSelection();
    if (rightSidebarOpen) {
      toggleSidebar();
    }
  };

  const hasSteps = allSteps.length > 0;
  const hasArtifacts = artifacts.length > 0;
  const currentStep = allSteps[currentStepIndex];
  const currentResults = currentStep?.results || [];
  const slidesArtifact = [...artifacts].reverse().find((artifact) => (artifact.slidesImages?.length ?? 0) > 0);
  const slidesFallback = slidesArtifact?.slidesImages;
  const slidesFallbackDownloadUrl = slidesArtifact?.downloadUrl;
  const slidesFallbackTitle = slidesArtifact?.filename;

  return (
    <div
      className="group text-sidebar-foreground block relative z-50"
      data-state={rightSidebarOpen ? "expanded" : "collapsed"}
      data-side="right"
    >
      {/* Sidebar gap */}
      <div
        className={cn(
          "relative bg-transparent transition-[width] duration-200 ease-linear",
          rightSidebarOpen ? "w-full md:w-96" : "w-0"
        )}
      />

      {/* Sidebar container */}
      <div
        className={cn(
          "fixed p-2 inset-y-0 z-10 flex h-svh transition-[right,width] duration-200 ease-linear",
          "right-0",
          rightSidebarOpen ? "w-full md:w-96" : "w-0 -right-60"
        )}
      >

        <div className="bg-secondary/50 backdrop-blur-2xl border flex h-full w-full flex-col rounded-2xl overflow-hidden gap-2">

          {/* Header */}
          <div className="flex flex-col gap-2 p-2 pb-0! shrink-0 ">
            <div className="flex items-center w-full pl-2 justify-between">
              <div className="flex flex-col gap-0.5">
                <span className="text-base font-medium font-studio">
                  Jan's Computer
                </span>
              </div>
              <Button
                variant="ghost"
                size="icon"
                className="size-7 text-muted-foreground hover:bg-foreground/10 shrink-0"
                onClick={handleClose}
              >
                <XIcon className="size-4" />
              </Button>
            </div>
          </div>


          {/* Panel artifact */}
            <div className="flex flex-col mx-2 overflow-auto bg-background border rounded-lg">
              <div className="sticky top-0 z-10 bg-background border-b px-3 py-2">
                <div className="flex items-center gap-1 text-xs text-muted-foreground">
                  <span className="shrink-0">Artifacts</span>
                  <span className="text-muted-foreground/60">({artifacts.length})</span>
                </div>
              </div>
              <div className="divide-y">
                {hasArtifacts ?  artifacts.map((artifact) => (
                  <ArtifactCard key={artifact.id} artifact={artifact} />
                )) : <div className="p-3 text-xs text-muted-foreground">Outputs created during the task land here.</div>}
              </div>
            </div>

          {/* Panel content */}
          <div className="flex flex-1 flex-col mx-2 overflow-auto bg-background border rounded-lg">
            {hasSteps ? (
              <>
                {/* Content Header */}
                <div className="sticky top-0 z-10 bg-background border-b px-3 py-2">
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="shrink-0">{currentStep.stepName}</span>
                    <span className="shrink-0">•</span>
                    <span className="line-clamp-1">
                      {currentStep.stepTitle}
                    </span>
                  </div>
                </div>
                {/* Results */}
                <div>
                  {currentResults.map((result, index) => (
                    <SearchResult
                      key={index}
                      result={result}
                      isLast={index === currentResults.length - 1}
                      slidesFallback={slidesFallback}
                      slidesFallbackDownloadUrl={slidesFallbackDownloadUrl}
                      slidesFallbackTitle={slidesFallbackTitle}
                    />
                  ))}
                </div>
              </>
            ) : hasArtifacts ? (
              <>
                {/* Preview Header */}
                <div className="sticky top-0 z-10 bg-background border-b px-3 py-2">
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="shrink-0">Preview</span>
                    <span className="shrink-0">•</span>
                    <span className="line-clamp-1">{artifacts[0].filename}</span>
                  </div>
                </div>
              </>
            ) : (
              <div className="flex items-center justify-center h-full">
                <p className="text-muted-foreground text-sm text-center">
                  No tasks running
                </p>
              </div>
            )}
          </div>

          {/* Pagination Footer */}
          {hasSteps && (
            <div className="mt-2">
              {/* Progress Bar */}
              <div className="w-full h-1 bg-primary/20 rounded-full overflow-hidden">
                <div
                  className="h-full bg-primary transition-all duration-300 ease-out"
                  style={{
                    width: `${((currentStepIndex + 1) / allSteps.length) * 100}%`,
                  }}
                />
              </div>
              <div className="shrink-0 p-3 border-t bg-background/50">
                <div className="flex items-center justify-between">
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-6"
                      onClick={prevStep}
                      disabled={currentStepIndex === 0}
                    >
                      <ChevronLeftIcon className="size-4" />
                    </Button>

                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-6"
                      onClick={nextStep}
                      disabled={currentStepIndex === allSteps.length - 1}
                    >
                      <ChevronRightIcon className="size-4" />
                    </Button>
                  </div>

                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground">
                      <span className="text-foreground">
                        {currentStepIndex + 1}
                      </span>{" "}
                      / {allSteps.length}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
});
