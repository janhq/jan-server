package steps

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
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
	return e.uploadFileArtifact(ctx, input, pptxBytes, filename, artifact.ContentTypeSlides, artifact.ContentTypeSlides.MimeTypeFor(), "Presentation", retentionPolicy)
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

func (e *SlideCreatorExecutor) uploadFileArtifact(ctx context.Context, input agent.ExecutionInput, content []byte, filename string, contentType artifact.ContentType, mimeType string, title string, retentionPolicy string) (*agent.ExecutionResult, error) {
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
			ID:          createdArtifact.ID,
			Type:        string(contentType),
			Filename:    filename,
			DownloadURL: downloadURL,
			Size:        int64(len(content)),
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
