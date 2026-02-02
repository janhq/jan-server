package responses

import (
	"encoding/json"
	"fmt"
	"time"

	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"
)

// PlanResponse represents a plan in API responses.
type PlanResponse struct {
	ID              string  `json:"id"`
	Object          string  `json:"object"`
	ResponseID      string  `json:"response_id"`
	Status          string  `json:"status"`
	Progress        float64 `json:"progress"`
	AgentType       string  `json:"agent_type"`
	EstimatedSteps  int     `json:"estimated_steps"`
	CompletedSteps  int     `json:"completed_steps"`
	CurrentTaskID   *string `json:"current_task_id,omitempty"`
	FinalArtifactID *string `json:"final_artifact_id,omitempty"`
	Error           *string `json:"error,omitempty"`
	CreatedAt       int64   `json:"created_at"`
	UpdatedAt       int64   `json:"updated_at"`
	CompletedAt     *int64  `json:"completed_at,omitempty"`
}

// PlanDetailResponse represents a plan with full details.
type PlanDetailResponse struct {
	PlanResponse
	Tasks []TaskResponse `json:"tasks"`
}

// PlanProgressResponse represents plan progress.
type PlanProgressResponse struct {
	PlanID         string                `json:"plan_id"`
	Status         string                `json:"status"`
	Progress       float64               `json:"progress"`
	EstimatedSteps int                   `json:"estimated_steps"`
	CompletedSteps int                   `json:"completed_steps"`
	FailedSteps    int                   `json:"failed_steps"`
	CurrentTask    *TaskProgressResponse `json:"current_task,omitempty"`
}

// TaskProgressResponse represents task progress.
type TaskProgressResponse struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// TaskResponse represents a task in API responses.
type TaskResponse struct {
	ID          string         `json:"id"`
	Object      string         `json:"object"`
	PlanID      string         `json:"plan_id"`
	Sequence    int            `json:"sequence"`
	TaskType    string         `json:"task_type"`
	Status      string         `json:"status"`
	Title       string         `json:"title"`
	Description *string        `json:"description,omitempty"`
	Error       *string        `json:"error,omitempty"`
	Steps       []StepResponse `json:"steps,omitempty"`
	CreatedAt   int64          `json:"created_at"`
	UpdatedAt   int64          `json:"updated_at"`
	CompletedAt *int64         `json:"completed_at,omitempty"`
}

// StepResponse represents a step in API responses.
type StepResponse struct {
	ID            string          `json:"id"`
	Object        string          `json:"object"`
	TaskID        string          `json:"task_id"`
	Sequence      int             `json:"sequence"`
	Action        string          `json:"action"`
	Status        string          `json:"status"`
	Title         string          `json:"title,omitempty"`
	Description   *string         `json:"description,omitempty"`
	RetryCount    int             `json:"retry_count"`
	MaxRetries    int             `json:"max_retries"`
	Error         *string         `json:"error,omitempty"`
	ErrorSeverity *string         `json:"error_severity,omitempty"`
	DurationMs    *int64          `json:"duration_ms,omitempty"`
	PlannedParams json.RawMessage `json:"planned_params,omitempty"` // Original planned parameters
	ActualParams  json.RawMessage `json:"actual_params,omitempty"`  // Actual execution parameters
	InputParams   json.RawMessage `json:"input_params,omitempty"`   // Deprecated: use PlannedParams
	OutputData    json.RawMessage `json:"output_data,omitempty"`
	StartedAt     *int64          `json:"started_at,omitempty"`
	CompletedAt   *int64          `json:"completed_at,omitempty"`
}

// ArtifactResponse represents an artifact in API responses.
type ArtifactResponse struct {
	ID              string          `json:"id"`
	Object          string          `json:"object"`
	ResponseID      string          `json:"response_id"`
	ConversationID  *string         `json:"conversation_id,omitempty"` // Thread ID for navigation
	PlanID          *string         `json:"plan_id,omitempty"`
	ContentType     string          `json:"content_type"`
	MimeType        string          `json:"mime_type"`
	Title           string          `json:"title"`
	Content         *string         `json:"content,omitempty"`      // Inline content (for markdown, code, etc.)
	StoragePath     *string         `json:"storage_path,omitempty"` // For file-based content
	SizeBytes       int64           `json:"size_bytes"`
	Version         int             `json:"version"`
	ParentID        *string         `json:"parent_id,omitempty"`
	IsLatest        bool            `json:"is_latest"`
	RetentionPolicy string          `json:"retention_policy"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       int64           `json:"created_at"`
	UpdatedAt       int64           `json:"updated_at"`
	ExpiresAt       *int64          `json:"expires_at,omitempty"`
}

// ArtifactListResponse represents a paginated list of artifacts (cursor-based).
type ArtifactListResponse struct {
	Object  string             `json:"object"`
	Data    []ArtifactResponse `json:"data"`
	FirstID string             `json:"first_id"`
	LastID  string             `json:"last_id"`
	HasMore bool               `json:"has_more"`
	Total   int64              `json:"total"`
}

// Mapping functions

// MapPlanToResponse converts a domain plan to an API response.
func MapPlanToResponse(p *plan.Plan) PlanResponse {
	resp := PlanResponse{
		ID:              p.ID,
		Object:          "plan",
		ResponseID:      p.ResponseID,
		Status:          string(p.Status),
		Progress:        p.Progress,
		AgentType:       string(p.AgentType),
		EstimatedSteps:  p.EstimatedSteps,
		CompletedSteps:  p.CompletedSteps,
		CurrentTaskID:   p.CurrentTaskID,
		FinalArtifactID: p.FinalArtifactID,
		Error:           p.ErrorMessage,
		CreatedAt:       p.CreatedAt.Unix(),
		UpdatedAt:       p.UpdatedAt.Unix(),
	}

	if p.CompletedAt != nil {
		ts := p.CompletedAt.Unix()
		resp.CompletedAt = &ts
	}

	return resp
}

// MapPlanDetailToResponse converts a domain plan with details to an API response.
func MapPlanDetailToResponse(p *plan.Plan) PlanDetailResponse {
	resp := PlanDetailResponse{
		PlanResponse: MapPlanToResponse(p),
		Tasks:        make([]TaskResponse, 0, len(p.Tasks)),
	}

	for _, task := range p.Tasks {
		resp.Tasks = append(resp.Tasks, MapTaskToResponse(&task))
	}

	return resp
}

// MapPlanDetailFromInterface converts an interface{} (expected to be *plan.Plan) to PlanDetailResponse.
// This is used for the full response endpoint where plan details come from the response service.
func MapPlanDetailFromInterface(p interface{}) *PlanDetailResponse {
	if p == nil {
		return nil
	}
	planPtr, ok := p.(*plan.Plan)
	if !ok {
		return nil
	}
	resp := MapPlanDetailToResponse(planPtr)
	return &resp
}

// MapPlanProgressToResponse converts a domain plan progress to an API response.
func MapPlanProgressToResponse(p *plan.PlanProgress) PlanProgressResponse {
	resp := PlanProgressResponse{
		PlanID:         p.PlanID,
		Status:         string(p.Status),
		Progress:       p.Progress,
		EstimatedSteps: p.EstimatedSteps,
		CompletedSteps: p.CompletedSteps,
		FailedSteps:    p.FailedSteps,
	}

	if p.CurrentTask != nil {
		resp.CurrentTask = &TaskProgressResponse{
			TaskID: p.CurrentTask.TaskID,
			Title:  p.CurrentTask.Title,
			Status: string(p.CurrentTask.Status),
		}
	}

	return resp
}

// MapTaskToResponse converts a domain task to an API response.
func MapTaskToResponse(t *plan.Task) TaskResponse {
	resp := TaskResponse{
		ID:          t.ID,
		Object:      "plan_task",
		PlanID:      t.PlanID,
		Sequence:    t.Sequence,
		TaskType:    string(t.TaskType),
		Status:      string(t.Status),
		Title:       t.Title,
		Description: t.Description,
		Error:       t.ErrorMessage,
		CreatedAt:   t.CreatedAt.Unix(),
		UpdatedAt:   t.UpdatedAt.Unix(),
	}

	if t.CompletedAt != nil {
		ts := t.CompletedAt.Unix()
		resp.CompletedAt = &ts
	}

	if len(t.Steps) > 0 {
		resp.Steps = make([]StepResponse, 0, len(t.Steps))
		for _, step := range t.Steps {
			resp.Steps = append(resp.Steps, MapStepToResponse(&step))
		}
	}

	return resp
}

// MapStepToResponse converts a domain step to an API response.
func MapStepToResponse(s *plan.Step) StepResponse {
	resp := StepResponse{
		ID:            s.ID,
		Object:        "plan_step",
		TaskID:        s.TaskID,
		Sequence:      s.Sequence,
		Action:        string(s.Action),
		Status:        string(s.Status),
		Title:         s.Title,
		Description:   s.Description,
		RetryCount:    s.RetryCount,
		MaxRetries:    s.MaxRetries,
		Error:         s.ErrorMessage,
		DurationMs:    s.DurationMs,
		PlannedParams: sanitizePlannedParams(s.PlannedParams),
		ActualParams:  s.ActualParams,
		InputParams:   sanitizeInputParams(s.InputParams), // Deprecated, kept for compatibility
		OutputData:    sanitizeStepOutputData(s.OutputData),
	}

	if s.ErrorSeverity != "" {
		sev := string(s.ErrorSeverity)
		resp.ErrorSeverity = &sev
	}

	if s.StartedAt != nil {
		ts := s.StartedAt.Unix()
		resp.StartedAt = &ts
	}

	if s.CompletedAt != nil {
		ts := s.CompletedAt.Unix()
		resp.CompletedAt = &ts
	}

	return resp
}

// sanitizePlannedParams removes the large schema field from planned parameters.
// The schema field can be very large and is not typically used by the API consumers.
func sanitizePlannedParams(plannedParams json.RawMessage) json.RawMessage {
	if len(plannedParams) == 0 {
		return plannedParams
	}

	var params map[string]interface{}
	if err := json.Unmarshal(plannedParams, &params); err != nil {
		// Not a JSON object, return as-is
		return plannedParams
	}

	// Remove the schema field if it exists
	if _, exists := params["schema"]; exists {
		delete(params, "schema")

		// Re-marshal without the schema field
		sanitized, err := json.Marshal(params)
		if err != nil {
			// If marshaling fails, return original
			return plannedParams
		}
		return sanitized
	}

	// No schema field found, return original
	return plannedParams
}

// sanitizeInputParams removes the large schema field from input parameters.
// The schema field can be very large and is not typically used by the API consumers.
func sanitizeInputParams(inputParams json.RawMessage) json.RawMessage {
	if len(inputParams) == 0 {
		return inputParams
	}

	var params map[string]interface{}
	if err := json.Unmarshal(inputParams, &params); err != nil {
		// Not a JSON object, return as-is
		return inputParams
	}

	// Remove the schema field if it exists
	if _, exists := params["schema"]; exists {
		delete(params, "schema")

		// Re-marshal without the schema field
		sanitized, err := json.Marshal(params)
		if err != nil {
			// If marshaling fails, return original
			return inputParams
		}
		return sanitized
	}

	// No schema field found, return original
	return inputParams
}

// sanitizeStepOutputData removes large binary content (like base64) from step output data.
// This prevents bloated API responses while preserving meaningful metadata.
func sanitizeStepOutputData(outputData json.RawMessage) json.RawMessage {
	if len(outputData) == 0 {
		return outputData
	}

	var data map[string]interface{}
	if err := json.Unmarshal(outputData, &data); err != nil {
		// Not a JSON object, return as-is but truncate if too large
		if len(outputData) > 10000 {
			return json.RawMessage(`{"truncated": true, "message": "output too large"}`)
		}
		return outputData
	}

	// Fields to strip (large binary content)
	stripFields := []string{
		"file_content_base64",
		"content_base64",
		"base64",
		"data_url",
	}

	modified := false
	for _, field := range stripFields {
		if val, exists := data[field]; exists {
			if str, ok := val.(string); ok && len(str) > 1000 {
				// Replace with a placeholder
				data[field] = "[content stripped - use download URL]"
				modified = true
			}
		}
	}

	// Also check nested content array (for tool outputs)
	if content, ok := data["content"].([]interface{}); ok {
		for i, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if text, ok := itemMap["text"].(string); ok && len(text) > 50000 {
					itemMap["text"] = text[:1000] + "... [truncated, " + fmt.Sprintf("%d", len(text)) + " bytes total]"
					content[i] = itemMap
					modified = true
				}
			}
		}
		if modified {
			data["content"] = content
		}
	}

	if !modified {
		return outputData
	}

	sanitized, err := json.Marshal(data)
	if err != nil {
		return outputData
	}
	return sanitized
}

// MapArtifactToResponse converts a domain artifact to an API response.
func MapArtifactToResponse(a *artifact.Artifact) ArtifactResponse {
	resp := ArtifactResponse{
		ID:              a.ID,
		Object:          "artifact",
		ResponseID:      a.ResponseID,
		ConversationID:  a.ConversationID,
		PlanID:          a.PlanID,
		ContentType:     string(a.ContentType),
		MimeType:        a.MimeType,
		Title:           a.Title,
		Content:         a.Content,
		StoragePath:     a.StoragePath,
		SizeBytes:       a.SizeBytes,
		Version:         a.Version,
		ParentID:        a.ParentID,
		IsLatest:        a.IsLatest,
		RetentionPolicy: string(a.RetentionPolicy),
		Metadata:        a.Metadata,
		CreatedAt:       a.CreatedAt.Unix(),
		UpdatedAt:       a.UpdatedAt.Unix(),
	}

	if a.ExpiresAt != nil {
		ts := a.ExpiresAt.Unix()
		resp.ExpiresAt = &ts
	}

	return resp
}

// MapArtifactsToResponse converts a slice of domain artifacts to API responses.
func MapArtifactsToResponse(artifacts []*artifact.Artifact) []ArtifactResponse {
	responses := make([]ArtifactResponse, 0, len(artifacts))
	for _, a := range artifacts {
		responses = append(responses, MapArtifactToResponse(a))
	}
	return responses
}

// Helper to convert time.Time to unix timestamp pointer
func timeToUnixPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	ts := t.Unix()
	return &ts
}
