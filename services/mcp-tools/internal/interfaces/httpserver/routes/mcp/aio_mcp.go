package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jan-server/services/mcp-tools/internal/infrastructure/aio"
	"jan-server/services/mcp-tools/internal/infrastructure/llmapi"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"

	sdkgo "github.com/agent-infra/sandbox-sdk-go"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// AIOMCP handles AIO Sandbox tools via direct SDK integration
type AIOMCP struct {
	client    *aio.Client
	llmClient *llmapi.Client
	enabled   bool
}

// NewAIOMCP creates a new AIO MCP handler
func NewAIOMCP(client *aio.Client, enabled bool) *AIOMCP {
	if client == nil || !client.IsEnabled() {
		return nil
	}
	return &AIOMCP{
		client:  client,
		enabled: enabled,
	}
}

// SetLLMClient sets the LLM-API client for tool call tracking
func (a *AIOMCP) SetLLMClient(client *llmapi.Client) {
	if a != nil {
		a.llmClient = client
	}
}

// RegisterTools registers AIO tools with the MCP server
func (a *AIOMCP) RegisterTools(server *mcpsdk.Server) {
	if a == nil || !a.enabled {
		return
	}

	// Register shell execution tool
	a.registerShellExec(server)

	// Register file operations
	a.registerFileRead(server)
	a.registerFileWrite(server)
	a.registerFileList(server)

	// Register browser operations
	a.registerBrowserGetInfo(server)

	// Register code execution (Python/Node.js)
	a.registerCodeExecute(server)

	// Register utility tools
	a.registerMarkitdownConvert(server)

	log.Info().
		Str("url", a.client.BaseURL()).
		Msg("AIO SDK tools registered")
}

// --- Shell Tools ---

type ShellExecArgs struct {
	Command        string `json:"command"`
	ToolCallID     string `json:"tool_call_id,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
}

func (a *AIOMCP) registerShellExec(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "aio_shell_exec",
		Description: "Execute shell commands in AIO Sandbox. Returns stdout, stderr, and exit code. Use for file operations, system commands, and automation tasks.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input ShellExecArgs) (*mcpsdk.CallToolResult, map[string]any, error) {
		startTime := time.Now()
		callCtx := extractAllContext(req)

		log.Info().
			Str("tool", "aio_shell_exec").
			Str("command", truncateString(input.Command, 100)).
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Msg("AIO shell exec requested")

		result, err := a.client.Shell().ExecCommand(ctx, &sdkgo.ShellExecRequest{
			Command: input.Command,
		})

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("aio_shell_exec", "aio-sandbox", status, duration.Seconds())

		if err != nil {
			log.Error().Err(err).Str("tool", "aio_shell_exec").Msg("Shell exec failed")
			return nil, nil, fmt.Errorf("shell exec failed: %w", err)
		}

		output := map[string]any{
			"stdout":      result.Data.Output,
			"exit_code":   result.Data.ExitCode,
			"duration_ms": duration.Milliseconds(),
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, output, nil
	})
}

// --- File Tools ---

type FileReadArgs struct {
	Path string `json:"path"`
}

func (a *AIOMCP) registerFileRead(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "aio_file_read",
		Description: "Read file contents from AIO Sandbox filesystem. Provide the absolute path to the file.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input FileReadArgs) (*mcpsdk.CallToolResult, map[string]any, error) {
		startTime := time.Now()

		log.Debug().
			Str("tool", "aio_file_read").
			Str("path", input.Path).
			Msg("AIO file read requested")

		result, err := a.client.File().ReadFile(ctx, &sdkgo.FileReadRequest{
			File: input.Path,
		})

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("aio_file_read", "aio-sandbox", status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("file read failed: %w", err)
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: result.Data.Content}},
		}, nil, nil
	})
}

type FileWriteArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (a *AIOMCP) registerFileWrite(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "aio_file_write",
		Description: "Write content to a file in AIO Sandbox filesystem. Creates parent directories if needed.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input FileWriteArgs) (*mcpsdk.CallToolResult, map[string]any, error) {
		startTime := time.Now()

		log.Debug().
			Str("tool", "aio_file_write").
			Str("path", input.Path).
			Int("content_len", len(input.Content)).
			Msg("AIO file write requested")

		result, err := a.client.File().WriteFile(ctx, &sdkgo.FileWriteRequest{
			File:    input.Path,
			Content: input.Content,
		})

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("aio_file_write", "aio-sandbox", status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("file write failed: %w", err)
		}

		output := map[string]any{
			"success":       true,
			"bytes_written": result.Data.BytesWritten,
			"path":          input.Path,
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

type FileListArgs struct {
	Path string `json:"path"`
}

func (a *AIOMCP) registerFileList(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "aio_file_list",
		Description: "List files and directories in AIO Sandbox filesystem. Returns file names, sizes, and types.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input FileListArgs) (*mcpsdk.CallToolResult, map[string]any, error) {
		startTime := time.Now()

		log.Debug().
			Str("tool", "aio_file_list").
			Str("path", input.Path).
			Msg("AIO file list requested")

		result, err := a.client.File().ListPath(ctx, &sdkgo.FileListRequest{
			Path: input.Path,
		})

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("aio_file_list", "aio-sandbox", status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("file list failed: %w", err)
		}

		outputJSON, _ := json.Marshal(result.Data)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// --- Browser Tools ---

func (a *AIOMCP) registerBrowserGetInfo(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "aio_browser_info",
		Description: "Get browser information from AIO Sandbox including CDP URL for automation.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input struct{}) (*mcpsdk.CallToolResult, map[string]any, error) {
		startTime := time.Now()

		log.Debug().
			Str("tool", "aio_browser_info").
			Msg("AIO browser info requested")

		result, err := a.client.Browser().GetInfo(ctx)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("aio_browser_info", "aio-sandbox", status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("browser info failed: %w", err)
		}

		output := map[string]any{
			"cdp_url": result.Data.CdpUrl,
			"vnc_url": result.Data.VncUrl,
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// --- Code Execution Tools ---

type CodeExecuteArgs struct {
	Code     string `json:"code"`
	Language string `json:"language"` // "python" or "nodejs"
}

func (a *AIOMCP) registerCodeExecute(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "aio_code_execute",
		Description: "Execute Python or Node.js code in AIO Sandbox. Set language to 'python' or 'nodejs'. Returns stdout, stderr, and execution results.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input CodeExecuteArgs) (*mcpsdk.CallToolResult, map[string]any, error) {
		startTime := time.Now()
		callCtx := extractAllContext(req)

		log.Info().
			Str("tool", "aio_code_execute").
			Str("language", input.Language).
			Int("code_len", len(input.Code)).
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Msg("AIO code execute requested")

		// Default to python if not specified
		lang := input.Language
		if lang == "" {
			lang = "python"
		}

		var output map[string]any
		var execErr error

		switch lang {
		case "python":
			result, err := a.client.Jupyter().ExecuteCode(ctx, &sdkgo.JupyterExecuteRequest{
				Code: input.Code,
			})
			if err != nil {
				execErr = err
			} else {
				// Parse Jupyter outputs
				var outputs []string
				for _, out := range result.Data.Outputs {
					if out.Text != nil && *out.Text != "" {
						outputs = append(outputs, *out.Text)
					}
				}
				output = map[string]any{
					"status":      result.Data.Status,
					"success":     result.Data.Status == "ok",
					"outputs":     outputs,
					"duration_ms": time.Since(startTime).Milliseconds(),
				}
			}
		case "nodejs", "javascript":
			result, err := a.client.Nodejs().ExecuteCode(ctx, &sdkgo.NodeJsExecuteRequest{
				Code: input.Code,
			})
			if err != nil {
				execErr = err
			} else {
				var stdout, stderr string
				if result.Data.Stdout != nil {
					stdout = *result.Data.Stdout
				}
				if result.Data.Stderr != nil {
					stderr = *result.Data.Stderr
				}
				output = map[string]any{
					"status":      result.Data.Status,
					"success":     result.Data.Status == "ok",
					"stdout":      stdout,
					"stderr":      stderr,
					"exit_code":   result.Data.ExitCode,
					"duration_ms": time.Since(startTime).Milliseconds(),
				}
			}
		default:
			execErr = fmt.Errorf("unsupported language: %s (use 'python' or 'nodejs')", lang)
		}

		duration := time.Since(startTime)
		status := "success"
		if execErr != nil {
			status = "error"
		}
		metrics.RecordToolCall("aio_code_execute", "aio-sandbox", status, duration.Seconds())

		if execErr != nil {
			log.Error().Err(execErr).Str("tool", "aio_code_execute").Msg("Code execution failed")
			return nil, nil, fmt.Errorf("code execution failed: %w", execErr)
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// --- Utility Tools ---

type MarkitdownConvertArgs struct {
	URL string `json:"url,omitempty"`
}

func (a *AIOMCP) registerMarkitdownConvert(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "aio_markitdown_convert",
		Description: "Convert a URL or document to Markdown format. Useful for extracting readable text from web pages.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input MarkitdownConvertArgs) (*mcpsdk.CallToolResult, map[string]any, error) {
		startTime := time.Now()

		log.Debug().
			Str("tool", "aio_markitdown_convert").
			Str("url", input.URL).
			Msg("AIO markitdown convert requested")

		result, err := a.client.Util().ConvertToMarkdown(ctx, &sdkgo.UtilConvertToMarkdownRequest{
			Uri: input.URL,
		})

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("aio_markitdown_convert", "aio-sandbox", status, duration.Seconds())

		if err != nil {
			return nil, nil, fmt.Errorf("markitdown convert failed: %w", err)
		}

		// The result.Data is interface{}, serialize the full response
		outputJSON, _ := json.Marshal(result)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// Helper function to truncate strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
