package aio

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	sdkgo "github.com/agent-infra/sandbox-sdk-go"
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

	sandboxdomain "jan-server/services/mcp-tools/internal/domain/sandbox"
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

// --- SandboxProvider Interface Implementation ---

// Name returns the provider name
func (c *Client) Name() string {
	return "aio"
}

// ShellExec executes a command in the sandbox shell
func (c *Client) ShellExec(ctx context.Context, command string) (*sandboxdomain.ShellResult, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("aio client not enabled")
	}

	startTime := time.Now()
	result, err := c.sdk.Shell.ExecCommand(ctx, &sdkgo.ShellExecRequest{
		Command: command,
	})
	if err != nil {
		return nil, fmt.Errorf("shell exec failed: %w", err)
	}

	var exitCode int
	if result.Data.ExitCode != nil {
		exitCode = *result.Data.ExitCode
	}

	stdout := ""
	if result.Data.Output != nil {
		stdout = *result.Data.Output
	}

	return &sandboxdomain.ShellResult{
		Stdout:     stdout,
		ExitCode:   exitCode,
		DurationMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// FileRead reads file contents from the sandbox filesystem
func (c *Client) FileRead(ctx context.Context, path string) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("aio client not enabled")
	}

	// Try download endpoint first
	if payload, err := c.DownloadFile(ctx, path); err == nil && len(payload) > 0 {
		return base64.StdEncoding.EncodeToString(payload), nil
	}

	// Fall back to code execution
	pythonCode := fmt.Sprintf(`import base64
with open(%q, 'rb') as f:
    content = f.read()
    encoded = base64.b64encode(content).decode('utf-8')
    print(encoded)`, path)

	result, err := c.sdk.Code.ExecuteCode(ctx, &sdkgo.CodeExecuteRequest{
		Language: sdkgo.LanguagePython,
		Code:     pythonCode,
	})
	if err != nil {
		return "", fmt.Errorf("code execution failed: %w", err)
	}

	if result.Data.ExitCode != nil && *result.Data.ExitCode != 0 {
		stderr := ""
		if result.Data.Stderr != nil {
			stderr = *result.Data.Stderr
		}
		return "", fmt.Errorf("script failed with exit code %d: %s", *result.Data.ExitCode, stderr)
	}

	content := ""
	if result.Data.Stdout != nil {
		content = *result.Data.Stdout
	}
	return strings.TrimSpace(content), nil
}

// FileWrite writes content to a file in the sandbox filesystem
func (c *Client) FileWrite(ctx context.Context, path, content string) (*sandboxdomain.FileWriteResult, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("aio client not enabled")
	}

	result, err := c.sdk.File.WriteFile(ctx, &sdkgo.FileWriteRequest{
		File:    path,
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("file write failed: %w", err)
	}

	bytesWritten := 0
	if result.Data.BytesWritten != nil {
		bytesWritten = *result.Data.BytesWritten
	}

	return &sandboxdomain.FileWriteResult{
		Success:      true,
		Path:         path,
		BytesWritten: bytesWritten,
	}, nil
}

// FileList lists files and directories at the given path
func (c *Client) FileList(ctx context.Context, path string) ([]sandboxdomain.FileInfo, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("aio client not enabled")
	}

	result, err := c.sdk.File.ListPath(ctx, &sdkgo.FileListRequest{
		Path: path,
	})
	if err != nil {
		return nil, fmt.Errorf("file list failed: %w", err)
	}

	files := make([]sandboxdomain.FileInfo, 0)
	if result.Data != nil && len(result.Data.Files) > 0 {
		for _, entry := range result.Data.Files {
			if entry == nil {
				continue
			}
			name := entry.Name
			isDir := entry.IsDirectory
			var size int64
			if entry.Size != nil {
				size = int64(*entry.Size)
			}
			files = append(files, sandboxdomain.FileInfo{
				Name:  name,
				IsDir: isDir,
				Size:  size,
			})
		}
	}

	return files, nil
}

// CodeExecute executes code in the specified language
func (c *Client) CodeExecute(ctx context.Context, code, language string) (*sandboxdomain.CodeResult, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("aio client not enabled")
	}

	startTime := time.Now()

	// Map language to SDK type
	var lang sdkgo.Language
	switch strings.ToLower(language) {
	case "python":
		lang = sdkgo.LanguagePython
	case "nodejs", "javascript":
		lang = sdkgo.LanguageJavascript
	default:
		return nil, fmt.Errorf("unsupported language: %s (use 'python' or 'nodejs')", language)
	}

	result, err := c.sdk.Code.ExecuteCode(ctx, &sdkgo.CodeExecuteRequest{
		Language: lang,
		Code:     code,
	})
	if err != nil {
		return nil, fmt.Errorf("code execution failed: %w", err)
	}

	var stdout, stderr string
	var exitCode int
	if result.Data.Stdout != nil {
		stdout = *result.Data.Stdout
	}
	if result.Data.Stderr != nil {
		stderr = *result.Data.Stderr
	}
	if result.Data.ExitCode != nil {
		exitCode = *result.Data.ExitCode
	}

	status := result.Data.Status
	success := status == "ok"

	return &sandboxdomain.CodeResult{
		Status:     status,
		Success:    success,
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		DurationMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// InstallPackages installs packages using the specified package manager
func (c *Client) InstallPackages(ctx context.Context, packages []string, manager string) (*sandboxdomain.InstallResult, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("aio client not enabled")
	}

	startTime := time.Now()

	if len(packages) == 0 {
		return nil, fmt.Errorf("no packages specified")
	}

	// For AIO, we use pip by default via code execution
	if manager == "" {
		manager = "pip"
	}

	var installCode string
	if manager == "pip" {
		packagesJSON, _ := json.Marshal(packages)
		installCode = fmt.Sprintf(
			"import subprocess, sys\npkgs = %s\nproc = subprocess.run([sys.executable, '-m', 'pip', 'install', '--quiet', '--disable-pip-version-check', *pkgs], capture_output=True, text=True)\nprint(proc.stdout)\nprint(proc.stderr, file=sys.stderr)\nif proc.returncode != 0:\n    raise RuntimeError(f\"pip install failed with exit code {proc.returncode}\")\n",
			string(packagesJSON),
		)
	} else if manager == "npm" {
		packagesStr := strings.Join(packages, " ")
		installCode = fmt.Sprintf(
			"import subprocess\nproc = subprocess.run(['npm', 'install', '--quiet', %q], capture_output=True, text=True, shell=True)\nprint(proc.stdout)\nprint(proc.stderr, file=sys.stderr)\nif proc.returncode != 0:\n    raise RuntimeError(f\"npm install failed with exit code {proc.returncode}\")\n",
			packagesStr,
		)
	} else {
		return nil, fmt.Errorf("unsupported package manager: %s (use 'pip' or 'npm')", manager)
	}

	result, err := c.sdk.Code.ExecuteCode(ctx, &sdkgo.CodeExecuteRequest{
		Language: sdkgo.LanguagePython,
		Code:     installCode,
	})

	duration := time.Since(startTime)

	if err != nil {
		return &sandboxdomain.InstallResult{
			Status:     "error",
			Success:    false,
			Packages:   packages,
			ExitCode:   -1,
			DurationMs: duration.Milliseconds(),
			Error:      err.Error(),
		}, nil
	}

	var output string
	var exitCode int
	if result.Data.Stdout != nil {
		output = *result.Data.Stdout
	}
	if result.Data.Stderr != nil {
		if output != "" {
			output += "\n"
		}
		output += *result.Data.Stderr
	}
	if result.Data.ExitCode != nil {
		exitCode = *result.Data.ExitCode
	}

	success := result.Data.Status == "ok"
	status := "success"
	var errorMsg string
	if !success {
		status = "error"
		errorMsg = fmt.Sprintf("install failed with status: %s", result.Data.Status)
	}

	return &sandboxdomain.InstallResult{
		Status:     status,
		Success:    success,
		Packages:   packages,
		Output:     output,
		ExitCode:   exitCode,
		DurationMs: duration.Milliseconds(),
		Error:      errorMsg,
	}, nil
}

// BrowserInfo returns browser automation URLs
func (c *Client) BrowserInfo(ctx context.Context) (*sandboxdomain.BrowserInfo, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("aio client not enabled")
	}

	result, err := c.sdk.Browser.GetInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("browser info failed: %w", err)
	}

	return &sandboxdomain.BrowserInfo{
		CdpURL: result.Data.CdpUrl,
		VncURL: result.Data.VncUrl,
	}, nil
}

// Screenshot is not supported by AIO - returns error
func (c *Client) Screenshot(ctx context.Context) (string, error) {
	return "", sandboxdomain.NewNotSupportedError("aio", "screenshot")
}

// Click is not supported by AIO - returns error
func (c *Client) Click(ctx context.Context, x, y int, button string) error {
	return sandboxdomain.NewNotSupportedError("aio", "click")
}

// Type is not supported by AIO - returns error
func (c *Client) Type(ctx context.Context, text string) error {
	return sandboxdomain.NewNotSupportedError("aio", "type")
}

// MarkitdownConvert converts a URL or document to Markdown format
func (c *Client) MarkitdownConvert(ctx context.Context, url string) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("aio client not enabled")
	}

	result, err := c.sdk.Util.ConvertToMarkdown(ctx, &sdkgo.UtilConvertToMarkdownRequest{
		Uri: url,
	})
	if err != nil {
		return "", fmt.Errorf("markitdown convert failed: %w", err)
	}

	// Serialize the result as JSON
	output, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(output), nil
}

// Verify that Client implements Provider interface
var _ sandboxdomain.Provider = (*Client)(nil)
