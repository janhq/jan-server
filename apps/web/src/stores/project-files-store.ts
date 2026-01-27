import { create } from "zustand";
import {
  projectFilesService,
  type ProjectFile,
  type CreateProjectFileRequest,
} from "@/services/project-files-service";
import { uploadMedia, fileToDataUrl } from "@/services/media-upload-service";

// Helper to extract media object ID from URL
function extractMediaObjectId(url: string): string {
  const match = url.match(/jan_[a-zA-Z0-9]+/);
  if (match) {
    return match[0];
  }
  return url;
}

interface ProjectFilesState {
  // Files grouped by project ID
  files: Record<string, ProjectFile[]>;
  // Loading state per project
  loading: Record<string, boolean>;
  // Error state per project
  errors: Record<string, string | null>;

  // Actions
  getFiles: (projectId: string) => Promise<ProjectFile[]>;
  uploadFile: (projectId: string, file: File, userId: string) => Promise<ProjectFile>;
  deleteFile: (projectId: string, fileId: string) => Promise<void>;
  reorderFiles: (projectId: string, fileIds: string[]) => Promise<void>;
  clearFiles: (projectId?: string) => void;
}

export const useProjectFiles = create<ProjectFilesState>((set, get) => ({
  files: {},
  loading: {},
  errors: {},

  getFiles: async (projectId: string) => {
    const { files } = get();

    // If we already have files for this project, return them
    if (files[projectId] && files[projectId].length > 0) {
      return files[projectId];
    }

    // Set loading state
    set((state) => ({
      loading: { ...state.loading, [projectId]: true },
      errors: { ...state.errors, [projectId]: null },
    }));

    try {
      const response = await projectFilesService.getProjectFiles(projectId);
      set((state) => ({
        files: { ...state.files, [projectId]: response.data },
        loading: { ...state.loading, [projectId]: false },
      }));
      return response.data;
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Failed to fetch files";
      set((state) => ({
        loading: { ...state.loading, [projectId]: false },
        errors: { ...state.errors, [projectId]: errorMessage },
      }));
      throw error;
    }
  },

  uploadFile: async (projectId: string, file: File, userId: string) => {
    set((state) => ({
      loading: { ...state.loading, [projectId]: true },
      errors: { ...state.errors, [projectId]: null },
    }));

    try {
      // 1. Convert file to data URL
      const dataUrl = await fileToDataUrl(file);

      // 2. Upload to media API
      const mediaResult = await uploadMedia(dataUrl, file.name, userId);

      // 3. Extract media object ID
      const mediaObjectId = extractMediaObjectId(mediaResult.id);

      // 4. Create project file with the media object
      const createRequest: CreateProjectFileRequest = {
        media_object_id: mediaObjectId,
        filename: file.name,
      };

      const projectFile = await projectFilesService.createProjectFile(
        projectId,
        createRequest,
      );

      // 5. Update state
      set((state) => ({
        files: {
          ...state.files,
          [projectId]: [...(state.files[projectId] || []), projectFile],
        },
        loading: { ...state.loading, [projectId]: false },
      }));

      return projectFile;
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Failed to upload file";
      set((state) => ({
        loading: { ...state.loading, [projectId]: false },
        errors: { ...state.errors, [projectId]: errorMessage },
      }));
      throw error;
    }
  },

  deleteFile: async (projectId: string, fileId: string) => {
    try {
      await projectFilesService.deleteProjectFile(projectId, fileId);

      set((state) => ({
        files: {
          ...state.files,
          [projectId]: (state.files[projectId] || []).filter(
            (f) => f.id !== fileId && f.public_id !== fileId,
          ),
        },
      }));
    } catch (error) {
      console.error("Failed to delete project file:", error);
      throw error;
    }
  },

  reorderFiles: async (projectId: string, fileIds: string[]) => {
    try {
      const fileOrders = fileIds.reduce<Record<string, number>>((acc, id, idx) => {
        acc[id] = idx;
        return acc;
      }, {});

      await projectFilesService.reorderProjectFiles(projectId, {
        file_orders: fileOrders,
      });

      // Reorder local state to match
      set((state) => {
        const currentFiles = state.files[projectId] || [];
        const reorderedFiles = fileIds
          .map((id) => currentFiles.find((f) => f.id === id || f.public_id === id))
          .filter((f): f is ProjectFile => f !== undefined);

        return {
          files: {
            ...state.files,
            [projectId]: reorderedFiles,
          },
        };
      });
    } catch (error) {
      console.error("Failed to reorder project files:", error);
      throw error;
    }
  },

  clearFiles: (projectId?: string) => {
    if (projectId) {
      set((state) => ({
        files: { ...state.files, [projectId]: [] },
        loading: { ...state.loading, [projectId]: false },
        errors: { ...state.errors, [projectId]: null },
      }));
    } else {
      set({ files: {}, loading: {}, errors: {} });
    }
  },
}));
