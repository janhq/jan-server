import { create } from "zustand";
import { artifactService, type ArtifactResponse } from "@/services/artifact-service";

export type ContentTypeFilter = "all" | "slides" | "document" | "research" | "code" | "image" | "markdown" | "html";

interface ArtifactGalleryState {
  // Data
  artifacts: ArtifactResponse[];
  total: number;
  hasMore: boolean;
  lastId: string | null;
  isLoading: boolean;
  isLoadingMore: boolean;
  error: string | null;

  // Filters
  contentTypeFilter: ContentTypeFilter;
  searchQuery: string;

  // Pagination
  limit: number;

  // Viewer state
  selectedArtifact: ArtifactResponse | null;
  isViewerOpen: boolean;
  isViewerLoading: boolean;

  // Category picker state
  isCategoryPickerOpen: boolean;

  // Actions
  fetchArtifacts: () => Promise<void>;
  loadMore: () => Promise<void>;
  setContentTypeFilter: (filter: ContentTypeFilter) => void;
  setSearchQuery: (query: string) => void;
  openViewer: (artifact: ArtifactResponse) => Promise<void>;
  closeViewer: () => void;
  openCategoryPicker: () => void;
  closeCategoryPicker: () => void;
  reset: () => void;
}

let fetchPromise: Promise<void> | null = null;

export const useArtifactGalleryStore = create<ArtifactGalleryState>((set, get) => ({
  // Initial state
  artifacts: [],
  total: 0,
  hasMore: false,
  lastId: null,
  isLoading: false,
  isLoadingMore: false,
  error: null,
  contentTypeFilter: "all",
  searchQuery: "",
  limit: 20,
  selectedArtifact: null,
  isViewerOpen: false,
  isViewerLoading: false,
  isCategoryPickerOpen: false,

  fetchArtifacts: async () => {
    // Prevent duplicate fetches
    if (fetchPromise) {
      return fetchPromise;
    }

    const { contentTypeFilter, searchQuery, limit } = get();

    fetchPromise = (async () => {
      try {
        set({ isLoading: true, error: null, lastId: null });

        const response = await artifactService.list({
          content_type: contentTypeFilter === "all" ? undefined : contentTypeFilter,
          search: searchQuery || undefined,
          limit,
          latest: true,
        });

        set({
          artifacts: response.data,
          total: response.total,
          hasMore: response.has_more,
          lastId: response.last_id || null,
          isLoading: false,
        });
      } catch (err) {
        set({
          error: err instanceof Error ? err.message : "Failed to fetch artifacts",
          isLoading: false,
        });
      } finally {
        fetchPromise = null;
      }
    })();

    return fetchPromise;
  },

  loadMore: async () => {
    const { hasMore, lastId, limit, contentTypeFilter, searchQuery, isLoadingMore } = get();

    // Check if there's more to load
    if (!hasMore || !lastId || isLoadingMore) {
      return;
    }

    try {
      set({ isLoadingMore: true });

      const response = await artifactService.list({
        content_type: contentTypeFilter === "all" ? undefined : contentTypeFilter,
        search: searchQuery || undefined,
        limit,
        after: lastId,
        latest: true,
      });

      set((state) => ({
        artifacts: [...state.artifacts, ...response.data],
        hasMore: response.has_more,
        lastId: response.last_id || null,
        isLoadingMore: false,
      }));
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : "Failed to load more artifacts",
        isLoadingMore: false,
      });
    }
  },

  setContentTypeFilter: (filter) => {
    set({ contentTypeFilter: filter });
    get().fetchArtifacts();
  },

  setSearchQuery: (query) => {
    set({ searchQuery: query });
  },

  openViewer: async (artifact) => {
    // Open viewer immediately with available data
    set({ selectedArtifact: artifact, isViewerOpen: true, isViewerLoading: true });

    try {
      // Fetch full artifact with content
      const fullArtifact = await artifactService.getById(artifact.id);

      // Only fetch content as text for text-based types (not images/slides/PDFs)
      const isTextType = ["research", "markdown", "html", "code"].includes(fullArtifact.content_type) ||
        (fullArtifact.content_type === "document" && fullArtifact.mime_type?.startsWith("text/"));

      if (!fullArtifact.content && fullArtifact.storage_path && isTextType) {
        try {
          const response = await fetch(fullArtifact.storage_path);
          if (response.ok) {
            const content = await response.text();
            fullArtifact.content = content;
          }
        } catch (fetchErr) {
          console.error("Failed to fetch content from storage:", fetchErr);
        }
      }

      set({ selectedArtifact: fullArtifact, isViewerLoading: false });
    } catch (err) {
      // Keep showing what we have, just stop loading
      set({ isViewerLoading: false });
      console.error("Failed to fetch full artifact:", err);
    }
  },

  closeViewer: () => {
    set({ selectedArtifact: null, isViewerOpen: false });
  },

  openCategoryPicker: () => {
    set({ isCategoryPickerOpen: true });
  },

  closeCategoryPicker: () => {
    set({ isCategoryPickerOpen: false });
  },

  reset: () => {
    set({
      artifacts: [],
      total: 0,
      hasMore: false,
      lastId: null,
      contentTypeFilter: "all",
      searchQuery: "",
      selectedArtifact: null,
      isViewerOpen: false,
      isViewerLoading: false,
      isCategoryPickerOpen: false,
      error: null,
    });
  },
}));
