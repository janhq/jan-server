package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"jan-server/services/mcp-tools/internal/infrastructure/connectorapi"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

const (
	DriveAPIURL = "https://www.googleapis.com/drive/v3"
)

// DriveMCP handles Google Drive connector MCP tools.
type DriveMCP struct {
	connectorClient *connectorapi.Client
	llmAPIURL       string
	enabled         bool
	httpClient      *http.Client
}

// NewDriveMCP creates a new Drive MCP handler.
func NewDriveMCP(llmAPIURL string, enabled bool) *DriveMCP {
	return &DriveMCP{
		connectorClient: connectorapi.NewClient(llmAPIURL),
		llmAPIURL:       llmAPIURL,
		enabled:         enabled,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterTools registers all Drive MCP tools.
func (d *DriveMCP) RegisterTools(server *mcp.Server) {
	if !d.enabled {
		log.Warn().Msg("Google Drive connector MCP tools disabled")
		return
	}

	d.registerSearchFiles(server)
	d.registerGetFileContent(server)
	d.registerListRecent(server)

	log.Info().Msg("Registered Google Drive connector MCP tools (3 tools)")
}

// getAccessToken gets the Drive access token for the user.
func (d *DriveMCP) getAccessToken(ctx context.Context) (string, error) {
	authToken, ok := ctx.Value("auth_token").(string)
	if !ok || authToken == "" {
		return "", fmt.Errorf("authentication required")
	}

	tokenResp, err := d.connectorClient.GetAccessToken(ctx, authToken, connectorapi.ConnectorTypeGoogleDrive)
	if err != nil {
		return "", fmt.Errorf("Google Drive not connected: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// driveRequest makes a request to the Drive API.
func (d *DriveMCP) driveRequest(ctx context.Context, method, path string) ([]byte, error) {
	accessToken, err := d.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, DriveAPIURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Drive API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type SearchFilesArgs struct {
	Query      string `json:"query"`
	MaxResults *int   `json:"max_results,omitempty"`
}

func (d *DriveMCP) registerSearchFiles(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "drive_search_files",
		Description: "Search files in Google Drive using Drive query syntax (e.g., \"name contains 'report'\", \"mimeType='application/vnd.google-apps.document'\"). Requires Google Drive connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchFilesArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Query == "" {
			return nil, nil, fmt.Errorf("query is required")
		}

		maxResults := 10
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 100 {
			maxResults = *input.MaxResults
		}

		path := fmt.Sprintf("/files?q=%s&pageSize=%d&fields=files(id,name,mimeType,modifiedTime,size,webViewLink,owners)",
			url.QueryEscape(input.Query), maxResults)
		respBody, err := d.driveRequest(ctx, http.MethodGet, path)
		if err != nil {
			metrics.RecordToolCall("drive_search_files", "drive", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("drive_search_files", "drive", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type GetDriveFileContentArgs struct {
	FileID     string  `json:"file_id"`
	ExportType *string `json:"export_type,omitempty"` // text/plain, text/html, application/pdf
}

func (d *DriveMCP) registerGetFileContent(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "drive_get_file_content",
		Description: "Get the content of a file from Google Drive. For Google Docs, use export_type to specify format (text/plain, text/html). Requires Google Drive connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetDriveFileContentArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.FileID == "" {
			return nil, nil, fmt.Errorf("file_id is required")
		}

		accessToken, err := d.getAccessToken(ctx)
		if err != nil {
			metrics.RecordToolCall("drive_get_file_content", "drive", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		// First, get file metadata to determine type
		metaPath := fmt.Sprintf("/files/%s?fields=id,name,mimeType,size", input.FileID)
		metaBody, err := d.driveRequest(ctx, http.MethodGet, metaPath)
		if err != nil {
			metrics.RecordToolCall("drive_get_file_content", "drive", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		var fileMeta struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			MimeType string `json:"mimeType"`
			Size     int64  `json:"size"`
		}
		if err := json.Unmarshal(metaBody, &fileMeta); err != nil {
			return nil, nil, fmt.Errorf("parse metadata: %w", err)
		}

		var contentURL string
		isGoogleDoc := false

		// Check if it's a Google Doc type
		switch fileMeta.MimeType {
		case "application/vnd.google-apps.document",
			"application/vnd.google-apps.spreadsheet",
			"application/vnd.google-apps.presentation":
			isGoogleDoc = true
			exportType := "text/plain"
			if input.ExportType != nil && *input.ExportType != "" {
				exportType = *input.ExportType
			}
			contentURL = fmt.Sprintf("%s/files/%s/export?mimeType=%s", DriveAPIURL, input.FileID, url.QueryEscape(exportType))
		default:
			// Regular file download
			contentURL = fmt.Sprintf("%s/files/%s?alt=media", DriveAPIURL, input.FileID)
		}

		// Fetch content
		contentReq, err := http.NewRequestWithContext(ctx, http.MethodGet, contentURL, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("create content request: %w", err)
		}
		contentReq.Header.Set("Authorization", "Bearer "+accessToken)

		contentResp, err := d.httpClient.Do(contentReq)
		if err != nil {
			metrics.RecordToolCall("drive_get_file_content", "drive", "error", time.Since(startTime).Seconds())
			return nil, nil, fmt.Errorf("fetch content: %w", err)
		}
		defer contentResp.Body.Close()

		if contentResp.StatusCode >= 400 {
			body, _ := io.ReadAll(contentResp.Body)
			metrics.RecordToolCall("drive_get_file_content", "drive", "error", time.Since(startTime).Seconds())
			return nil, nil, fmt.Errorf("Drive API error %d: %s", contentResp.StatusCode, string(body))
		}

		// Limit content size
		limitedReader := io.LimitReader(contentResp.Body, 100*1024) // 100KB limit
		content, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, nil, fmt.Errorf("read content: %w", err)
		}

		result := map[string]interface{}{
			"id":            fileMeta.ID,
			"name":          fileMeta.Name,
			"mime_type":     fileMeta.MimeType,
			"is_google_doc": isGoogleDoc,
			"content":       string(content),
			"truncated":     len(content) >= 100*1024,
		}

		metrics.RecordToolCall("drive_get_file_content", "drive", "success", time.Since(startTime).Seconds())
		return nil, result, nil
	})
}

type ListRecentArgs struct {
	MaxResults *int `json:"max_results,omitempty"`
}

func (d *DriveMCP) registerListRecent(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "drive_list_recent",
		Description: "List recently modified files in Google Drive. Requires Google Drive connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListRecentArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		maxResults := 10
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 50 {
			maxResults = *input.MaxResults
		}

		path := fmt.Sprintf("/files?orderBy=modifiedTime desc&pageSize=%d&fields=files(id,name,mimeType,modifiedTime,size,webViewLink,owners)", maxResults)
		respBody, err := d.driveRequest(ctx, http.MethodGet, path)
		if err != nil {
			metrics.RecordToolCall("drive_list_recent", "drive", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("drive_list_recent", "drive", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}
