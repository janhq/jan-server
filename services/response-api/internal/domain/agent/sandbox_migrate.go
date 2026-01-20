package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// AIOSandboxClient handles direct API calls to AIO Sandbox, bypassing MCP tools
type AIOSandboxClient struct {
	baseURL    string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewAIOSandboxClient creates a new AIO Sandbox client
func NewAIOSandboxClient(baseURL string, logger zerolog.Logger) *AIOSandboxClient {
	if baseURL == "" {
		logger.Warn().Msg("AIO_URL not configured, AIO Sandbox features will be disabled")
		return nil
	}

	return &AIOSandboxClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		logger: logger,
	}
}

// AIOCodeRequest represents the request body for /v1/code/execute
type AIOCodeRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

// AIOCodeResponse represents the response from /v1/code/execute
type AIOCodeResponse struct {
	Data *struct {
		Status   *string `json:"status"`
		Stdout   *string `json:"stdout"`
		Stderr   *string `json:"stderr"`
		ExitCode *int    `json:"exit_code"`
	} `json:"data"`
}

// ExecuteCode executes Python code in the AIO sandbox
func (c *AIOSandboxClient) ExecuteCode(ctx context.Context, code string, language string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("AIO Sandbox client not initialized (AIO_URL not configured)")
	}

	reqBody := AIOCodeRequest{
		Code:     code,
		Language: language,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/code/execute", c.baseURL)
	c.logger.Debug().
		Str("url", url).
		Str("language", language).
		Int("code_size", len(code)).
		Msg("Executing code in AIO Sandbox")

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error().
			Int("status_code", resp.StatusCode).
			Str("body", string(body)).
			Msg("AIO Sandbox returned error status")
		return "", fmt.Errorf("AIO Sandbox error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp AIOCodeResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Data == nil {
		return "", fmt.Errorf("no data in response")
	}

	stdout := ""
	if apiResp.Data.Stdout != nil {
		stdout = *apiResp.Data.Stdout
	}

	stderr := ""
	if apiResp.Data.Stderr != nil {
		stderr = *apiResp.Data.Stderr
	}

	exitCode := 0
	if apiResp.Data.ExitCode != nil {
		exitCode = *apiResp.Data.ExitCode
	}

	c.logger.Debug().
		Int("exit_code", exitCode).
		Int("stdout_len", len(stdout)).
		Int("stderr_len", len(stderr)).
		Msg("AIO Sandbox execution complete")

	if exitCode != 0 {
		c.logger.Error().
			Int("exit_code", exitCode).
			Str("stderr", stderr).
			Msg("AIO Sandbox execution failed")
		return "", fmt.Errorf("execution failed (exit %d): %s", exitCode, stderr)
	}

	// Combine stdout and stderr for full output
	output := stdout
	if stderr != "" {
		output += "\n" + stderr
	}

	return output, nil
}

// RenderSlidesPPTX renders a DeckSpec JSON to PPTX using the AIO Sandbox
// This bypasses MCP tools for better stability and control
func (c *AIOSandboxClient) RenderSlidesPPTX(ctx context.Context, deckSpecJSON string, renderScript string, outputPath string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("AIO Sandbox client not initialized (AIO_URL not configured)")
	}

	// Default output path if not specified
	if outputPath == "" {
		outputPath = "/home/gem/output.pptx"
	}

	c.logger.Info().
		Int("deck_spec_size", len(deckSpecJSON)).
		Int("render_script_size", len(renderScript)).
		Str("output_path", outputPath).
		Msg("Starting slide rendering in AIO Sandbox")

	// Base64 encode the deck JSON and render script to avoid escaping issues
	deckJSONB64 := base64.StdEncoding.EncodeToString([]byte(deckSpecJSON))
	renderScriptB64 := base64.StdEncoding.EncodeToString([]byte(renderScript))

	// Create combined Python code that does everything in one execution
	// This is critical - AIO Sandbox doesn't persist files between calls
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

	// Execute the combined code
	output, err := c.ExecuteCode(ctx, combinedCode, "python")
	if err != nil {
		return nil, fmt.Errorf("execute render script: %w", err)
	}

	c.logger.Debug().
		Int("output_size", len(output)).
		Msg("Received output from AIO Sandbox")

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
			Str("magic", fmt.Sprintf("%v", pptxData[:min(4, len(pptxData))])).
			Msg("Invalid PPTX magic bytes")
		return nil, fmt.Errorf("invalid PPTX: does not have ZIP magic bytes")
	}

	c.logger.Info().
		Int("pptx_size", len(pptxData)).
		Msg("✓ Successfully rendered PPTX")

	return pptxData, nil
}

// escapeForPython escapes a string for safe embedding in Python code
func escapeForPython(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
