package steps

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/media"

	"github.com/rs/zerolog/log"
)

func (e *SlideCreatorExecutor) executeArtifactCreation(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(input.StepParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_creator] failed to parse artifact parameters")
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
	config, _ := params["config"].(map[string]interface{})
	retentionPolicy := strings.TrimSpace(stringValue(config, "retention_policy"))
	if retentionPolicy == "" {
		retentionPolicy = "session"
	}

	switch action {
	case "store_html_artifact":
		return e.storeHTMLArtifact(ctx, input, retentionPolicy)
	case "store_pptx_artifact":
		return e.storePPTXArtifact(ctx, input, retentionPolicy)
	case "store_slide_images":
		return e.storeSlideImagesArtifact(ctx, input, retentionPolicy)
	case "store_artifact":
		return e.storeGenericArtifact(ctx, input, params, retentionPolicy)
	default:
		return &agent.ExecutionResult{Status: status.StatusCompleted}, nil
	}
}

func (e *SlideCreatorExecutor) storeHTMLArtifact(ctx context.Context, input agent.ExecutionInput, retentionPolicy string) (*agent.ExecutionResult, error) {
	outDir := extractOutputDir(input)
	if outDir == "" {
		outDir = outputDirForPlan(input)
	}
	files, err := collectHTMLBundleFiles(outDir)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "HTML_BUNDLE_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	zipPath := filepath.Join(outDir, "slides-html.zip")
	if err := zipFiles(zipPath, files, outDir); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "ZIP_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "ZIP_READ_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	return e.uploadFileArtifact(ctx, input, zipBytes, "slides-html.zip", artifact.ContentTypeHTML, "application/zip", "Slide HTML Bundle", retentionPolicy)
}

func (e *SlideCreatorExecutor) storePPTXArtifact(ctx context.Context, input agent.ExecutionInput, retentionPolicy string) (*agent.ExecutionResult, error) {
	outDir := extractOutputDir(input)
	if outDir == "" {
		outDir = outputDirForPlan(input)
	}

	pptxPath := extractPPTXPath(input)
	if pptxPath == "" {
		matches, _ := filepath.Glob(filepath.Join(outDir, "*.pptx"))
		sort.Strings(matches)
		if len(matches) > 0 {
			pptxPath = matches[0]
		}
	}
	if pptxPath == "" {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "PPTX_MISSING", Message: "pptx file not found in output directory", Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	pptxBytes, err := os.ReadFile(pptxPath)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "PPTX_READ_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	filename := filepath.Base(pptxPath)
	slideImages := e.uploadSlidePreviewImages(ctx, input, collectSlideImages(outDir))
	return e.uploadFileArtifact(ctx, input, pptxBytes, filename, artifact.ContentTypeSlides, artifact.ContentTypeSlides.MimeTypeFor(), "Presentation", retentionPolicy, slideImages...)
}

func (e *SlideCreatorExecutor) storeSlideImagesArtifact(ctx context.Context, input agent.ExecutionInput, retentionPolicy string) (*agent.ExecutionResult, error) {
	outDir := extractOutputDir(input)
	if outDir == "" {
		outDir = outputDirForPlan(input)
	}

	images := collectSlideImages(outDir)
	if len(images) == 0 {
		stepOutput := &plan.StepOutput{
			Status:    "skipped",
			Type:      "artifact_create",
			Content:   "no slide images found",
			CreatedAt: time.Now(),
		}
		outputBytes, _ := json.Marshal(stepOutput)
		return &agent.ExecutionResult{Status: status.StatusCompleted, Output: outputBytes}, nil
	}

	zipPath := filepath.Join(outDir, "slide-images.zip")
	if err := zipFiles(zipPath, images, outDir); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "ZIP_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "ZIP_READ_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	return e.uploadFileArtifact(ctx, input, zipBytes, "slide-images.zip", artifact.ContentTypeImage, "application/zip", "Slide Images", retentionPolicy)
}

func (e *SlideCreatorExecutor) uploadFileArtifact(ctx context.Context, input agent.ExecutionInput, content []byte, filename string, contentType artifact.ContentType, mimeType string, title string, retentionPolicy string, slidesImages ...plan.MediaArtifactImage) (*agent.ExecutionResult, error) {
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

	responseID := ""
	conversationID := ""
	userID := ""
	if input.PlanContext != nil {
		responseID = input.PlanContext.ResponseID
		conversationID = input.PlanContext.ConversationID
		userID = input.PlanContext.UserID
	}

	mediaArtifact, err := e.mediaClient.UploadArtifact(ctx, &media.UploadRequest{
		Content:        content,
		Filename:       filename,
		ContentType:    mimeType,
		ConversationID: conversationID,
		ResponseID:     responseID,
		UserID:         userID,
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "UPLOAD_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	createdArtifact, err := e.artifactService.Create(ctx, artifact.CreateParams{
		ResponseID:      responseID,
		ContentType:     contentType,
		MimeType:        &mimeType,
		Title:           title,
		StoragePath:     &mediaArtifact.DownloadURL,
		SizeBytes:       int64(len(content)),
		RetentionPolicy: artifact.RetentionPolicy(retentionPolicy),
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "ARTIFACT_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
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
			ID:           createdArtifact.ID,
			Type:         string(contentType),
			Filename:     filename,
			DownloadURL:  downloadURL,
			Size:         int64(len(content)),
			ContentType:  mimeType,
			SlidesImages: slidesImages,
		},
	}
	outputBytes, _ := json.Marshal(stepOutput)

	return &agent.ExecutionResult{
		Status:     status.StatusCompleted,
		Output:     outputBytes,
		ArtifactID: &createdArtifact.ID,
	}, nil
}

func (e *SlideCreatorExecutor) uploadSlidePreviewImages(ctx context.Context, input agent.ExecutionInput, imagePaths []string) []plan.MediaArtifactImage {
	if e.mediaClient == nil || len(imagePaths) == 0 {
		return nil
	}

	responseID := ""
	conversationID := ""
	userID := ""
	if input.PlanContext != nil {
		responseID = input.PlanContext.ResponseID
		conversationID = input.PlanContext.ConversationID
		userID = input.PlanContext.UserID
	}

	results := make([]plan.MediaArtifactImage, 0, len(imagePaths))
	for _, path := range imagePaths {
		content, err := os.ReadFile(path)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("[slide_creator] failed to read slide preview")
			continue
		}
		filename := filepath.Base(path)
		mediaArtifact, err := e.mediaClient.UploadArtifact(ctx, &media.UploadRequest{
			Content:        content,
			Filename:       filename,
			ContentType:    "",
			ConversationID: conversationID,
			ResponseID:     responseID,
			UserID:         userID,
		})
		if err != nil {
			log.Warn().Err(err).Str("filename", filename).Msg("[slide_creator] failed to upload slide preview")
			continue
		}
		id := strings.TrimSuffix(filename, filepath.Ext(filename))
		url := mediaArtifact.DownloadURL
		results = append(results, plan.MediaArtifactImage{
			ID:    id,
			URL:   url,
			Thumb: url,
		})
	}
	return results
}

func collectHTMLBundleFiles(outDir string) ([]string, error) {
	files := []string{}

	index := filepath.Join(outDir, "index.html")
	if fileExistsOnDisk(index) {
		files = append(files, index)
	}

	slides, _ := filepath.Glob(filepath.Join(outDir, "slide-*.html"))
	sort.Strings(slides)
	files = append(files, slides...)

	for _, name := range []string{"plan.json", "charts.json", "images.json"} {
		path := filepath.Join(outDir, name)
		if fileExistsOnDisk(path) {
			files = append(files, path)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no HTML outputs found in %s", outDir)
	}

	return files, nil
}

func collectSlideImages(outDir string) []string {
	all := []string{}
	for _, pattern := range []string{"slide-*.png", "slide-*.jpg", "slide-*.jpeg"} {
		matches, _ := filepath.Glob(filepath.Join(outDir, pattern))
		all = append(all, matches...)
	}
	sort.Strings(all)
	return all
}

func zipFiles(zipPath string, files []string, baseDir string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to zip")
	}

	file, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	writer := zip.NewWriter(file)

	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		header.Method = zip.Deflate

		entry, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(entry, src); err != nil {
			_ = src.Close()
			return err
		}
		_ = src.Close()
	}

	return writer.Close()
}

func fileExistsOnDisk(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func extractPPTXPath(input agent.ExecutionInput) string {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}
	for i := len(outputs) - 1; i >= 0; i-- {
		var payload map[string]any
		if err := json.Unmarshal(outputs[i], &payload); err != nil {
			continue
		}
		if path, ok := payload["pptx_path"].(string); ok && strings.TrimSpace(path) != "" {
			return path
		}
	}
	return ""
}

func (e *SlideCreatorExecutor) storeGenericArtifact(ctx context.Context, input agent.ExecutionInput, params map[string]interface{}, retentionPolicy string) (*agent.ExecutionResult, error) {
	config, _ := params["config"].(map[string]interface{})
	format, _ := config["format"].(string)
	if format == "" {
		format = "pptx"
	}
	artifactType, _ := params["artifact_type"].(string)

	if input.PreviousOutput != nil {
		var skillOutput planners.SkillExecuteOutput
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
			log.Warn().Err(err).Str("response_id", responseID).Msg("[slide_creator] failed to upload artifact to media-api, falling back to inline storage")
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

func (e *SlideCreatorExecutor) uploadSkillArtifact(ctx context.Context, input agent.ExecutionInput, skillOutput planners.SkillExecuteOutput, artifactType string, retentionPolicy string) (*agent.ExecutionResult, error) {
	log.Debug().
		Str("artifact_type", artifactType).
		Str("filename", skillOutput.FileName).
		Str("mime_type", skillOutput.MimeType).
		Msg("[slide_creator] uploadSkillArtifact started")
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

func (e *SlideCreatorExecutor) readBinaryFileFromSandbox(ctx context.Context, path string, input agent.ExecutionInput) ([]byte, error) {
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

func firstTextContent(content []tool.MCPContent) string {
	for _, item := range content {
		if strings.TrimSpace(item.Text) != "" {
			return item.Text
		}
	}
	return ""
}
