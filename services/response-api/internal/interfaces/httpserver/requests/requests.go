package requests

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
	Type     string                 `json:"type"`
	Tool     string                 `json:"tool,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Function struct {
		Name string `json:"name,omitempty"`
	} `json:"function,omitempty"`
}

// CreateResponseRequest models POST /v1/responses input.
type CreateResponseRequest struct {
	Model                string                 `json:"model" binding:"required"`
	Input                interface{}            `json:"input" binding:"required"`
	SystemPrompt         *string                `json:"system_prompt,omitempty"`
	MaxTokens            *int                   `json:"max_tokens,omitempty"`
	Temperature          *float64               `json:"temperature,omitempty"`
	Tools                []ToolDefinition       `json:"tools,omitempty"`
	ToolChoice           *ToolChoice            `json:"tool_choice,omitempty"`
	Stream               *bool                  `json:"stream,omitempty"`
	Background           *bool                  `json:"background,omitempty"`
	Store                *bool                  `json:"store,omitempty"`
	Data                 *bool                  `json:"data,omitempty"` // Include all conversation items (tool calls, results) in response
	PreviousResponseID   *string                `json:"previous_response_id,omitempty"`
	Conversation         *string                `json:"conversation,omitempty"`
	ParentConversationID *string                `json:"parent_conversation_id,omitempty"` // llm-api conversation ID (for artifact linking)
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	User                 string                 `json:"user,omitempty"`
}
