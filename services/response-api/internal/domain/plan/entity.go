// Package plan defines plan-related domain entities and services.
package plan

import (
	"bytes"
	"encoding/json"
	"time"

	"jan-server/services/response-api/internal/domain/status"
)

// Plan represents a multi-step execution plan created by an agent.
type Plan struct {
	ID              string        `json:"id"`
	ResponseID      string        `json:"response_id"`
	Model           string        `json:"model"` // Model to use for LLM steps
	Status          status.Status `json:"status"`
	Progress        float64       `json:"progress"` // 0-100 percentage
	AgentType       AgentType     `json:"agent_type"`
	PlanningConfig  PlanConfig    `json:"planning_config,omitempty"`
	EstimatedSteps  int           `json:"estimated_steps"`
	CompletedSteps  int           `json:"completed_steps"`
	CurrentTaskID   *string       `json:"current_task_id,omitempty"`
	FinalArtifactID *string       `json:"final_artifact_id,omitempty"`
	UserSelection   *string       `json:"user_selection,omitempty"` // JSON: user's choices
	ErrorMessage    *string       `json:"error_message,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`

	// Relations (loaded conditionally)
	Tasks []Task `json:"tasks,omitempty"`
}

// AgentType identifies the type of agent handling the plan.
type AgentType string

const (
	AgentTypeSlideCreator         AgentType = "slide_creator"
	AgentTypeDeepResearch         AgentType = "deep_research"
	AgentTypeDocGenerator         AgentType = "doc_generator"
	AgentTypePDFGenerator         AgentType = "pdf_generator"
	AgentTypeSpreadsheetGenerator AgentType = "spreadsheet_generator"
	AgentTypeCustom               AgentType = "custom"
)

// String returns the string representation of the agent type.
func (a AgentType) String() string {
	return string(a)
}

// PlanConfig contains configuration for plan execution.
type PlanConfig struct {
	MaxRetries        int           `json:"max_retries"`
	TimeoutPerStep    time.Duration `json:"timeout_per_step"`
	EnableFallback    bool          `json:"enable_fallback"`
	UserApproval      bool          `json:"user_approval"`
	StreamProgress    bool          `json:"stream_progress"`
	ArtifactRetention string        `json:"artifact_retention"` // "ephemeral", "session", "permanent"
}

// DefaultPlanConfig returns the default configuration for a plan.
func DefaultPlanConfig() PlanConfig {
	return PlanConfig{
		MaxRetries:        3,
		TimeoutPerStep:    5 * time.Minute,
		EnableFallback:    true,
		UserApproval:      false,
		StreamProgress:    true,
		ArtifactRetention: "session",
	}
}

// Task represents a group of related steps in a plan.
type Task struct {
	ID           string        `json:"id"`
	PlanID       string        `json:"plan_id"`
	Sequence     int           `json:"sequence"`
	TaskType     TaskType      `json:"task_type"`
	Status       status.Status `json:"status"`
	Title        string        `json:"title"`
	Description  *string       `json:"description,omitempty"`
	ErrorMessage *string       `json:"error_message,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`

	// Relations (loaded conditionally)
	Steps []Step `json:"steps,omitempty"`
}

// TaskType identifies the category of task.
type TaskType string

const (
	TaskTypeResearch     TaskType = "research"
	TaskTypeGeneration   TaskType = "generation"
	TaskTypeValidation   TaskType = "validation"
	TaskTypeTransform    TaskType = "transform"
	TaskTypeUserInput    TaskType = "user_input"
	TaskTypeFinalization TaskType = "finalization"
	TaskTypeExecution    TaskType = "execution"
)

// String returns the string representation of the task type.
func (t TaskType) String() string {
	return string(t)
}

// Step represents an individual action in a task.
type Step struct {
	ID            string               `json:"id"`
	TaskID        string               `json:"task_id"`
	Sequence      int                  `json:"sequence"`
	Action        ActionType           `json:"action"`
	Status        status.Status        `json:"status"`
	Title         string               `json:"title"`
	Description   *string              `json:"description,omitempty"`
	PlannedParams json.RawMessage      `json:"planned_params,omitempty"` // Original planned parameters
	ActualParams  json.RawMessage      `json:"actual_params,omitempty"`  // Actual execution parameters (may differ if agent changed tool)
	InputParams   json.RawMessage      `json:"input_params,omitempty"`   // Deprecated: use PlannedParams
	OutputData    json.RawMessage      `json:"output_data,omitempty"`
	RetryCount    int                  `json:"retry_count"`
	MaxRetries    int                  `json:"max_retries"`
	ErrorMessage  *string              `json:"error_message,omitempty"`
	ErrorSeverity status.ErrorSeverity `json:"error_severity,omitempty"`
	DurationMs    *int64               `json:"duration_ms,omitempty"`
	StartedAt     *time.Time           `json:"started_at,omitempty"`
	CompletedAt   *time.Time           `json:"completed_at,omitempty"`

	// Relations (loaded conditionally)
	Details []StepDetail `json:"details,omitempty"`
}

// ActionType identifies the action performed by a step.
type ActionType string

const (
	ActionTypeLLMCall        ActionType = "llm_call"
	ActionTypeToolCall       ActionType = "tool_call"
	ActionTypeArtifactCreate ActionType = "artifact_create"
	ActionTypeArtifactUpdate ActionType = "artifact_update"
	ActionTypeUserPrompt     ActionType = "user_prompt"
	ActionTypeValidation     ActionType = "validation"
	ActionTypeTransform      ActionType = "transform"
	ActionTypeSkillExecute   ActionType = "skill_execute"
)

// String returns the string representation of the action type.
func (a ActionType) String() string {
	return string(a)
}

// StepDetail links a step to its execution artifacts.
type StepDetail struct {
	ID                 string          `json:"id"`
	StepID             string          `json:"step_id"`
	DetailType         DetailType      `json:"detail_type"`
	ConversationItemID *string         `json:"conversation_item_id,omitempty"`
	ToolCallID         *string         `json:"tool_call_id,omitempty"`
	ToolExecutionID    *string         `json:"tool_execution_id,omitempty"`
	ArtifactID         *string         `json:"artifact_id,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

// DetailType identifies the type of step detail.
type DetailType string

const (
	DetailTypeMessage      DetailType = "message"
	DetailTypeToolCall     DetailType = "tool_call"
	DetailTypeToolResult   DetailType = "tool_result"
	DetailTypeArtifact     DetailType = "artifact"
	DetailTypeUserResponse DetailType = "user_response"
)

// String returns the string representation of the detail type.
func (d DetailType) String() string {
	return string(d)
}

// PlanProgress holds aggregated progress information.
type PlanProgress struct {
	PlanID         string        `json:"plan_id"`
	Status         status.Status `json:"status"`
	Progress       float64       `json:"progress"`
	EstimatedSteps int           `json:"estimated_steps"`
	CompletedSteps int           `json:"completed_steps"`
	CurrentTask    *TaskProgress `json:"current_task,omitempty"`
	FailedSteps    int           `json:"failed_steps"`
}

// TaskProgress holds progress for a single task.
type TaskProgress struct {
	TaskID      string        `json:"task_id"`
	Title       string        `json:"title"`
	Status      status.Status `json:"status"`
	TotalSteps  int           `json:"total_steps"`
	CurrentStep int           `json:"current_step"`
}

// CanRetry checks if the step can be retried.
func (s *Step) CanRetry() bool {
	return s.RetryCount < s.MaxRetries && s.ErrorSeverity.IsRetryable()
}

// IncrementRetry increments the retry count.
func (s *Step) IncrementRetry() {
	s.RetryCount++
}

// UpdateProgress recalculates the plan's progress based on completed steps.
func (p *Plan) UpdateProgress() {
	if p.EstimatedSteps == 0 {
		p.Progress = 0
		return
	}
	p.Progress = float64(p.CompletedSteps) / float64(p.EstimatedSteps) * 100
	if p.Progress > 100 {
		p.Progress = 100
	}
}

// IsCompleted returns true if the plan is in a terminal completed state.
func (p *Plan) IsCompleted() bool {
	return p.Status == status.StatusCompleted
}

// IsFailed returns true if the plan is in a failed state.
func (p *Plan) IsFailed() bool {
	return p.Status == status.StatusFailed
}

// StepOutput represents the structured output of a step execution.
// Each step should have its own unique output, not shared across steps.
type StepOutput struct {
	Status     string          `json:"status"`
	Type       string          `json:"type,omitempty"` // "llm_response", "tool_result", etc.
	Content    string          `json:"content,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
	CallID     string          `json:"call_id,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	TokensUsed *int            `json:"tokens_used,omitempty"`
	Artifact   *MediaArtifact  `json:"artifact,omitempty"`
}

// MediaArtifact represents an artifact uploaded to media-api.
// These are accessible files with jan_file_* IDs and download URLs.
type MediaArtifact struct {
	ID           string               `json:"id"`   // jan_file_xxx format
	Type         string               `json:"type"` // code, image, document, etc.
	Filename     string               `json:"filename"`
	DownloadURL  string               `json:"download_url"` // Pre-signed or permanent URL
	Size         int64                `json:"size"`
	ContentType  string               `json:"content_type"` // MIME type
	SlidesImages []MediaArtifactImage `json:"slides_images,omitempty"`
}

// MediaArtifactImage represents a slide preview image.
type MediaArtifactImage struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Thumb string `json:"thumb,omitempty"`
}

// Citation represents a source citation from research steps.
type Citation struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Snippet   string `json:"snippet,omitempty"`
	FetchedAt string `json:"fetched_at,omitempty"`
}

// SetPlannedParams sets the planned parameters for a step.
// This should be called when creating the step from the plan.
func (s *Step) SetPlannedParams(params json.RawMessage) {
	s.PlannedParams = params
	// For backward compatibility, also set InputParams
	s.InputParams = params
}

// SetActualParams sets the actual execution parameters.
// This should be called when the agent decides to use different parameters.
func (s *Step) SetActualParams(params json.RawMessage) {
	s.ActualParams = params
}

// GetEffectiveParams returns the actual params if set, otherwise planned params.
func (s *Step) GetEffectiveParams() json.RawMessage {
	if len(s.ActualParams) > 0 && !isJSONNull(s.ActualParams) {
		return s.ActualParams
	}
	if len(s.PlannedParams) > 0 && !isJSONNull(s.PlannedParams) {
		return s.PlannedParams
	}
	return s.InputParams
}

func isJSONNull(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

// HasOutputData returns true if the step has output data set.
func (s *Step) HasOutputData() bool {
	return len(s.OutputData) > 0 && string(s.OutputData) != "null"
}

// SetOutputData sets the output data from a StepOutput struct.
func (s *Step) SetOutputData(output *StepOutput) error {
	if output == nil {
		return nil
	}
	data, err := json.Marshal(output)
	if err != nil {
		return err
	}
	s.OutputData = data
	return nil
}

// GetStepOutput parses and returns the step output.
func (s *Step) GetStepOutput() (*StepOutput, error) {
	if !s.HasOutputData() {
		return nil, nil
	}
	var output StepOutput
	if err := json.Unmarshal(s.OutputData, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

// ValidateCompletedStep checks that a completed step has all required data.
// Returns an error if the step is completed but missing output_data.
func (s *Step) ValidateCompletedStep() error {
	if s.Status == status.StatusCompleted && !s.HasOutputData() {
		return &ValidationError{
			StepID:  s.ID,
			Message: "completed step must have output_data",
		}
	}
	return nil
}

// ValidationError represents a validation error for plan/step data.
type ValidationError struct {
	StepID  string `json:"step_id,omitempty"`
	TaskID  string `json:"task_id,omitempty"`
	PlanID  string `json:"plan_id,omitempty"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	if e.StepID != "" {
		return "step " + e.StepID + ": " + e.Message
	}
	if e.TaskID != "" {
		return "task " + e.TaskID + ": " + e.Message
	}
	if e.PlanID != "" {
		return "plan " + e.PlanID + ": " + e.Message
	}
	return e.Message
}

// Validate validates the entire plan including all tasks and steps.
func (p *Plan) Validate() error {
	for _, task := range p.Tasks {
		for _, step := range task.Steps {
			if err := step.ValidateCompletedStep(); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExtractCitations extracts all citations from search tool results in the plan.
func (p *Plan) ExtractCitations() []Citation {
	citations := make([]Citation, 0)
	seen := make(map[string]bool)

	for _, task := range p.Tasks {
		for _, step := range task.Steps {
			stepCitations := step.extractCitationsFromOutput()
			for _, c := range stepCitations {
				if !seen[c.URL] {
					seen[c.URL] = true
					citations = append(citations, c)
				}
			}
		}
	}

	return citations
}

// extractCitationsFromOutput extracts citations from a step's output data.
func (s *Step) extractCitationsFromOutput() []Citation {
	citations := make([]Citation, 0)
	if !s.HasOutputData() {
		return citations
	}

	// Try to parse as StepOutput with result containing search results
	var output StepOutput
	if err := json.Unmarshal(s.OutputData, &output); err != nil {
		return citations
	}

	// Check if this is a search tool result
	if output.ToolName == "" || output.Result == nil {
		return citations
	}

	return append(citations, extractSearchCitations(output.Result)...)
}

func extractSearchCitations(raw json.RawMessage) []Citation {
	citations := make([]Citation, 0)

	// Try direct search result structure.
	var searchResult struct {
		Citations []string `json:"citations"`
		Results   []struct {
			Title     string `json:"title"`
			URL       string `json:"url"`
			SourceURL string `json:"source_url"`
			Snippet   string `json:"snippet"`
			FetchedAt string `json:"fetched_at"`
		} `json:"results"`
	}

	if err := json.Unmarshal(raw, &searchResult); err == nil {
		for _, r := range searchResult.Results {
			url := r.URL
			if url == "" {
				url = r.SourceURL
			}
			if url != "" {
				citations = append(citations, Citation{
					Title:     r.Title,
					URL:       url,
					Snippet:   r.Snippet,
					FetchedAt: r.FetchedAt,
				})
			}
		}
		if len(citations) > 0 {
			return citations
		}
	}

	// Fallback: MCP tool results embed JSON in content.text.
	var mcpResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &mcpResult); err != nil {
		return citations
	}

	for _, content := range mcpResult.Content {
		if content.Type != "text" || content.Text == "" {
			continue
		}
		var nested json.RawMessage
		if err := json.Unmarshal([]byte(content.Text), &nested); err != nil {
			continue
		}
		citations = append(citations, extractSearchCitations(nested)...)
	}

	return citations
}

// ExtractArtifacts extracts all artifacts from step outputs in the plan.
func (p *Plan) ExtractArtifacts() []MediaArtifact {
	artifacts := make([]MediaArtifact, 0)
	seen := make(map[string]bool)

	for _, task := range p.Tasks {
		for _, step := range task.Steps {
			output, err := step.GetStepOutput()
			if err != nil || output == nil || output.Artifact == nil {
				continue
			}
			if !seen[output.Artifact.ID] {
				seen[output.Artifact.ID] = true
				artifacts = append(artifacts, *output.Artifact)
			}
		}
	}

	return artifacts
}
