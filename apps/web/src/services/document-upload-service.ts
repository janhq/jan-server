import { fetchJsonWithAuth } from "@/lib/api-client";
import { uploadMedia, blobUrlToDataUrl } from "./media-upload-service";

declare const JAN_API_BASE_URL: string;

// Document scan request
export type DocumentScanRequest = {
  media_object_id: string;
  filename?: string;
};

// Document scan response from API
export type DocumentScanResponse = {
  id: string;
  public_id: string;
  media_object_id: string;
  filename: string;
  mime_type: string;
  file_size: number;
  processing_status: "pending" | "processing" | "completed" | "failed";
  extracted_text?: string;
  page_count?: number;
  word_count?: number;
  error_message?: string;
  created_at: string;
  updated_at: string;
};

// Document types that require OCR scanning
const DOCUMENT_MIME_TYPES = [
  "application/pdf",
  "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  "text/plain",
  "text/markdown",
  "text/html",
  "application/rtf",
];

/**
 * Check if a MIME type is a document type that requires OCR
 */
export function isDocumentType(mimeType: string): boolean {
  return DOCUMENT_MIME_TYPES.includes(mimeType);
}

/**
 * Scan a document using the OCR service
 *
 * @param mediaObjectId - The media object ID (jan_* ID) from media upload
 * @param filename - Optional filename for display
 * @param abortSignal - Optional abort signal for cancellation
 * @returns DocumentScanResponse with extracted text
 */
export async function scanDocument(
  mediaObjectId: string,
  filename?: string,
  abortSignal?: AbortSignal,
): Promise<DocumentScanResponse> {
  const payload: DocumentScanRequest = {
    media_object_id: mediaObjectId,
    filename,
  };

  const response = await fetchJsonWithAuth<DocumentScanResponse>(
    `${JAN_API_BASE_URL}v1/documents/scan`,
    {
      method: "POST",
      body: JSON.stringify(payload),
      signal: abortSignal,
    },
  );

  return response;
}

/**
 * Get document content by ID
 *
 * @param documentId - The document content ID
 * @param abortSignal - Optional abort signal for cancellation
 * @returns DocumentScanResponse
 */
export async function getDocument(
  documentId: string,
  abortSignal?: AbortSignal,
): Promise<DocumentScanResponse> {
  const response = await fetchJsonWithAuth<DocumentScanResponse>(
    `${JAN_API_BASE_URL}v1/documents/${documentId}`,
    {
      method: "GET",
      signal: abortSignal,
    },
  );

  return response;
}

/**
 * Get just the extracted text content from a document
 *
 * @param documentId - The document content ID
 * @param abortSignal - Optional abort signal for cancellation
 * @returns Object with text content
 */
export async function getDocumentContent(
  documentId: string,
  abortSignal?: AbortSignal,
): Promise<{ text: string }> {
  const response = await fetchJsonWithAuth<{ text: string }>(
    `${JAN_API_BASE_URL}v1/documents/${documentId}/content`,
    {
      method: "GET",
      signal: abortSignal,
    },
  );

  return response;
}

/**
 * Upload and scan a document in one operation
 *
 * This function:
 * 1. Uploads the file to media-api
 * 2. Triggers OCR scanning via the document scan endpoint
 * 3. Returns the document with extracted text
 *
 * @param dataUrl - The base64 data URL of the file
 * @param filename - The original filename
 * @param userId - The user or conversation ID for tracking
 * @param abortSignal - Optional abort signal for cancellation
 * @returns DocumentScanResponse with extracted text
 */
export async function uploadAndScanDocument(
  dataUrl: string,
  filename: string,
  userId: string,
  abortSignal?: AbortSignal,
): Promise<DocumentScanResponse> {
  // 1. Upload to media-api
  const mediaResult = await uploadMedia(dataUrl, filename, userId, abortSignal);

  // Extract media object ID from the URL
  // The mediaResult.id is the direct URL, we need to extract the jan_* ID
  const mediaObjectId = extractMediaObjectId(mediaResult.id);

  // 2. Scan the document
  const scanResult = await scanDocument(mediaObjectId, filename, abortSignal);

  return scanResult;
}

/**
 * Upload and scan a document from a blob URL
 *
 * @param blobUrl - The blob URL of the file
 * @param filename - The original filename
 * @param userId - The user or conversation ID for tracking
 * @param abortSignal - Optional abort signal for cancellation
 * @returns DocumentScanResponse with extracted text
 */
export async function uploadAndScanDocumentFromBlob(
  blobUrl: string,
  filename: string,
  userId: string,
  abortSignal?: AbortSignal,
): Promise<DocumentScanResponse> {
  const dataUrl = await blobUrlToDataUrl(blobUrl);
  return uploadAndScanDocument(dataUrl, filename, userId, abortSignal);
}

/**
 * Extract the media object ID (jan_*) from a media URL
 *
 * The URL format is typically: https://domain/media/jan_XXXX/filename
 * We need to extract "jan_XXXX"
 */
function extractMediaObjectId(url: string): string {
  // Try to find jan_* pattern in the URL
  const match = url.match(/jan_[a-zA-Z0-9]+/);
  if (match) {
    return match[0];
  }

  // If no jan_* found, the URL itself might be the ID
  // This handles cases where the ID is returned directly
  return url;
}

export const documentUploadService = {
  scanDocument,
  getDocument,
  getDocumentContent,
  uploadAndScanDocument,
  uploadAndScanDocumentFromBlob,
  isDocumentType,
};
