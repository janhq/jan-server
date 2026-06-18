// Package artifactexec provides a generic executor for ActionTypeArtifactCreate
// steps. It stores the output of a preceding step (typically a skill execution)
// as an artifact via the media service. It is shared by the document, PDF,
// spreadsheet, and deep-research agents.
package artifactexec

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners"
	"jan-server/services/response-api/internal/domain/agent/planners/skill"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/media"

	"github.com/rs/zerolog/log"
)

// GenericArtifactExecutor handles ActionTypeArtifactCreate steps by persisting
// the previous step's output as an artifact.
type GenericArtifactExecutor struct {
	mcpClient       planners.MCPClient
	artifactService artifact.Service
	mediaClient     *media.Client
	aioBaseURL      string
}

// NewGenericArtifactExecutor creates a new generic artifact executor.
func NewGenericArtifactExecutor(mcpClient planners.MCPClient, artifactService artifact.Service, mediaClient *media.Client, cfg *config.Config) *GenericArtifactExecutor {
	aioBaseURL := ""
	if cfg != nil {
		aioBaseURL = strings.TrimSpace(cfg.AIOURL)
	}
	return &GenericArtifactExecutor{
		mcpClient:       mcpClient,
		artifactService: artifactService,
		mediaClient:     mediaClient,
		aioBaseURL:      aioBaseURL,
	}
}

// CanExecute checks if this executor can handle the given action type.
func (e *GenericArtifactExecutor) CanExecute(action plan.ActionType) bool {
	return action == plan.ActionTypeArtifactCreate
}

// Rollback attempts to undo a step's effects.
func (e *GenericArtifactExecutor) Rollback(ctx context.Context, step *plan.Step) error {
	return nil
}

// Execute runs a single step and returns the result.
func (e *GenericArtifactExecutor) Execute(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Str("action", string(step.Action)).Int("sequence", step.Sequence).Msg("[artifact] Execute started")
	switch step.Action {
	case plan.ActionTypeArtifactCreate:
		return e.executeArtifactCreation(ctx, step, input)
	default:
		return &agent.ExecutionResult{Status: status.StatusCompleted}, nil
	}
}

func (e *GenericArtifactExecutor) executeArtifactCreation(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(input.StepParams, &params); err != nil {
		log.Error().Err(err).Msg("[artifact] failed to parse artifact parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  "failed to parse artifact parameters",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	action, _ := params["action"].(string)
	cfg, _ := params["config"].(map[string]interface{})
	retentionPolicy := strings.TrimSpace(stringValue(cfg, "retention_policy"))
	if retentionPolicy == "" {
		retentionPolicy = "session"
	}

	switch action {
	case "store_artifact":
		return e.storeGenericArtifact(ctx, input, params, retentionPolicy)
	default:
		return &agent.ExecutionResult{Status: status.StatusCompleted}, nil
	}
}

func (e *GenericArtifactExecutor) storeGenericArtifact(ctx context.Context, input agent.ExecutionInput, params map[string]interface{}, retentionPolicy string) (*agent.ExecutionResult, error) {
	cfg, _ := params["config"].(map[string]interface{})
	format, _ := cfg["format"].(string)
	artifactType, _ := params["artifact_type"].(string)

	if input.PreviousOutput != nil {
		var skillOutput skill.ExecuteOutput
		if err := json.Unmarshal(input.PreviousOutput, &skillOutput); err == nil && skillOutput.Success {
			return e.uploadSkillArtifact(ctx, input, skillOutput, artifactType, retentionPolicy)
		}
	}

	content := ""
	if input.PreviousOutput != nil {
		var prevOutput map[string]interface{}
		if err := json.Unmarshal(input.PreviousOutput, &prevOutput); err == nil {
			if c, ok := prevOutput["content"].(string); ok {
				content = c
			}
		}
		if content == "" {
			content = string(input.PreviousOutput)
		}
	}
	if content == "" {
		content = "Artifact content unavailable."
	}

	responseID := ""
	conversationID := ""
	userID := ""
	if input.PlanContext != nil {
		responseID = input.PlanContext.ResponseID
		conversationID = input.PlanContext.ConversationID
		userID = input.PlanContext.UserID
	}

	contentType := resolveArtifactContentType(artifactType, format)
	title := resolveArtifactTitle(artifactType)
	filename := resolveArtifactFilename(artifactType, format)

	var storagePath *string
	downloadURL := ""
	if e.mediaClient != nil {
		mediaArtifact, err := e.mediaClient.UploadArtifact(ctx, &media.UploadRequest{
			Content:        []byte(content),
			Filename:       filename,
			ContentType:    contentType.MimeTypeFor(),
			ConversationID: conversationID,
			ResponseID:     responseID,
			UserID:         userID,
		})
		if err != nil {
			log.Warn().Err(err).Str("response_id", responseID).Msg("[artifact] failed to upload artifact to media-api, falling back to inline storage")
		} else {
			storagePath = &mediaArtifact.DownloadURL
			downloadURL = mediaArtifact.DownloadURL
		}
	}

	var contentPtr *string
	if storagePath == nil {
		contentPtr = &content
	}

	createdArtifact, err := e.artifactService.Create(ctx, artifact.CreateParams{
		ResponseID:      responseID,
		ContentType:     contentType,
		Title:           title,
		Content:         contentPtr,
		StoragePath:     storagePath,
		SizeBytes:       int64(len(content)),
		RetentionPolicy: artifact.RetentionPolicy(retentionPolicy),
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "ARTIFACT_ERROR",
				Message:  fmt.Sprintf("failed to create artifact: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	if downloadURL == "" {
		downloadURL = fmt.Sprintf("/responses/v1/artifacts/%s/download", createdArtifact.ID)
	}

	stepOutput := &plan.StepOutput{
		Status:    "completed",
		Type:      "artifact_create",
		CreatedAt: time.Now(),
		Artifact: &plan.MediaArtifact{
			ID:          createdArtifact.ID,
			Type:        string(contentType),
			Filename:    filename,
			DownloadURL: downloadURL,
			Size:        int64(len(content)),
			ContentType: contentType.MimeTypeFor(),
		},
	}
	outputBytes, _ := json.Marshal(stepOutput)

	return &agent.ExecutionResult{
		Status:     status.StatusCompleted,
		Output:     outputBytes,
		ArtifactID: &createdArtifact.ID,
	}, nil
}

func (e *GenericArtifactExecutor) uploadSkillArtifact(ctx context.Context, input agent.ExecutionInput, skillOutput skill.ExecuteOutput, artifactType string, retentionPolicy string) (*agent.ExecutionResult, error) {
	log.Debug().
		Str("artifact_type", artifactType).
		Str("filename", skillOutput.FileName).
		Str("mime_type", skillOutput.MimeType).
		Msg("[artifact] uploadSkillArtifact started")
	if e.mediaClient == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MEDIA_CLIENT_MISSING",
				Message:  "media client not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	var decoded []byte
	var err error
	if strings.TrimSpace(skillOutput.FileContentBase64) != "" {
		decoded, err = base64.StdEncoding.DecodeString(skillOutput.FileContentBase64)
		if err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "FILE_DECODE_ERROR",
					Message:  "failed to decode skill file content",
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
	} else if strings.TrimSpace(skillOutput.OutputPath) != "" {
		decoded, err = e.readBinaryFileFromSandbox(ctx, skillOutput.OutputPath, input)
		if err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "FILE_READ_ERROR",
					Message:  err.Error(),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
	} else {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "FILE_MISSING",
				Message:  "no skill output file available for upload",
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	responseID := ""
	conversationID := ""
	userID := ""
	if input.PlanContext != nil {
		responseID = input.PlanContext.ResponseID
		conversationID = input.PlanContext.ConversationID
		userID = input.PlanContext.UserID
	}

	fileName := strings.TrimSpace(skillOutput.FileName)
	if fileName == "" {
		fileName = resolveArtifactFilename(artifactType, "")
	}
	mimeType := strings.TrimSpace(skillOutput.MimeType)
	if mimeType == "" {
		mimeType = resolveArtifactContentType(artifactType, "").MimeTypeFor()
	}

	mediaArtifact, err := e.mediaClient.UploadArtifact(ctx, &media.UploadRequest{
		Content:        decoded,
		Filename:       fileName,
		ContentType:    mimeType,
		ConversationID: conversationID,
		ResponseID:     responseID,
		UserID:         userID,
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "UPLOAD_ERROR",
				Message:  fmt.Sprintf("failed to upload artifact: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	contentType := resolveArtifactContentType(artifactType, "")
	createdArtifact, err := e.artifactService.Create(ctx, artifact.CreateParams{
		ResponseID:      responseID,
		ContentType:     contentType,
		MimeType:        &mimeType,
		Title:           resolveArtifactTitle(artifactType),
		StoragePath:     &mediaArtifact.DownloadURL,
		SizeBytes:       int64(len(decoded)),
		RetentionPolicy: artifact.RetentionPolicy(retentionPolicy),
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "ARTIFACT_ERROR",
				Message:  fmt.Sprintf("failed to create artifact: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	downloadURL := mediaArtifact.DownloadURL
	if downloadURL == "" {
		downloadURL = fmt.Sprintf("/responses/v1/artifacts/%s/download", createdArtifact.ID)
	}

	stepOutput := &plan.StepOutput{
		Status:    "completed",
		Type:      "artifact_create",
		CreatedAt: time.Now(),
		Artifact: &plan.MediaArtifact{
			ID:          createdArtifact.ID,
			Type:        string(contentType),
			Filename:    fileName,
			DownloadURL: downloadURL,
			Size:        int64(len(decoded)),
			ContentType: mimeType,
		},
	}
	outputBytes, _ := json.Marshal(stepOutput)

	return &agent.ExecutionResult{
		Status:     status.StatusCompleted,
		Output:     outputBytes,
		ArtifactID: &createdArtifact.ID,
	}, nil
}

func (e *GenericArtifactExecutor) readBinaryFileFromSandbox(ctx context.Context, path string, input agent.ExecutionInput) ([]byte, error) {
	if strings.TrimSpace(e.aioBaseURL) != "" {
		if payload, err := downloadAIOFile(ctx, e.aioBaseURL, path); err == nil {
			return payload, nil
		}
	}

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
		Name: "sandbox_code_execute",
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

func downloadAIOFile(ctx context.Context, baseURL string, path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("empty path")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	escaped := url.QueryEscape(path)
	reqURL := baseURL + "/v1/file/download?path=" + escaped
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read download response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func resolveArtifactContentType(artifactType string, format string) artifact.ContentType {
	switch artifactType {
	case "report":
		return artifact.ContentTypeResearch
	case "document":
		return artifact.ContentTypeDocument
	case "spreadsheet":
		return artifact.ContentTypeTable
	case "markdown":
		return artifact.ContentTypeMarkdown
	default:
		if format == "markdown" {
			return artifact.ContentTypeMarkdown
		}
		return artifact.ContentTypeDocument
	}
}

func resolveArtifactTitle(artifactType string) string {
	switch artifactType {
	case "report":
		return "Research Report"
	case "document":
		return "Document"
	case "spreadsheet":
		return "Spreadsheet"
	default:
		return "Document"
	}
}

func resolveArtifactFilename(artifactType string, format string) string {
	switch artifactType {
	case "report":
		return "research_report.md"
	case "document":
		if format == "pdf" {
			return "document.pdf"
		}
		return "document.md"
	case "spreadsheet":
		return "spreadsheet.xlsx"
	case "markdown":
		return "content.md"
	default:
		if format == "pdf" {
			return "document.pdf"
		}
		return "document.md"
	}
}

func firstTextContent(content []tool.MCPContent) string {
	for _, item := range content {
		if strings.TrimSpace(item.Text) != "" {
			return item.Text
		}
	}
	return ""
}

func stringValue(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}
