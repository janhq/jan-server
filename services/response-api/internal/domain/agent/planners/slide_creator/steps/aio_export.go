package steps

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	slidecreatorassets "jan-server/services/response-api/assets/slide_creator"
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

type nodeJSExecuteRequest struct {
	Code    string            `json:"code"`
	Timeout *int              `json:"timeout,omitempty"`
	Stdin   *string           `json:"stdin,omitempty"`
	Files   map[string]string `json:"files,omitempty"`
}

type nodeJSExecuteResponse struct {
	Data *struct {
		Status   *string `json:"status"`
		Stdout   *string `json:"stdout"`
		Stderr   *string `json:"stderr"`
		ExitCode *int    `json:"exit_code"`
	} `json:"data"`
}

type filePayload struct {
	Path string `json:"path"`
	B64  string `json:"b64"`
}

type fileListRequest struct {
	Path       string   `json:"path"`
	Recursive  *bool    `json:"recursive,omitempty"`
	ShowHidden *bool    `json:"show_hidden,omitempty"`
	FileTypes  []string `json:"file_types,omitempty"`
}

type fileInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"is_directory"`
	Extension   string `json:"extension"`
}

type fileListResult struct {
	Path  string     `json:"path"`
	Files []fileInfo `json:"files"`
}

type responseWrapper[T any] struct {
	Success bool    `json:"success"`
	Message *string `json:"message"`
	Data    *T      `json:"data"`
}

// Shell execution types for AIO API
type shellExecRequest struct {
	ID        *string  `json:"id,omitempty"`
	ExecDir   *string  `json:"exec_dir,omitempty"`
	Command   string   `json:"command"`
	AsyncMode bool     `json:"async_mode"`
	Timeout   *float64 `json:"timeout,omitempty"`
}

type shellCommandResult struct {
	SessionID string  `json:"session_id"`
	Command   string  `json:"command"`
	Status    string  `json:"status"`
	Output    *string `json:"output"`
	ExitCode  *int    `json:"exit_code"`
}

type shellExecResponse struct {
	Success bool               `json:"success"`
	Message *string            `json:"message"`
	Data    *shellCommandResult `json:"data"`
}

// ensurePlaywrightBrowsers installs Playwright browsers via AIO shell API
// Key fixes: cleans up incomplete installations, copies chrome to headless shell path if missing
func ensurePlaywrightBrowsers(ctx context.Context, baseURL, cacheDir string) error {
	browsersDir := cacheDir + "/pw_browsers"
	nodeEnvDir := cacheDir + "/node_env"

	// Commands to run (using sudo for system deps)
	commands := []struct {
		name    string
		cmd     string
		timeout float64
	}{
		{
			name:    "Create cache directories",
			cmd:     fmt.Sprintf("mkdir -p %s %s %s/outputs && chmod -R 777 %s 2>/dev/null || true", nodeEnvDir, browsersDir, cacheDir, cacheDir),
			timeout: 30,
		},
		{
			name:    "Initialize npm package",
			cmd:     fmt.Sprintf("cd %s && [ -f package.json ] || npm init -y 2>/dev/null", nodeEnvDir),
			timeout: 30,
		},
		{
			name:    "Install npm packages",
			cmd:     fmt.Sprintf("cd %s && PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npm install --no-fund --no-audit playwright pptxgenjs 2>&1", nodeEnvDir),
			timeout: 180,
		},
		{
			name:    "Install Playwright system dependencies",
			cmd:     fmt.Sprintf("cd %s && sudo npx playwright install-deps 2>&1 || true", nodeEnvDir),
			timeout: 180,
		},
		{
			name:    "Clean up incomplete browser installations",
			cmd:     fmt.Sprintf(`cd %s && for d in chromium_headless_shell-*; do if [ -d "$d" ]; then exe="$d/chrome-headless-shell-linux64/chrome-headless-shell"; if [ ! -f "$exe" ]; then echo "Removing incomplete: $d"; rm -rf "$d"; fi; fi; done 2>/dev/null; echo "Cleanup done"`, browsersDir),
			timeout: 30,
		},
		{
			name:    "Install Playwright browsers",
			cmd:     fmt.Sprintf(`cd %s && echo "Playwright version:" && npx playwright --version 2>&1 && echo "Installing browsers..." && PLAYWRIGHT_BROWSERS_PATH=%s npx playwright install chromium 2>&1`, nodeEnvDir, browsersDir),
			timeout: 300,
		},
		{
			name:    "Check and fix headless shell",
			cmd:     fmt.Sprintf(`cd %s && echo "=== Current browser structure ===" && ls -la 2>&1 && EXPECTED="%s/chromium_headless_shell-1200/chrome-headless-shell-linux64/chrome-headless-shell" && if [ -f "$EXPECTED" ]; then echo "Headless shell exists at $EXPECTED"; else echo "Headless shell MISSING at $EXPECTED" && echo "Checking directory structure:" && ls -laR chromium_headless_shell-* 2>&1 | head -30 && CHROME="%s/chromium-1200/chrome-linux64/chrome" && if [ -f "$CHROME" ]; then echo "Chrome found, creating headless shell structure..." && mkdir -p chromium_headless_shell-1200/chrome-headless-shell-linux64 && cp "$CHROME" "$EXPECTED" 2>&1 && chmod +x "$EXPECTED" 2>&1 && echo "Copied chrome to $EXPECTED"; else echo "Chrome also missing at $CHROME"; fi; fi`, browsersDir, browsersDir, browsersDir),
			timeout: 60,
		},
		{
			name:    "Verify browser paths",
			cmd:     fmt.Sprintf(`echo '=== Browser directories ===' && ls -la %s/ 2>&1 && echo '=== All chrome executables ===' && find %s -type f \( -name 'chrome' -o -name 'chrome-headless-shell' -o -name 'chromium' \) 2>/dev/null && echo '=== Checking headless shell ===' && ls -la %s/chromium_headless_shell-*/chrome-headless-shell-linux64/ 2>&1 || echo 'Headless shell dir not found' && echo '=== Checking regular chromium ===' && ls -la %s/chromium-*/chrome-linux64/ 2>&1 || echo 'Regular chromium dir not found'`, browsersDir, browsersDir, browsersDir, browsersDir),
			timeout: 30,
		},
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/v1/shell/exec"
	client := &http.Client{Timeout: 10 * time.Minute}

	for _, c := range commands {
		log.Debug().Str("command", c.name).Msg("[slide_creator] Running shell command")

		req := shellExecRequest{
			Command:   c.cmd,
			AsyncMode: false,
			Timeout:   &c.timeout,
		}

		body, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			return fmt.Errorf("execute request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("shell exec failed %d: %s", resp.StatusCode, truncateForLog(string(respBody), 1000))
		}

		var result shellExecResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		if result.Data != nil {
			if result.Data.Output != nil && *result.Data.Output != "" {
				// Log output for debugging (first few lines)
				output := *result.Data.Output
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
			if result.Data.ExitCode != nil && *result.Data.ExitCode != 0 {
				log.Warn().Int("exit_code", *result.Data.ExitCode).Str("command", c.name).Msg("[slide_creator] Shell command exited with non-zero code")
			}
		}
	}

	return nil
}

func (e *SlideCreatorExecutor) executeExportPPTX(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	if strings.TrimSpace(e.aioBaseURL) == "" {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "AIO_NOT_CONFIGURED",
				Message:  "AIO_URL not configured for PPTX export",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
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
	// CDP disabled entirely - it was timing out, so we use local browsers instead
	useCDP := false

	payloads, rootName, err := collectAIOInputFiles(outDir)
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

	inputsJSON, _ := json.Marshal(payloads)
	files := map[string]string{
		"inputs.json":         string(inputsJSON),
		"export_pptx_full.js": slidecreatorassets.ExportPPTXFullScript,
	}

	baseURL := discoverAIOBase(ctx, aioBaseCandidates(e.aioBaseURL))
	if baseURL == "" {
		baseURL = strings.TrimRight(e.aioBaseURL, "/")
	}

	// CDP disabled - always use local browsers
	cdpURL := ""

	outputFile := "presentation.pptx"
	cacheDir := "/home/gem/.cache/pptx_export"

	// Ensure Playwright browsers are installed via shell API
	// This cleans up incomplete installations and copies chrome to headless shell path if needed
	log.Info().Str("cache_dir", cacheDir).Msg("[slide_creator] Ensuring Playwright browsers are installed")
	if err := ensurePlaywrightBrowsers(ctx, baseURL, cacheDir); err != nil {
		log.Warn().Err(err).Msg("[slide_creator] Playwright browser installation may have failed, continuing anyway")
		// Continue anyway - the JS code will also try to install
	}

	code := buildAIOWrapperCode(rootName, outputFile, mode, cdpURL, cacheDir, useCDP)

	stdout, err := executeAIONodeJS(ctx, baseURL, code, files, 5*time.Minute)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "AIO_EXPORT_FAILED",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	pptxB64, err := extractBase64(stdout)
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

	imageNames := []string{}
	images, imgErr := extractImages(stdout)
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

	if len(imageNames) == 0 {
		outputDir := extractOutputDirFromStdout(stdout)
		if outputDir == "" {
			outputDir = filepath.ToSlash(filepath.Join(cacheDir, "outputs"))
		}
		if outputDir != "" {
			downloaded, err := downloadSlideImages(ctx, baseURL, outputDir, outDir)
			if err != nil {
				log.Warn().Err(err).Msg("[slide_creator] failed to download slide images from AIO")
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

func collectAIOInputFiles(outDir string) ([]filePayload, string, error) {
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

func buildAIOWrapperCode(inputDir, outFile, mode, cdpURL, cacheDir string, useCDP bool) string {
	// CDP is disabled entirely - it was timing out, so we use local browsers instead
	// The useCDP and cdpURL parameters are kept for compatibility but ignored
	code := fmt.Sprintf(`
const fs = require("fs");
const path = require("path");
const { spawnSync, execSync } = require("child_process");

const workdir = process.cwd();
const INPUT_DIR = %s;
const OUT_FILE = %s;
const MODE = %s;

// CDP is unreliable (often times out), so we disable it and use local browsers
const USE_CDP = false;
const CDP_URL = "";
const CACHE_DIR = (process.env.AIO_CACHE_DIR && process.env.AIO_CACHE_DIR.trim()) ? process.env.AIO_CACHE_DIR.trim() : %s;

const RUN_ID = String(Date.now());
const OUTPUT_DIR = path.join(CACHE_DIR, "outputs", RUN_ID);
const OUT_PATH = path.join(OUTPUT_DIR, path.basename(OUT_FILE));

function run(cmd, args, opts) {
  const res = spawnSync(cmd, args, Object.assign({ encoding: "utf8", maxBuffer: 50 * 1024 * 1024 }, opts || {}));
  if (res.stdout) process.stdout.write(res.stdout);
  if (res.stderr) process.stderr.write(res.stderr);
  const code = (typeof res.status === "number") ? res.status : 1;
  return { code };
}

console.log("[AIO] workdir=" + workdir);
console.log("[AIO] cache_dir=" + CACHE_DIR);

const files = JSON.parse(fs.readFileSync(path.join(workdir, "inputs.json"), "utf8"));
const exportJs = fs.readFileSync(path.join(workdir, "export_pptx_full.js"), "utf8");

// Write input files into temp workdir
for (const f of files) {
  const fullPath = path.join(workdir, f.path);
  fs.mkdirSync(path.dirname(fullPath), { recursive: true });
  fs.writeFileSync(fullPath, Buffer.from(f.b64, "base64"));
}
fs.writeFileSync(path.join(workdir, "export_pptx_full.js"), exportJs);
console.log("[AIO] wrote " + files.length + " input files + export_pptx_full.js");

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
  console.log("[AIO] Installing npm deps (playwright, pptxgenjs) into " + nodeEnvDir);
  const envNpm = Object.assign({}, process.env, {
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: "1",
  });
  let r = run("npm", ["install", "--no-fund", "--no-audit", "playwright", "pptxgenjs"], { cwd: nodeEnvDir, env: envNpm });
  if (r.code !== 0) {
    console.error("[AIO] ERROR: npm install failed");
    process.exit(2);
  }
} else {
  console.log("[AIO] npm deps already present in cache");
}

// Check browser installation
console.log("[AIO] Checking browser installation in " + browsersDir);
try {
  const dirs = fs.readdirSync(browsersDir);
  console.log("[AIO] Browser dirs: " + dirs.join(", "));
} catch (e) {
  console.log("[AIO] Could not list browser dirs: " + e.message);
}

// Install browsers if needed (using the cached playwright)
const pwBin = path.join(nodeEnvDir, "node_modules", ".bin", "playwright");
if (fs.existsSync(pwBin)) {
  console.log("[AIO] Ensuring browsers are installed via " + pwBin);
  const envPwInstall = Object.assign({}, process.env, {
    PLAYWRIGHT_BROWSERS_PATH: browsersDir,
  });
  let r = run(pwBin, ["install", "chromium"], { cwd: nodeEnvDir, env: envPwInstall });
  if (r.code !== 0) {
    console.warn("[AIO] WARNING: playwright install chromium returned " + r.code);
  }
}

// Find actual chrome executable
let chromeExe = "";
let headlessShellExe = "";
try {
  // Find regular chrome
  const findChrome = execSync("find " + browsersDir + " -type f -name 'chrome' -executable 2>/dev/null | grep -v headless | head -1", { encoding: "utf8" });
  chromeExe = findChrome.trim();
  if (chromeExe) {
    console.log("[AIO] Found chrome executable: " + chromeExe);
  }

  // Find headless shell
  const findHeadless = execSync("find " + browsersDir + " -type f -name 'chrome-headless-shell' -executable 2>/dev/null | head -1", { encoding: "utf8" });
  headlessShellExe = findHeadless.trim();
  if (headlessShellExe) {
    console.log("[AIO] Found headless shell executable: " + headlessShellExe);
  }

  // If no headless shell, try to find any chrome in headless_shell dir
  if (!headlessShellExe) {
    const findAny = execSync("find " + browsersDir + "/chromium_headless_shell* -type f -executable 2>/dev/null | head -1", { encoding: "utf8" });
    headlessShellExe = findAny.trim();
    if (headlessShellExe) {
      console.log("[AIO] Found alternative headless executable: " + headlessShellExe);
    }
  }
} catch (e) {
  console.log("[AIO] Error finding chrome executables: " + e.message);
}

// If headless shell is missing but chrome exists, create a copy
if (!headlessShellExe && chromeExe) {
  console.log("[AIO] Headless shell missing, copying chrome to headless shell path...");
  const targetDir = path.join(browsersDir, "chromium_headless_shell-1200", "chrome-headless-shell-linux64");
  const targetFile = path.join(targetDir, "chrome-headless-shell");
  try {
    fs.mkdirSync(targetDir, { recursive: true });
    // Remove existing file if any
    try { fs.unlinkSync(targetFile); } catch (e) {}
    // Copy chrome to headless shell location
    fs.copyFileSync(chromeExe, targetFile);
    fs.chmodSync(targetFile, 0o755);
    headlessShellExe = targetFile;
    console.log("[AIO] Copied chrome to: " + targetFile);
  } catch (e) {
    console.log("[AIO] Could not copy chrome: " + e.message);
  }
}

const envRun = Object.assign({}, process.env, {
  NODE_PATH: path.join(nodeEnvDir, "node_modules"),
  AIO_CACHE_DIR: CACHE_DIR,
  AIO_CDP_URL: "", // Disabled
  PLAYWRIGHT_BROWSERS_PATH: browsersDir,
  PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: "0",
});

// Set executable paths if found
if (chromeExe) {
  envRun.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH = chromeExe;
}
if (headlessShellExe) {
  envRun.PLAYWRIGHT_CHROMIUM_HEADLESS_SHELL_EXECUTABLE_PATH = headlessShellExe;
}

const args = [path.join(workdir, "export_pptx_full.js"), "--in", INPUT_DIR, "--out", OUT_PATH, "--mode", MODE];
console.log("[AIO] Running: node " + args.join(" "));
const res = spawnSync("node", args, { cwd: workdir, env: envRun, stdio: "inherit", maxBuffer: 50 * 1024 * 1024 });
if (typeof res.status === "number" && res.status !== 0) process.exit(res.status);

// Return PPTX as base64
const pptxPath = OUT_PATH;
const pptx = fs.readFileSync(pptxPath);
console.log("===OUTPUT_DIR===");
console.log(OUTPUT_DIR);
console.log("===OUTPUT_DIR_START===");
console.log(OUTPUT_DIR);
console.log("===OUTPUT_DIR_END===");
console.log("===BASE64_START===");
console.log(pptx.toString("base64"));
console.log("===BASE64_END===");

// Try to get slide images
let images = {};
try {
  const outDir = OUTPUT_DIR;
  const imageFiles = fs.readdirSync(outDir).filter((f) => /^slide-\d+\.png$/i.test(f));
  for (const f of imageFiles) {
    images[f] = fs.readFileSync(path.join(outDir, f)).toString("base64");
  }
} catch (e) {
  console.log("[AIO] Could not read slide images: " + e.message);
}
console.log("===IMAGES_START===");
console.log(JSON.stringify(images));
console.log("===IMAGES_END===");
`,
		jsQuote(inputDir),
		jsQuote(outFile),
		jsQuote(mode),
		jsQuote(cacheDir),
	)

	return strings.TrimSpace(code) + "\n"
}

func jsQuote(s string) string {
	encoded, _ := json.Marshal(s)
	return string(encoded)
}

func executeAIONodeJS(ctx context.Context, baseURL, code string, files map[string]string, timeout time.Duration) (string, error) {
	timeoutSec := int(mathRoundSeconds(timeout))
	if timeoutSec < 1 {
		timeoutSec = 1
	}
	if timeoutSec > 300 {
		timeoutSec = 300
	}

	stdin := ""
	payload := nodeJSExecuteRequest{
		Code:    code,
		Timeout: &timeoutSec,
		Stdin:   &stdin,
		Files:   files,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/v1/nodejs/execute"
	client := &http.Client{Timeout: 10 * time.Minute}
	maxAttempts := 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			if isRetryableNetErr(err) && attempt < maxAttempts {
				time.Sleep(time.Duration(attempt*2) * time.Second)
				continue
			}
			return "", err
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			if isRetryableNetErr(readErr) && attempt < maxAttempts {
				time.Sleep(time.Duration(attempt*2) * time.Second)
				continue
			}
			return "", readErr
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("aio nodejs error %d: %s", resp.StatusCode, truncateForLog(string(respBody), 2000))
		}

		var decoded nodeJSExecuteResponse
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			return "", fmt.Errorf("nodejs response decode failed: %v", err)
		}
		if decoded.Data == nil {
			return "", errors.New("nodejs response missing data")
		}

		stdout := ""
		stderr := ""
		exitCode := 0
		if decoded.Data.Stdout != nil {
			stdout = *decoded.Data.Stdout
		}
		if decoded.Data.Stderr != nil {
			stderr = *decoded.Data.Stderr
		}
		if decoded.Data.ExitCode != nil {
			exitCode = *decoded.Data.ExitCode
		}
		if exitCode != 0 {
			msg := stderr
			if msg == "" {
				msg = stdout
			}
			return "", fmt.Errorf("nodejs exit %d: %s", exitCode, truncateForLog(msg, 4000))
		}

		if stderr != "" {
			stdout += "\n" + stderr
		}
		return stdout, nil
	}

	return "", errors.New("nodejs request failed after retries")
}

func extractBase64(output string) (string, error) {
	start := strings.Index(output, "===BASE64_START===")
	end := strings.Index(output, "===BASE64_END===")
	if start == -1 || end == -1 || end <= start {
		return "", errors.New("base64 markers not found")
	}
	chunk := strings.TrimSpace(output[start+len("===BASE64_START===") : end])
	if chunk == "" {
		return "", errors.New("base64 content empty")
	}
	return chunk, nil
}

func extractImages(output string) (map[string]string, error) {
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

func extractOutputDirFromStdout(output string) string {
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

func aioBaseCandidates(raw string) []string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	add := func(u string) {
		u = strings.TrimRight(strings.TrimSpace(u), "/")
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	add(base)
	for _, suffix := range []string{"/zh/api", "/api"} {
		if strings.HasSuffix(base, suffix) {
			add(strings.TrimSuffix(base, suffix))
		}
	}
	return out
}

func discoverAIOBase(ctx context.Context, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	client := &http.Client{Timeout: 10 * time.Second}
	for _, base := range candidates {
		endpoint := strings.TrimRight(base, "/") + "/v1/nodejs/info"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		_, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return base
		}
	}
	return candidates[0]
}

func fetchBrowserCDPURL(ctx context.Context, baseURL string) (string, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/browser/info"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("browser/info status=%s body=%s", resp.Status, truncateForLog(string(body), 800))
	}

	var decoded struct {
		Data *struct {
			CDPURL *string `json:"cdp_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", err
	}
	if decoded.Data == nil || decoded.Data.CDPURL == nil || strings.TrimSpace(*decoded.Data.CDPURL) == "" {
		return "", errors.New("cdp_url missing in response")
	}
	return *decoded.Data.CDPURL, nil
}

func isLocalhostURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if ne, ok := err.(net.Error); ok && (ne.Timeout() || ne.Temporary()) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "unexpected EOF") || strings.Contains(msg, "connection reset") || strings.Contains(msg, "broken pipe")
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "... (truncated)"
}

func mathRoundSeconds(d time.Duration) int64 {
	seconds := d.Seconds()
	if seconds < 0 {
		return 0
	}
	return int64(seconds + 0.5)
}

func downloadSlideImages(ctx context.Context, baseURL, outputDir, localDir string) ([]string, error) {
	showHidden := true
	req := fileListRequest{
		Path:       outputDir,
		ShowHidden: &showHidden,
		FileTypes:  []string{".png"},
	}
	var resp responseWrapper[fileListResult]
	if err := postJSON(ctx, strings.TrimRight(baseURL, "/")+"/v1/file/list", req, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}
	var names []string
	for _, f := range resp.Data.Files {
		if f.IsDirectory {
			continue
		}
		name := f.Name
		if !strings.HasPrefix(strings.ToLower(name), "slide-") || !strings.HasSuffix(strings.ToLower(name), ".png") {
			continue
		}
		localPath := filepath.Join(localDir, name)
		if err := downloadFile(ctx, baseURL, f.Path, localPath); err != nil {
			return names, err
		}
		names = append(names, name)
	}
	return names, nil
}

func downloadFile(ctx context.Context, baseURL, remotePath, localPath string) error {
	u := strings.TrimRight(baseURL, "/") + "/v1/file/download?path=" + url.QueryEscape(remotePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed %d: %s", resp.StatusCode, truncateForLog(string(body), 800))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(localPath, data, 0644)
}

func postJSON(ctx context.Context, url string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AIO error %d: %s", resp.StatusCode, truncateForLog(string(respBody), 1200))
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("response decode failed: %v; body=%s", err, truncateForLog(string(respBody), 1200))
		}
	}
	return nil
}
