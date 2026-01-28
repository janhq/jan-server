package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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

// RenderSlidesPPTX renders a DeckSpec JSON to PPTX using the sandbox via MCP tools
func (c *SandboxClient) RenderSlidesPPTX(ctx context.Context, deckSpecJSON string, renderScript string, outputPath string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("Sandbox client not initialized")
	}

	// Default output path if not specified
	if outputPath == "" {
		outputPath = "/home/gem/output.pptx"
	}

	c.logger.Info().
		Int("deck_spec_size", len(deckSpecJSON)).
		Int("render_script_size", len(renderScript)).
		Str("output_path", outputPath).
		Msg("Starting slide rendering via MCP sandbox tools")

	// Base64 encode the deck JSON and render script to avoid escaping issues
	deckJSONB64 := base64.StdEncoding.EncodeToString([]byte(deckSpecJSON))
	renderScriptB64 := base64.StdEncoding.EncodeToString([]byte(renderScript))

	// Create combined Python code that does everything in one execution
	// This is critical - sandbox doesn't persist files between calls
	combinedCode := fmt.Sprintf(`
import sys
import os
import subprocess
import base64

print("=== Step 1: Writing JSON file ===")
json_path = "/home/gem/slide_spec.json"
deck_json = base64.b64decode(%s).decode('utf-8')
os.makedirs(os.path.dirname(json_path), exist_ok=True)
with open(json_path, 'w') as f:
    f.write(deck_json)
print(f"✓ JSON written: {len(deck_json)} bytes to {json_path}")
print(f"✓ File exists: {os.path.exists(json_path)}")

print("\n=== Step 2: Writing render_deck.py ===")
render_path = "/home/gem/render_deck.py"
render_script_content = base64.b64decode(%s).decode('utf-8')
with open(render_path, 'w') as f:
    f.write(render_script_content)
print(f"✓ Script written: {len(render_script_content)} bytes to {render_path}")
print(f"✓ File exists: {os.path.exists(render_path)}")

# Verify file was written correctly
with open(render_path, 'r') as f:
    written_content = f.read()
    if len(written_content) != len(render_script_content):
        print(f"ERROR: File size mismatch! Expected {len(render_script_content)}, got {len(written_content)}", file=sys.stderr)
        sys.exit(1)
    print(f"✓ File content verified: {len(written_content)} bytes")

print("\n=== Step 3: Executing render script ===")
output_path = %s

# Diagnostic: Check if python-pptx is available
try:
    import pptx
    print(f"✓ python-pptx is available: {pptx.__version__}")
except ImportError as e:
    print(f"ERROR: python-pptx not available: {e}", file=sys.stderr)
    sys.exit(1)

# Diagnostic: Check script syntax
try:
    with open(render_path, 'r') as f:
        compile(f.read(), render_path, 'exec')
    print(f"✓ Render script syntax is valid")
except SyntaxError as e:
    print(f"ERROR: Render script has syntax error: {e}", file=sys.stderr)
    sys.exit(1)

# Test if script can at least print something
print(f"✓ Testing script execution...")
test_result = subprocess.run(
    [sys.executable, '-c', 'import sys; print("PYTHON_WORKS", file=sys.stderr); sys.stderr.flush()'],
    capture_output=True,
    text=True,
    timeout=5
)
if "PYTHON_WORKS" in test_result.stderr:
    print(f"✓ Python subprocess can produce stderr output")
else:
    print(f"WARNING: Python subprocess test failed: stdout={test_result.stdout}, stderr={test_result.stderr}", file=sys.stderr)

print(f"✓ About to execute: {sys.executable} {render_path} {json_path} {output_path}")
import sys
sys.stdout.flush()
try:
    result = subprocess.run(
        [sys.executable, render_path, json_path, output_path],
        capture_output=True,
        text=True,
        timeout=120,
        env=os.environ.copy()  # Ensure environment is passed
    )
    print(f"✓ Render script exit code: {result.returncode}")
    print(f"STDOUT ({len(result.stdout)} bytes):")
    if result.stdout:
        print(result.stdout)
    else:
        print("  (empty)")
    print(f"STDERR ({len(result.stderr)} bytes):")
    if result.stderr:
        print(result.stderr)
    else:
        print("  (empty)")
    sys.stdout.flush()

    if result.returncode != 0:
        print(f"ERROR: Render script failed with code {result.returncode}", file=sys.stderr)
        sys.exit(result.returncode)
except subprocess.TimeoutExpired:
    print("ERROR: Render script timed out after 120 seconds", file=sys.stderr)
    sys.exit(1)
except Exception as e:
    print(f"ERROR executing render script: {e}", file=sys.stderr)
    sys.exit(1)

print("\n=== Step 4: Reading output file ===")
if not os.path.exists(output_path):
    print(f"ERROR: Output file {output_path} was not created", file=sys.stderr)
    print(f"Files in {os.path.dirname(output_path)}: {os.listdir(os.path.dirname(output_path))}", file=sys.stderr)
    sys.exit(1)

with open(output_path, 'rb') as f:
    pptx_data = f.read()

print(f"✓ Generated PPTX size: {len(pptx_data)} bytes")

if len(pptx_data) < 1000:
    print(f"ERROR: Generated PPTX is too small ({len(pptx_data)} bytes)", file=sys.stderr)
    sys.exit(1)

print("\n=== Step 5: Base64 encoding ===")
encoded = base64.b64encode(pptx_data).decode('ascii')
print(f"✓ Encoded length: {len(encoded)} characters")
print("=== BASE64_START ===")
print(encoded)
print("=== BASE64_END ===")
`, strconv.Quote(deckJSONB64), strconv.Quote(renderScriptB64), strconv.Quote(outputPath))

	c.logger.Debug().
		Int("combined_code_size", len(combinedCode)).
		Msg("Generated combined Python code")

	// Execute the combined code via MCP tool
	output, err := c.ExecuteCode(ctx, combinedCode, "python")
	if err != nil {
		return nil, fmt.Errorf("execute render script: %w", err)
	}

	c.logger.Debug().
		Int("output_size", len(output)).
		Msg("Received output from sandbox")

	// Check for errors in output
	if strings.Contains(output, "ERROR:") {
		c.logger.Error().
			Str("output", output).
			Msg("Render script reported errors")
		return nil, fmt.Errorf("render script failed: see stderr in output")
	}

	// Extract base64 content between markers
	startMarker := "=== BASE64_START ==="
	endMarker := "=== BASE64_END ==="
	startIdx := strings.Index(output, startMarker)
	endIdx := strings.Index(output, endMarker)

	if startIdx == -1 || endIdx == -1 {
		c.logger.Error().
			Bool("has_start", startIdx != -1).
			Bool("has_end", endIdx != -1).
			Msg("Base64 markers not found in output")
		return nil, fmt.Errorf("no base64 markers found in output")
	}

	base64Content := strings.TrimSpace(output[startIdx+len(startMarker) : endIdx])

	if base64Content == "" {
		return nil, fmt.Errorf("empty base64 content")
	}

	c.logger.Info().
		Int("base64_length", len(base64Content)).
		Msg("Extracted base64-encoded PPTX")

	// Decode the PPTX
	pptxData, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	// Verify it's a valid PPTX (should start with PK zip magic bytes)
	if len(pptxData) < 4 || pptxData[0] != 'P' || pptxData[1] != 'K' {
		c.logger.Error().
			Int("size", len(pptxData)).
			Str("magic", fmt.Sprintf("%v", pptxData[:minInt(4, len(pptxData))])).
			Msg("Invalid PPTX magic bytes")
		return nil, fmt.Errorf("invalid PPTX: does not have ZIP magic bytes")
	}

	c.logger.Info().
		Int("pptx_size", len(pptxData)).
		Msg("✓ Successfully rendered PPTX")

	return pptxData, nil
}

// Helper functions

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
