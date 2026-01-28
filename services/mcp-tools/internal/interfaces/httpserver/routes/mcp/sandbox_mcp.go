package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jan-server/services/mcp-tools/internal/domain/sandbox"
	"jan-server/services/mcp-tools/internal/infrastructure/llmapi"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// SandboxMCP handles unified sandbox tools via the Provider interface
// Works with both AIO and E2B backends transparently
type SandboxMCP struct {
	provider  sandbox.Provider
	llmClient *llmapi.Client
}

// Tool descriptions - single source of truth
const (
	descShellExec         = "Execute shell commands in the sandbox. Returns stdout, stderr, and exit code. Use for file operations, system commands, and automation tasks."
	descFileRead          = "Read file contents from the sandbox filesystem."
	descFileWrite         = "Write content to a file in the sandbox filesystem."
	descFileList          = "List files and directories in a sandbox path."
	descCodeExecute       = "Execute code in a specific programming language in the sandbox."
	descInstallPackages   = "Install packages in the sandbox environment."
	descMarkitdownConvert = "Convert a URL or document to Markdown format. Useful for extracting readable text from web pages."
	descScreenshot        = "Take a screenshot of the sandbox desktop."
	descClick             = "Click at a specific position on the sandbox desktop."
	descType              = "Type text on the sandbox desktop."
	descBrowserInfo       = "Get browser information from the sandbox."
)

// NewSandboxMCP creates a new Sandbox MCP handler
func NewSandboxMCP(provider sandbox.Provider) *SandboxMCP {
	if provider == nil || !provider.IsEnabled() {
		return nil
	}
	return &SandboxMCP{
		provider: provider,
	}
}

// SetLLMClient sets the LLM-API client for tool call tracking
func (s *SandboxMCP) SetLLMClient(client *llmapi.Client) {
	if s != nil {
		s.llmClient = client
	}
}

// ProviderName returns the name of the active provider
func (s *SandboxMCP) ProviderName() string {
	if s == nil || s.provider == nil {
		return ""
	}
	return s.provider.Name()
}

// setProviderContext extracts user/conversation context and returns an enhanced context.
// The returned context should be passed to provider methods.
func (s *SandboxMCP) setProviderContext(ctx context.Context, req *mcpsdk.CallToolRequest) context.Context {
	if s == nil || s.provider == nil {
		return ctx
	}

	callCtx := extractAllContext(req)
	userID := callCtx["user_id"]
	conversationID := callCtx["conversation_id"]

	// Extract conversation_id from tracking context (X-Conversation-ID header)
	// Sandbox tools only require X-Conversation-ID and Authorization, not X-Tool-Call-ID
	if conversationID == "" && ctx != nil {
		conversationID = GetConversationID(ctx)
	}

	if userID == "" && ctx != nil {
		if val := ctx.Value("user_id"); val != nil {
			if str, ok := val.(string); ok {
				userID = str
			}
		}
	}

	if userID == "" && conversationID == "" {
		return ctx
	}

	// Inject user context into the context for thread-safe access by provider
	return sandbox.WithUserContext(ctx, userID, conversationID)
}

// RegisterTools registers sandbox tools with the MCP server
func (s *SandboxMCP) RegisterTools(server *mcpsdk.Server) {
	if s == nil || s.provider == nil || !s.provider.IsEnabled() {
		return
	}

	providerName := s.provider.Name()

	// Core tools (available for both AIO and E2B)
	s.registerShellExec(server)
	s.registerFileRead(server)
	s.registerFileWrite(server)
	s.registerFileList(server)
	s.registerCodeExecute(server)
	s.registerInstallPackages(server)
	s.registerMarkitdownConvert(server)

	// Conditional tools based on provider capabilities
	if providerName == "aio" {
		s.registerBrowserInfo(server)
	}
	if providerName == "e2b" {
		s.registerScreenshot(server)
		s.registerClick(server)
		s.registerType(server)
	}

	log.Info().
		Str("provider", providerName).
		Msg("Sandbox tools registered")
}

// GetToolDefinitions returns the tool definitions for sandbox execution tools.
// Used by MCPRoute to include these tools in tools/list response.
func (s *SandboxMCP) GetToolDefinitions() []ToolDefinition {
	if s == nil || s.provider == nil || !s.provider.IsEnabled() {
		return nil
	}

	providerName := s.provider.Name()

	// Core tools (available for both AIO and E2B)
	tools := []ToolDefinition{
		{
			Name:        "sandbox_shell_exec",
			Description: descShellExec,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Shell command to execute",
					},
				},
				"required": []string{"command"},
			},
		},
		{
			Name:        "sandbox_file_read",
			Description: descFileRead,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "sandbox_file_write",
			Description: descFileWrite,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to write",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Content to write to the file",
					},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "sandbox_file_list",
			Description: descFileList,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to list",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "sandbox_code_execute",
			Description: descCodeExecute,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"code": map[string]interface{}{
						"type":        "string",
						"description": "Code to execute",
					},
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Programming language (python, javascript, etc.)",
					},
				},
				"required": []string{"code", "language"},
			},
		},
		{
			Name:        "sandbox_install_packages",
			Description: descInstallPackages,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"packages": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of packages to install",
					},
					"manager": map[string]interface{}{
						"type":        "string",
						"description": "Package manager to use (pip, npm, apt, etc.)",
					},
				},
				"required": []string{"packages"},
			},
		},
		{
			Name:        "sandbox_markitdown_convert",
			Description: descMarkitdownConvert,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL to convert to markdown",
					},
				},
			},
		},
	}

	// E2B-specific desktop tools
	if providerName == "e2b" {
		tools = append(tools, []ToolDefinition{
			{
				Name:        "sandbox_screenshot",
				Description: descScreenshot,
				InputSchema: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
			{
				Name:        "sandbox_click",
				Description: descClick,
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"x": map[string]interface{}{
							"type":        "integer",
							"description": "X coordinate to click",
						},
						"y": map[string]interface{}{
							"type":        "integer",
							"description": "Y coordinate to click",
						},
					},
					"required": []string{"x", "y"},
				},
			},
			{
				Name:        "sandbox_type",
				Description: descType,
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{
							"type":        "string",
							"description": "Text to type",
						},
					},
					"required": []string{"text"},
				},
			},
		}...)
	}

	// AIO-specific tools
	if providerName == "aio" {
		tools = append(tools, ToolDefinition{
			Name:        "sandbox_browser_info",
			Description: descBrowserInfo,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		})
	}

	return tools
}

// --- Shell Tool ---

type SandboxShellExecArgs struct {
	Command string `json:"command"`
}

func (s *SandboxMCP) registerShellExec(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_shell_exec",
		Description: descShellExec,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxShellExecArgs) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()
		callCtx := extractAllContext(req)

		log.Info().
			Str("tool", "sandbox_shell_exec").
			Str("command", truncateSandboxString(input.Command, 100)).
			Str("provider", providerName).
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Msg("Sandbox shell exec requested")

		result, err := s.provider.ShellExec(ctx, input.Command)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_shell_exec", providerName, status, duration.Seconds())

		if err != nil {
			log.Error().Err(err).Str("tool", "sandbox_shell_exec").Msg("Shell exec failed")
			return nil, nil, fmt.Errorf("shell exec failed: %w", err)
		}

		output := map[string]any{
			"stdout":      result.Stdout,
			"stderr":      result.Stderr,
			"exit_code":   result.ExitCode,
			"duration_ms": duration.Milliseconds(),
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, output, nil
	})
}

// --- File Tools ---

type SandboxFileReadArgs struct {
	Path string `json:"path"`
}

func (s *SandboxMCP) registerFileRead(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_file_read",
		Description: descFileRead,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxFileReadArgs) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()

		log.Debug().
			Str("tool", "sandbox_file_read").
			Str("path", input.Path).
			Str("provider", providerName).
			Msg("Sandbox file read requested")

		content, err := s.provider.FileRead(ctx, input.Path)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_file_read", providerName, status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("file read failed: %w", err)
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: content}},
		}, nil, nil
	})
}

type SandboxFileWriteArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (s *SandboxMCP) registerFileWrite(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_file_write",
		Description: descFileWrite,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxFileWriteArgs) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()

		log.Debug().
			Str("tool", "sandbox_file_write").
			Str("path", input.Path).
			Int("content_len", len(input.Content)).
			Str("provider", providerName).
			Msg("Sandbox file write requested")

		result, err := s.provider.FileWrite(ctx, input.Path, input.Content)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_file_write", providerName, status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("file write failed: %w", err)
		}

		output := map[string]any{
			"success":       result.Success,
			"bytes_written": result.BytesWritten,
			"path":          result.Path,
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

type SandboxFileListArgs struct {
	Path string `json:"path"`
}

func (s *SandboxMCP) registerFileList(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_file_list",
		Description: descFileList,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxFileListArgs) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()

		log.Debug().
			Str("tool", "sandbox_file_list").
			Str("path", input.Path).
			Str("provider", providerName).
			Msg("Sandbox file list requested")

		files, err := s.provider.FileList(ctx, input.Path)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_file_list", providerName, status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("file list failed: %w", err)
		}

		outputJSON, _ := json.Marshal(files)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// --- Code Execution Tool ---

type SandboxCodeExecuteArgs struct {
	Code     string `json:"code"`
	Language string `json:"language"` // "python" or "nodejs"
}

func (s *SandboxMCP) registerCodeExecute(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_code_execute",
		Description: descCodeExecute,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxCodeExecuteArgs) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()
		callCtx := extractAllContext(req)

		log.Info().
			Str("tool", "sandbox_code_execute").
			Str("language", input.Language).
			Int("code_len", len(input.Code)).
			Str("code_hash", hashSandboxString(input.Code)).
			Str("provider", providerName).
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Msg("Sandbox code execute requested")

		// Default to python if not specified
		lang := input.Language
		if lang == "" {
			lang = "python"
		}

		result, err := s.provider.CodeExecute(ctx, input.Code, lang)

		duration := time.Since(startTime)
		metricsStatus := "success"
		if err != nil {
			metricsStatus = "error"
		}
		metrics.RecordToolCall("sandbox_code_execute", providerName, metricsStatus, duration.Seconds())

		if err != nil {
			log.Error().
				Err(err).
				Str("tool", "sandbox_code_execute").
				Str("language", lang).
				Str("provider", providerName).
				Msg("Code execution failed")
			return nil, nil, fmt.Errorf("code execution failed: %w", err)
		}

		// Check for execution errors
		if !result.Success {
			errMsg := fmt.Sprintf("code execution status: %s", result.Status)
			if result.Stderr != "" {
				stderr := result.Stderr
				if len(stderr) > 500 {
					stderr = stderr[:500] + "..."
				}
				errMsg = fmt.Sprintf("%s\nstderr: %s", errMsg, stderr)
			}
			if result.ExitCode != 0 {
				errMsg = fmt.Sprintf("%s (exit code: %d)", errMsg, result.ExitCode)
			}
			log.Error().
				Str("tool", "sandbox_code_execute").
				Str("language", lang).
				Str("status", result.Status).
				Int("exit_code", result.ExitCode).
				Str("provider", providerName).
				Msg("Code execution failed")
			return nil, nil, fmt.Errorf("%s", errMsg)
		}

		output := map[string]any{
			"status":      result.Status,
			"success":     result.Success,
			"stdout":      result.Stdout,
			"stderr":      result.Stderr,
			"exit_code":   result.ExitCode,
			"duration_ms": duration.Milliseconds(),
		}

		log.Info().
			Str("tool", "sandbox_code_execute").
			Str("language", lang).
			Str("status", result.Status).
			Int("exit_code", result.ExitCode).
			Int64("duration_ms", duration.Milliseconds()).
			Str("provider", providerName).
			Msg("Sandbox code execute completed")

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// --- Package Installation Tool ---

type SandboxInstallPackagesArgs struct {
	Packages []string `json:"packages"`
	Manager  string   `json:"manager,omitempty"` // "pip" or "npm", defaults to "pip"
}

func (s *SandboxMCP) registerInstallPackages(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_install_packages",
		Description: descInstallPackages,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxInstallPackagesArgs) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()
		callCtx := extractAllContext(req)

		if len(input.Packages) == 0 {
			return nil, nil, fmt.Errorf("no packages specified")
		}

		manager := input.Manager
		if manager == "" {
			manager = "pip"
		}

		log.Info().
			Str("tool", "sandbox_install_packages").
			Strs("packages", input.Packages).
			Str("manager", manager).
			Str("provider", providerName).
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Msg("Sandbox install packages requested")

		result, err := s.provider.InstallPackages(ctx, input.Packages, manager)

		duration := time.Since(startTime)
		status := "success"
		if err != nil || (result != nil && !result.Success) {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_install_packages", providerName, status, duration.Seconds())

		if err != nil {
			log.Error().Err(err).
				Strs("packages", input.Packages).
				Str("tool", "sandbox_install_packages").
				Str("provider", providerName).
				Msg("Package installation failed")

			output := map[string]any{
				"status":             "error",
				"success":            false,
				"packages_requested": input.Packages,
				"duration_ms":        duration.Milliseconds(),
				"error":              err.Error(),
			}
			outputJSON, _ := json.Marshal(output)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
				IsError: true,
			}, output, nil
		}

		if !result.Success {
			log.Error().
				Strs("packages", input.Packages).
				Str("tool", "sandbox_install_packages").
				Str("error", result.Error).
				Str("provider", providerName).
				Msg("Package installation failed")

			output := map[string]any{
				"status":             result.Status,
				"success":            result.Success,
				"packages_requested": input.Packages,
				"output":             result.Output,
				"exit_code":          result.ExitCode,
				"duration_ms":        duration.Milliseconds(),
				"error":              result.Error,
			}
			outputJSON, _ := json.Marshal(output)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
				IsError: true,
			}, output, nil
		}

		log.Info().
			Strs("packages", input.Packages).
			Int64("duration_ms", duration.Milliseconds()).
			Str("provider", providerName).
			Msg("Packages installed successfully")

		output := map[string]any{
			"status":             result.Status,
			"success":            result.Success,
			"packages_installed": result.Packages,
			"output":             result.Output,
			"exit_code":          result.ExitCode,
			"duration_ms":        duration.Milliseconds(),
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, output, nil
	})
}

// --- Utility Tools ---

type SandboxMarkitdownConvertArgs struct {
	URL string `json:"url,omitempty"`
}

func (s *SandboxMCP) registerMarkitdownConvert(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_markitdown_convert",
		Description: descMarkitdownConvert,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxMarkitdownConvertArgs) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()

		log.Debug().
			Str("tool", "sandbox_markitdown_convert").
			Str("url", input.URL).
			Str("provider", providerName).
			Msg("Sandbox markitdown convert requested")

		result, err := s.provider.MarkitdownConvert(ctx, input.URL)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_markitdown_convert", providerName, status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("markitdown convert failed: %w", err)
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: result}},
		}, nil, nil
	})
}

// --- AIO-specific Tools ---

func (s *SandboxMCP) registerBrowserInfo(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_browser_info",
		Description: descBrowserInfo,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input struct{}) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()

		log.Debug().
			Str("tool", "sandbox_browser_info").
			Str("provider", providerName).
			Msg("Sandbox browser info requested")

		result, err := s.provider.BrowserInfo(ctx)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_browser_info", providerName, status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("browser info failed: %w", err)
		}

		output := map[string]any{
			"cdp_url": result.CdpURL,
			"vnc_url": result.VncURL,
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// --- E2B-specific Tools ---

func (s *SandboxMCP) registerScreenshot(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_screenshot",
		Description: descScreenshot,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input struct{}) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()

		log.Debug().
			Str("tool", "sandbox_screenshot").
			Str("provider", providerName).
			Msg("Sandbox screenshot requested")

		result, err := s.provider.Screenshot(ctx)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_screenshot", providerName, status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("screenshot failed: %w", err)
		}

		imageData, err := decodeSandboxImage(result)
		if err != nil {
			log.Error().
				Err(err).
				Str("tool", "sandbox_screenshot").
				Str("provider", providerName).
				Msg("Screenshot decode failed")
			return nil, nil, fmt.Errorf("screenshot decode failed: %w", err)
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.ImageContent{
				Data:     imageData,
				MIMEType: "image/png",
			}},
		}, nil, nil
	})
}

type SandboxClickArgs struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Button string `json:"button,omitempty"` // "left", "right", "middle"
}

func (s *SandboxMCP) registerClick(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_click",
		Description: descClick,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxClickArgs) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()

		log.Debug().
			Str("tool", "sandbox_click").
			Int("x", input.X).
			Int("y", input.Y).
			Str("button", input.Button).
			Str("provider", providerName).
			Msg("Sandbox click requested")

		button := input.Button
		if button == "" {
			button = "left"
		}

		err := s.provider.Click(ctx, input.X, input.Y, button)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_click", providerName, status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("click failed: %w", err)
		}

		output := map[string]any{
			"x":      input.X,
			"y":      input.Y,
			"button": button,
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

type SandboxTypeArgs struct {
	Text string `json:"text"`
}

func (s *SandboxMCP) registerType(server *mcpsdk.Server) {
	providerName := s.provider.Name()
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_type",
		Description: descType,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxTypeArgs) (*mcpsdk.CallToolResult, any, error) {
		ctx = s.setProviderContext(ctx, req)
		startTime := time.Now()

		log.Debug().
			Str("tool", "sandbox_type").
			Int("text_len", len(input.Text)).
			Str("provider", providerName).
			Msg("Sandbox type requested")

		err := s.provider.Type(ctx, input.Text)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_type", providerName, status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("type failed: %w", err)
		}

		output := map[string]any{
			"text_length": len(input.Text),
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// Helper functions (namespaced to avoid conflicts with aio_mcp.go)

func truncateSandboxString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func hashSandboxString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func decodeSandboxImage(data string) ([]byte, error) {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return nil, fmt.Errorf("empty image data")
	}

	if strings.HasPrefix(trimmed, "data:") {
		if comma := strings.Index(trimmed, ","); comma != -1 {
			trimmed = trimmed[comma+1:]
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(trimmed)
	}
	if err != nil {
		return nil, err
	}

	return decoded, nil
}
