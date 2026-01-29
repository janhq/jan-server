package steps

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	slidecreatorassets "jan-server/services/response-api/assets/slide_creator"
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"

	"github.com/rs/zerolog/log"
)

// ensurePlaywrightBrowsersMCP installs Playwright browsers via MCP sandbox_shell_exec tool
// Key fixes: cleans up incomplete installations, copies chrome to headless shell path if missing
func ensurePlaywrightBrowsersMCP(ctx context.Context, mcpClient planners.MCPClient, cacheDir, conversationID, requestID string) error {
	// cacheDir is relative (e.g., ".cache/pptx_export")
	// Use $HOME for cross-provider compatibility (E2B: /home/user, AIO: /home/gem)
	fullCacheDir := "$HOME/" + cacheDir
	browsersDir := fullCacheDir + "/pw_browsers"
	nodeEnvDir := fullCacheDir + "/node_env"

	// Commands to run (using sudo for system deps)
	commands := []struct {
		name string
		cmd  string
	}{
		{
			name: "Create cache directories",
			cmd:  fmt.Sprintf("mkdir -p %s %s %s/outputs && chmod -R 777 %s 2>/dev/null || true", nodeEnvDir, browsersDir, fullCacheDir, fullCacheDir),
		},
		{
			name: "Initialize npm package",
			cmd:  fmt.Sprintf("cd %s && [ -f package.json ] || npm init -y 2>/dev/null", nodeEnvDir),
		},
		{
			name: "Install npm packages",
			cmd:  fmt.Sprintf("cd %s && PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npm install --no-fund --no-audit playwright pptxgenjs 2>&1", nodeEnvDir),
		},
		{
			name: "Install Playwright system dependencies",
			cmd:  fmt.Sprintf("cd %s && sudo npx playwright install-deps 2>&1 || true", nodeEnvDir),
		},
		{
			name: "Clean up incomplete browser installations",
			cmd:  fmt.Sprintf(`cd %s && for d in chromium_headless_shell-*; do if [ -d "$d" ]; then exe="$d/chrome-headless-shell-linux64/chrome-headless-shell"; if [ ! -f "$exe" ]; then echo "Removing incomplete: $d"; rm -rf "$d"; fi; fi; done 2>/dev/null; echo "Cleanup done"`, browsersDir),
		},
		{
			name: "Install Playwright browsers",
			cmd:  fmt.Sprintf(`cd %s && echo "Playwright version:" && npx playwright --version 2>&1 && echo "Installing browsers..." && PLAYWRIGHT_BROWSERS_PATH=%s npx playwright install chromium 2>&1`, nodeEnvDir, browsersDir),
		},
		{
			name: "Check and fix headless shell",
			cmd:  fmt.Sprintf(`cd %s && echo "=== Current browser structure ===" && ls -la 2>&1 && EXPECTED="%s/chromium_headless_shell-1200/chrome-headless-shell-linux64/chrome-headless-shell" && if [ -f "$EXPECTED" ]; then echo "Headless shell exists at $EXPECTED"; else echo "Headless shell MISSING at $EXPECTED" && echo "Checking directory structure:" && ls -laR chromium_headless_shell-* 2>&1 | head -30 && CHROME="%s/chromium-1200/chrome-linux64/chrome" && if [ -f "$CHROME" ]; then echo "Chrome found, creating headless shell structure..." && mkdir -p chromium_headless_shell-1200/chrome-headless-shell-linux64 && cp "$CHROME" "$EXPECTED" 2>&1 && chmod +x "$EXPECTED" 2>&1 && echo "Copied chrome to $EXPECTED"; else echo "Chrome also missing at $CHROME"; fi; fi`, browsersDir, browsersDir, browsersDir),
		},
		{
			name: "Verify browser paths",
			cmd:  fmt.Sprintf(`echo '=== Browser directories ===' && ls -la %s/ 2>&1 && echo '=== All chrome executables ===' && find %s -type f \( -name 'chrome' -o -name 'chrome-headless-shell' -o -name 'chromium' \) 2>/dev/null && echo '=== Checking headless shell ===' && ls -la %s/chromium_headless_shell-*/chrome-headless-shell-linux64/ 2>&1 || echo 'Headless shell dir not found' && echo '=== Checking regular chromium ===' && ls -la %s/chromium-*/chrome-linux64/ 2>&1 || echo 'Regular chromium dir not found'`, browsersDir, browsersDir, browsersDir, browsersDir),
		},
	}

	for _, c := range commands {
		log.Debug().Str("command", c.name).Msg("[slide_creator] Running shell command via MCP")

		result, err := mcpClient.CallTool(ctx, tool.CallRequest{
			Name: "sandbox_shell_exec",
			Arguments: map[string]interface{}{
				"command": c.cmd,
			},
			ConversationID: conversationID,
			RequestID:      requestID,
		})
		if err != nil {
			return fmt.Errorf("sandbox_shell_exec failed for %s: %w", c.name, err)
		}

		if result.IsError {
			return fmt.Errorf("shell exec error for %s: %s", c.name, result.Error)
		}

		// Extract output for logging
		output := extractTextContent(result)
		if output != "" {
			lines := strings.Split(output, "\n")
			for i, line := range lines {
				if i < 5 {
					log.Debug().Str("line", line).Msg("[slide_creator] Shell output")
				} else if i == 5 {
					log.Debug().Int("remaining", len(lines)-5).Msg("[slide_creator] ... more lines")
					break
				}
			}
		}
	}

	return nil
}

func (e *SlideCreatorExecutor) executeExportPPTXViaMCP(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	if e.mcpClient == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MCP_NOT_CONFIGURED",
				Message:  "MCP client not configured for PPTX export",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Start sandbox automatically before PPTX export (E2B only)
	if e.sandboxHelper != nil {
		opts := agent.StartSandboxOptions{
			Timeout: 1800, // 30 minutes
		}
		if input.PlanContext != nil {
			opts.ConversationID = input.PlanContext.ConversationID
			opts.RequestID = input.PlanContext.ResponseID
		}
		if opts.ConversationID != "" {
			if err := e.sandboxHelper.EnsureSandboxStarted(ctx, opts); err != nil {
				log.Warn().Err(err).Msg("[slide_creator] failed to start sandbox, continuing anyway")
				// Don't fail - sandbox might already be running
			}
		}
	}

	outDir := extractOutputDir(input)
	if outDir == "" {
		outDir = outputDirForPlan(input)
	}
	if _, err := os.Stat(outDir); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "OUTPUT_DIR_ERROR",
				Message:  fmt.Sprintf("output directory not found: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	mode := strings.ToLower(strings.TrimSpace(stringValue(params, "mode")))
	if mode == "" {
		mode = "dom"
	}

	payloads, rootName, err := collectInputFiles(outDir)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "EXPORT_INPUT_ERROR",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	outputFile := "presentation.pptx"
	// Cache dir is relative to workspace - the JS code will resolve it using process.cwd()
	// e2b-service sets cwd to /home/user/workspace/<conversation_id>
	// So ".cache/pptx_export" becomes /home/user/workspace/<conv_id>/.cache/pptx_export
	cacheDir := ".cache/pptx_export"

	// Extract conversation context for sandbox tool calls
	var conversationID, requestID string
	if input.PlanContext != nil {
		conversationID = input.PlanContext.ConversationID
		requestID = input.PlanContext.ResponseID
	}

	// Ensure Playwright browsers are installed via MCP sandbox_shell_exec
	log.Info().Str("cache_dir", cacheDir).Str("conversation_id", conversationID).Msg("[slide_creator] Ensuring Playwright browsers are installed via MCP")
	if err := ensurePlaywrightBrowsersMCP(ctx, e.mcpClient, cacheDir, conversationID, requestID); err != nil {
		log.Warn().Err(err).Msg("[slide_creator] Playwright browser installation may have failed, continuing anyway")
	}

	// Build the combined code that will be executed in the sandbox
	inputsJSON, _ := json.Marshal(payloads)
	code := buildSandboxWrapperCode(rootName, outputFile, mode, cacheDir, string(inputsJSON), slidecreatorassets.ExportPPTXFullScript)

	log.Debug().Int("code_size", len(code)).Msg("[slide_creator] Executing code via MCP sandbox_code_execute")

	// Execute via MCP sandbox_code_execute tool
	result, err := e.mcpClient.CallTool(ctx, tool.CallRequest{
		Name: "sandbox_code_execute",
		Arguments: map[string]interface{}{
			"code":     code,
			"language": "nodejs",
		},
		ConversationID: conversationID,
		RequestID:      requestID,
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "SANDBOX_EXPORT_FAILED",
				Message:  fmt.Sprintf("sandbox_code_execute failed: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	if result.IsError {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "SANDBOX_EXPORT_ERROR",
				Message:  fmt.Sprintf("sandbox execution error: %s", result.Error),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	stdout := extractTextContent(result)

	log.Debug().Int("output_size", len(stdout)).Msg("[slide_creator] Received output from sandbox")

	// Extract PPTX base64
	pptxB64, err := extractBase64FromOutput(stdout)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "EXPORT_PARSE_ERROR",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	pptxBytes, err := base64.StdEncoding.DecodeString(pptxB64)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "EXPORT_DECODE_ERROR",
				Message:  fmt.Sprintf("failed to decode PPTX base64: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	pptxPath := filepath.Join(outDir, outputFile)
	if err := os.WriteFile(pptxPath, pptxBytes, 0644); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "EXPORT_WRITE_ERROR",
				Message:  fmt.Sprintf("failed to write PPTX: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	// Extract slide images from output
	imageNames := []string{}
	images, imgErr := extractImagesFromOutput(stdout)
	if imgErr != nil {
		log.Warn().Err(imgErr).Msg("[slide_creator] failed to parse slide images from export output")
	}
	for name, b64 := range images {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			log.Warn().Err(err).Str("image", name).Msg("[slide_creator] failed to decode slide image")
			continue
		}
		if err := os.WriteFile(filepath.Join(outDir, name), data, 0644); err != nil {
			log.Warn().Err(err).Str("image", name).Msg("[slide_creator] failed to write slide image")
			continue
		}
		imageNames = append(imageNames, name)
	}
	sort.Strings(imageNames)

	// If no images were extracted from stdout, try to download from sandbox
	if len(imageNames) == 0 {
		outputDirFromStdout := extractOutputDirFromOutput(stdout)
		if outputDirFromStdout == "" {
			// Fallback: only used when OUTPUT_DIR marker not in stdout (code execution likely crashed)
			// This relative path works for E2B (resolved against workspace) but may fail for AIO
			// depending on Python cwd. Best-effort recovery only.
			outputDirFromStdout = ".outputs"
			log.Warn().Str("fallback_dir", outputDirFromStdout).Msg("[slide_creator] OUTPUT_DIR not found in stdout, using fallback")
		}
		if outputDirFromStdout != "" {
			downloaded, err := downloadSlideImagesMCP(ctx, e.mcpClient, outputDirFromStdout, outDir, conversationID, requestID)
			if err != nil {
				log.Warn().Err(err).Msg("[slide_creator] failed to download slide images from sandbox")
			} else if len(downloaded) > 0 {
				imageNames = append(imageNames, downloaded...)
				sort.Strings(imageNames)
			}
		}
	}

	output := map[string]interface{}{
		"type":        "pptx_export",
		"output_dir":  outDir,
		"pptx_path":   pptxPath,
		"pptx_file":   outputFile,
		"pptx_size":   len(pptxBytes),
		"images":      imageNames,
		"image_count": len(imageNames),
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

// collectInputFiles collects files from output directory for sandbox export
func collectInputFiles(outDir string) ([]filePayload, string, error) {
	root := filepath.Clean(outDir)
	rootName := filepath.Base(root)
	if rootName == "" || rootName == "." {
		rootName = "slides"
	}

	payloads := []filePayload{}
	allowed := map[string]bool{
		".html":  true,
		".json":  true,
		".css":   true,
		".svg":   true,
		".png":   true,
		".jpg":   true,
		".jpeg":  true,
		".webp":  true,
		".gif":   true,
		".txt":   true,
		".md":    true,
		".woff":  true,
		".woff2": true,
		".ttf":   true,
		".otf":   true,
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".pptx" || ext == ".zip" {
			return nil
		}
		if !allowed[ext] {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		payloads = append(payloads, filePayload{
			Path: filepath.ToSlash(filepath.Join(rootName, rel)),
			B64:  base64.StdEncoding.EncodeToString(data),
		})
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	if len(payloads) == 0 {
		return nil, "", fmt.Errorf("no exportable files found in %s", outDir)
	}
	return payloads, rootName, nil
}

// buildSandboxWrapperCode builds the NodeJS code to execute in the sandbox
func buildSandboxWrapperCode(inputDir, outFile, mode, cacheDir, inputsJSON, exportScript string) string {
	// Base64 encode inputs to avoid escaping issues
	inputsB64 := base64.StdEncoding.EncodeToString([]byte(inputsJSON))
	exportScriptB64 := base64.StdEncoding.EncodeToString([]byte(exportScript))

	code := fmt.Sprintf(`
const fs = require("fs");
const path = require("path");
const { spawnSync, execSync } = require("child_process");

const workdir = process.cwd();
const INPUT_DIR = %s;
const OUT_FILE = %s;
const MODE = %s;
// CACHE_DIR uses $HOME for cross-provider compatibility (E2B: /home/user, AIO: /home/gem)
// Cache can be shared across conversations since it's just browsers/npm packages
const CACHE_DIR = path.join(process.env.HOME || workdir, %s);

const RUN_ID = String(Date.now());
// OUTPUT_DIR must be inside workspace for sandbox_file_read compatibility
// (e2b-service rejects paths that escape workspace)
const OUTPUT_DIR = path.join(workdir, ".outputs", RUN_ID);
const OUT_PATH = path.join(OUTPUT_DIR, path.basename(OUT_FILE));

function run(cmd, args, opts) {
  const res = spawnSync(cmd, args, Object.assign({ encoding: "utf8", maxBuffer: 50 * 1024 * 1024 }, opts || {}));
  if (res.stdout) process.stdout.write(res.stdout);
  if (res.stderr) process.stderr.write(res.stderr);
  const code = (typeof res.status === "number") ? res.status : 1;
  return { code };
}

console.log("[SANDBOX] workdir=" + workdir);
console.log("[SANDBOX] cache_dir=" + CACHE_DIR);

// Decode base64 inputs
const inputsB64 = %s;
const exportScriptB64 = %s;
const files = JSON.parse(Buffer.from(inputsB64, 'base64').toString('utf8'));
const exportJs = Buffer.from(exportScriptB64, 'base64').toString('utf8');

// Write input files into temp workdir
for (const f of files) {
  const fullPath = path.join(workdir, f.path);
  fs.mkdirSync(path.dirname(fullPath), { recursive: true });
  fs.writeFileSync(fullPath, Buffer.from(f.b64, "base64"));
}
fs.writeFileSync(path.join(workdir, "export_pptx_full.js"), exportJs);
console.log("[SANDBOX] wrote " + files.length + " input files + export_pptx_full.js");

// Persistent env dirs
const nodeEnvDir = path.join(CACHE_DIR, "node_env");
const browsersDir = path.join(CACHE_DIR, "pw_browsers");
fs.mkdirSync(nodeEnvDir, { recursive: true });
fs.mkdirSync(browsersDir, { recursive: true });
fs.mkdirSync(OUTPUT_DIR, { recursive: true });

const pkgJson = path.join(nodeEnvDir, "package.json");
if (!fs.existsSync(pkgJson)) {
  fs.writeFileSync(pkgJson, JSON.stringify({ name: "pptx_export_env", private: true }, null, 2));
}

// Ensure deps
const nmPlaywright = path.join(nodeEnvDir, "node_modules", "playwright");
const nmPptx = path.join(nodeEnvDir, "node_modules", "pptxgenjs");
if (!fs.existsSync(nmPlaywright) || !fs.existsSync(nmPptx)) {
  console.log("[SANDBOX] Installing npm deps (playwright, pptxgenjs) into " + nodeEnvDir);
  const envNpm = Object.assign({}, process.env, {
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: "1",
  });
  let r = run("npm", ["install", "--no-fund", "--no-audit", "playwright", "pptxgenjs"], { cwd: nodeEnvDir, env: envNpm });
  if (r.code !== 0) {
    console.error("[SANDBOX] ERROR: npm install failed");
    process.exit(2);
  }
} else {
  console.log("[SANDBOX] npm deps already present in cache");
}

// Check browser installation
console.log("[SANDBOX] Checking browser installation in " + browsersDir);
try {
  const dirs = fs.readdirSync(browsersDir);
  console.log("[SANDBOX] Browser dirs: " + dirs.join(", "));
} catch (e) {
  console.log("[SANDBOX] Could not list browser dirs: " + e.message);
}

// Install browsers if needed
const pwBin = path.join(nodeEnvDir, "node_modules", ".bin", "playwright");
if (fs.existsSync(pwBin)) {
  console.log("[SANDBOX] Ensuring browsers are installed via " + pwBin);
  const envPwInstall = Object.assign({}, process.env, {
    PLAYWRIGHT_BROWSERS_PATH: browsersDir,
  });
  let r = run(pwBin, ["install", "chromium"], { cwd: nodeEnvDir, env: envPwInstall });
  if (r.code !== 0) {
    console.warn("[SANDBOX] WARNING: playwright install chromium returned " + r.code);
  }
}

// Find chrome executables
let chromeExe = "";
let headlessShellExe = "";
try {
  const findChrome = execSync("find " + browsersDir + " -type f -name 'chrome' -executable 2>/dev/null | grep -v headless | head -1", { encoding: "utf8" });
  chromeExe = findChrome.trim();
  if (chromeExe) console.log("[SANDBOX] Found chrome: " + chromeExe);

  const findHeadless = execSync("find " + browsersDir + " -type f -name 'chrome-headless-shell' -executable 2>/dev/null | head -1", { encoding: "utf8" });
  headlessShellExe = findHeadless.trim();
  if (headlessShellExe) console.log("[SANDBOX] Found headless shell: " + headlessShellExe);

  if (!headlessShellExe) {
    const findAny = execSync("find " + browsersDir + "/chromium_headless_shell* -type f -executable 2>/dev/null | head -1", { encoding: "utf8" });
    headlessShellExe = findAny.trim();
    if (headlessShellExe) console.log("[SANDBOX] Found alternative headless: " + headlessShellExe);
  }
} catch (e) {
  console.log("[SANDBOX] Error finding chrome: " + e.message);
}

// Copy chrome to headless shell path if missing
if (!headlessShellExe && chromeExe) {
  console.log("[SANDBOX] Copying chrome to headless shell path...");
  const targetDir = path.join(browsersDir, "chromium_headless_shell-1200", "chrome-headless-shell-linux64");
  const targetFile = path.join(targetDir, "chrome-headless-shell");
  try {
    fs.mkdirSync(targetDir, { recursive: true });
    try { fs.unlinkSync(targetFile); } catch (e) {}
    fs.copyFileSync(chromeExe, targetFile);
    fs.chmodSync(targetFile, 0o755);
    headlessShellExe = targetFile;
    console.log("[SANDBOX] Copied chrome to: " + targetFile);
  } catch (e) {
    console.log("[SANDBOX] Could not copy chrome: " + e.message);
  }
}

const envRun = Object.assign({}, process.env, {
  NODE_PATH: path.join(nodeEnvDir, "node_modules"),
  AIO_CACHE_DIR: CACHE_DIR,
  PLAYWRIGHT_BROWSERS_PATH: browsersDir,
  PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: "0",
});

if (chromeExe) envRun.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH = chromeExe;
if (headlessShellExe) envRun.PLAYWRIGHT_CHROMIUM_HEADLESS_SHELL_EXECUTABLE_PATH = headlessShellExe;

const args = [path.join(workdir, "export_pptx_full.js"), "--in", INPUT_DIR, "--out", OUT_PATH, "--mode", MODE];
console.log("[SANDBOX] Running: node " + args.join(" "));
const res = spawnSync("node", args, { cwd: workdir, env: envRun, stdio: "inherit", maxBuffer: 50 * 1024 * 1024 });
if (typeof res.status === "number" && res.status !== 0) process.exit(res.status);

// Return PPTX as base64
const pptx = fs.readFileSync(OUT_PATH);
// Output absolute path for cross-provider compatibility:
// - E2B: e2b-service accepts absolute paths within workspace (passes startswith check)
// - AIO: Python opens absolute path directly without cwd dependency
console.log("===OUTPUT_DIR===");
console.log(OUTPUT_DIR);
console.log("===OUTPUT_DIR_START===");
console.log(OUTPUT_DIR);
console.log("===OUTPUT_DIR_END===");
console.log("===BASE64_START===");
console.log(pptx.toString("base64"));
console.log("===BASE64_END===");

// Get slide images
let images = {};
try {
  const imageFiles = fs.readdirSync(OUTPUT_DIR).filter((f) => /^slide-\d+\.png$/i.test(f));
  for (const f of imageFiles) {
    images[f] = fs.readFileSync(path.join(OUTPUT_DIR, f)).toString("base64");
  }
} catch (e) {
  console.log("[SANDBOX] Could not read slide images: " + e.message);
}
console.log("===IMAGES_START===");
console.log(JSON.stringify(images));
console.log("===IMAGES_END===");
`,
		jsQuoteMCP(inputDir),
		jsQuoteMCP(outFile),
		jsQuoteMCP(mode),
		jsQuoteMCP(cacheDir),
		jsQuoteMCP(inputsB64),
		jsQuoteMCP(exportScriptB64),
	)

	return strings.TrimSpace(code) + "\n"
}

func jsQuoteMCP(s string) string {
	encoded, _ := json.Marshal(s)
	return string(encoded)
}

// extractTextContent extracts text content from MCP result
// If the content is a JSON object with a "stdout" field, extracts that instead
func extractTextContent(result *tool.Result) string {
	if result == nil {
		return ""
	}
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			text := content.Text
			// Check if this is a JSON response with stdout field (from sandbox_code_execute)
			var codeResult struct {
				Stdout string `json:"stdout"`
			}
			if err := json.Unmarshal([]byte(text), &codeResult); err == nil && codeResult.Stdout != "" {
				return codeResult.Stdout
			}
			return text
		}
	}
	return ""
}

func extractBase64FromOutput(output string) (string, error) {
	start := strings.Index(output, "===BASE64_START===")
	end := strings.Index(output, "===BASE64_END===")
	if start == -1 || end == -1 || end <= start {
		return "", fmt.Errorf("base64 markers not found in output")
	}
	chunk := strings.TrimSpace(output[start+len("===BASE64_START===") : end])
	if chunk == "" {
		return "", fmt.Errorf("base64 content empty")
	}
	return chunk, nil
}

func extractImagesFromOutput(output string) (map[string]string, error) {
	start := strings.Index(output, "===IMAGES_START===")
	end := strings.Index(output, "===IMAGES_END===")
	if start == -1 || end == -1 || end <= start {
		return map[string]string{}, nil
	}
	chunk := strings.TrimSpace(output[start+len("===IMAGES_START===") : end])
	if chunk == "" {
		return map[string]string{}, nil
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(chunk), &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func extractOutputDirFromOutput(output string) string {
	singleMarker := "===OUTPUT_DIR==="
	if idx := strings.Index(output, singleMarker); idx != -1 {
		rest := output[idx+len(singleMarker):]
		rest = strings.TrimLeft(rest, "\r\n")
		if rest == "" {
			return ""
		}
		if lineEnd := strings.IndexAny(rest, "\r\n"); lineEnd != -1 {
			rest = rest[:lineEnd]
		}
		return strings.TrimSpace(rest)
	}
	start := strings.Index(output, "===OUTPUT_DIR_START===")
	end := strings.Index(output, "===OUTPUT_DIR_END===")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	chunk := strings.TrimSpace(output[start+len("===OUTPUT_DIR_START===") : end])
	return chunk
}

// downloadSlideImagesMCP downloads slide images from sandbox using MCP tools
func downloadSlideImagesMCP(ctx context.Context, mcpClient planners.MCPClient, outputDir, localDir, conversationID, requestID string) ([]string, error) {
	// List files in sandbox output directory via MCP
	listResult, err := mcpClient.CallTool(ctx, tool.CallRequest{
		Name: "sandbox_file_list",
		Arguments: map[string]interface{}{
			"path": outputDir,
		},
		ConversationID: conversationID,
		RequestID:      requestID,
	})
	if err != nil {
		return nil, fmt.Errorf("sandbox_file_list failed: %w", err)
	}

	if listResult.IsError {
		return nil, fmt.Errorf("file list error: %s", listResult.Error)
	}

	// Parse file list from result
	listOutput := extractTextContent(listResult)
	if listOutput == "" {
		return nil, nil
	}

	// Try to parse as JSON array (mcp-tools format: [{"name": "...", "is_dir": false, "size": 123}, ...])
	var fileArray []struct {
		Name  string `json:"name"`
		IsDir bool   `json:"is_dir"`
		Size  int64  `json:"size"`
	}
	if err := json.Unmarshal([]byte(listOutput), &fileArray); err != nil {
		// Try legacy format: {"files": [...]}
		var fileList struct {
			Files []struct {
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"files"`
		}
		if err := json.Unmarshal([]byte(listOutput), &fileList); err != nil {
			// If not JSON, try line-by-line parsing
			return parseAndDownloadImages(ctx, mcpClient, listOutput, outputDir, localDir, conversationID, requestID)
		}
		// Convert legacy format to array format
		for _, f := range fileList.Files {
			fileArray = append(fileArray, struct {
				Name  string `json:"name"`
				IsDir bool   `json:"is_dir"`
				Size  int64  `json:"size"`
			}{Name: f.Name})
		}
	}

	var names []string
	for _, f := range fileArray {
		name := f.Name
		if f.IsDir {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(name), "slide-") || !strings.HasSuffix(strings.ToLower(name), ".png") {
			continue
		}

		// Read file from sandbox (path is relative to workspace)
		path := outputDir + "/" + name

		readResult, err := mcpClient.CallTool(ctx, tool.CallRequest{
			Name: "sandbox_file_read",
			Arguments: map[string]interface{}{
				"path": path,
			},
			ConversationID: conversationID,
			RequestID:      requestID,
		})
		if err != nil {
			log.Warn().Err(err).Str("file", name).Msg("[slide_creator] failed to read image from sandbox")
			continue
		}

		if readResult.IsError {
			log.Warn().Str("error", readResult.Error).Str("file", name).Msg("[slide_creator] sandbox file read error")
			continue
		}

		content := extractTextContent(readResult)
		if content == "" {
			continue
		}

		// Content might be base64 encoded
		var data []byte
		if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
			data = decoded
		} else {
			data = []byte(content)
		}

		localPath := filepath.Join(localDir, name)
		if err := os.WriteFile(localPath, data, 0644); err != nil {
			log.Warn().Err(err).Str("file", name).Msg("[slide_creator] failed to write image")
			continue
		}
		names = append(names, name)
	}

	return names, nil
}

func parseAndDownloadImages(ctx context.Context, mcpClient planners.MCPClient, listOutput, outputDir, localDir, conversationID, requestID string) ([]string, error) {
	var names []string
	lines := strings.Split(listOutput, "\n")
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(name), "slide-") || !strings.HasSuffix(strings.ToLower(name), ".png") {
			continue
		}

		path := outputDir + "/" + name

		readResult, err := mcpClient.CallTool(ctx, tool.CallRequest{
			Name: "sandbox_file_read",
			Arguments: map[string]interface{}{
				"path": path,
			},
			ConversationID: conversationID,
			RequestID:      requestID,
		})
		if err != nil {
			continue
		}
		if readResult.IsError {
			continue
		}

		content := extractTextContent(readResult)
		if content == "" {
			continue
		}

		var data []byte
		if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
			data = decoded
		} else {
			data = []byte(content)
		}

		localPath := filepath.Join(localDir, name)
		if err := os.WriteFile(localPath, data, 0644); err != nil {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}
