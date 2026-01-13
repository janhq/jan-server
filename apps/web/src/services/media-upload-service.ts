import { fetchJsonWithAuth } from "@/lib/api-client";

declare const JAN_API_BASE_URL: string;

// Response from the media upload API
type MediaUploadAPIResponse = {
  mime: string;
  bytes: number;
  deduped: boolean;
  url: string;
};

// Normalized response used by the app
export type MediaUploadResponse = MediaUploadAPIResponse & {
  id: string;
};

// Request payload for media upload
export type MediaUploadRequest = {
  source: {
    type: "data_url";
    data_url: string;
  };
  filename: string;
  user_id: string;
};

// Upload status for tracking
export type UploadStatus = "pending" | "uploading" | "completed" | "failed";

// Error type for upload failures
export type MediaUploadError = {
  code: "network" | "timeout" | "server" | "unknown";
  message: string;
};

/**
 * Convert a File to a base64 data URL
 */
export async function fileToDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onloadend = () => {
      if (typeof reader.result === "string") {
        resolve(reader.result);
      } else {
        reject(new Error("Failed to convert file to data URL"));
      }
    };
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}

/**
 * Convert a blob URL to a base64 data URL
 */
export async function blobUrlToDataUrl(blobUrl: string): Promise<string> {
  const response = await fetch(blobUrl);
  const blob = await response.blob();
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onloadend = () => {
      if (typeof reader.result === "string") {
        resolve(reader.result);
      } else {
        reject(new Error("Failed to convert blob to data URL"));
      }
    };
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(blob);
  });
}

/**
 * Upload media to the Jan API
 *
 * @param dataUrl - The base64 data URL of the file (e.g., "data:image/jpeg;base64,...")
 * @param filename - The original filename
 * @param userId - The user or conversation ID for tracking
 * @param abortSignal - Optional abort signal for cancellation
 * @returns MediaUploadResponse with the media URL
 */
export async function uploadMedia(
  dataUrl: string,
  filename: string,
  userId: string,
  abortSignal?: AbortSignal,
): Promise<MediaUploadResponse> {
  const payload: MediaUploadRequest = {
    source: {
      type: "data_url",
      data_url: dataUrl,
    },
    filename,
    user_id: userId,
  };

  const response = await fetchJsonWithAuth<MediaUploadAPIResponse>(
    `${JAN_API_BASE_URL}media/v1/media`,
    {
      method: "POST",
      body: JSON.stringify(payload),
      signal: abortSignal,
    },
  );
  const directUrl = response.url;
  return {
    ...response,
    id: directUrl,
    url: directUrl,
  };
}

/**
 * Upload a File object directly
 *
 * @param file - The File object to upload
 * @param userId - The user or conversation ID for tracking
 * @param abortSignal - Optional abort signal for cancellation
 * @returns MediaUploadResponse with the media URL
 */
export async function uploadFile(
  file: File,
  userId: string,
  abortSignal?: AbortSignal,
): Promise<MediaUploadResponse> {
  const dataUrl = await fileToDataUrl(file);
  return uploadMedia(dataUrl, file.name, userId, abortSignal);
}

/**
 * Use the direct media URL for chat messages.
 *
 * @param mediaUrl - The media URL returned from upload
 * @returns Direct media URL
 */
export function createJanMediaUrl(mediaUrl: string, _mimeType: string): string {
  return mediaUrl;
}

export const mediaUploadService = {
  uploadMedia,
  uploadFile,
  fileToDataUrl,
  blobUrlToDataUrl,
  createJanMediaUrl,
};
