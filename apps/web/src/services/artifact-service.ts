import { fetchJsonWithAuth, fetchWithAuth } from "@/lib/api-client";

declare const JAN_API_BASE_URL: string;

// File System Access API types (for "Save As" dialog)
interface SaveFilePickerOptions {
  suggestedName?: string;
  types?: Array<{
    description?: string;
    accept: Record<string, string[]>;
  }>;
}

// Artifacts have their own Kong route at /v1/artifacts (not under /responses)
const API_BASE = JAN_API_BASE_URL;

export interface ArtifactResponse {
  id: string;
  response_id: string;
  conversation_id?: string; // Thread ID for navigation
  plan_id?: string;
  content_type: string;
  mime_type: string;
  title: string;
  content?: string;
  storage_path?: string;
  size_bytes: number;
  version: number;
  parent_id?: string;
  is_latest: boolean;
  retention_policy: string;
  metadata?: Record<string, unknown>;
  created_at: number; // Unix timestamp in seconds
  updated_at: number; // Unix timestamp in seconds
  expires_at?: number; // Unix timestamp in seconds
}

export interface ArtifactListResponse {
  object: string;
  data: ArtifactResponse[];
  first_id: string;
  last_id: string;
  has_more: boolean;
  total: number;
}

export interface ListArtifactsParams {
  content_type?: string;
  search?: string;
  latest?: boolean;
  limit?: number;
  after?: string; // Cursor for pagination
  order?: "asc" | "desc";
}

export const artifactService = {
  /**
   * List all artifacts for the authenticated user
   */
  list: async (params: ListArtifactsParams = {}): Promise<ArtifactListResponse> => {
    const searchParams = new URLSearchParams();
    if (params.content_type) searchParams.set("content_type", params.content_type);
    if (params.search) searchParams.set("search", params.search);
    if (params.latest !== undefined) searchParams.set("latest", String(params.latest));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.after) searchParams.set("after", params.after);
    if (params.order) searchParams.set("order", params.order);

    const queryString = searchParams.toString();
    const url = `${API_BASE}v1/artifacts${queryString ? `?${queryString}` : ""}`;

    return fetchJsonWithAuth<ArtifactListResponse>(url);
  },

  /**
   * Get a single artifact by ID
   */
  getById: async (artifactId: string): Promise<ArtifactResponse> => {
    return fetchJsonWithAuth<ArtifactResponse>(
      `${API_BASE}v1/artifacts/${artifactId}`,
    );
  },

  /**
   * Get all versions of an artifact
   */
  getVersions: async (artifactId: string): Promise<ArtifactResponse[]> => {
    return fetchJsonWithAuth<ArtifactResponse[]>(
      `${API_BASE}v1/artifacts/${artifactId}/versions`,
    );
  },

  /**
   * Delete an artifact
   */
  delete: async (artifactId: string): Promise<void> => {
    const response = await fetchWithAuth(
      `${API_BASE}v1/artifacts/${artifactId}`,
      { method: "DELETE" },
    );
    if (!response.ok) {
      throw new Error("Failed to delete artifact");
    }
  },

  /**
   * Get download URL for an artifact
   */
  getDownloadUrl: (artifactId: string): string => {
    return `${API_BASE}v1/artifacts/${artifactId}/download`;
  },

  /**
   * Download an artifact with "Save As" dialog
   * Uses File System Access API when available, falls back to regular download
   */
  download: async (artifactId: string, filename?: string): Promise<void> => {
    // First get the artifact metadata for filename and mime type
    const artifact = await artifactService.getById(artifactId);
    const suggestedName = filename || artifact.title || "download";
    const mimeType = artifact.mime_type || "application/octet-stream";

    let blob: Blob;

    if (artifact.content) {
      // Inline content - create blob from content
      blob = new Blob([artifact.content], { type: mimeType });
    } else if (artifact.storage_path) {
      // Try fetching from storage URL (should be public/signed)
      // If that fails, fall back to authenticated backend endpoint
      try {
        const response = await fetch(artifact.storage_path);
        if (response.ok) {
          blob = await response.blob();
        } else {
          throw new Error("Storage fetch failed");
        }
      } catch {
        // Fallback: use authenticated backend download endpoint
        const response = await fetchWithAuth(`${API_BASE}v1/artifacts/${artifactId}/download`);
        if (!response.ok) {
          throw new Error("Failed to download artifact");
        }
        blob = await response.blob();
      }
    } else {
      throw new Error("Artifact has no downloadable content");
    }

    // Try to use File System Access API for "Save As" dialog
    if ("showSaveFilePicker" in window) {
      try {
        const handle = await (window as Window & { showSaveFilePicker: (options?: SaveFilePickerOptions) => Promise<FileSystemFileHandle> }).showSaveFilePicker({
          suggestedName,
          types: [{
            description: "File",
            accept: { [mimeType]: [] },
          }],
        });
        const writable = await handle.createWritable();
        await writable.write(blob);
        await writable.close();
        return;
      } catch (err) {
        // User cancelled or API failed - fall back to regular download
        if ((err as Error).name === "AbortError") {
          return; // User cancelled
        }
      }
    }

    // Fallback: regular download
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = suggestedName;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  },
};
