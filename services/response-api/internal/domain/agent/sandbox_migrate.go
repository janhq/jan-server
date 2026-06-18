package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/tool"
)

// SandboxClient handles sandbox operations via MCP tools
// This replaces direct AIO API calls with MCP tool calls for unified sandbox access
type SandboxClient struct {
	mcpClient tool.MCPClient
	logger    zerolog.Logger
}

// NewSandboxClient creates a new Sandbox client that uses MCP tools
func NewSandboxClient(mcpClient tool.MCPClient, logger zerolog.Logger) *SandboxClient {
	if mcpClient == nil {
		logger.Warn().Msg("MCP client not provided, Sandbox features will be disabled")
		return nil
	}

	return &SandboxClient{
		mcpClient: mcpClient,
		logger:    logger,
	}
}

// ExecuteCode executes code in the sandbox via MCP sandbox_code_execute tool
func (c *SandboxClient) ExecuteCode(ctx context.Context, code string, language string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("Sandbox client not initialized")
	}

	c.logger.Debug().
		Str("language", language).
		Int("code_size", len(code)).
		Msg("Executing code via MCP sandbox_code_execute tool")

	// Call the sandbox_code_execute MCP tool
	result, err := c.mcpClient.CallTool(ctx, tool.CallRequest{
		Name: "sandbox_code_execute",
		Arguments: map[string]interface{}{
			"code":     code,
			"language": language,
		},
	})
	if err != nil {
		return "", fmt.Errorf("sandbox_code_execute failed: %w", err)
	}

	if result.IsError {
		return "", fmt.Errorf("sandbox execution error: %s", result.Error)
	}

	// Extract text content from result
	var output string
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			output = content.Text
			break
		}
	}

	// Parse the JSON response to extract execution details
	var execResult struct {
		Status   string `json:"status"`
		Success  bool   `json:"success"`
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
	}

	if err := json.Unmarshal([]byte(output), &execResult); err != nil {
		// If it's not JSON, return the raw output
		c.logger.Debug().
			Int("output_size", len(output)).
			Msg("Sandbox execution returned non-JSON output")
		return output, nil
	}

	c.logger.Debug().
		Str("status", execResult.Status).
		Bool("success", execResult.Success).
		Int("exit_code", execResult.ExitCode).
		Int("stdout_len", len(execResult.Stdout)).
		Int("stderr_len", len(execResult.Stderr)).
		Msg("Sandbox execution complete")

	if !execResult.Success || execResult.ExitCode != 0 {
		c.logger.Error().
			Int("exit_code", execResult.ExitCode).
			Str("stderr", execResult.Stderr).
			Msg("Sandbox execution failed")
		return "", fmt.Errorf("execution failed (exit %d): %s", execResult.ExitCode, execResult.Stderr)
	}

	// Combine stdout and stderr for full output
	result_output := execResult.Stdout
	if execResult.Stderr != "" {
		result_output += "\n" + execResult.Stderr
	}

	return result_output, nil
}

// ShellExec executes a shell command in the sandbox via MCP sandbox_shell_exec tool
func (c *SandboxClient) ShellExec(ctx context.Context, command string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("Sandbox client not initialized")
	}

	c.logger.Debug().
		Str("command", truncateForLog(command, 100)).
		Msg("Executing shell command via MCP sandbox_shell_exec tool")

	result, err := c.mcpClient.CallTool(ctx, tool.CallRequest{
		Name: "sandbox_shell_exec",
		Arguments: map[string]interface{}{
			"command": command,
		},
	})
	if err != nil {
		return "", fmt.Errorf("sandbox_shell_exec failed: %w", err)
	}

	if result.IsError {
		return "", fmt.Errorf("shell execution error: %s", result.Error)
	}

	// Extract text content from result
	var output string
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			output = content.Text
			break
		}
	}

	return output, nil
}

// FileRead reads a file from the sandbox via MCP sandbox_file_read tool
func (c *SandboxClient) FileRead(ctx context.Context, path string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("Sandbox client not initialized")
	}

	result, err := c.mcpClient.CallTool(ctx, tool.CallRequest{
		Name: "sandbox_file_read",
		Arguments: map[string]interface{}{
			"path": path,
		},
	})
	if err != nil {
		return "", fmt.Errorf("sandbox_file_read failed: %w", err)
	}

	if result.IsError {
		return "", fmt.Errorf("file read error: %s", result.Error)
	}

	// Extract text content from result
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			return content.Text, nil
		}
	}

	return "", fmt.Errorf("no content in file read result")
}

// FileWrite writes content to a file in the sandbox via MCP sandbox_file_write tool
func (c *SandboxClient) FileWrite(ctx context.Context, path, content string) error {
	if c == nil {
		return fmt.Errorf("Sandbox client not initialized")
	}

	result, err := c.mcpClient.CallTool(ctx, tool.CallRequest{
		Name: "sandbox_file_write",
		Arguments: map[string]interface{}{
			"path":    path,
			"content": content,
		},
	})
	if err != nil {
		return fmt.Errorf("sandbox_file_write failed: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("file write error: %s", result.Error)
	}

	return nil
}

// InstallPackages installs packages in the sandbox via MCP sandbox_install_packages tool
func (c *SandboxClient) InstallPackages(ctx context.Context, packages []string, manager string) error {
	if c == nil {
		return fmt.Errorf("Sandbox client not initialized")
	}

	if manager == "" {
		manager = "pip"
	}

	c.logger.Debug().
		Strs("packages", packages).
		Str("manager", manager).
		Msg("Installing packages via MCP sandbox_install_packages tool")

	result, err := c.mcpClient.CallTool(ctx, tool.CallRequest{
		Name: "sandbox_install_packages",
		Arguments: map[string]interface{}{
			"packages": packages,
			"manager":  manager,
		},
	})
	if err != nil {
		return fmt.Errorf("sandbox_install_packages failed: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("package installation error: %s", result.Error)
	}

	return nil
}

// Helper functions

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Legacy compatibility - these are deprecated and will be removed
// Use SandboxClient instead

// AIOSandboxClient is deprecated - use SandboxClient instead
// Kept for backwards compatibility during migration
type AIOSandboxClient = SandboxClient

// NewAIOSandboxClient is deprecated - use NewSandboxClient instead
// This function now creates a nil client since we require MCP
func NewAIOSandboxClient(baseURL string, logger zerolog.Logger) *AIOSandboxClient {
	logger.Warn().
		Str("base_url", baseURL).
		Msg("NewAIOSandboxClient is deprecated - direct AIO calls are no longer supported. Use NewSandboxClient with MCP client instead.")
	return nil
}
