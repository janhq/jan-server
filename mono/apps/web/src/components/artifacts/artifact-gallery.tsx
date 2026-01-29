import { useEffect, useState, useCallback } from "react";
import { useArtifactGalleryStore } from "@/stores/artifact-gallery-store";
import { artifactService } from "@/services/artifact-service";
import { ArtifactCard } from "./artifact-card";
import { ArtifactFilterTabs } from "./artifact-filter-tabs";
import { ArtifactCategoryPicker } from "./artifact-category-picker";
import { SlideViewer } from "@/components/misc/slide-viewer";
import { MdViewer } from "@/components/misc/md-viewer";
import { HtmlViewer } from "@/components/misc/html-viewer";
import { ImageViewer } from "@/components/misc/image-viewer";
import { Button } from "@janhq/interfaces/button";
import { Input } from "@janhq/interfaces/input";
import { Plus, Search, Loader2 } from "lucide-react";
import { useDebounce } from "@/hooks/use-debounce";

export function ArtifactGallery() {
  const {
    artifacts,
    total,
    hasMore,
    isLoading,
    isLoadingMore,
    error,
    searchQuery,
    selectedArtifact,
    isViewerOpen,
    isViewerLoading,
    isCategoryPickerOpen,
    fetchArtifacts,
    setSearchQuery,
    loadMore,
    closeViewer,
    openCategoryPicker,
    closeCategoryPicker,
  } = useArtifactGalleryStore();

  const [localSearch, setLocalSearch] = useState(searchQuery);
  const debouncedSearch = useDebounce(localSearch, 300);

  useEffect(() => {
    fetchArtifacts();
  }, [fetchArtifacts]);

  useEffect(() => {
    if (debouncedSearch !== searchQuery) {
      setSearchQuery(debouncedSearch);
      fetchArtifacts();
    }
  }, [debouncedSearch, searchQuery, setSearchQuery, fetchArtifacts]);

  // Parse slide images from artifact metadata if available
  const getSlideImages = useCallback(() => {
    if (!selectedArtifact?.metadata) return [];
    const metadata = selectedArtifact.metadata as Record<string, unknown>;
    const slidesImages = metadata.slides_images as Array<{ id: string; url: string; thumb: string }> | undefined;
    return slidesImages?.map((img) => ({
      id: img.id,
      thumb: img.thumb || img.url,
    })) || [];
  }, [selectedArtifact]);

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-background border-b px-6 py-4">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold">Artifacts</h1>
            <p className="text-muted-foreground text-sm">
              {total} {total === 1 ? "artifact" : "artifacts"} created
            </p>
          </div>
          <Button onClick={openCategoryPicker} className="gap-1.5">
            <Plus className="size-4" />
            New artifact
          </Button>
        </div>

        {/* Search */}
        <div className="relative max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            placeholder="Search artifacts..."
            className="pl-9"
            value={localSearch}
            onChange={(e) => setLocalSearch(e.target.value)}
          />
        </div>
      </div>

      {/* Filter Tabs */}
      <ArtifactFilterTabs />

      {/* Grid */}
      <div className="flex-1 overflow-auto p-6">
        {isLoading && artifacts.length === 0 ? (
          <div className="flex items-center justify-center h-64">
            <Loader2 className="size-8 animate-spin text-muted-foreground" />
          </div>
        ) : error ? (
          <div className="text-center text-destructive py-12">{error}</div>
        ) : artifacts.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 text-center">
            <div className="rounded-full bg-muted p-4 mb-4">
              <Search className="size-8 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-medium mb-2">No artifacts yet</h3>
            <p className="text-muted-foreground mb-4 max-w-sm">
              Create your first artifact by starting a new conversation and asking for slides, documents, or other content.
            </p>
            <Button onClick={openCategoryPicker} className="gap-1.5">
              <Plus className="size-4" />
              Create artifact
            </Button>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {artifacts.map((artifact) => (
                <ArtifactCard key={artifact.id} artifact={artifact} />
              ))}
            </div>

            {hasMore && (
              <div className="flex justify-center mt-6">
                <Button variant="outline" onClick={loadMore} disabled={isLoadingMore}>
                  {isLoadingMore ? <Loader2 className="size-4 animate-spin mr-2" /> : null}
                  Load more
                </Button>
              </div>
            )}
          </>
        )}
      </div>

      {/* Category Picker Modal */}
      <ArtifactCategoryPicker
        open={isCategoryPickerOpen}
        onClose={closeCategoryPicker}
      />

      {/* Viewer Modals */}
      {isViewerOpen && selectedArtifact && selectedArtifact.content_type === "slides" && (
        <SlideViewer
          slides={getSlideImages()}
          title={selectedArtifact.title}
          onDownload={() => artifactService.download(selectedArtifact.id, selectedArtifact.title)}
          conversationId={selectedArtifact.conversation_id}
          onClose={closeViewer}
        />
      )}

      {isViewerOpen && selectedArtifact && (
        selectedArtifact.content_type === "research" ||
        selectedArtifact.content_type === "markdown" ||
        (selectedArtifact.content_type === "document" && selectedArtifact.mime_type?.startsWith("text/"))
      ) && (
        <MdViewer
          content={selectedArtifact.content || ""}
          title={selectedArtifact.title}
          onDownload={() => artifactService.download(selectedArtifact.id, selectedArtifact.title)}
          conversationId={selectedArtifact.conversation_id}
          isLoading={isViewerLoading}
          onClose={closeViewer}
        />
      )}

      {isViewerOpen && selectedArtifact && selectedArtifact.content_type === "html" && (
        <HtmlViewer
          content={selectedArtifact.content || ""}
          title={selectedArtifact.title}
          onDownload={() => artifactService.download(selectedArtifact.id, selectedArtifact.title)}
          conversationId={selectedArtifact.conversation_id}
          isLoading={isViewerLoading}
          onClose={closeViewer}
        />
      )}

      {isViewerOpen && selectedArtifact && selectedArtifact.content_type === "code" && (
        <MdViewer
          content={`\`\`\`\n${selectedArtifact.content || ""}\n\`\`\``}
          title={selectedArtifact.title}
          onDownload={() => artifactService.download(selectedArtifact.id, selectedArtifact.title)}
          conversationId={selectedArtifact.conversation_id}
          isLoading={isViewerLoading}
          onClose={closeViewer}
        />
      )}

      {isViewerOpen && selectedArtifact && selectedArtifact.content_type === "image" && selectedArtifact.storage_path && (
        <ImageViewer
          src={selectedArtifact.storage_path}
          title={selectedArtifact.title}
          onDownload={() => artifactService.download(selectedArtifact.id, selectedArtifact.title)}
          conversationId={selectedArtifact.conversation_id}
          isLoading={isViewerLoading}
          onClose={closeViewer}
        />
      )}
    </div>
  );
}
