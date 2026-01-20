package slide_generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	sliderenderer "jan-server/services/response-api/assets/slide_renderer"
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/media"

	"github.com/rs/zerolog/log"
)

func (e *SlideGeneratorExecutor) executeUploadSlideSpec(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executeUploadSlideSpec started")
	deckJSON, err := e.assembleDeck(input)
	if err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to assemble deck")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "ASSEMBLY_ERROR",
				Message:  fmt.Sprintf("Failed to assemble deck: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	targetPath, _ := params["target_path"].(string)
	if targetPath == "" {
		responseID := "slide_spec"
		if input.PlanContext != nil && input.PlanContext.ResponseID != "" {
			responseID = input.PlanContext.ResponseID
		}
		targetPath = fmt.Sprintf("/home/gem/slide_specs/slide_spec_%s.json", responseID)
	}

	code := fmt.Sprintf(`import json

spec = json.loads(%s)

with open(%q, "w", encoding="utf-8") as f:
    json.dump(spec, f, indent=2)

print(json.dumps({"success": True, "path": %q, "size": len(json.dumps(spec))}))
`, strconv.Quote(string(deckJSON)), targetPath, targetPath)

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

	log.Debug().Str("target_path", targetPath).Int("deck_size", len(deckJSON)).Msg("[slide_generator] uploading slide spec")
	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil || (result != nil && result.IsError) {
		log.Error().Err(err).Bool("is_error", result != nil && result.IsError).Msg("[slide_generator] failed to upload slide spec")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "UPLOAD_ERROR",
				Message:  "Failed to upload slide spec to sandbox",
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	log.Debug().Str("target_path", targetPath).Int("size", len(deckJSON)).Msg("[slide_generator] slide spec uploaded successfully")

	output := map[string]interface{}{
		"type": "file_uploaded",
		"path": targetPath,
		"size": len(deckJSON),
		"json": string(deckJSON), // Include the DeckSpec JSON for the next step
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Msg("[slide_generator] executeUploadSlideSpec completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeRenderScript(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Info().Bool("renderer_enabled", e.rendererEnabled).Msg("[slide_generator] executeRenderScript started")

	if !e.rendererEnabled {
		log.Warn().Msg("[slide_generator] slide renderer is disabled")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "RENDER_DISABLED",
				Message:  "slide renderer is disabled",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Check if AIO client is available
	if e.aioClient == nil {
		log.Error().Msg("[slide_generator] AIO sandbox client not initialized (AIO_URL not configured)")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "AIO_NOT_CONFIGURED",
				Message:  "AIO_URL environment variable not set",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	renderScriptContent := e.loadRenderDeckScript()
	if strings.TrimSpace(renderScriptContent) == "" {
		log.Error().Msg("[slide_generator] render_deck.py script is empty")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "RENDER_SCRIPT_MISSING",
				Message:  "render_deck.py content is empty",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Get DeckSpec JSON from previous step
	var deckSpecJSON string
	if input.PreviousOutput != nil {
		var prevOutput map[string]interface{}
		if err := json.Unmarshal(input.PreviousOutput, &prevOutput); err == nil {
			// Try to get the JSON directly
			if jsonStr, ok := prevOutput["json"].(string); ok {
				deckSpecJSON = jsonStr
			} else if data, ok := prevOutput["data"]; ok {
				// Try to marshal the data object
				if jsonBytes, err := json.Marshal(data); err == nil {
					deckSpecJSON = string(jsonBytes)
				}
			}
		}
	}

	if deckSpecJSON == "" {
		log.Error().Msg("[slide_generator] No DeckSpec JSON from previous step")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_INPUT",
				Message:  "No DeckSpec JSON from previous step",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Get output path from params or use response ID
	outputPath := "/home/gem/output.pptx"
	if params != nil {
		if path, ok := params["output_path"].(string); ok && path != "" {
			outputPath = path
		}
	}

	log.Info().
		Int("deck_spec_size", len(deckSpecJSON)).
		Int("render_script_size", len(renderScriptContent)).
		Str("output_path", outputPath).
		Msg("[slide_generator] Starting PPTX rendering via direct AIO sandbox")

	// Use direct AIO sandbox client (bypasses unstable MCP layer)
	// This executes everything in a single sandbox call with clear step-by-step logging
	pptxData, err := e.aioClient.RenderSlidesPPTX(ctx, deckSpecJSON, renderScriptContent, outputPath)
	if err != nil {
		log.Error().Err(err).Msg("[slide_generator] PPTX rendering failed")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "RENDER_ERROR",
				Message:  fmt.Sprintf("Render failed: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	// Encode to base64 for artifact storage
	base64Content := base64.StdEncoding.EncodeToString(pptxData)

	log.Info().
		Int("pptx_size", len(pptxData)).
		Int("base64_length", len(base64Content)).
		Msg("[slide_generator] ✓ PPTX rendering completed successfully")

	output := map[string]interface{}{
		"type":      "render_output",
		"base64":    base64Content,
		"filename":  "presentation.pptx",
		"mime_type": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"size":      len(pptxData),
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func truncateForLogString(data string, maxLen int) string {
	if data == "" {
		return ""
	}
	if len(data) <= maxLen {
		return data
	}
	return data[:maxLen] + "..."
}

func (e *SlideGeneratorExecutor) executeArtifactCreation(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Msg("[slide_generator] executeArtifactCreation started")
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to parse artifact parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  "failed to parse artifact parameters",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	config, _ := params["config"].(map[string]interface{})
	format, _ := config["format"].(string)
	if format == "" {
		format = "pptx"
	}
	artifactType, _ := params["artifact_type"].(string)
	isSlideArtifact := artifactType == "slides" || format == "pptx" || format == "pdf"

	if isSlideArtifact && input.PreviousOutput != nil {
		var skillOutput planners.SkillExecuteOutput
		if err := json.Unmarshal(input.PreviousOutput, &skillOutput); err == nil && skillOutput.Success {
			retentionPolicy, _ := config["retention_policy"].(string)
			if retentionPolicy == "" {
				retentionPolicy = "session"
			}
			return e.uploadSkillArtifact(ctx, step, input, skillOutput, artifactType, retentionPolicy)
		}
		if renderOutput := extractRenderOutput(input.PreviousOutput); renderOutput != nil {
			retentionPolicy, _ := config["retention_policy"].(string)
			if retentionPolicy == "" {
				retentionPolicy = "session"
			}
			return e.uploadRenderedArtifact(ctx, step, input, renderOutput, artifactType, retentionPolicy)
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

	retentionPolicy, _ := config["retention_policy"].(string)
	if retentionPolicy == "" {
		retentionPolicy = "session"
	}

	contentType := resolveArtifactContentType(artifactType, format)
	title := resolveArtifactTitle(artifactType)
	filename := resolveArtifactFilename(artifactType, format)

	var storagePath *string
	var downloadURL string
	var mediaID string

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
			log.Warn().Err(err).Str("response_id", responseID).Msg("failed to upload artifact to media-api, falling back to inline storage")
		} else {
			storagePath = &mediaArtifact.DownloadURL
			downloadURL = mediaArtifact.DownloadURL
			mediaID = mediaArtifact.ID
			log.Debug().
				Str("media_id", mediaID).
				Str("download_url", downloadURL).
				Str("response_id", responseID).
				Msg("artifact uploaded to media-api")
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

	// Use media URL if available, otherwise fall back to artifact API path
	if downloadURL == "" {
		downloadURL = fmt.Sprintf("/responses/v1/artifacts/%s/download", createdArtifact.ID)
	}
	log.Debug().
		Str("artifact_id", createdArtifact.ID).
		Str("download_url", downloadURL).
		Int64("size", int64(len(content))).
		Str("content_type", string(contentType)).
		Msg("[slide_generator] artifact created successfully")
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

func (e *SlideGeneratorExecutor) uploadSkillArtifact(ctx context.Context, step *plan.Step, input agent.ExecutionInput, skillOutput planners.SkillExecuteOutput, artifactType string, retentionPolicy string) (*agent.ExecutionResult, error) {
	log.Debug().
		Str("artifact_type", artifactType).
		Str("filename", skillOutput.FileName).
		Str("mime_type", skillOutput.MimeType).
		Msg("[slide_generator] uploadSkillArtifact started")
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
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	contentType := resolveArtifactContentType(artifactType, "")
	title := resolveArtifactTitle(artifactType)
	createdArtifact, err := e.artifactService.Create(ctx, artifact.CreateParams{
		ResponseID:      responseID,
		ContentType:     contentType,
		MimeType:        &mimeType,
		Title:           title,
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

	// Use media URL if available, otherwise fall back to artifact API path
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

func (e *SlideGeneratorExecutor) uploadRenderedArtifact(ctx context.Context, step *plan.Step, input agent.ExecutionInput, renderOutput *slideRenderOutput, artifactType string, retentionPolicy string) (*agent.ExecutionResult, error) {
	log.Debug().
		Str("artifact_type", artifactType).
		Str("filename", renderOutput.FileName).
		Int("size", renderOutput.Size).
		Msg("[slide_generator] uploadRenderedArtifact started")
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
	if renderOutput == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "FILE_MISSING",
				Message:  "no render output available for upload",
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	if strings.TrimSpace(renderOutput.Base64) == "" {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "FILE_MISSING",
				Message:  "render output missing base64 payload",
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(renderOutput.Base64)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "FILE_READ_ERROR",
				Message:  fmt.Sprintf("base64 decode failed: %v", err),
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

	fileName := strings.TrimSpace(renderOutput.FileName)
	if fileName == "" {
		fileName = resolveArtifactFilename(artifactType, "")
	}
	mimeType := strings.TrimSpace(renderOutput.MimeType)
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
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	contentType := resolveArtifactContentType(artifactType, "")
	title := resolveArtifactTitle(artifactType)
	createdArtifact, err := e.artifactService.Create(ctx, artifact.CreateParams{
		ResponseID:      responseID,
		ContentType:     contentType,
		MimeType:        &mimeType,
		Title:           title,
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

	// Use media URL if we uploaded to media-api
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

func (e *SlideGeneratorExecutor) readBinaryFileFromSandbox(ctx context.Context, path string, input agent.ExecutionInput) ([]byte, error) {
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
		return artifact.ContentTypeSlides
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
		return "Presentation"
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
			return "presentation.pdf"
		}
		return "presentation.pptx"
	}
}

func (e *SlideGeneratorExecutor) loadRenderDeckScript() string {
	if e.rendererScriptPath != "" {
		if payload, err := os.ReadFile(e.rendererScriptPath); err == nil {
			return string(payload)
		}
		log.Warn().
			Str("path", e.rendererScriptPath).
			Msg("Failed to read SLIDE_RENDERER_SCRIPT, falling back to embedded script")
	}
	return sliderenderer.RenderDeckScript
}

func cloneSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	payload, _ := json.Marshal(schema)
	var clone map[string]any
	_ = json.Unmarshal(payload, &clone)
	return clone
}

// extractJSONFromResponse extracts JSON from a response that may be wrapped in markdown code blocks
func extractJSONFromResponse(response string) string {
	response = strings.TrimSpace(response)

	// Remove markdown code block markers if present
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	}

	return strings.TrimSpace(response)
}

type slideRenderOutput struct {
	FileName string `json:"filename"`
	MimeType string `json:"mime_type"`
	Base64   string `json:"base64"`
	Size     int    `json:"size"`
}

func extractRenderOutput(previousOutput json.RawMessage) *slideRenderOutput {
	if len(previousOutput) == 0 {
		return nil
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(previousOutput, &parsed); err != nil {
		return nil
	}
	if parsed["type"] != "render_output" {
		return nil
	}
	payload, _ := json.Marshal(parsed)
	var output slideRenderOutput
	if err := json.Unmarshal(payload, &output); err != nil {
		return nil
	}
	return &output
}
