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

	// Register package installation
	a.registerInstallPackages(server)

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

		content, err := a.readFileBase64(ctx, input.Path)

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
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: content}},
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
			Str("code_hash", hashString(input.Code)).
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Str("conversation_id", callCtx["conversation_id"]).
			Str("user_id", callCtx["user_id"]).
			Msg("AIO code execute requested")

		// Default to python if not specified
		lang := input.Language
		if lang == "" {
			lang = "python"
		}

		var (
			output          map[string]any
			execErr         error
			stdout          string
			stderr          string
			exitCode        int
			execStatus      string
			responsePreview string
		)

		switch lang {
		case "python":
			// Use Code.ExecuteCode for unified runtime (same environment as Shell)
			// This ensures pip-installed packages are available
			result, err := a.client.Code().ExecuteCode(ctx, &sdkgo.CodeExecuteRequest{
				Language: sdkgo.LanguagePython,
				Code:     input.Code,
			})
			if err != nil {
				execErr = err
			} else {
				responsePreview = marshalForLog(result, 800)
				if result.Data.Stdout != nil {
					stdout = *result.Data.Stdout
				}
				if result.Data.Stderr != nil {
					stderr = *result.Data.Stderr
				}
				if result.Data.ExitCode != nil {
					exitCode = *result.Data.ExitCode
				}
				execStatus = result.Data.Status
				output = map[string]any{
					"status":      execStatus,
					"success":     execStatus == "ok",
					"stdout":      stdout,
					"stderr":      stderr,
					"exit_code":   exitCode,
					"duration_ms": time.Since(startTime).Milliseconds(),
				}
				if execStatus != "ok" {
					// Include stderr in error message for Python
					errMsg := fmt.Sprintf("code execution status: %s", execStatus)
					if stderr != "" {
						if len(stderr) > 500 {
							stderr = stderr[:500] + "..."
						}
						errMsg = fmt.Sprintf("%s\nstderr: %s", errMsg, stderr)
					}
					if exitCode != 0 {
						errMsg = fmt.Sprintf("%s (exit code: %d)", errMsg, exitCode)
					}
					execErr = fmt.Errorf("%s", errMsg)
				}
			}
		case "nodejs", "javascript":
			// Use Code.ExecuteCode for unified runtime (same environment as Shell)
			result, err := a.client.Code().ExecuteCode(ctx, &sdkgo.CodeExecuteRequest{
				Language: sdkgo.LanguageJavascript,
				Code:     input.Code,
			})
			if err != nil {
				execErr = err
			} else {
				responsePreview = marshalForLog(result, 800)
				if result.Data.Stdout != nil {
					stdout = *result.Data.Stdout
				}
				if result.Data.Stderr != nil {
					stderr = *result.Data.Stderr
				}
				if result.Data.ExitCode != nil {
					exitCode = *result.Data.ExitCode
				}
				execStatus = result.Data.Status
				output = map[string]any{
					"status":      execStatus,
					"success":     execStatus == "ok",
					"stdout":      stdout,
					"stderr":      stderr,
					"exit_code":   exitCode,
					"duration_ms": time.Since(startTime).Milliseconds(),
				}
				if execStatus != "ok" {
					// Include stderr in error message for Node.js
					errMsg := fmt.Sprintf("code execution status: %s", execStatus)
					if stderr != "" {
						if len(stderr) > 500 {
							stderr = stderr[:500] + "..."
						}
						errMsg = fmt.Sprintf("%s\nstderr: %s", errMsg, stderr)
					}
					if exitCode != 0 {
						errMsg = fmt.Sprintf("%s (exit code: %d)", errMsg, exitCode)
					}
					execErr = fmt.Errorf("%s", errMsg)
				}
			}
		default:
			execErr = fmt.Errorf("unsupported language: %s (use 'python' or 'nodejs')", lang)
		}

		duration := time.Since(startTime)
		metricsStatus := "success"
		if execErr != nil {
			metricsStatus = "error"
		}
		metrics.RecordToolCall("aio_code_execute", "aio-sandbox", metricsStatus, duration.Seconds())

		if execErr != nil {
			log.Error().
				Err(execErr).
				Str("tool", "aio_code_execute").
				Str("language", lang).
				Str("exec_status", execStatus).
				Str("metrics_status", metricsStatus).
				Int("exit_code", exitCode).
				Str("code_hash", hashString(input.Code)).
				Int("code_len", len(input.Code)).
				Int("stdout_len", len(stdout)).
				Int("stderr_len", len(stderr)).
				Str("stderr_preview", truncateString(stderr, 200)).
				Str("response_preview", responsePreview).
				Str("tool_call_id", callCtx["tool_call_id"]).
				Str("request_id", callCtx["request_id"]).
				Str("conversation_id", callCtx["conversation_id"]).
				Str("user_id", callCtx["user_id"]).
				Msg("Code execution failed")
			return nil, nil, fmt.Errorf("code execution failed: %w", execErr)
		}

		log.Info().
			Str("tool", "aio_code_execute").
			Str("language", lang).
			Str("exec_status", execStatus).
			Str("metrics_status", metricsStatus).
			Int("exit_code", exitCode).
			Int("stdout_len", len(stdout)).
			Int("stderr_len", len(stderr)).
			Int64("duration_ms", duration.Milliseconds()).
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Str("conversation_id", callCtx["conversation_id"]).
			Str("user_id", callCtx["user_id"]).
			Msg("AIO code execute completed")

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// --- Package Installation ---

type InstallPackagesArgs struct {
	Packages []string `json:"packages"`
}

func (a *AIOMCP) registerInstallPackages(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "aio_install_packages",
		Description: "Install Python packages using pip in the AIO Sandbox. Use this before running code that requires external libraries.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input InstallPackagesArgs) (*mcpsdk.CallToolResult, map[string]any, error) {
		startTime := time.Now()
		callCtx := extractAllContext(req)

		if len(input.Packages) == 0 {
			return nil, nil, fmt.Errorf("no packages specified")
		}

		// Build pip install command
		packagesStr := ""
		for i, pkg := range input.Packages {
			if i > 0 {
				packagesStr += " "
			}
			packagesStr += pkg
		}
		command := fmt.Sprintf("pip install --quiet --disable-pip-version-check %s", packagesStr)

		log.Info().
			Str("tool", "aio_install_packages").
			Strs("packages", input.Packages).
			Str("command", command).
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Msg("AIO install packages requested")

		var (
			status         = "success"
			execErr        error
			pipOutput      string
			exitCode       int
			responsePreview string
		)

		packagesJSON, _ := json.Marshal(input.Packages)
		installCode := fmt.Sprintf(
			"import subprocess, sys\npkgs = %s\nproc = subprocess.run([sys.executable, '-m', 'pip', 'install', '--quiet', '--disable-pip-version-check', *pkgs], capture_output=True, text=True)\nprint(proc.stdout)\nprint(proc.stderr, file=sys.stderr)\nif proc.returncode != 0:\n    raise RuntimeError(f\"pip install failed with exit code {proc.returncode}\")\n",
			string(packagesJSON),
		)

		codeResult, err := a.client.Code().ExecuteCode(ctx, &sdkgo.CodeExecuteRequest{
			Language: sdkgo.LanguagePython,
			Code:     installCode,
		})
		if err != nil {
			execErr = err
		} else {
			responsePreview = marshalForLog(codeResult, 800)
			if codeResult.Data.Stdout != nil {
				pipOutput = *codeResult.Data.Stdout
			}
			if codeResult.Data.Stderr != nil {
				if pipOutput != "" {
					pipOutput += "\n"
				}
				pipOutput += *codeResult.Data.Stderr
			}
			if codeResult.Data.ExitCode != nil {
				exitCode = *codeResult.Data.ExitCode
			}
			if codeResult.Data.Status != "ok" {
				execErr = fmt.Errorf("code execution status: %s", codeResult.Data.Status)
			}
		}

		if execErr != nil {
			log.Warn().
				Err(execErr).
				Strs("packages", input.Packages).
				Str("response_preview", responsePreview).
				Msg("Code-based package install failed; falling back to shell")

			result, err := a.client.Shell().ExecCommand(ctx, &sdkgo.ShellExecRequest{
				Command: command,
			})
			if err != nil {
				status = "error"
				execErr = err
			} else if result.Data.ExitCode != nil && *result.Data.ExitCode != 0 {
				status = "error"
				execErr = fmt.Errorf("pip install failed with exit code %d: %s", *result.Data.ExitCode, result.Data.Output)
			} else {
				execErr = nil
			}
			if result.Data.Output != nil {
				pipOutput = *result.Data.Output
			}
			if result.Data.ExitCode != nil {
				exitCode = *result.Data.ExitCode
			}
		}

		duration := time.Since(startTime)

		metrics.RecordToolCall("aio_install_packages", "aio-sandbox", status, duration.Seconds())

		if execErr != nil {
			log.Error().Err(execErr).
				Strs("packages", input.Packages).
				Str("tool", "aio_install_packages").
				Msg("Package installation failed")

			output := map[string]any{
				"status":             "error",
				"success":            false,
				"packages_requested": input.Packages,
				"pip_output":         pipOutput,
				"exit_code":          exitCode,
				"duration_ms":        duration.Milliseconds(),
				"error":              execErr.Error(),
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
			Msg("Packages installed successfully")

		output := map[string]any{
			"status":             "success",
			"success":            true,
			"packages_installed": input.Packages,
			"pip_output":         pipOutput,
			"exit_code":          exitCode,
			"duration_ms":        duration.Milliseconds(),
		}

		outputJSON, _ := json.Marshal(output)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, output, nil
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

// readFileBase64 reads a file from the sandbox and returns it as base64-encoded string.
// Prefer the download endpoint, then fall back to code execution when needed.
func (a *AIOMCP) readFileBase64(ctx context.Context, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("empty path")
	}

	if payload, err := a.client.DownloadFile(ctx, path); err == nil {
		if len(payload) == 0 {
			return "", fmt.Errorf("downloaded file is empty")
		}
		return base64.StdEncoding.EncodeToString(payload), nil
	}

	pythonCode := fmt.Sprintf(`import base64
with open(%q, 'rb') as f:
    content = f.read()
    encoded = base64.b64encode(content).decode('utf-8')
    print(encoded)`, path)

	result, err := a.client.Code().ExecuteCode(ctx, &sdkgo.CodeExecuteRequest{
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
		if stderr == "" && len(result.Data.Outputs) > 0 {
			stderr = extractStreamText(result.Data.Outputs, "stderr")
		}
		return "", fmt.Errorf("script failed with exit code %d: %s",
			*result.Data.ExitCode, stderr)
	}

	content := ""
	if result.Data.Stdout != nil {
		content = *result.Data.Stdout
	}
	if content == "" && len(result.Data.Outputs) > 0 {
		content = extractStreamText(result.Data.Outputs, "stdout")
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("no output from file read")
	}

	return strings.TrimSpace(content), nil
}

// Helper function to truncate strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func marshalForLog(v any, maxLen int) string {
	if v == nil {
		return ""
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return truncateString(string(raw), maxLen)
}

func extractStreamText(outputs []map[string]interface{}, streamName string) string {
	for _, output := range outputs {
		outType, _ := output["output_type"].(string)
		name, _ := output["name"].(string)
		if outType != "stream" || name != streamName {
			continue
		}
		if text, ok := output["text"].(string); ok {
			return text
		}
	}
	return ""
}
