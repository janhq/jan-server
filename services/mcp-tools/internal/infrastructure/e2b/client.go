package e2b

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps HTTP calls to the e2b-service
type Client struct {
	baseURL    string
	httpClient *http.Client
	enabled    bool
}

// ClientConfig holds E2B client configuration
type ClientConfig struct {
	BaseURL string
	Timeout time.Duration
	Enabled bool
}

// NewClient creates a new E2B service client
func NewClient(cfg ClientConfig) *Client {
	if !cfg.Enabled || cfg.BaseURL == "" {
		return &Client{enabled: false}
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		enabled: true,
	}
}

// IsEnabled returns whether the client is configured
func (c *Client) IsEnabled() bool {
	return c != nil && c.enabled
}

// BaseURL returns the configured base URL
func (c *Client) BaseURL() string {
	if c == nil {
		return ""
	}
	return c.baseURL
}

// CreateSandboxRequest represents the request to create a sandbox
type CreateSandboxRequest struct {
	Timeout int `json:"timeout,omitempty"`
}

// CreateSandboxResponse represents the response from creating a sandbox
type CreateSandboxResponse struct {
	SandboxID  string `json:"sandbox_id"`
	Status     string `json:"status"`
	ViewURL    string `json:"view_url,omitempty"`    // View-only VNC URL (no interaction)
	ControlURL string `json:"control_url,omitempty"` // Full control VNC URL (takeover)
}

// CreateSandbox creates a new E2B sandbox
func (c *Client) CreateSandbox(ctx context.Context, timeout int) (*CreateSandboxResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := CreateSandboxRequest{Timeout: timeout}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result CreateSandboxResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// SandboxStatus represents the status of a sandbox
type SandboxStatus struct {
	SandboxID  string `json:"sandbox_id"`
	Status     string `json:"status"`
	ViewURL    string `json:"view_url,omitempty"`    // View-only VNC URL (no interaction)
	ControlURL string `json:"control_url,omitempty"` // Full control VNC URL (takeover)
}

// GetSandboxStatus gets the status of a sandbox
func (c *Client) GetSandboxStatus(ctx context.Context, sandboxID string) (*SandboxStatus, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/sandboxes/"+sandboxID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result SandboxStatus
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// StopSandbox stops and cleans up a sandbox
func (c *Client) StopSandbox(ctx context.Context, sandboxID string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("E2B client not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/api/v1/sandboxes/"+sandboxID, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// RunCommandRequest represents the request to run a command
type RunCommandRequest struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// RunCommandResponse represents the response from running a command
type RunCommandResponse struct {
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
}

// RunCommand executes a shell command in the sandbox
func (c *Client) RunCommand(ctx context.Context, sandboxID, command string, timeout int) (*RunCommandResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := RunCommandRequest{
		Command: command,
		Timeout: timeout,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes/"+sandboxID+"/command", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result RunCommandResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// ReadFileRequest represents the request to read a file
type ReadFileRequest struct {
	Path string `json:"path"`
}

// ReadFileResponse represents the response from reading a file
type ReadFileResponse struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Error    string `json:"error,omitempty"`
}

// ReadFile reads a file from the sandbox
func (c *Client) ReadFile(ctx context.Context, sandboxID, path string) (*ReadFileResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := ReadFileRequest{Path: path}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes/"+sandboxID+"/files/read", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result ReadFileResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// WriteFileRequest represents the request to write a file
type WriteFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WriteFileResponse represents the response from writing a file
type WriteFileResponse struct {
	Path         string `json:"path"`
	Status       string `json:"status"`
	BytesWritten int    `json:"bytes_written,omitempty"`
	Error        string `json:"error,omitempty"`
}

// WriteFile writes content to a file in the sandbox
func (c *Client) WriteFile(ctx context.Context, sandboxID, path, content string) (*WriteFileResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := WriteFileRequest{
		Path:    path,
		Content: content,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes/"+sandboxID+"/files/write", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result WriteFileResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// ListFilesRequest represents the request to list files
type ListFilesRequest struct {
	Path string `json:"path"`
}

// FileInfo represents information about a file
type FileInfo struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

// ListFilesResponse represents the response from listing files
type ListFilesResponse struct {
	Path  string     `json:"path"`
	Files []FileInfo `json:"files"`
	Error string     `json:"error,omitempty"`
}

// ListFiles lists files in a directory in the sandbox
func (c *Client) ListFiles(ctx context.Context, sandboxID, path string) (*ListFilesResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := ListFilesRequest{Path: path}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes/"+sandboxID+"/files/list", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result ListFilesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// ScreenshotResponse represents the response from taking a screenshot
type ScreenshotResponse struct {
	Image    string `json:"image"`
	Format   string `json:"format"`
	Encoding string `json:"encoding"`
	Error    string `json:"error,omitempty"`
}

// Screenshot takes a screenshot of the sandbox desktop
func (c *Client) Screenshot(ctx context.Context, sandboxID string) (*ScreenshotResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes/"+sandboxID+"/screenshot", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result ScreenshotResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// OpenURLRequest represents the request to open a URL
type OpenURLRequest struct {
	URL string `json:"url"`
}

// OpenURLResponse represents the response from opening a URL
type OpenURLResponse struct {
	URL    string `json:"url"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// OpenURL opens a URL in the sandbox browser
func (c *Client) OpenURL(ctx context.Context, sandboxID, url string) (*OpenURLResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := OpenURLRequest{URL: url}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes/"+sandboxID+"/browser/open", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result OpenURLResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// ClickRequest represents the request to perform a mouse click
type ClickRequest struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Button string `json:"button,omitempty"`
}

// ClickResponse represents the response from a mouse click
type ClickResponse struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Button string `json:"button"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Click performs a mouse click in the sandbox
func (c *Client) Click(ctx context.Context, sandboxID string, x, y int, button string) (*ClickResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	if button == "" {
		button = "left"
	}

	reqBody := ClickRequest{
		X:      x,
		Y:      y,
		Button: button,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes/"+sandboxID+"/input/click", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result ClickResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// TypeTextRequest represents the request to type text
type TypeTextRequest struct {
	Text string `json:"text"`
}

// TypeTextResponse represents the response from typing text
type TypeTextResponse struct {
	Text   string `json:"text"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// TypeText types text in the sandbox
func (c *Client) TypeText(ctx context.Context, sandboxID, text string) (*TypeTextResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := TypeTextRequest{Text: text}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes/"+sandboxID+"/input/type", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result TypeTextResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
	ID      interface{}            `json:"id,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	Result  interface{}            `json:"result,omitempty"`
	Error   map[string]interface{} `json:"error,omitempty"`
	ID      interface{}            `json:"id,omitempty"`
}

// CallMCP calls an MCP method on the sandbox
func (c *Client) CallMCP(ctx context.Context, sandboxID, method string, params map[string]interface{}) (*MCPResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := MCPRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sandboxes/"+sandboxID+"/mcp", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result MCPResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// === User-centric API methods ===

// SandboxRecord represents a user's sandbox from the database
type SandboxRecord struct {
	PublicID       string  `json:"public_id"`
	UserID         string  `json:"user_id"`
	E2BSandboxID   *string `json:"e2b_sandbox_id,omitempty"`
	Status         string  `json:"status"`
	ViewURL        *string `json:"view_url,omitempty"`    // View-only VNC URL (no interaction)
	ControlURL     *string `json:"control_url,omitempty"` // Full control VNC URL (takeover)
	SandboxURL     *string `json:"sandbox_url,omitempty"` // External sandbox URL for MCP proxy
	StartedAt      *string `json:"started_at,omitempty"`
	TimeoutAt      *string `json:"timeout_at,omitempty"`
	PausedAt       *string `json:"paused_at,omitempty"`
	PauseExpiresAt *string `json:"pause_expires_at,omitempty"`
	ErrorMessage   *string `json:"error_message,omitempty"`
	CreatedAt      *string `json:"created_at,omitempty"`
}

// WorkspaceInfo represents workspace information
type WorkspaceInfo struct {
	PublicID       string  `json:"public_id"`
	ConversationID string  `json:"conversation_id"`
	WorkspacePath  string  `json:"workspace_path"`
	CreatedAt      *string `json:"created_at,omitempty"`
}

// WorkspaceWithSandbox represents workspace with its parent sandbox
type WorkspaceWithSandbox struct {
	Sandbox   SandboxRecord `json:"sandbox"`
	Workspace WorkspaceInfo `json:"workspace"`
}

// GetSandboxByUser gets a user's sandbox
func (c *Client) GetSandboxByUser(ctx context.Context, userID string) (*SandboxRecord, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/users/"+userID+"/sandbox", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == 404 {
		return nil, nil // No sandbox found
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result SandboxRecord
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// StartSandboxRequest represents request to start a sandbox
type StartSandboxRequest struct {
	Timeout int `json:"timeout,omitempty"`
}

// StartSandboxForUser starts a sandbox for a user
func (c *Client) StartSandboxForUser(ctx context.Context, userID string, timeout int) (*SandboxRecord, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	var body []byte
	var err error
	if timeout > 0 {
		reqBody := StartSandboxRequest{Timeout: timeout}
		body, err = json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/users/"+userID+"/sandbox", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result SandboxRecord
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// StopSandboxForUser stops a user's sandbox
func (c *Client) StopSandboxForUser(ctx context.Context, userID string) (*SandboxRecord, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/api/v1/users/"+userID+"/sandbox", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result SandboxRecord
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// PauseSandbox pauses a user's sandbox
func (c *Client) PauseSandbox(ctx context.Context, userID string) (*SandboxRecord, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/users/"+userID+"/sandbox/pause", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result SandboxRecord
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// ResumeSandboxRequest represents request to resume a sandbox
type ResumeSandboxRequest struct {
	Timeout int `json:"timeout,omitempty"`
}

// ResumeSandbox resumes a paused sandbox
func (c *Client) ResumeSandbox(ctx context.Context, userID string, timeout int) (*SandboxRecord, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	var body []byte
	var err error
	if timeout > 0 {
		reqBody := ResumeSandboxRequest{Timeout: timeout}
		body, err = json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/users/"+userID+"/sandbox/resume", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result SandboxRecord
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// ExtendTimeoutRequest represents request to extend sandbox timeout
type ExtendTimeoutRequest struct {
	AdditionalSeconds int `json:"additional_seconds"`
}

// ExtendTimeout extends the timeout of a running sandbox
func (c *Client) ExtendTimeout(ctx context.Context, userID string, additionalSeconds int) (*SandboxRecord, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := ExtendTimeoutRequest{AdditionalSeconds: additionalSeconds}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/users/"+userID+"/sandbox/extend", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result SandboxRecord
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// ResetSandbox resets a user's sandbox (stops, deletes, starts fresh)
func (c *Client) ResetSandbox(ctx context.Context, userID string) (*SandboxRecord, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/users/"+userID+"/sandbox/reset", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result SandboxRecord
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// GetOrCreateWorkspace gets or creates a workspace for a conversation
func (c *Client) GetOrCreateWorkspace(ctx context.Context, userID, conversationID string) (*WorkspaceWithSandbox, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox/workspace/%s", c.baseURL, userID, conversationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result WorkspaceWithSandbox
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// === User-centric MCP Proxy API ===

// MCPToolDefinition represents an MCP tool definition from e2b-api
type MCPToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// CallUserMCP sends an MCP JSON-RPC request to e2b-api for a specific user.
// This proxies MCP calls to /api/v1/users/{user_id}/mcp which automatically
// handles sandbox lifecycle (start/resume if needed).
func (c *Client) CallUserMCP(ctx context.Context, userID, method string, params map[string]interface{}) (*MCPResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	reqBody := MCPRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox/mcp", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result MCPResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// ListUserTools fetches available MCP tools from e2b-api for a user.
// This calls tools/list on the user's sandbox MCP endpoint.
func (c *Client) ListUserTools(ctx context.Context, userID string) ([]MCPToolDefinition, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	resp, err := c.CallUserMCP(ctx, userID, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("call tools/list: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: code=%v message=%v", resp.Error["code"], resp.Error["message"])
	}

	// Parse result.tools array
	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	toolsRaw, ok := resultMap["tools"]
	if !ok {
		return nil, nil // No tools
	}

	toolsArray, ok := toolsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected tools type: %T", toolsRaw)
	}

	var tools []MCPToolDefinition
	for _, t := range toolsArray {
		toolMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		tool := MCPToolDefinition{
			Name:        getString(toolMap, "name"),
			Description: getString(toolMap, "description"),
		}

		if schema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
			tool.InputSchema = schema
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

// CallUserToolMCP calls a specific tool via MCP for a user.
// This is a convenience method that wraps CallUserMCP with tools/call method.
func (c *Client) CallUserToolMCP(ctx context.Context, userID, toolName string, arguments map[string]interface{}) (*MCPResponse, error) {
	return c.CallUserMCP(ctx, userID, "tools/call", map[string]interface{}{
		"name":      toolName,
		"arguments": arguments,
	})
}

// getString safely gets a string from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// GetStaticTools fetches static E2B tool definitions from e2b-api.
// This endpoint doesn't require a running sandbox - it returns the
// predefined tool definitions for discovery at startup.
func (c *Client) GetStaticTools(ctx context.Context) ([]MCPToolDefinition, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/mcp/tools", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response: {"tools": [...]}
	var result struct {
		Tools []struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			InputSchema map[string]interface{} `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	tools := make([]MCPToolDefinition, 0, len(result.Tools))
	for _, t := range result.Tools {
		tools = append(tools, MCPToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	return tools, nil
}
