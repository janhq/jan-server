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
	useCDP := true
	if v, ok := parseBoolFromInterface(params["use_cdp"]); ok {
		useCDP = v
	}

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

	cdpURL := ""
	if useCDP {
		if u, err := fetchBrowserCDPURL(ctx, baseURL); err == nil {
			cdpURL = strings.TrimSpace(u)
		}
		if cdpURL != "" && isLocalhostURL(cdpURL) {
			useCDP = false
			cdpURL = ""
		}
	}

	outputFile := "presentation.pptx"
	cacheDir := "/home/gem/.cache/pptx_export"
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
	code := fmt.Sprintf(`
const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

const workdir = process.cwd();
const INPUT_DIR = %s;
const OUT_FILE = %s;
const MODE = %s;

const USE_CDP = %v;
const CDP_URL = (process.env.AIO_CDP_URL && process.env.AIO_CDP_URL.trim()) ? process.env.AIO_CDP_URL.trim() : %s;
const CACHE_DIR = (process.env.AIO_CACHE_DIR && process.env.AIO_CACHE_DIR.trim()) ? process.env.AIO_CACHE_DIR.trim() : %s;

const OUTPUT_DIR = path.join(CACHE_DIR, "outputs");
const OUT_PATH = path.join(OUTPUT_DIR, path.basename(OUT_FILE));

function run(cmd, args, opts) {
  const res = spawnSync(cmd, args, Object.assign({ encoding: "utf8" }, opts || {}));
  if (res.stdout) process.stdout.write(res.stdout);
  if (res.stderr) process.stderr.write(res.stderr);
  const code = (typeof res.status === "number") ? res.status : 1;
  return { code };
}

console.log("[AIO] workdir=" + workdir);
console.log("[AIO] cache_dir=" + CACHE_DIR);
if (USE_CDP) console.log("[AIO] cdp_url=" + (CDP_URL || "(empty)"));

const files = JSON.parse(fs.readFileSync(path.join(workdir, "inputs.json"), "utf8"));
const exportJs = fs.readFileSync(path.join(workdir, "export_pptx_full.js"), "utf8");

for (const f of files) {
  const fullPath = path.join(workdir, f.path);
  fs.mkdirSync(path.dirname(fullPath), { recursive: true });
  fs.writeFileSync(fullPath, Buffer.from(f.b64, "base64"));
}
fs.writeFileSync(path.join(workdir, "export_pptx_full.js"), exportJs);
console.log("[AIO] wrote " + files.length + " input files + export_pptx_full.js");

const nodeEnvDir = path.join(CACHE_DIR, "node_env");
const browsersDir = path.join(CACHE_DIR, "pw_browsers");
fs.mkdirSync(nodeEnvDir, { recursive: true });
fs.mkdirSync(browsersDir, { recursive: true });
fs.mkdirSync(OUTPUT_DIR, { recursive: true });

const pkgJson = path.join(nodeEnvDir, "package.json");
if (!fs.existsSync(pkgJson)) {
  fs.writeFileSync(pkgJson, JSON.stringify({ name: "pptx_export_env", private: true }, null, 2));
}

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

const shimPath = path.join(workdir, "playwright_shim.cjs");
const shimContent = [
  "const pw = require(\"playwright\");",
  "const cdp = (process.env.AIO_CDP_URL || \"\").trim();",
  "if (!cdp) {",
  "  console.log(\"[AIO] Shim: AIO_CDP_URL empty; will use local launch()\");",
  "  return;",
  "}",
  "console.log(\"[AIO] Shim: overriding chromium.launch() to use connectOverCDP()\");",
  "const chromium = pw.chromium;",
  "",
  "const origLaunch = chromium.launch.bind(chromium);",
  "chromium.launch = async function(opts) {",
  "  try {",
  "    return await chromium.connectOverCDP(cdp);",
  "  } catch (e) {",
  "    console.warn(\"[AIO] Shim: connectOverCDP failed; falling back to launch(): \" + (e && e.message ? e.message : e));",
  "    return await origLaunch(opts || {});",
  "  }",
  "};",
  "",
  "const origPersistent = chromium.launchPersistentContext.bind(chromium);",
  "chromium.launchPersistentContext = async function(userDataDir, opts) {",
  "  try {",
  "    const browser = await chromium.connectOverCDP(cdp);",
  "    const contexts = browser.contexts();",
  "    if (contexts && contexts.length) return contexts[0];",
  "    return await browser.newContext(opts || {});",
  "  } catch (e) {",
  "    console.warn(\"[AIO] Shim: connectOverCDP(persistent) failed; falling back: \" + (e && e.message ? e.message : e));",
  "    return await origPersistent(userDataDir, opts || {});",
  "  }",
  "};"
].join("\n");
fs.writeFileSync(shimPath, shimContent, "utf8");

const pwBin = path.join(nodeEnvDir, "node_modules", ".bin", "playwright");
if (!USE_CDP || !CDP_URL) {
  console.log("[AIO] No CDP (or disabled). Ensuring local Playwright browsers in " + browsersDir);
  const envPwInstall = Object.assign({}, process.env, {
    PLAYWRIGHT_BROWSERS_PATH: browsersDir,
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: "0",
  });
  let r = run(pwBin, ["install", "chromium"], { cwd: nodeEnvDir, env: envPwInstall });
  if (r.code !== 0) {
    console.error("[AIO] ERROR: playwright install chromium failed");
    process.exit(2);
  }
} else {
  console.log("[AIO] Using CDP; skipping browser download");
}

const envRun = Object.assign({}, process.env, {
  NODE_PATH: path.join(nodeEnvDir, "node_modules"),
  AIO_CACHE_DIR: CACHE_DIR,
  AIO_CDP_URL: CDP_URL,
  PLAYWRIGHT_BROWSERS_PATH: browsersDir,
  PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: "0",
});

const args = ["-r", shimPath, path.join(workdir, "export_pptx_full.js"), "--in", INPUT_DIR, "--out", OUT_PATH, "--mode", MODE];
console.log("[AIO] Running: node " + args.join(" "));
const res = spawnSync("node", args, { cwd: workdir, env: envRun, stdio: "inherit" });
if (typeof res.status === "number" && res.status !== 0) process.exit(res.status);

const pptxPath = OUT_PATH;
const pptx = fs.readFileSync(pptxPath);
console.log("===OUTPUT_DIR_START===");
console.log(OUTPUT_DIR);
console.log("===OUTPUT_DIR_END===");
console.log("===BASE64_START===");
console.log(pptx.toString("base64"));
console.log("===BASE64_END===");
const imageFiles = fs.readdirSync(OUTPUT_DIR).filter((f) => /^slide-\\d+\\.png$/i.test(f));
const images = {};
for (const f of imageFiles) {
  images[f] = fs.readFileSync(path.join(OUTPUT_DIR, f)).toString("base64");
}
console.log("===IMAGES_START===");
console.log(JSON.stringify(images));
console.log("===IMAGES_END===");
`,
		jsQuote(inputDir),
		jsQuote(outFile),
		jsQuote(mode),
		useCDP,
		jsQuote(cdpURL),
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
