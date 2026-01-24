package skill

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/plan"
	domainskill "jan-server/services/response-api/internal/domain/skill"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/utils/idgen"

	"github.com/rs/zerolog/log"
)

// extractJSONFromMarkdown extracts JSON content from markdown code blocks.
// It handles ```json ... ``` or ``` ... ``` wrapped content.
func extractJSONFromMarkdown(content string) string {
	content = strings.TrimSpace(content)

	// Try to extract from ```json ... ``` or ``` ... ``` blocks
	// Pattern matches ```json or ``` at the start, content, then ``` at the end
	patterns := []string{
		"```json\\s*\\n([\\s\\S]*?)\\n```",
		"```\\s*\\n([\\s\\S]*?)\\n```",
		"```json([\\s\\S]*?)```",
		"```([\\s\\S]*?)```",
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(content); len(matches) > 1 {
			return strings.TrimSpace(stripJSONComments(matches[1]))
		}
	}

	// No code block found, return as-is
	return strings.TrimSpace(stripJSONComments(content))
}

// stripJSONComments removes // and /* */ comments while preserving string literals.
func stripJSONComments(input string) string {
	if input == "" {
		return input
	}
	var b strings.Builder
	b.Grow(len(input))
	inString := false
	escaped := false
	inLine := false
	inBlock := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		var next byte
		if i+1 < len(input) {
			next = input[i+1]
		}

		if inLine {
			if ch == '\n' {
				inLine = false
				b.WriteByte(ch)
			}
			continue
		}
		if inBlock {
			if ch == '*' && next == '/' {
				inBlock = false
				i++
				continue
			}
			if ch == '\n' {
				b.WriteByte(ch)
			}
			continue
		}

		if inString {
			b.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			b.WriteByte(ch)
			continue
		}
		if ch == '/' && next == '/' {
			inLine = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			inBlock = true
			i++
			continue
		}

		b.WriteByte(ch)
	}

	return b.String()
}

// Executor handles ActionTypeSkillExecute steps.
type Executor struct {
	mcpClient          MCPClient
	llmProvider        LLMProvider
	skillService       domainskill.Service
	enabled            bool
	maxInstallRetries  int
	maxCodeFixRetries  int
	maxFileSize        int64
	skillEnabledByType map[domainskill.SkillType]bool
	defaultTimeout     time.Duration
}

// NewExecutor creates a new skill executor.
func NewExecutor(
	mcpClient MCPClient,
	llmProvider LLMProvider,
	skillService domainskill.Service,
	enabled bool,
	maxInstallRetries int,
	maxCodeFixRetries int,
	maxFileSize int64,
	defaultTimeout time.Duration,
	skillEnabledByType map[domainskill.SkillType]bool,
) *Executor {
	if skillEnabledByType == nil {
		skillEnabledByType = map[domainskill.SkillType]bool{}
	}
	return &Executor{
		mcpClient:          mcpClient,
		llmProvider:        llmProvider,
		skillService:       skillService,
		enabled:            enabled,
		maxInstallRetries:  maxInstallRetries,
		maxCodeFixRetries:  maxCodeFixRetries,
		maxFileSize:        maxFileSize,
		defaultTimeout:     defaultTimeout,
		skillEnabledByType: skillEnabledByType,
	}
}

// Execute runs a skill execution step.
func (e *Executor) Execute(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().
		Str("step_id", step.ID).
		Str("action", string(step.Action)).
		Bool("enabled", e.enabled).
		Msg("[skill_executor] Execute started")
	if !e.enabled {
		log.Warn().Msg("[skill_executor] skill executor is disabled")
		return e.failedResult("SKILL_DISABLED", "skill execution is disabled", status.ErrorSeverityFatal), nil
	}

	var params executeParams
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		log.Error().Err(err).Msg("[skill_executor] failed to parse skill parameters")
		return e.failedResult("PARSE_ERROR", "failed to parse skill parameters", status.ErrorSeverityFatal), nil
	}

	log.Debug().
		Str("skill_type", string(params.SkillType)).
		Interface("options", params.Options).
		Msg("[skill_executor] parsed skill parameters")

	if params.SkillType == "" {
		log.Error().Msg("[skill_executor] missing skill type")
		return e.failedResult("MISSING_SKILL_TYPE", "skill_type is required", status.ErrorSeverityFatal), nil
	}

	if enabled, ok := e.skillEnabledByType[params.SkillType]; ok && !enabled {
		log.Warn().Str("skill_type", string(params.SkillType)).Msg("[skill_executor] skill type is disabled")
		return e.failedResult("SKILL_TYPE_DISABLED", fmt.Sprintf("skill %s is disabled", params.SkillType), status.ErrorSeverityFatal), nil
	}

	// Try to get content from previous output first
	content := e.extractContentFromPreviousOutput(input.PreviousOutput)

	// If no content from immediate previous step, search accumulated outputs
	// This handles cases where skill_execute is in a different task than the LLM step
	if content == nil && len(input.AccumulatedOutputs) > 0 {
		content = e.findContentInAccumulatedOutputs(input.AccumulatedOutputs, params.SkillType)
	}

	if content == nil {
		return e.failedResult("NO_CONTENT", "no content from previous step", status.ErrorSeverityRetryable), nil
	}

	outputPath := e.resolveOutputPath(params)
	code, err := e.skillService.GenerateCode(ctx, domainskill.GenerateCodeRequest{
		SkillType:  params.SkillType,
		Content:    content,
		Options:    params.Options,
		OutputPath: outputPath,
	})
	if err != nil {
		return e.failedResult("CODE_GEN_ERROR", err.Error(), status.ErrorSeverityRetryable), nil
	}

	code = agent.NormalizeSandboxFilePaths(code)

	return e.executeWithRetry(ctx, step, input, code, params, outputPath, 0, nil, 0)
}

// CanExecute checks if this executor can handle the given action type.
func (e *Executor) CanExecute(action plan.ActionType) bool {
	return action == plan.ActionTypeSkillExecute
}

// Rollback attempts to undo a step's effects (optional).
func (e *Executor) Rollback(ctx context.Context, step *plan.Step) error {
	// Skill execution is not reversible in the sandbox context.
	return nil
}

type executeParams struct {
	SkillType  domainskill.SkillType  `json:"skill_type"`
	OutputPath string                 `json:"output_path,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

func (e *Executor) executeWithRetry(
	ctx context.Context,
	step *plan.Step,
	input agent.ExecutionInput,
	code string,
	params executeParams,
	outputPath string,
	installRetryCount int,
	installedPackages []string,
	codeFixRetryCount int,
) (*agent.ExecutionResult, error) {
	callReq := tool.CallRequest{
		Name: "aio_code_execute",
		Arguments: map[string]interface{}{
			"language": "python",
			"code":     code,
		},
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return e.failedResult("EXECUTION_ERROR", err.Error(), status.ErrorSeverityRetryable), nil
	}

	hasError := result.IsError || e.hasErrorInResult(result)
	if hasError {
		errorText := e.extractErrorText(result)
		codeErr := agent.ParseCodeExecutionError(errorText)

		if agent.IsRetryableWithInstall(codeErr) && installRetryCount < e.maxInstallRetries {
			packageName := agent.ResolvePackageName(codeErr.ModuleName)
			if !contains(installedPackages, packageName) {
				if _, installErr := e.installPackage(ctx, packageName, input); installErr == nil {
					newPackages := append(installedPackages, packageName)
					return e.executeWithRetry(ctx, step, input, code, params, outputPath, installRetryCount+1, newPackages, codeFixRetryCount)
				}
			}
		}

		if e.llmProvider != nil && codeFixRetryCount < e.maxCodeFixRetries {
			fixedCode, fixErr := e.llmProvider.FixCode(ctx, code, errorText, "python")
			if fixErr == nil && fixedCode != "" && fixedCode != code {
				return e.executeWithRetry(ctx, step, input, fixedCode, params, outputPath, installRetryCount, installedPackages, codeFixRetryCount+1)
			}
		}

		severity := status.ErrorSeverityRetryable
		if installRetryCount >= e.maxInstallRetries && codeFixRetryCount >= e.maxCodeFixRetries {
			severity = status.ErrorSeverityFatal
		}
		return e.failedResult("SKILL_EXECUTION_FAILED", errorText, severity), nil
	}

	fileContent, err := e.readFileFromSandbox(ctx, outputPath, input)
	if err != nil {
		return e.failedResult("FILE_READ_ERROR", err.Error(), status.ErrorSeverityRetryable), nil
	}

	if e.maxFileSize > 0 && int64(len(fileContent)) > e.maxFileSize {
		return e.failedResult("FILE_TOO_LARGE", "generated file exceeds max size", status.ErrorSeverityFatal), nil
	}

	fileName := e.getFileName(params)
	mimeType := e.getMimeType(params.SkillType)
	output := ExecuteOutput{
		Success:    true,
		SkillType:  string(params.SkillType),
		OutputPath: outputPath,
		FileName:   fileName,
		MimeType:   mimeType,
		FileSize:   int64(len(fileContent)),
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *Executor) installPackage(ctx context.Context, packageName string, input agent.ExecutionInput) (*tool.Result, error) {
	callReq := tool.CallRequest{
		Name: "aio_install_packages",
		Arguments: map[string]interface{}{
			"packages": []string{packageName},
		},
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}
	return e.mcpClient.CallTool(ctx, callReq)
}

// readFileFromSandbox reads a file from the sandbox, supporting both text and binary files.
// For binary files (like PPTX, DOCX), it uses base64 encoding via shell command.
func (e *Executor) readFileFromSandbox(ctx context.Context, path string, input agent.ExecutionInput) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(path))

	// For binary files, use shell command with base64 encoding
	binaryExts := map[string]bool{
		".pptx": true, ".docx": true, ".xlsx": true, ".pdf": true,
		".zip": true, ".png": true, ".jpg": true, ".jpeg": true,
		".gif": true, ".bmp": true, ".bin": true, ".exe": true,
	}

	if binaryExts[ext] {
		return e.readBinaryFileFromSandbox(ctx, path, input)
	}

	// For text files, use the standard file read
	callReq := tool.CallRequest{
		Name: "aio_file_read",
		Arguments: map[string]interface{}{
			"path": path,
		},
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("sandbox file read returned nil result")
	}
	if result.IsError {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = firstTextContent(result.Content)
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("file not found or unreadable: %s", path)
		}
		return nil, fmt.Errorf("sandbox file read error: %s", errMsg)
	}

	rawText := firstTextContent(result.Content)
	if rawText == "" {
		return nil, fmt.Errorf("sandbox file read returned empty content")
	}

	return decodeSandboxContent(rawText, ext)
}

// readBinaryFileFromSandbox reads a binary file using Python code execution with base64 encoding.
// This is more reliable than shell base64 command for large binary files.
func (e *Executor) readBinaryFileFromSandbox(ctx context.Context, path string, input agent.ExecutionInput) ([]byte, error) {
	// Use Python to read and base64-encode the file, which is more reliable than shell base64 for large binary files
	code := fmt.Sprintf(`import base64
import json
import os

path = %q
if not os.path.exists(path):
    print(json.dumps({"error": "file not found", "path": path}))
else:
    with open(path, "rb") as f:
        data = f.read()
    encoded = base64.b64encode(data).decode("ascii")
    print(json.dumps({"base64": encoded, "size": len(data)}))
`, path)

	callReq := tool.CallRequest{
		Name: "aio_code_execute",
		Arguments: map[string]interface{}{
			"language": "python",
			"code":     code,
		},
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return nil, fmt.Errorf("code execute failed: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("code execute returned nil result")
	}
	if result.IsError {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = firstTextContent(result.Content)
		}
		return nil, fmt.Errorf("code execute error: %s", errMsg)
	}

	rawText := firstTextContent(result.Content)
	if rawText == "" {
		return nil, fmt.Errorf("code execute returned empty content")
	}

	// Parse the code execution result
	var execResult struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal([]byte(rawText), &execResult); err != nil {
		return nil, fmt.Errorf("failed to parse code execute result: %w", err)
	}

	if execResult.Status != "" && execResult.Status != "ok" {
		return nil, fmt.Errorf("code execute status: %s, stderr: %s", execResult.Status, execResult.Stderr)
	}

	// Parse the Python script output
	stdout := strings.TrimSpace(execResult.Stdout)
	if stdout == "" {
		return nil, fmt.Errorf("code execute returned empty stdout")
	}

	var fileResult struct {
		Base64 string `json:"base64"`
		Size   int    `json:"size"`
		Error  string `json:"error"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &fileResult); err != nil {
		return nil, fmt.Errorf("failed to parse file read result: %w, stdout: %s", err, stdout)
	}

	if fileResult.Error != "" {
		return nil, fmt.Errorf("file read error: %s (path: %s)", fileResult.Error, fileResult.Path)
	}

	if fileResult.Base64 == "" {
		return nil, fmt.Errorf("file read returned empty base64")
	}

	decoded, err := base64.StdEncoding.DecodeString(fileResult.Base64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	return decoded, nil
}

func (e *Executor) extractContentFromPreviousOutput(output json.RawMessage) interface{} {
	if len(output) == 0 {
		return nil
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err == nil {
		if content, ok := parsed["content"]; ok {
			return content
		}
		if data, ok := parsed["data"]; ok {
			return data
		}
		if result, ok := parsed["result"]; ok {
			return result
		}
		return parsed
	}

	return string(output)
}

// findContentInAccumulatedOutputs searches through accumulated outputs from previous tasks
// to find content suitable for skill execution (e.g., JSON slides data from LLM generation)
func (e *Executor) findContentInAccumulatedOutputs(outputs []json.RawMessage, skillType domainskill.SkillType) interface{} {
	// Search backwards through accumulated outputs to find the most recent valid content
	for i := len(outputs) - 1; i >= 0; i-- {
		output := outputs[i]
		if len(output) == 0 {
			continue
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(output, &parsed); err != nil {
			continue
		}

		// Look for content field that contains skill-relevant data
		if content, ok := parsed["content"]; ok {
			if extracted := e.extractAndValidateContent(content, skillType); extracted != nil {
				return extracted
			}
		}

		// Check if the output itself is valid skill content
		if extracted := e.extractAndValidateContent(parsed, skillType); extracted != nil {
			return extracted
		}

		// Also check data field
		if data, ok := parsed["data"]; ok {
			if extracted := e.extractAndValidateContent(data, skillType); extracted != nil {
				return extracted
			}
		}

		// Check result field
		if result, ok := parsed["result"]; ok {
			if extracted := e.extractAndValidateContent(result, skillType); extracted != nil {
				return extracted
			}
		}
	}

	return nil
}

// extractAndValidateContent extracts content from a value and validates it for the skill type.
// If the content is a string with markdown code blocks, it extracts and parses the JSON.
// Returns the parsed content if valid, nil otherwise.
func (e *Executor) extractAndValidateContent(content interface{}, skillType domainskill.SkillType) interface{} {
	// Handle string content - extract JSON from markdown if needed
	if str, ok := content.(string); ok {
		cleanedStr := extractJSONFromMarkdown(str)

		var parsed interface{}
		if err := json.Unmarshal([]byte(cleanedStr), &parsed); err != nil {
			return nil
		}

		// Check if parsed content is valid
		if e.isValidSkillContentValue(parsed, skillType) {
			return parsed
		}
		return nil
	}

	// For non-string content, validate directly
	if e.isValidSkillContentValue(content, skillType) {
		return content
	}
	return nil
}

// isValidSkillContentValue checks if the already-parsed content is valid for the given skill type.
func (e *Executor) isValidSkillContentValue(content interface{}, skillType domainskill.SkillType) bool {
	// Handle array content (slides are typically an array)
	if arr, ok := content.([]interface{}); ok {
		if len(arr) > 0 {
			// Check if first element looks like a slide
			if slide, ok := arr[0].(map[string]interface{}); ok {
				if _, hasTitle := slide["slide_title"]; hasTitle {
					return true
				}
				if _, hasContent := slide["content"]; hasContent {
					return true
				}
			}
		}
		// For slides, an array is valid
		if skillType == domainskill.SkillTypeSlides && len(arr) > 0 {
			return true
		}
	}

	// Handle map content
	contentMap, ok := content.(map[string]interface{})
	if !ok {
		return false
	}

	switch skillType {
	case domainskill.SkillTypeSlides:
		// For slides, look for "slides" array or presentation structure
		if _, hasSlides := contentMap["slides"]; hasSlides {
			return true
		}
		if _, hasSlideTitle := contentMap["slide_title"]; hasSlideTitle {
			return true
		}
	case domainskill.SkillTypeDocs:
		if _, hasSections := contentMap["sections"]; hasSections {
			return true
		}
		if _, hasContent := contentMap["content"]; hasContent {
			return true
		}
	case domainskill.SkillTypeSpreadsheets:
		if _, hasSheets := contentMap["sheets"]; hasSheets {
			return true
		}
		if _, hasData := contentMap["data"]; hasData {
			return true
		}
	case domainskill.SkillTypePDFs:
		if _, hasPages := contentMap["pages"]; hasPages {
			return true
		}
		if _, hasContent := contentMap["content"]; hasContent {
			return true
		}
	}

	return len(contentMap) > 0
}

func (e *Executor) resolveOutputPath(params executeParams) string {
	if strings.TrimSpace(params.OutputPath) != "" {
		return params.OutputPath
	}

	ext := e.getFileExtension(params.SkillType)
	id := idgen.MustGenerateSecureID("skill", 12)
	return "/tmp/" + id + ext
}

func (e *Executor) getFileExtension(skillType domainskill.SkillType) string {
	switch skillType {
	case domainskill.SkillTypeSlides:
		return ".pptx"
	case domainskill.SkillTypeDocs:
		return ".docx"
	case domainskill.SkillTypePDFs:
		return ".pdf"
	case domainskill.SkillTypeSpreadsheets:
		return ".xlsx"
	default:
		return ".bin"
	}
}

func (e *Executor) getFileName(params executeParams) string {
	ext := e.getFileExtension(params.SkillType)
	id := idgen.MustGenerateSecureID(string(params.SkillType), 10)
	return id + ext
}

func (e *Executor) getMimeType(skillType domainskill.SkillType) string {
	switch skillType {
	case domainskill.SkillTypeSlides:
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case domainskill.SkillTypeDocs:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case domainskill.SkillTypePDFs:
		return "application/pdf"
	case domainskill.SkillTypeSpreadsheets:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		return "application/octet-stream"
	}
}

func (e *Executor) failedResult(code string, message string, severity status.ErrorSeverity) *agent.ExecutionResult {
	return &agent.ExecutionResult{
		Status: status.StatusFailed,
		Error: &agent.ExecutionError{
			Code:     code,
			Message:  message,
			Severity: severity,
		},
	}
}

func (e *Executor) hasErrorInResult(result *tool.Result) bool {
	if result == nil || len(result.Content) == 0 {
		return false
	}

	for _, content := range result.Content {
		if content.Type != "text" || content.Text == "" {
			continue
		}
		if strings.Contains(content.Text, "Traceback (most recent call last)") {
			return true
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(content.Text), &parsed); err == nil {
			if errDetails, ok := parsed["error_details"].(map[string]interface{}); ok {
				if errName, ok := errDetails["error_name"].(string); ok && errName != "" {
					return true
				}
			}
			if isError, ok := parsed["is_error"].(bool); ok && isError {
				return true
			}
			if success, ok := parsed["success"].(bool); ok && !success {
				return true
			}
		}
	}
	return false
}

func (e *Executor) extractErrorText(result *tool.Result) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	texts := make([]string, 0, len(result.Content))
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			texts = append(texts, content.Text)
		}
	}
	return strings.Join(texts, "\n")
}

func firstTextContent(contents []tool.MCPContent) string {
	for _, content := range contents {
		if content.Type == "text" && content.Text != "" {
			return content.Text
		}
	}
	return ""
}

func decodeSandboxContent(rawText string, ext string) ([]byte, error) {
	trimmed := strings.TrimSpace(rawText)
	if trimmed == "" {
		return nil, fmt.Errorf("empty sandbox content")
	}

	if decoded, ok := tryDecodeBase64(trimmed); ok {
		return decoded, nil
	}

	return []byte(rawText), nil
}

func tryDecodeBase64(value string) ([]byte, bool) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, true
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return decoded, true
	}
	return nil, false
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

var _ agent.Executor = (*Executor)(nil)
