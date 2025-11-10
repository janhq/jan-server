package dto

// ToolFunctionDefinition describes a function passed to OpenAI compatible APIs.
type ToolFunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolDefinition describes a tool in the HTTP contract.
type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

// ToolChoice allows callers to force or disable tools.
type ToolChoice struct {
	Type     string `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

// CreateResponseRequest models POST /v1/responses input.
type CreateResponseRequest struct {
	Model              string                 `json:"model" binding:"required"`
	Input              interface{}            `json:"input" binding:"required"`
	SystemPrompt       *string                `json:"system_prompt,omitempty"`
	MaxTokens          *int                   `json:"max_tokens,omitempty"`
	Temperature        *float64               `json:"temperature,omitempty"`
	Tools              []ToolDefinition       `json:"tools,omitempty"`
	ToolChoice         *ToolChoice            `json:"tool_choice,omitempty"`
	Stream             *bool                  `json:"stream,omitempty"`
	PreviousResponseID *string                `json:"previous_response_id,omitempty"`
	Conversation       *string                `json:"conversation,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	User               string                 `json:"user,omitempty"`
}
