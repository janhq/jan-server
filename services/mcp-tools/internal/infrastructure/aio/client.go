package aio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/agent-infra/sandbox-sdk-go/browser"
	"github.com/agent-infra/sandbox-sdk-go/client"
	"github.com/agent-infra/sandbox-sdk-go/code"
	"github.com/agent-infra/sandbox-sdk-go/file"
	"github.com/agent-infra/sandbox-sdk-go/jupyter"
	"github.com/agent-infra/sandbox-sdk-go/mcp"
	"github.com/agent-infra/sandbox-sdk-go/nodejs"
	"github.com/agent-infra/sandbox-sdk-go/option"
	"github.com/agent-infra/sandbox-sdk-go/sandbox"
	"github.com/agent-infra/sandbox-sdk-go/shell"
	"github.com/agent-infra/sandbox-sdk-go/util"
)

// Client wraps the AIO Sandbox SDK client
type Client struct {
	sdk        *client.Client
	baseURL    string
	enabled    bool
	httpClient *http.Client
}

// ClientConfig holds AIO client configuration
type ClientConfig struct {
	BaseURL string
	Timeout time.Duration
	Enabled bool
}

// NewClient creates a new AIO SDK client
func NewClient(cfg ClientConfig) *Client {
	if !cfg.Enabled || cfg.BaseURL == "" {
		return &Client{enabled: false}
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	httpClient := &http.Client{Timeout: timeout}

	sdk := client.NewClient(
		option.WithBaseURL(cfg.BaseURL),
		option.WithHTTPClient(httpClient),
	)

	return &Client{
		sdk:        sdk,
		baseURL:    cfg.BaseURL,
		enabled:    true,
		httpClient: httpClient,
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

// DownloadFile downloads a file from the sandbox by path.
func (c *Client) DownloadFile(ctx context.Context, path string) ([]byte, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("client not enabled")
	}
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("empty path")
	}
	escaped := url.QueryEscape(path)
	url := strings.TrimRight(c.baseURL, "/") + "/v1/file/download?path=" + escaped
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read download response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download status=%d body=%s", resp.StatusCode, string(body))
	}
	return body, nil
}

// Shell returns the shell operations client
func (c *Client) Shell() *shell.Client {
	if !c.IsEnabled() {
		return nil
	}
	return c.sdk.Shell
}

// File returns the file operations client
func (c *Client) File() *file.Client {
	if !c.IsEnabled() {
		return nil
	}
	return c.sdk.File
}

// Browser returns the browser operations client
func (c *Client) Browser() *browser.Client {
	if !c.IsEnabled() {
		return nil
	}
	return c.sdk.Browser
}

// Jupyter returns the Jupyter operations client
func (c *Client) Jupyter() *jupyter.Client {
	if !c.IsEnabled() {
		return nil
	}
	return c.sdk.Jupyter
}

// Code returns the code execution client
func (c *Client) Code() *code.Client {
	if !c.IsEnabled() {
		return nil
	}
	return c.sdk.Code
}

// Nodejs returns the Node.js execution client
func (c *Client) Nodejs() *nodejs.Client {
	if !c.IsEnabled() {
		return nil
	}
	return c.sdk.Nodejs
}

// Mcp returns the MCP operations client (for hybrid usage)
func (c *Client) Mcp() *mcp.Client {
	if !c.IsEnabled() {
		return nil
	}
	return c.sdk.Mcp
}

// Sandbox returns the sandbox context client
func (c *Client) Sandbox() *sandbox.Client {
	if !c.IsEnabled() {
		return nil
	}
	return c.sdk.Sandbox
}

// Util returns the utility client (e.g., markitdown)
func (c *Client) Util() *util.Client {
	if !c.IsEnabled() {
		return nil
	}
	return c.sdk.Util
}
