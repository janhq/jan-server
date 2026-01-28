package prompt

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	openai "github.com/sashabaranov/go-openai"

	"jan-server/services/llm-api/internal/domain/document"
)

const projectFilesModuleName = "project_files"

// ProjectFilesModule injects project file contents into the conversation context.
// This allows AI to reference documents that have been attached to a project.
type ProjectFilesModule struct {
	projectFileService *document.ProjectFileService
	maxChars           int // Maximum characters to include from all files
}

// NewProjectFilesModule creates a new project files module.
func NewProjectFilesModule(projectFileService *document.ProjectFileService) *ProjectFilesModule {
	return &ProjectFilesModule{
		projectFileService: projectFileService,
		maxChars:           100000, // Default 100K chars max
	}
}

// NewProjectFilesModuleWithLimit creates a new project files module with a custom character limit.
func NewProjectFilesModuleWithLimit(projectFileService *document.ProjectFileService, maxChars int) *ProjectFilesModule {
	if maxChars <= 0 {
		maxChars = 100000
	}
	return &ProjectFilesModule{
		projectFileService: projectFileService,
		maxChars:           maxChars,
	}
}

// Name returns the module identifier.
func (m *ProjectFilesModule) Name() string {
	return projectFilesModuleName
}

// ShouldApply determines if project files should be injected.
func (m *ProjectFilesModule) ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	if promptCtx == nil {
		return false
	}
	if promptCtx.Preferences != nil && isModuleDisabled(promptCtx.Preferences, m.Name()) {
		return false
	}
	// Only apply if there's a project with an internal ID
	return promptCtx.ProjectID != nil && *promptCtx.ProjectID > 0
}

// Apply prepends project file contents as a system message.
func (m *ProjectFilesModule) Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messages, err
		}
	}
	if promptCtx == nil || promptCtx.ProjectID == nil || *promptCtx.ProjectID == 0 {
		return messages, nil
	}

	// Fetch project files with their extracted content
	files, err := m.projectFileService.GetProjectFilesWithContent(ctx, *promptCtx.ProjectID)
	if err != nil {
		log.Warn().Err(err).Uint("project_id", *promptCtx.ProjectID).Msg("ProjectFilesModule: Failed to fetch project files")
		return messages, nil // Don't fail the request, just skip files
	}

	if len(files) == 0 {
		return messages, nil
	}

	// Build the project files context
	var sb strings.Builder
	sb.WriteString("<project_files>\n")
	sb.WriteString("The following files have been attached to this project. Reference them when relevant to the conversation.\n\n")

	totalChars := 0
	filesIncluded := 0

	for _, f := range files {
		if f.DocumentContent == nil {
			continue
		}
		if f.DocumentContent.ProcessingStatus != document.ProcessingStatusCompleted {
			continue
		}
		if strings.TrimSpace(f.DocumentContent.ExtractedText) == "" {
			continue
		}

		text := f.DocumentContent.ExtractedText
		// Truncate if necessary
		if totalChars+len(text) > m.maxChars {
			remaining := m.maxChars - totalChars
			if remaining > 0 {
				text = text[:remaining] + "\n[Content truncated due to length]"
			} else {
				sb.WriteString(fmt.Sprintf("\n[File %s omitted due to context length limit]\n", f.DocumentContent.Filename))
				continue
			}
		}

		sb.WriteString(fmt.Sprintf("<file name=\"%s\"", f.DocumentContent.Filename))
		if f.DocumentContent.PageCount != nil && *f.DocumentContent.PageCount > 0 {
			sb.WriteString(fmt.Sprintf(" pages=\"%d\"", *f.DocumentContent.PageCount))
		}
		if f.DocumentContent.WordCount != nil && *f.DocumentContent.WordCount > 0 {
			sb.WriteString(fmt.Sprintf(" words=\"%d\"", *f.DocumentContent.WordCount))
		}
		sb.WriteString(">\n")
		sb.WriteString(text)
		sb.WriteString("\n</file>\n\n")

		totalChars += len(text)
		filesIncluded++
	}

	sb.WriteString("</project_files>")

	if filesIncluded == 0 {
		return messages, nil
	}

	log.Debug().
		Uint("project_id", *promptCtx.ProjectID).
		Int("files_included", filesIncluded).
		Int("total_chars", totalChars).
		Msg("ProjectFilesModule: Injecting project files context")

	// Insert project files context after project instructions (if present)
	// Use a priority slightly lower than project instructions
	return prependInstructionSystemMessage(messages, sb.String(), projectFilesModuleName), nil
}
