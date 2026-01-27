import { fetchJsonWithAuth } from "@/lib/api-client";

declare const JAN_API_BASE_URL: string;

// Document scan request
export type DocumentScanRequest = {
  media_object_id: string;
  filename?: string;
};

// Document scan response from API
export type DocumentScanResponse = {
  id: string;
  object?: string;
  media_object_id: string;
  filename: string;
  mime_type: string;
  file_size?: number;
  processing_status: "pending" | "processing" | "completed" | "failed";
  extracted_text?: string;
  page_count?: number;
  word_count?: number;
  error_message?: string;
  created_at?: string;
  updated_at?: string;
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

