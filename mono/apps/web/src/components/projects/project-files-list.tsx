import { useEffect, useRef, useState } from "react";
import { useProjectFiles } from "@/stores/project-files-store";
import type { ProjectFile } from "@/services/project-files-service";
import { Button } from "@janhq/interfaces/button";
import {
  FileIcon,
  FileTextIcon,
  Trash2Icon,
  Loader2Icon,
  AlertCircleIcon,
  UploadIcon,
  FilesIcon,
} from "lucide-react";
import { toast } from "@janhq/interfaces/sonner";
import { cn } from "@/lib/utils";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@janhq/interfaces/dialog";

// Accepted document types for project files
const ACCEPTED_DOCUMENT_TYPES = [
  "application/pdf",
  "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  "text/plain",
  "text/markdown",
  "text/html",
  "application/rtf",
].join(",");

const MAX_FILE_SIZE = 50 * 1024 * 1024; // 50MB

interface ProjectFilesListProps {
  projectId: string;
  userId: string;
}

function getFileIcon(mimeType?: string) {
  if (!mimeType) return <FileIcon className="size-4" />;

  if (mimeType.includes("pdf")) {
    return <FileTextIcon className="size-4 text-red-500" />;
  }
  if (mimeType.includes("word") || mimeType.includes("document")) {
    return <FileTextIcon className="size-4 text-blue-500" />;
  }
  if (mimeType.includes("sheet") || mimeType.includes("excel")) {
    return <FileTextIcon className="size-4 text-green-500" />;
  }
  if (mimeType.includes("presentation") || mimeType.includes("powerpoint")) {
    return <FileTextIcon className="size-4 text-orange-500" />;
  }
  if (mimeType.startsWith("text/")) {
    return <FileTextIcon className="size-4 text-gray-500" />;
  }

  return <FileIcon className="size-4" />;
}

function formatFileSize(bytes?: number): string {
  if (!bytes) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function FileItem({
  file,
  onDelete,
}: {
  file: ProjectFile;
  onDelete: (fileId: string) => void;
}) {
  const [isDeleting, setIsDeleting] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const content = file.document_content;
  const filename = content?.filename || "Unknown file";
  const status = content?.processing_status;
  const isProcessing = status === "pending" || status === "processing";
  const isFailed = status === "failed";

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await onDelete(file.id);
      toast.success("File removed from project");
    } catch {
      toast.error("Failed to remove file");
    } finally {
      setIsDeleting(false);
      setShowDeleteDialog(false);
    }
  };

  return (
    <>
      <div
        className={cn(
          "group flex items-center gap-3 p-3 rounded-lg border bg-background hover:bg-accent/50 transition-colors",
          isFailed && "border-destructive/50 bg-destructive/5",
        )}
      >
        <div className="shrink-0">
          {isProcessing ? (
            <Loader2Icon className="size-4 animate-spin text-muted-foreground" />
          ) : isFailed ? (
            <AlertCircleIcon className="size-4 text-destructive" />
          ) : (
            getFileIcon(content?.mime_type)
          )}
        </div>

        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium truncate">{filename}</p>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            {content?.file_size && (
              <span>{formatFileSize(content.file_size)}</span>
            )}
            {content?.page_count && <span>{content.page_count} pages</span>}
            {isProcessing && <span className="text-blue-500">Processing...</span>}
            {isFailed && (
              <span className="text-destructive">
                {content?.error_message || "Processing failed"}
              </span>
            )}
          </div>
        </div>

        <Button
          variant="ghost"
          size="icon"
          className="shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
          onClick={() => setShowDeleteDialog(true)}
          disabled={isDeleting}
        >
          {isDeleting ? (
            <Loader2Icon className="size-4 animate-spin" />
          ) : (
            <Trash2Icon className="size-4 text-muted-foreground hover:text-destructive" />
          )}
        </Button>
      </div>

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Remove file from project?</DialogTitle>
            <DialogDescription>
              This will remove &quot;{filename}&quot; from the project. The file
              will no longer be available as context for conversations.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline" className="rounded-full">
                Cancel
              </Button>
            </DialogClose>
            <Button
              onClick={handleDelete}
              variant="destructive"
              className="rounded-full"
            >
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function ProjectFilesList({ projectId, userId }: ProjectFilesListProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isUploading, setIsUploading] = useState(false);

  const { files, loading, errors, getFiles, uploadFile, deleteFile } =
    useProjectFiles();

  const projectFiles = files[projectId] || [];
  const isLoading = loading[projectId];
  const error = errors[projectId];

  useEffect(() => {
    if (projectId) {
      getFiles(projectId).catch(() => {
        // Error is handled in store
      });
    }
  }, [projectId, getFiles]);

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFiles = e.target.files;
    if (!selectedFiles || selectedFiles.length === 0) return;

    const file = selectedFiles[0];

    // Validate file type
    if (!ACCEPTED_DOCUMENT_TYPES.split(",").includes(file.type)) {
      toast.error("File type not supported. Please upload a document.");
      return;
    }

    // Validate file size
    if (file.size > MAX_FILE_SIZE) {
      toast.error("File size must be under 50MB.");
      return;
    }

    setIsUploading(true);
    try {
      await uploadFile(projectId, file, userId);
      toast.success("File uploaded successfully");
    } catch {
      toast.error("Failed to upload file");
    } finally {
      setIsUploading(false);
      // Reset input
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  const handleDelete = async (fileId: string) => {
    await deleteFile(projectId, fileId);
  };

  if (isLoading && projectFiles.length === 0) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2Icon className="size-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <AlertCircleIcon className="size-6 text-destructive mb-2" />
        <p className="text-sm text-destructive">{error}</p>
        <Button
          variant="outline"
          size="sm"
          className="mt-4"
          onClick={() => getFiles(projectId)}
        >
          Retry
        </Button>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      <input
        ref={fileInputRef}
        type="file"
        accept={ACCEPTED_DOCUMENT_TYPES}
        className="hidden"
        onChange={handleFileChange}
      />

      <div className="flex items-center justify-between mb-3">
        <span className="text-base font-semibold inline-block">Files</span>
        <Button
          variant="outline"
          size="sm"
          className="rounded-full"
          onClick={handleUploadClick}
          disabled={isUploading}
        >
          {isUploading ? (
            <>
              <Loader2Icon className="size-4 animate-spin" />
              <span>Uploading...</span>
            </>
          ) : (
            <>
              <UploadIcon className="size-4" />
              <span>Upload</span>
            </>
          )}
        </Button>
      </div>

      {projectFiles.length === 0 ? (
        <div className="flex-1 bg-muted/50 rounded-2xl flex items-center justify-center text-center px-4 py-6">
          <div className="px-8 w-full">
            <FilesIcon className="text-muted-foreground size-6 mx-auto mb-2" />
            <p className="text-base mb-2">Add files to this project</p>
            <p className="text-sm text-muted-foreground">
              Upload documents that provide Jan with context for more accurate
              answers
            </p>
          </div>
        </div>
      ) : (
        <div className="flex-1 space-y-2 overflow-y-auto">
          {projectFiles.map((file) => (
            <FileItem key={file.id} file={file} onDelete={handleDelete} />
          ))}
        </div>
      )}
    </div>
  );
}
