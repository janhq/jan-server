// Package media provides a client for interacting with the media-api service.
package media

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"

	"jan-server/services/response-api/internal/domain/plan"
)

// Client implements the media-api client interface.
type Client struct {
	httpClient *resty.Client
	baseURL    string
}

// NewClient constructs a new media-api client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: resty.New().
			SetBaseURL(baseURL).
			SetHeader("Content-Type", "application/json"),
	}
}

// UploadRequest contains the parameters for uploading an artifact.
type UploadRequest struct {
	Content        []byte // Raw file content
	Filename       string
	ContentType    string // MIME type
	ConversationID string
	ResponseID     string
	UserID         string
	Metadata       map[string]interface{}
}

// UploadResponse contains the result of an upload operation.
type UploadResponse struct {
	ID          string `json:"id"`           // jan_file_xxx format
	DownloadURL string `json:"download_url"` // Accessible URL
	Size        int64  `json:"bytes"`
	MimeType    string `json:"mime"`
}

// UploadArtifact uploads content to media-api and returns artifact metadata.
func (c *Client) UploadArtifact(ctx context.Context, req *UploadRequest) (*plan.MediaArtifact, error) {
	if req == nil || len(req.Content) == 0 {
		return nil, fmt.Errorf("empty content")
	}

	// Determine content type if not provided
	contentType := req.ContentType
	if contentType == "" {
		contentType = detectContentType(req.Filename)
	}

	// Build data URL for media-api ingest endpoint
	dataURL := fmt.Sprintf("data:%s;base64,%s",
		contentType,
		base64.StdEncoding.EncodeToString(req.Content))

	payload := map[string]interface{}{
		"source": map[string]interface{}{
			"type":     "data_url",
			"data_url": dataURL,
		},
		"filename": req.Filename,
		"user_id":  req.UserID,
	}

	log.Debug().
		Str("filename", req.Filename).
		Str("content_type", contentType).
		Int("size", len(req.Content)).
		Str("response_id", req.ResponseID).
		Msg("Uploading artifact to media-api")

	httpReq := c.httpClient.R().
		SetContext(ctx).
		SetBody(payload)

	resp, err := httpReq.Post("/v1/files/ingest")
	if err != nil {
		return nil, fmt.Errorf("media-api request failed: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("media-api error: %s", resp.String())
	}

	var result UploadResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse media-api response: %w", err)
	}

	// Build download URL if not provided
	downloadURL := result.DownloadURL
	if downloadURL == "" {
		downloadURL = fmt.Sprintf("%s/v1/files/%s/content", c.baseURL, result.ID)
	}

	return &plan.MediaArtifact{
		ID:          result.ID,
		Type:        detectArtifactType(req.Filename, contentType),
		Filename:    req.Filename,
		DownloadURL: downloadURL,
		Size:        result.Size,
		ContentType: contentType,
	}, nil
}

// UploadFromPath uploads a file from a sandbox/container path.
// This is used when tools write files to ephemeral paths like /tmp.
type SandboxFileReader interface {
	ReadFile(ctx context.Context, path string) ([]byte, error)
}

// UploadFromSandbox reads a file from sandbox and uploads it to media-api.
func (c *Client) UploadFromSandbox(ctx context.Context, sandboxReader SandboxFileReader, path string, req *UploadRequest) (*plan.MediaArtifact, error) {
	content, err := sandboxReader.ReadFile(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from sandbox: %w", err)
	}

	req.Content = content
	if req.Filename == "" {
		req.Filename = filepath.Base(path)
	}

	return c.UploadArtifact(ctx, req)
}

// detectContentType determines MIME type from filename.
func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Common programming file extensions
	codeTypes := map[string]string{
		".py":    "text/x-python",
		".js":    "text/javascript",
		".ts":    "text/typescript",
		".go":    "text/x-go",
		".java":  "text/x-java",
		".rb":    "text/x-ruby",
		".rs":    "text/x-rust",
		".cpp":   "text/x-c++src",
		".c":     "text/x-csrc",
		".h":     "text/x-chdr",
		".hpp":   "text/x-c++hdr",
		".cs":    "text/x-csharp",
		".php":   "text/x-php",
		".swift": "text/x-swift",
		".kt":    "text/x-kotlin",
		".scala": "text/x-scala",
		".sql":   "text/x-sql",
		".sh":    "text/x-shellscript",
		".bash":  "text/x-shellscript",
		".ps1":   "text/x-powershell",
		".yaml":  "text/yaml",
		".yml":   "text/yaml",
		".toml":  "text/toml",
		".json":  "application/json",
		".xml":   "application/xml",
		".html":  "text/html",
		".css":   "text/css",
		".md":    "text/markdown",
		".txt":   "text/plain",
	}

	if mimeType, ok := codeTypes[ext]; ok {
		return mimeType
	}

	// Try standard mime type detection
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		return mimeType
	}

	return "application/octet-stream"
}

// detectArtifactType determines artifact type from filename and MIME type.
func detectArtifactType(filename, mimeType string) string {
	// Check MIME type first
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.HasPrefix(mimeType, "text/x-") ||
		strings.Contains(mimeType, "javascript") ||
		strings.Contains(mimeType, "typescript"):
		return "code"
	case mimeType == "application/pdf":
		return "document"
	case strings.Contains(mimeType, "spreadsheet") ||
		strings.Contains(mimeType, "csv"):
		return "data"
	case strings.Contains(mimeType, "presentation"):
		return "presentation"
	case strings.Contains(mimeType, "document") ||
		strings.Contains(mimeType, "word"):
		return "document"
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filename))
	codeExts := []string{".py", ".js", ".ts", ".go", ".java", ".rb", ".rs", ".cpp", ".c", ".cs", ".php", ".swift", ".kt", ".scala", ".sql", ".sh", ".bash", ".ps1"}
	for _, e := range codeExts {
		if ext == e {
			return "code"
		}
	}

	switch ext {
	case ".md", ".txt":
		return "text"
	case ".json", ".yaml", ".yml", ".toml", ".xml":
		return "data"
	case ".html", ".css":
		return "web"
	case ".pptx", ".ppt":
		return "presentation"
	case ".docx", ".doc", ".pdf":
		return "document"
	case ".xlsx", ".xls", ".csv":
		return "spreadsheet"
	}

	return "file"
}
