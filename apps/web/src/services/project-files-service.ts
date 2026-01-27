import { fetchJsonWithAuth } from "@/lib/api-client";

declare const JAN_API_BASE_URL: string;

// Project file response from API
export type ProjectFile = {
  id: string;
  object?: string;
  project_id?: string;
  display_order: number;
  created_by?: string;
  created_at?: number | string;
  updated_at?: string;
  // Legacy/back-compat fields (may not be present)
  public_id?: string;
  document_content: {
    id: string;
    object?: string;
    media_object_id?: string;
    filename?: string;
    mime_type?: string;
    file_size?: number;
    processing_status: "pending" | "processing" | "completed" | "failed";
    extracted_text?: string;
    page_count?: number;
    word_count?: number;
    error_message?: string;
    created_at?: string;
    updated_at?: string;
  } | null;
};

// List response with pagination
export type ProjectFilesListResponse = {
  data: ProjectFile[];
  has_more: boolean;
  last_id?: string;
};

// Create project file request
export type CreateProjectFileRequest = {
  media_object_id: string;
  filename?: string;
};

// Reorder project files request
export type ReorderProjectFilesRequest = {
  file_orders: Record<string, number>;
};

/**
 * Get files for a project
 */
export async function getProjectFiles(
  projectId: string,
  abortSignal?: AbortSignal,
): Promise<ProjectFilesListResponse> {
  const response = await fetchJsonWithAuth<ProjectFilesListResponse>(
    `${JAN_API_BASE_URL}v1/projects/${projectId}/files`,
    {
      method: "GET",
      signal: abortSignal,
    },
  );
  return response;
}

/**
 * Get a single project file by ID
 */
export async function getProjectFile(
  projectId: string,
  fileId: string,
  abortSignal?: AbortSignal,
): Promise<ProjectFile> {
  const response = await fetchJsonWithAuth<ProjectFile>(
    `${JAN_API_BASE_URL}v1/projects/${projectId}/files/${fileId}`,
    {
      method: "GET",
      signal: abortSignal,
    },
  );
  return response;
}

/**
 * Add a file to a project
 */
export async function createProjectFile(
  projectId: string,
  data: CreateProjectFileRequest,
  abortSignal?: AbortSignal,
): Promise<ProjectFile> {
  const response = await fetchJsonWithAuth<ProjectFile>(
    `${JAN_API_BASE_URL}v1/projects/${projectId}/files`,
    {
      method: "POST",
      body: JSON.stringify(data),
      signal: abortSignal,
    },
  );
  return response;
}

/**
 * Delete a project file
 */
export async function deleteProjectFile(
  projectId: string,
  fileId: string,
  abortSignal?: AbortSignal,
): Promise<void> {
  await fetchJsonWithAuth<void>(
    `${JAN_API_BASE_URL}v1/projects/${projectId}/files/${fileId}`,
    {
      method: "DELETE",
      signal: abortSignal,
    },
  );
}

/**
 * Reorder project files
 */
export async function reorderProjectFiles(
  projectId: string,
  data: ReorderProjectFilesRequest,
  abortSignal?: AbortSignal,
): Promise<{ success: boolean }> {
  const response = await fetchJsonWithAuth<{ success: boolean }>(
    `${JAN_API_BASE_URL}v1/projects/${projectId}/files/reorder`,
    {
      method: "PATCH",
      body: JSON.stringify(data),
      signal: abortSignal,
    },
  );
  return response;
}

export const projectFilesService = {
  getProjectFiles,
  getProjectFile,
  createProjectFile,
  deleteProjectFile,
  reorderProjectFiles,
};
