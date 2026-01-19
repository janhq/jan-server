import { memo, useState } from "react";
import {
  useRightSidebarStore,
  type SearchResultItem,
  type ArtifactItem,
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
  FileIcon,
  PresentationIcon,
  FileTextIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";
import DocViewer, { DocViewerRenderers } from "react-doc-viewer";

const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
};

const getArtifactIcon = (contentType: string, mimeType: string) => {
  if (contentType === "slides" || mimeType.includes("presentation")) {
    return PresentationIcon;
  }
  if (contentType === "document" || mimeType.includes("document") || mimeType.includes("pdf")) {
    return FileTextIcon;
  }
  return FileIcon;
};

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
        <div className="size-10 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
          <Icon className="size-5 text-primary" />
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

// Preview component for bottom Panel content section
const ArtifactPreview = ({ artifact }: { artifact: ArtifactItem }) => {


  return (
    <div className="h-full flex flex-col">
      <DocViewer
        documents={[{ uri: artifact.downloadUrl, fileType: artifact.mimeType }]}
        pluginRenderers={DocViewerRenderers}
        config={{
          header: {
            disableHeader: true,
            disableFileName: true,
          },
        }}
        style={{ flex: 1, backgroundColor: "transparent" }}
      />

    </div>
  );
};

const Favicon = ({ url }: { url: string }) => {
  const [error, setError] = useState(false);
  if (error) {
    return <LinkIcon className="size-4 shrink-0 mt-0.5 text-muted-foreground" />;
  }
  try {
    const domain = new URL(url).hostname;
    const faviconUrl = `https://www.google.com/s2/favicons?domain=${domain}&sz=32`;

    return (
      <img
        src={faviconUrl}
        alt=""
        className="size-4 shrink-0 mt-0.5 rounded-full"
        onError={() => setError(true)}
      />
    );
  } catch {
    return <LinkIcon className="size-4 shrink-0 mt-0.5 text-muted-foreground" />;
  }
};

const SearchResult = ({
  result,
  isLast,
}: {
  result: SearchResultItem;
  isLast?: boolean;
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
    return (
      <div>
        {/* <ArtifactCard artifact={result.artifact} /> */}
        <div className="p-3">
          <div className="rounded-lg border bg-muted/50 overflow-hidden" style={{ height: 400 }}>
            <ArtifactPreview artifact={result.artifact} />
          </div>
        </div>
      </div>
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

          {/* Panel progress */}
          <div className="flex flex-1 flex-col mx-2 overflow-auto bg-background border rounded-lg max-h-20">
            <div className="sticky top-0 z-10 bg-background border-b px-3 py-2">
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className="shrink-0">Progress</span>
              </div>
            </div>
          </div>

          {/* Panel artifact */}
          {hasArtifacts && (
            <div className="flex flex-col mx-2 overflow-auto bg-background border rounded-lg">
              <div className="sticky top-0 z-10 bg-background border-b px-3 py-2">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="shrink-0">Artifacts</span>
                  <span className="text-muted-foreground/60">({artifacts.length})</span>
                </div>
              </div>
              <div className="divide-y">
                {artifacts.map((artifact) => (
                  <ArtifactCard key={artifact.id} artifact={artifact} />
                ))}
              </div>
            </div>
          )}

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
                {/* Artifact Preview */}
                <div className="flex-1">
                  <ArtifactPreview artifact={artifacts[0]} />
                </div>
              </>
            ) : (
              <div className="flex items-center justify-center h-full">
                <p className="text-muted-foreground text-sm text-center">
                  Click on a step to view search results
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
