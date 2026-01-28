package e2b

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	sandboxdomain "jan-server/services/mcp-tools/internal/domain/sandbox"
)

// Client implements sandbox.Provider for E2B backend via e2b-service HTTP API
type Client struct {
	baseURL    string
	timeout    time.Duration
	enabled    bool
	httpClient *http.Client
}

// ClientConfig holds E2B client configuration
type ClientConfig struct {
	BaseURL string
	Timeout time.Duration
	Enabled bool
}

// NewClient creates a new E2B client
func NewClient(cfg ClientConfig) *Client {
	if !cfg.Enabled || cfg.BaseURL == "" {
		return &Client{enabled: false}
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &Client{
		baseURL:    cfg.BaseURL,
		timeout:    timeout,
		enabled:    true,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// IsEnabled returns whether the client is configured and ready
func (c *Client) IsEnabled() bool {
	return c != nil && c.enabled
}

// Name returns the provider name
func (c *Client) Name() string {
	return "e2b"
}

// BaseURL returns the configured base URL
func (c *Client) BaseURL() string {
	if c == nil {
		return ""
	}
	return c.baseURL
}

// SetContext is a no-op for E2B client.
// User context is now passed via context.Context using sandboxdomain.WithUserContext().
// This method exists for interface compatibility but does nothing.
func (c *Client) SetContext(userID, conversationID string) {
	// No-op: context is passed via context.Context, not stored on struct
	// This avoids race conditions in concurrent requests
}

// mcpRequest represents an MCP JSON-RPC request
type mcpRequest struct {
	Jsonrpc string         `json:"jsonrpc"`
	ID      int            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// mcpResponse represents an MCP JSON-RPC response
type mcpResponse struct {
	Jsonrpc string         `json:"jsonrpc"`
	ID      int            `json:"id"`
	Result  map[string]any `json:"result,omitempty"`
	Error   *mcpError      `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// callMCP makes an MCP call to the e2b-service
func (c *Client) callMCP(ctx context.Context, method string, params map[string]any) (*mcpResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("e2b client not enabled")
	}

	userID := sandboxdomain.GetUserID(ctx)
	if userID == "" {
		return nil, fmt.Errorf("user_id not set in context - use sandboxdomain.WithUserContext()")
	}

	req := mcpRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox/mcp", c.baseURL, userID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, string(respBody))
	}

	var mcpResp mcpResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("mcp error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}

	return &mcpResp, nil
}

// callTool is a helper to call a specific tool via MCP
func (c *Client) callTool(ctx context.Context, toolName string, arguments map[string]any) (map[string]any, error) {
	resp, err := c.callMCP(ctx, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": arguments,
	})
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// extractTextFromResult extracts text content from MCP tool result
func extractTextFromResult(result map[string]any) (string, error) {
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		return "", fmt.Errorf("no content in result")
	}

	for _, item := range content {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if itemMap["type"] == "text" {
			if text, ok := itemMap["text"].(string); ok {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("no text content found")
}

// extractImageFromResult extracts base64 image data from MCP tool result
func extractImageFromResult(result map[string]any) (string, error) {
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		return "", fmt.Errorf("no content in result")
	}

	for _, item := range content {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if itemMap["type"] == "image" {
			if data, ok := itemMap["data"].(string); ok {
				return data, nil
			}
		}
	}

	return "", fmt.Errorf("no image content found")
}

// --- SandboxProvider Interface Implementation ---

// ShellExec executes a command in the sandbox shell
func (c *Client) ShellExec(ctx context.Context, command string) (*sandboxdomain.ShellResult, error) {
	convID := sandboxdomain.GetConversationID(ctx)
	if convID == "" {
		return nil, fmt.Errorf("conversation_id not set in context")
	}

	startTime := time.Now()
	result, err := c.callTool(ctx, "e2b_desktop_shell", map[string]any{
		"conversation_id": convID,
		"command":         command,
	})
	if err != nil {
		return nil, fmt.Errorf("shell exec failed: %w", err)
	}

	text, err := extractTextFromResult(result)
	if err != nil {
		return nil, err
	}

	// Parse the result text (format: "stdout: ...\nstderr: ...\nexit_code: ...")
	// For simplicity, we return the full text as stdout
	return &sandboxdomain.ShellResult{
		Stdout:     text,
		DurationMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// FileRead reads file contents from the sandbox filesystem
func (c *Client) FileRead(ctx context.Context, path string) (string, error) {
	convID := sandboxdomain.GetConversationID(ctx)
	if convID == "" {
		return "", fmt.Errorf("conversation_id not set in context")
	}

	result, err := c.callTool(ctx, "e2b_desktop_file_read", map[string]any{
		"conversation_id": convID,
		"path":            path,
	})
	if err != nil {
		return "", fmt.Errorf("file read failed: %w", err)
	}

	return extractTextFromResult(result)
}

// FileWrite writes content to a file in the sandbox filesystem
func (c *Client) FileWrite(ctx context.Context, path, content string) (*sandboxdomain.FileWriteResult, error) {
	convID := sandboxdomain.GetConversationID(ctx)
	if convID == "" {
		return nil, fmt.Errorf("conversation_id not set in context")
	}

	result, err := c.callTool(ctx, "e2b_desktop_file_write", map[string]any{
		"conversation_id": convID,
		"path":            path,
		"content":         content,
	})
	if err != nil {
		return nil, fmt.Errorf("file write failed: %w", err)
	}

	text, err := extractTextFromResult(result)
	if err != nil {
		return nil, err
	}

	return &sandboxdomain.FileWriteResult{
		Success:      true,
		Path:         path,
		BytesWritten: len(text), // Approximate
	}, nil
}

// FileList lists files and directories at the given path
func (c *Client) FileList(ctx context.Context, path string) ([]sandboxdomain.FileInfo, error) {
	convID := sandboxdomain.GetConversationID(ctx)
	if convID == "" {
		return nil, fmt.Errorf("conversation_id not set in context")
	}

	result, err := c.callTool(ctx, "e2b_desktop_file_list", map[string]any{
		"conversation_id": convID,
		"path":            path,
	})
	if err != nil {
		return nil, fmt.Errorf("file list failed: %w", err)
	}

	text, err := extractTextFromResult(result)
	if err != nil {
		return nil, err
	}

	// Parse the text result into FileInfo list
	// The format is: "[DIR] name (size bytes)" per line
	files := make([]sandboxdomain.FileInfo, 0)
	for _, line := range splitLines(text) {
		if line == "" || line == "Empty directory" {
			continue
		}
		isDir := false
		if len(line) > 6 && line[:6] == "[DIR] " {
			isDir = true
			line = line[6:]
		}
		// Extract name (before parenthesis)
		name := line
		if idx := findLastParen(line); idx > 0 {
			name = line[:idx-1]
		}
		files = append(files, sandboxdomain.FileInfo{
			Name:  name,
			IsDir: isDir,
		})
	}

	return files, nil
}

// CodeExecute executes code in the specified language
func (c *Client) CodeExecute(ctx context.Context, code, language string) (*sandboxdomain.CodeResult, error) {
	convID := sandboxdomain.GetConversationID(ctx)
	if convID == "" {
		return nil, fmt.Errorf("conversation_id not set in context")
	}

	startTime := time.Now()
	result, err := c.callTool(ctx, "e2b_desktop_code_execute", map[string]any{
		"conversation_id": convID,
		"code":            code,
		"language":        language,
	})
	if err != nil {
		return nil, fmt.Errorf("code execute failed: %w", err)
	}

	text, err := extractTextFromResult(result)
	if err != nil {
		return nil, err
	}

	// Parse the JSON result
	var codeResult struct {
		Status     string `json:"status"`
		Success    bool   `json:"success"`
		Stdout     string `json:"stdout"`
		Stderr     string `json:"stderr"`
		ExitCode   int    `json:"exit_code"`
		DurationMs int64  `json:"duration_ms"`
		Error      string `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(text), &codeResult); err != nil {
		// If parsing fails, return raw text as stdout
		return &sandboxdomain.CodeResult{
			Status:     "ok",
			Success:    true,
			Stdout:     text,
			DurationMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	return &sandboxdomain.CodeResult{
		Status:     codeResult.Status,
		Success:    codeResult.Success,
		Stdout:     codeResult.Stdout,
		Stderr:     codeResult.Stderr,
		ExitCode:   codeResult.ExitCode,
		DurationMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// InstallPackages installs packages using the specified package manager
func (c *Client) InstallPackages(ctx context.Context, packages []string, manager string) (*sandboxdomain.InstallResult, error) {
	startTime := time.Now()

	if manager == "" {
		manager = "pip"
	}

	result, err := c.callTool(ctx, "e2b_desktop_packages", map[string]any{
		"packages":        packages,
		"package_manager": manager,
	})
	if err != nil {
		return &sandboxdomain.InstallResult{
			Status:     "error",
			Success:    false,
			Packages:   packages,
			DurationMs: time.Since(startTime).Milliseconds(),
			Error:      err.Error(),
		}, nil
	}

	text, err := extractTextFromResult(result)
	if err != nil {
		return &sandboxdomain.InstallResult{
			Status:     "error",
			Success:    false,
			Packages:   packages,
			DurationMs: time.Since(startTime).Milliseconds(),
			Error:      err.Error(),
		}, nil
	}

	// Parse the JSON result
	var installResult struct {
		Status     string   `json:"status"`
		Success    bool     `json:"success"`
		Packages   []string `json:"packages_installed"`
		Output     string   `json:"output"`
		ExitCode   int      `json:"exit_code"`
		DurationMs int64    `json:"duration_ms"`
		Error      string   `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(text), &installResult); err != nil {
		return &sandboxdomain.InstallResult{
			Status:     "success",
			Success:    true,
			Packages:   packages,
			Output:     text,
			DurationMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	return &sandboxdomain.InstallResult{
		Status:     installResult.Status,
		Success:    installResult.Success,
		Packages:   installResult.Packages,
		Output:     installResult.Output,
		ExitCode:   installResult.ExitCode,
		DurationMs: time.Since(startTime).Milliseconds(),
		Error:      installResult.Error,
	}, nil
}

// BrowserInfo is not supported by E2B - returns error
func (c *Client) BrowserInfo(ctx context.Context) (*sandboxdomain.BrowserInfo, error) {
	return nil, sandboxdomain.NewNotSupportedError("e2b", "browser_info")
}

// Screenshot takes a desktop screenshot
func (c *Client) Screenshot(ctx context.Context) (string, error) {
	result, err := c.callTool(ctx, "e2b_desktop_screenshot", map[string]any{})
	if err != nil {
		return "", fmt.Errorf("screenshot failed: %w", err)
	}

	return extractImageFromResult(result)
}

// Click performs a mouse click at the specified coordinates
func (c *Client) Click(ctx context.Context, x, y int, button string) error {
	if button == "" {
		button = "left"
	}

	_, err := c.callTool(ctx, "e2b_desktop_click", map[string]any{
		"x":      x,
		"y":      y,
		"button": button,
	})
	if err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	return nil
}

// Type types text on the desktop
func (c *Client) Type(ctx context.Context, text string) error {
	_, err := c.callTool(ctx, "e2b_desktop_type", map[string]any{
		"text": text,
	})
	if err != nil {
		return fmt.Errorf("type failed: %w", err)
	}

	return nil
}

// MarkitdownConvert is not directly supported by E2B
// We could implement it via shell command if needed
func (c *Client) MarkitdownConvert(ctx context.Context, url string) (string, error) {
	// E2B doesn't have a built-in markitdown tool
	// We can try running it via shell if available
	result, err := c.ShellExec(ctx, fmt.Sprintf("markitdown '%s' 2>/dev/null || curl -s '%s' | html2text 2>/dev/null || echo 'markitdown not available'", url, url))
	if err != nil {
		return "", fmt.Errorf("markitdown convert failed: %w", err)
	}
	return result.Stdout, nil
}

// EnsureSandboxRunning ensures the sandbox is running for the given user
func (c *Client) EnsureSandboxRunning(ctx context.Context, userID string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("e2b client not enabled")
	}

	// POST to /api/v1/users/{user_id}/sandbox to start sandbox
	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("start sandbox failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// EnsureWorkspace ensures a sandbox and workspace exist for the given user and conversation.
func (c *Client) EnsureWorkspace(ctx context.Context, userID, conversationID string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("e2b client not enabled")
	}
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(conversationID) == "" {
		return fmt.Errorf("user_id and conversation_id are required")
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox/workspace/%s", c.baseURL, userID, conversationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ensure workspace failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// helper functions

func splitLines(s string) []string {
	var lines []string
	var current []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, string(current))
			current = nil
		} else {
			current = append(current, s[i])
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}

func findLastParen(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '(' {
			return i
		}
	}
	return -1
}

// Verify that Client implements Provider interface
var _ sandboxdomain.Provider = (*Client)(nil)

// Verify that Client implements Manager interface
var _ sandboxdomain.Manager = (*Client)(nil)

// --- Manager Interface Implementation ---

// sandboxResponse represents the response from e2b-service sandbox endpoints
type sandboxResponse struct {
	PublicID       string  `json:"public_id"`
	E2BSandboxID   string  `json:"e2b_sandbox_id"`
	UserID         string  `json:"user_id"`
	Status         string  `json:"status"`
	ViewURL        *string `json:"view_url"`
	ControlURL     *string `json:"control_url"`
	StartedAt      *string `json:"started_at"`
	TimeoutAt      *string `json:"timeout_at"`
	PausedAt       *string `json:"paused_at"`
	PauseExpiresAt *string `json:"pause_expires_at"`
	ErrorMessage   *string `json:"error_message"`
}

// parseTime parses a time string, returning zero time on error
func parseTime(s *string) time.Time {
	if s == nil || *s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// toSandboxInfo converts sandboxResponse to SandboxInfo
func (r *sandboxResponse) toSandboxInfo() *sandboxdomain.SandboxInfo {
	info := &sandboxdomain.SandboxInfo{
		SandboxID: r.E2BSandboxID,
		Status:    r.Status,
		TimeoutAt: parseTime(r.TimeoutAt),
		StartedAt: parseTime(r.StartedAt),
	}
	if r.ViewURL != nil {
		info.ViewURL = *r.ViewURL
	}
	if r.ControlURL != nil {
		info.ControlURL = *r.ControlURL
	}
	return info
}

// toSandboxState converts sandboxResponse to SandboxState
func (r *sandboxResponse) toSandboxState() *sandboxdomain.SandboxState {
	state := &sandboxdomain.SandboxState{
		Status:         r.Status,
		SandboxID:      r.E2BSandboxID,
		TimeoutAt:      parseTime(r.TimeoutAt),
		StartedAt:      parseTime(r.StartedAt),
		PausedAt:       parseTime(r.PausedAt),
		PauseExpiresAt: parseTime(r.PauseExpiresAt),
	}
	if r.ViewURL != nil {
		state.ViewURL = *r.ViewURL
	}
	if r.ControlURL != nil {
		state.ControlURL = *r.ControlURL
	}
	return state
}

// Start creates or resumes a sandbox for the given user and conversation.
func (c *Client) Start(ctx context.Context, userID, conversationID string, timeout int) (*sandboxdomain.SandboxInfo, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("e2b client not enabled")
	}

	// Apply timeout on sandbox start (workspace endpoint does not accept timeout)
	if timeout > 0 {
		startURL := fmt.Sprintf("%s/api/v1/users/%s/sandbox", c.baseURL, userID)
		startBody, err := json.Marshal(map[string]int{"timeout": timeout})
		if err != nil {
			return nil, fmt.Errorf("marshal start request: %w", err)
		}
		startReq, err := http.NewRequestWithContext(ctx, http.MethodPost, startURL, bytes.NewReader(startBody))
		if err != nil {
			return nil, fmt.Errorf("create start request: %w", err)
		}
		startReq.Header.Set("Content-Type", "application/json")

		startResp, err := c.httpClient.Do(startReq)
		if err != nil {
			return nil, fmt.Errorf("start sandbox request: %w", err)
		}
		defer startResp.Body.Close()

		startRespBody, err := io.ReadAll(startResp.Body)
		if err != nil {
			return nil, fmt.Errorf("read start response: %w", err)
		}
		if startResp.StatusCode < 200 || startResp.StatusCode >= 300 {
			return nil, fmt.Errorf("start sandbox failed with status %d: %s", startResp.StatusCode, string(startRespBody))
		}
	}

	// Use workspace endpoint to ensure workspace exists
	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox/workspace/%s", c.baseURL, userID, conversationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("start sandbox failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response - workspace endpoint returns workspace with nested sandbox
	var workspaceResp struct {
		Sandbox sandboxResponse `json:"sandbox"`
	}
	if err := json.Unmarshal(body, &workspaceResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return workspaceResp.Sandbox.toSandboxInfo(), nil
}

// Stop terminates and deletes a user's sandbox.
func (c *Client) Stop(ctx context.Context, userID string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("e2b client not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return sandboxdomain.NewSandboxNotFoundError(userID)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop sandbox failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Pause pauses a running sandbox.
func (c *Client) Pause(ctx context.Context, userID string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("e2b client not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox/pause", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return sandboxdomain.NewSandboxNotFoundError(userID)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pause sandbox failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Extend extends the timeout of a running sandbox.
func (c *Client) Extend(ctx context.Context, userID string, additionalSeconds int) (*sandboxdomain.SandboxInfo, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("e2b client not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox/extend", c.baseURL, userID)
	reqBody, _ := json.Marshal(map[string]int{"additional_seconds": additionalSeconds})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == 404 {
		return nil, sandboxdomain.NewSandboxNotFoundError(userID)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("extend sandbox failed with status %d: %s", resp.StatusCode, string(body))
	}

	var sandboxResp sandboxResponse
	if err := json.Unmarshal(body, &sandboxResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return sandboxResp.toSandboxInfo(), nil
}

// GetState returns the current state of a user's sandbox.
func (c *Client) GetState(ctx context.Context, userID string) (*sandboxdomain.SandboxState, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("e2b client not enabled")
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// No sandbox exists - return nil state (not an error)
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get sandbox state failed with status %d: %s", resp.StatusCode, string(body))
	}

	var sandboxResp sandboxResponse
	if err := json.Unmarshal(body, &sandboxResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return sandboxResp.toSandboxState(), nil
}

// IsRunning returns true if the user has a running sandbox.
func (c *Client) IsRunning(ctx context.Context, userID string) (bool, error) {
	state, err := c.GetState(ctx, userID)
	if err != nil {
		return false, err
	}
	return state != nil && state.Status == "running", nil
}

// CallTool executes a tool via MCP protocol in the sandbox.
func (c *Client) CallTool(ctx context.Context, userID, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("e2b client not enabled")
	}

	// Build MCP tools/call request
	mcpReq := mcpRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	body, err := json.Marshal(mcpReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox/mcp", c.baseURL, userID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mcp call failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var mcpResp mcpResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("mcp error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}

	return mcpResp.Result, nil
}

// GetDynamicTools fetches tools from MCP servers running inside the sandbox.
func (c *Client) GetDynamicTools(ctx context.Context, userID string) ([]sandboxdomain.DynamicTool, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("e2b client not enabled")
	}

	// Call tools/list on the sandbox MCP endpoint
	req := mcpRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/users/%s/sandbox/mcp", c.baseURL, userID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Sandbox not running or not available - return empty list
		return []sandboxdomain.DynamicTool{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Sandbox not running - return empty list
		return []sandboxdomain.DynamicTool{}, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var mcpResp mcpResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if mcpResp.Error != nil {
		// MCP error - sandbox might not be ready
		return []sandboxdomain.DynamicTool{}, nil
	}

	// Parse tools from result
	tools := make([]sandboxdomain.DynamicTool, 0)
	if mcpResp.Result == nil {
		return tools, nil
	}

	toolsRaw, ok := mcpResp.Result["tools"].([]interface{})
	if !ok {
		return tools, nil
	}

	for _, t := range toolsRaw {
		toolMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := toolMap["name"].(string)
		if name == "" {
			continue
		}

		// Skip e2b_desktop_* and e2b_sandbox_* tools (these are static)
		// Only return dynamic tools (like browser tools from search-mcp-server)
		if strings.HasPrefix(name, "e2b_desktop_") || strings.HasPrefix(name, "e2b_sandbox_") {
			continue
		}

		desc, _ := toolMap["description"].(string)
		schema, _ := toolMap["inputSchema"].(map[string]interface{})

		tools = append(tools, sandboxdomain.DynamicTool{
			Name:        name,
			Description: desc,
			InputSchema: schema,
		})
	}

	return tools, nil
}
